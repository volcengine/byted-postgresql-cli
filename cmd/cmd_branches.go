// Copyright (c) 2026 Beijing Volcano Engine Technology Co., Ltd.
// SPDX-License-Identifier: MIT
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in all
// copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
// SOFTWARE.

package cli

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/spf13/cobra"

	"github.com/volcengine/byted-postgresql-cli/internal/volcengine"
)

var (
	branchFields       = []string{"BranchName", "BranchId", "BranchStatus", "Default", "Protected", "Archived", "InitSource", "CreateTime", "UpdateTime"}
	branchDetailFields = []string{"BranchName", "BranchId", "WorkspaceId", "BranchStatus", "Default", "Protected", "Archived", "InitSource", "CreationSource", "StartParentTime", "CreateTime", "UpdateTime"}
)

// newBranchesCmd manages a workspace's branches. Every branch operation is
// scoped to one PostgreSQL workspace via --workspace-id (falling back to the
// the --workspace-id flag).
func newBranchesCmd(ctx ProviderContext) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "branches",
		Aliases: []string{"branch"},
		Short:   "Manage workspace branches",
	}

	var workspaceID string

	workspaceResolverFn := func(runCmd *cobra.Command) (string, error) {
		g := fromCtx(runCmd)
		client, err := g.NewVolcClient(runCmd.Context())
		if err != nil {
			return "", err
		}
		return resolveWorkspace(runCmd.Context(), g, client, workspaceID)
	}

	children := []*cobra.Command{
		newBranchesListCmd(workspaceResolverFn),
		newBranchesGetCmd(workspaceResolverFn),
		newBranchesDefaultCmd(workspaceResolverFn),
		newBranchesChildrenCmd(workspaceResolverFn),
		newBranchesCreateCmd(workspaceResolverFn),
		newBranchesUpdateCmd(workspaceResolverFn),
		newBranchesDeleteCmd(workspaceResolverFn),
		newBranchesRestartCmd(workspaceResolverFn),
		newBranchesSetDefaultCmd(workspaceResolverFn),
		newBranchesRestoreWindowCmd(workspaceResolverFn),
		newBranchesRestorableCmd(workspaceResolverFn),
		newBranchesRestoreCmd(workspaceResolverFn),
		newBranchesDiffCmd(workspaceResolverFn),
	}
	for _, child := range children {
		if child.Name() == "children" {
			if list, _, err := child.Find([]string{"list"}); err == nil && list != nil {
				list.Flags().StringVar(&workspaceID, "workspace-id", "", "Workspace ID (required in non-interactive mode)")
			}
		} else {
			child.Flags().StringVar(&workspaceID, "workspace-id", "", "Workspace ID (required in non-interactive mode)")
		}
		cmd.AddCommand(child)
	}
	return cmd
}

func newBranchesDiffCmd(resolve workspaceResolver) *cobra.Command {
	var sourceBranch, sourceDatabase, sourceSchema, targetBranch, targetDatabase, targetSchema string
	var force bool
	cmd := &cobra.Command{
		Use:   "diff --source-branch-id <id> --source-database-name <name> --source-schema-name <name> --target-branch-id <id> --target-database-name <name> --target-schema-name <name>",
		Short: "Compare schemas between two branches",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			g := fromCtx(cmd)
			client, err := g.NewVolcClient(cmd.Context())
			if err != nil {
				return err
			}
			wsID, err := resolve(cmd)
			if err != nil {
				return err
			}
			sourceBranch, err = selectDiffBranch(cmd, client, wsID, sourceBranch, "source branch", "source-branch-id")
			if err != nil {
				return err
			}
			sourceDatabase, err = selectDiffDatabase(cmd, client, wsID, sourceBranch, sourceDatabase, "source database", "source-database-name")
			if err != nil {
				return err
			}
			sourceSchema, err = selectDiffSchema(cmd, client, wsID, sourceBranch, sourceDatabase, sourceSchema, "source schema", "source-schema-name")
			if err != nil {
				return err
			}
			targetBranch, err = selectDiffBranch(cmd, client, wsID, targetBranch, "target branch", "target-branch-id")
			if err != nil {
				return err
			}
			targetDatabase, err = selectDiffDatabase(cmd, client, wsID, targetBranch, targetDatabase, "target database", "target-database-name")
			if err != nil {
				return err
			}
			targetSchema, err = selectDiffSchema(cmd, client, wsID, targetBranch, targetDatabase, targetSchema, "target schema", "target-schema-name")
			if err != nil {
				return err
			}
			jobID, err := client.CreateSchemaDiff(cmd.Context(), volcengine.CreateSchemaDiffParams{
				WorkspaceID: wsID, SourceBranchID: sourceBranch, SourceDatabaseName: sourceDatabase, SourceSchemaName: sourceSchema,
				TargetBranchID: targetBranch, TargetDatabaseName: targetDatabase, TargetSchemaName: targetSchema, ForceReCompare: force,
			})
			if err != nil {
				return err
			}
			return g.Writer().WriteItem(map[string]string{"SchemaDiffJobId": jobID}, []string{"SchemaDiffJobId"})
		},
	}
	cmd.Flags().StringVar(&sourceBranch, "source-branch-id", "", "Source branch ID (required)")
	cmd.Flags().StringVar(&sourceDatabase, "source-database-name", "", "Source database name (required)")
	cmd.Flags().StringVar(&sourceSchema, "source-schema-name", "", "Source schema name (required)")
	cmd.Flags().StringVar(&targetBranch, "target-branch-id", "", "Target branch ID (required)")
	cmd.Flags().StringVar(&targetDatabase, "target-database-name", "", "Target database name (required)")
	cmd.Flags().StringVar(&targetSchema, "target-schema-name", "", "Target schema name (required)")
	cmd.Flags().BoolVar(&force, "force-recompare", false, "Force a fresh schema comparison")
	return cmd
}

func selectDiffBranch(cmd *cobra.Command, client *volcengine.Client, workspaceID, explicit, title, flagName string) (string, error) {
	if strings.TrimSpace(explicit) != "" {
		return explicit, nil
	}
	if !interactiveTarget() {
		return "", fmt.Errorf("--%s is required in non-interactive mode", flagName)
	}
	result, err := client.DescribeBranches(cmd.Context(), volcengine.DescribeBranchesParams{WorkspaceID: workspaceID, Limit: volcengine.InteractiveListLimit})
	if err != nil {
		return "", err
	}
	items := make([]promptItem, 0, len(result.Branches))
	for _, branch := range result.Branches {
		items = append(items, promptItem{summary: branch.BranchID, details: branch.BranchName + ", " + branch.BranchStatus})
	}
	return promptList(cmd.Context(), "Select "+title, items)
}

func selectDiffDatabase(cmd *cobra.Command, client *volcengine.Client, workspaceID, branchID, explicit, title, flagName string) (string, error) {
	if strings.TrimSpace(explicit) != "" {
		return explicit, nil
	}
	result, err := client.DescribeDatabases(cmd.Context(), volcengine.DescribeDatabasesParams{WorkspaceID: workspaceID, BranchID: branchID, Limit: volcengine.InteractiveListLimit})
	if err != nil {
		return "", err
	}
	if len(result.Databases) == 0 {
		return "", fmt.Errorf("no databases found for %s", branchID)
	}
	if !interactiveTarget() {
		return "", fmt.Errorf("--%s is required in non-interactive mode", flagName)
	}
	items := make([]promptItem, 0, len(result.Databases))
	for _, database := range result.Databases {
		items = append(items, promptItem{summary: database.DatabaseName, details: database.DatabaseOwner})
	}
	return promptList(cmd.Context(), "Select "+title, items)
}

func selectDiffSchema(cmd *cobra.Command, client *volcengine.Client, workspaceID, branchID, database, explicit, title, flagName string) (string, error) {
	if strings.TrimSpace(explicit) != "" {
		return explicit, nil
	}
	names, err := client.DescribeBranchSchemaNames(cmd.Context(), workspaceID, branchID, database)
	if err != nil {
		return "", err
	}
	if len(names) == 0 {
		return "", fmt.Errorf("no schemas found for database %s", database)
	}
	if !interactiveTarget() {
		return "", fmt.Errorf("--%s is required in non-interactive mode", flagName)
	}
	items := make([]promptItem, 0, len(names))
	for _, name := range names {
		items = append(items, promptItem{summary: name})
	}
	return promptList(cmd.Context(), "Select "+title, items)
}

type workspaceResolver func(*cobra.Command) (string, error)

func newBranchesListCmd(resolve workspaceResolver) *cobra.Command {
	var (
		search string
		limit  int
		offset int
		all    bool
	)
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List branches in a workspace",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			g := fromCtx(cmd)
			wsID, err := resolve(cmd)
			if err != nil {
				return err
			}
			client, err := g.NewVolcClient(cmd.Context())
			if err != nil {
				return err
			}
			limit = requestedListLimit(cmd, limit)
			params := volcengine.DescribeBranchesParams{WorkspaceID: wsID, Search: search, Limit: limit, Offset: offset}
			var result volcengine.DescribeBranchesResult
			if all {
				result, err = client.DescribeAllBranches(cmd.Context(), params)
			} else {
				result, err = client.DescribeBranches(cmd.Context(), params)
			}
			if err != nil {
				return err
			}
			return writeListPage(cmd, g, result.Branches, branchFields, result.Total, limit, offset, "branches")
		},
	}
	cmd.Flags().StringVar(&search, "search", "", "Filter branches by name")
	cmd.Flags().IntVar(&limit, "limit", 0, "Maximum number of branches to return (default 10, 0=100, max 100)")
	cmd.Flags().IntVar(&offset, "offset", 0, "Number of branches to skip")
	cmd.Flags().BoolVar(&all, "all", false, "List every branch (paginates through all pages)")
	return cmd
}

func newBranchesGetCmd(resolve workspaceResolver) *cobra.Command {
	var branchID string
	cmd := &cobra.Command{
		Use:   "get --branch-id <branch-id>",
		Short: "Get a branch by id",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			g := fromCtx(cmd)
			if strings.TrimSpace(branchID) == "" {
				return fmt.Errorf("--branch-id is required")
			}
			wsID, err := resolve(cmd)
			if err != nil {
				return err
			}
			client, err := g.NewVolcClient(cmd.Context())
			if err != nil {
				return err
			}
			result, err := client.DescribeBranchDetail(cmd.Context(), wsID, branchID)
			if err != nil {
				return err
			}
			return g.Writer().WriteItem(result.Branch, branchDetailFields)
		},
	}
	cmd.Flags().StringVar(&branchID, "branch-id", "", "Branch ID (required)")
	return cmd
}

func newBranchesDefaultCmd(resolve workspaceResolver) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "default",
		Short: "Show the workspace's default branch",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			g := fromCtx(cmd)
			wsID, err := resolve(cmd)
			if err != nil {
				return err
			}
			client, err := g.NewVolcClient(cmd.Context())
			if err != nil {
				return err
			}
			result, err := client.ResolveDefaultBranch(cmd.Context(), wsID)
			if err != nil {
				return err
			}
			return g.Writer().WriteItem(result, branchFields)
		},
	}
	return cmd
}

func newBranchesChildrenCmd(resolve workspaceResolver) *cobra.Command {
	var (
		parentID string
		limit    int
		offset   int
	)
	run := func(cmd *cobra.Command, args []string) error {
		g := fromCtx(cmd)
		if parentID == "" {
			return fmt.Errorf("--parent-branch-id is required")
		}
		wsID, err := resolve(cmd)
		if err != nil {
			return err
		}
		client, err := g.NewVolcClient(cmd.Context())
		if err != nil {
			return err
		}
		result, err := client.DescribeChildBranches(cmd.Context(), volcengine.DescribeChildBranchesParams{
			WorkspaceID: wsID, ParentID: parentID, Limit: requestedListLimit(cmd, limit), Offset: offset,
		})
		if err != nil {
			return err
		}
		return writeListPage(cmd, g, result.Branches, branchFields, result.Total, limit, offset, "branches")
	}
	cmd := &cobra.Command{
		Use:   "children",
		Short: "List child branches under a parent branch",
		Args:  cobra.NoArgs,
	}
	list := &cobra.Command{
		Use:   "list --parent-branch-id <branch-id>",
		Short: "List child branches under a parent branch",
		Args:  cobra.NoArgs,
		RunE:  run,
	}
	list.Flags().StringVar(&parentID, "parent-branch-id", "", "Parent branch ID (required)")
	list.Flags().IntVar(&limit, "limit", 0, "Maximum number of branches to return (default 10, 0=100, max 100)")
	list.Flags().IntVar(&offset, "offset", 0, "Number of branches to skip")
	cmd.AddCommand(list)
	return cmd
}

func newBranchesCreateCmd(resolve workspaceResolver) *cobra.Command {
	var (
		name       string
		parentID   string
		parentTime string
	)
	cmd := &cobra.Command{
		Use:     "create --name <name>",
		Short:   "Create a branch",
		Example: "byted-postgresql-cli branches create --workspace-id ws-xxx --name feature-a",
		Args:    cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			g := fromCtx(cmd)
			if name == "" {
				return fmt.Errorf("--name is required")
			}
			if err := validateBranchName(name); err != nil {
				return err
			}
			wsID, err := resolve(cmd)
			if err != nil {
				return err
			}
			client, err := g.NewVolcClient(cmd.Context())
			if err != nil {
				return err
			}
			result, err := client.CreateBranch(cmd.Context(), volcengine.CreateBranchParams{
				WorkspaceID: wsID, Name: name, ParentID: parentID, ParentTime: parentTime,
			})
			if err != nil {
				return err
			}
			return g.Writer().WriteItem(result.NormalizedBranch(), branchDetailFields)
		},
	}
	cmd.Flags().StringVar(&name, "name", "", "Branch name")
	cmd.Flags().StringVar(&parentID, "parent-id", "", "Parent branch id (defaults to the default branch)")
	cmd.Flags().StringVar(&parentTime, "parent-time", "", "Point-in-time on the parent to branch from (RFC3339)")
	_ = cmd.MarkFlagRequired("name")
	return cmd
}

func validateBranchName(name string) error {
	if err := validateWorkspaceName(name); err != nil {
		return fmt.Errorf("branch name %q must be 2-64 characters and contain only letters, digits, Chinese characters, '-' and '_'", name)
	}
	return nil
}

func newBranchesUpdateCmd(resolve workspaceResolver) *cobra.Command {
	var (
		branchID  string
		name      string
		protected string
	)
	cmd := &cobra.Command{
		Use:   "update --branch-id <branch-id> [--name <new-name>] [--protected true|false]",
		Short: "Rename a branch or change branch protection",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			g := fromCtx(cmd)
			if strings.TrimSpace(branchID) == "" {
				return fmt.Errorf("--branch-id is required")
			}
			if !cmd.Flags().Changed("name") && !cmd.Flags().Changed("protected") {
				return fmt.Errorf("specify --name and/or --protected")
			}
			wsID, err := resolve(cmd)
			if err != nil {
				return err
			}
			client, err := g.NewVolcClient(cmd.Context())
			if err != nil {
				return err
			}
			var protectedValue *bool
			if cmd.Flags().Changed("protected") {
				value, err := strconv.ParseBool(protected)
				if err != nil {
					return fmt.Errorf("--protected must be true or false")
				}
				protectedValue = &value
			}
			result, err := client.UpdateBranch(cmd.Context(), volcengine.UpdateBranchParams{
				WorkspaceID: wsID, BranchID: branchID, Name: changedString(cmd, "name", name), Protected: protectedValue,
			})
			if err != nil {
				return err
			}
			return g.Writer().WriteItem(result.Branch, branchDetailFields)
		},
	}
	cmd.Flags().StringVar(&name, "name", "", "New branch name")
	cmd.Flags().StringVar(&protected, "protected", "", "Set branch protection to true or false")
	cmd.Flags().StringVar(&branchID, "branch-id", "", "Branch ID (required)")
	return cmd
}

func changedString(cmd *cobra.Command, name, value string) *string {
	if !cmd.Flags().Changed(name) {
		return nil
	}
	return &value
}

func newBranchesDeleteCmd(resolve workspaceResolver) *cobra.Command {
	var branchID string
	var yes bool
	cmd := &cobra.Command{
		Use:     "delete --branch-id <branch-id>",
		Aliases: []string{"rm"},
		Short:   "Delete a branch",
		Args:    cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			g := fromCtx(cmd)
			if strings.TrimSpace(branchID) == "" {
				return fmt.Errorf("--branch-id is required")
			}
			wsID, err := resolve(cmd)
			if err != nil {
				return err
			}
			if !yes {
				summary := fmt.Sprintf("Delete branch %q? This operation cannot be undone.", branchID)
				if err := confirmDestructiveAction(cmd, branchID, summary, "delete"); err != nil {
					return err
				}
			}
			client, err := g.NewVolcClient(cmd.Context())
			if err != nil {
				return err
			}
			if _, err := client.DeleteBranch(cmd.Context(), wsID, branchID); err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Branch %s deleted\n", branchID)
			return nil
		},
	}
	cmd.Flags().StringVar(&branchID, "branch-id", "", "Branch ID (required)")
	cmd.Flags().BoolVarP(&yes, "yes", "y", false, "Skip the confirmation prompt")
	return cmd
}

func newBranchesRestartCmd(resolve workspaceResolver) *cobra.Command {
	var branchID string
	var computeIDs []string
	cmd := &cobra.Command{
		Use:   "restart --branch-id <branch-id>",
		Short: "Restart a branch's computes",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			g := fromCtx(cmd)
			if strings.TrimSpace(branchID) == "" {
				return fmt.Errorf("--branch-id is required")
			}
			wsID, err := resolve(cmd)
			if err != nil {
				return err
			}
			client, err := g.NewVolcClient(cmd.Context())
			if err != nil {
				return err
			}
			if _, err := client.RestartBranch(cmd.Context(), volcengine.RestartBranchParams{
				WorkspaceID: wsID, BranchID: branchID, ComputeIDs: computeIDs,
			}); err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Branch %s restarting\n", branchID)
			return nil
		},
	}
	cmd.Flags().StringVar(&branchID, "branch-id", "", "Branch ID (required)")
	cmd.Flags().StringSliceVar(&computeIDs, "compute-id", nil, "Restart only these compute ids (repeatable)")
	return cmd
}

func newBranchesSetDefaultCmd(resolve workspaceResolver) *cobra.Command {
	var branchID string
	cmd := &cobra.Command{
		Use:   "set-default --branch-id <branch-id>",
		Short: "Set a branch as the workspace's default",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			g := fromCtx(cmd)
			if strings.TrimSpace(branchID) == "" {
				return fmt.Errorf("--branch-id is required")
			}
			wsID, err := resolve(cmd)
			if err != nil {
				return err
			}
			client, err := g.NewVolcClient(cmd.Context())
			if err != nil {
				return err
			}
			result, err := client.SetAsDefaultBranch(cmd.Context(), wsID, branchID)
			if err != nil {
				return err
			}
			return g.Writer().WriteItem(result.Branch, branchDetailFields)
		},
	}
	cmd.Flags().StringVar(&branchID, "branch-id", "", "Branch ID (required)")
	return cmd
}

func newBranchesRestoreWindowCmd(resolve workspaceResolver) *cobra.Command {
	var branchID string
	cmd := &cobra.Command{
		Use:   "restore-window --branch-id <branch-id>",
		Short: "Show a branch's point-in-time restore window",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			g := fromCtx(cmd)
			if strings.TrimSpace(branchID) == "" {
				return fmt.Errorf("--branch-id is required")
			}
			wsID, err := resolve(cmd)
			if err != nil {
				return err
			}
			client, err := g.NewVolcClient(cmd.Context())
			if err != nil {
				return err
			}
			window, err := client.GetRestoreWindow(cmd.Context(), wsID, branchID)
			if err != nil {
				return err
			}
			return g.Writer().WriteItem(window, []string{"BranchId", "WindowSizeSeconds", "BranchCreateTime", "StartTime", "EndTime"})
		},
	}
	cmd.Flags().StringVar(&branchID, "branch-id", "", "Branch ID (required)")
	return cmd
}

func newBranchesRestorableCmd(resolve workspaceResolver) *cobra.Command {
	var (
		atTime string
		search string
		limit  int
		offset int
	)
	cmd := &cobra.Command{
		Use:   "restorable",
		Short: "List branches restorable at a point in time",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if strings.TrimSpace(atTime) == "" {
				return fmt.Errorf("--time is required")
			}
			g := fromCtx(cmd)
			wsID, err := resolve(cmd)
			if err != nil {
				return err
			}
			client, err := g.NewVolcClient(cmd.Context())
			if err != nil {
				return err
			}
			result, err := client.DescribeRestorableBranches(cmd.Context(), volcengine.DescribeRestorableBranchesParams{
				WorkspaceID: wsID, Time: atTime, Search: search, Limit: requestedListLimit(cmd, limit), Offset: offset,
			})
			if err != nil {
				return err
			}
			return writeListPage(cmd, g, result.Branches, branchFields, result.Total, limit, offset, "branches")
		},
	}
	cmd.Flags().StringVar(&atTime, "time", "", "Point in time to check (RFC3339)")
	cmd.Flags().StringVar(&search, "search", "", "Filter branches by name")
	cmd.Flags().IntVar(&limit, "limit", 0, "Maximum number of branches to return (default 10, 0=100, max 100)")
	cmd.Flags().IntVar(&offset, "offset", 0, "Number of branches to skip")
	return cmd
}

func newBranchesRestoreCmd(resolve workspaceResolver) *cobra.Command {
	var (
		branchID       string
		atTime         string
		sourceBranchID string
	)
	cmd := &cobra.Command{
		Use:   "restore --branch-id <branch-id>",
		Short: "Restore a branch to a point in time",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			g := fromCtx(cmd)
			if strings.TrimSpace(branchID) == "" {
				return fmt.Errorf("--branch-id is required")
			}
			wsID, err := resolve(cmd)
			if err != nil {
				return err
			}
			client, err := g.NewVolcClient(cmd.Context())
			if err != nil {
				return err
			}
			result, err := client.BranchRestore(cmd.Context(), volcengine.BranchRestoreParams{
				WorkspaceID: wsID, BranchID: branchID, Time: atTime, SourceBranchID: sourceBranchID,
			})
			if err != nil {
				return err
			}
			return g.Writer().WriteItem(result, []string{"WorkspaceId", "BranchId", "SourceBranchId", "Time", "BackupBranchId"})
		},
	}
	cmd.Flags().StringVar(&atTime, "time", "", "Point in time to restore to (RFC3339)")
	cmd.Flags().StringVar(&sourceBranchID, "source-branch-id", "", "Source branch to restore data from")
	cmd.Flags().StringVar(&branchID, "branch-id", "", "Branch ID (required)")
	return cmd
}
