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
	"strings"

	"github.com/spf13/cobra"

	"github.com/volcengine/byted-postgresql-cli/internal/volcengine"
)

var databaseFields = []string{"DatabaseName", "DatabaseOwner", "DatabaseDesc", "BranchId", "CreateTime", "UpdateTime"}

// newDatabasesCmd manages the logical databases inside a branch. Every database
// operation is scoped to one PostgreSQL workspace + branch via --workspace-id
// and --branch-id (--branch-id defaults to the workspace's default branch).
func newDatabasesCmd(ctx ProviderContext) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "databases",
		Aliases: []string{"database"},
		Short:   "Manage branch databases",
	}

	var workspaceID, branchID string

	// resolve returns a client plus the resolved workspace/branch ids. An empty
	// --branch-id falls back to the workspace's default branch.
	resolve := func(cmd *cobra.Command) (*volcengine.Client, string, string, error) {
		g := fromCtx(cmd)
		client, err := g.NewVolcClient(cmd.Context())
		if err != nil {
			return nil, "", "", err
		}
		workspace, err := resolveWorkspace(cmd.Context(), g, client, workspaceID)
		if err != nil {
			return nil, "", "", err
		}
		branch, err := resolveBranch(cmd.Context(), g, client, workspace, branchID)
		return client, workspace, branch, err
	}

	var (
		search string
		limit  int
		offset int
	)
	list := &cobra.Command{
		Use:   "list",
		Short: "List databases",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			g := fromCtx(cmd)
			client, wsID, bid, err := resolve(cmd)
			if err != nil {
				return err
			}
			result, err := client.DescribeDatabases(cmd.Context(), volcengine.DescribeDatabasesParams{
				WorkspaceID: wsID, BranchID: bid, Search: search, Limit: requestedListLimit(cmd, limit), Offset: offset,
			})
			if err != nil {
				return err
			}
			return writeListPage(cmd, g, result.Databases, databaseFields, result.Total, limit, offset, "databases")
		},
	}
	list.Flags().StringVar(&search, "search", "", "Filter databases by name")
	list.Flags().IntVar(&limit, "limit", 0, "Maximum number of databases to return (default 10, 0=100, max 100)")
	list.Flags().IntVar(&offset, "offset", 0, "Number of databases to skip")
	cmd.AddCommand(list)

	var name, desc, roleName string
	var yes bool
	create := &cobra.Command{
		Use:   "create",
		Short: "Create a database",
		Long:  "The branch's default database owner is used when --role-name is omitted. Pass --role-name to choose a different branch role.",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			g := fromCtx(cmd)
			client, wsID, bid, err := resolve(cmd)
			if err != nil {
				return err
			}
			accounts, err := client.DescribeDBAccounts(cmd.Context(), volcengine.DescribeDBAccountsParams{
				WorkspaceID: wsID,
				BranchID:    bid,
			})
			if err != nil {
				return err
			}
			owner, err := selectDatabaseOwner(cmd, accounts.Accounts, roleName)
			if err != nil {
				return err
			}
			db, err := client.CreateDatabase(cmd.Context(), volcengine.CreateDatabaseParams{
				WorkspaceID: wsID, BranchID: bid, DatabaseName: name, Owner: owner, Description: desc,
			})
			if err != nil {
				return err
			}
			return g.Writer().WriteItem(db, databaseFields)
		},
	}
	create.Flags().StringVar(&name, "name", "", "Database name")
	create.Flags().StringVar(&desc, "description", "", "Database description")
	create.Flags().StringVar(&roleName, "role-name", "", "Database owner role (defaults to the only role on the branch)")
	_ = create.MarkFlagRequired("name")
	cmd.AddCommand(create)

	del := &cobra.Command{
		Use:     "delete --name <name>",
		Aliases: []string{"rm", "drop"},
		Short:   "Delete a database",
		Args:    cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if strings.TrimSpace(name) == "" {
				return fmt.Errorf("--name is required")
			}
			client, wsID, bid, err := resolve(cmd)
			if err != nil {
				return err
			}
			if !yes {
				summary := fmt.Sprintf("Delete database %q? This operation cannot be undone.", name)
				if err := confirmDestructiveAction(cmd, name, summary, "delete"); err != nil {
					return err
				}
			}
			if err := client.DropDatabase(cmd.Context(), wsID, bid, name); err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Database %s deleted\n", name)
			return nil
		},
	}
	del.Flags().StringVar(&name, "name", "", "Database name (required)")
	del.Flags().BoolVarP(&yes, "yes", "y", false, "Skip the confirmation prompt")
	cmd.AddCommand(del)
	for _, child := range cmd.Commands() {
		child.Flags().StringVar(&workspaceID, "workspace-id", "", "Workspace ID (required in non-interactive mode)")
		child.Flags().StringVar(&branchID, "branch-id", "", "Branch ID (defaults to the workspace's default branch)")
	}

	return cmd
}

func resolveDatabaseOwner(accounts []volcengine.DBAccount) (string, error) {
	return resolveSingleAccount(accounts)
}

func selectDatabaseOwner(cmd *cobra.Command, accounts []volcengine.DBAccount, explicit string) (string, error) {
	if strings.TrimSpace(explicit) != "" {
		for _, account := range accounts {
			if account.AccountName == explicit {
				return explicit, nil
			}
		}
		return "", fmt.Errorf("role %q not found on branch; run `roles list` to inspect available roles", explicit)
	}
	if len(accounts) == 0 {
		return "", fmt.Errorf("branch has no roles; create a role first with `roles create`")
	}
	if !interactiveTarget() {
		if len(accounts) == 1 {
			return accounts[0].AccountName, nil
		}
		return "", fmt.Errorf("branch has multiple roles; pass --role-name in non-interactive mode")
	}
	items := make([]promptItem, 0, len(accounts))
	for _, account := range accounts {
		items = append(items, promptItem{
			summary: account.AccountName,
			details: account.AccountDesc,
		})
	}
	return promptList(cmd.Context(), "Select database owner", items)
}
