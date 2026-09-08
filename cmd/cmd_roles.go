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

var roleFields = []string{"AccountName", "AccountDesc", "BranchId", "CreateTime", "UpdateTime"}

// newRolesCmd manages a branch's database accounts (Postgres roles). Every role
// operation is scoped to one PostgreSQL workspace + branch via --workspace-id
// and --branch-id (--branch-id defaults to the workspace's default branch).
func newRolesCmd(ctx ProviderContext) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "roles",
		Aliases: []string{"role", "accounts", "account"},
		Short:   "List branch roles and database accounts",
	}

	var workspaceID, branchID string

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
		Short: "List roles",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			g := fromCtx(cmd)
			client, wsID, bid, err := resolve(cmd)
			if err != nil {
				return err
			}
			result, err := client.DescribeDBAccounts(cmd.Context(), volcengine.DescribeDBAccountsParams{
				WorkspaceID: wsID, BranchID: bid, Search: search, Limit: requestedListLimit(cmd, limit), Offset: offset,
			})
			if err != nil {
				return err
			}
			return writeListPage(cmd, g, result.Accounts, roleFields, result.Total, limit, offset, "roles")
		},
	}
	list.Flags().StringVar(&search, "search", "", "Filter roles by name")
	list.Flags().IntVar(&limit, "limit", 0, "Maximum number of roles to return (default 10, 0=100, max 100)")
	list.Flags().IntVar(&offset, "offset", 0, "Number of roles to skip")
	cmd.AddCommand(list)

	var createName, password, description string
	create := &cobra.Command{
		Use:   "create --name <name> --password <password>",
		Short: "Create a PostgreSQL role",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			g := fromCtx(cmd)
			client, wsID, bid, err := resolve(cmd)
			if err != nil {
				return err
			}
			account, err := client.CreateDBAccount(cmd.Context(), volcengine.CreateDBAccountParams{
				WorkspaceID: wsID, BranchID: bid, AccountName: createName, Password: password, Description: description,
			})
			if err != nil {
				return err
			}
			return g.Writer().WriteItem(account, roleFields)
		},
	}
	create.Flags().StringVar(&createName, "name", "", "Role name (required)")
	create.Flags().StringVar(&password, "password", "", "Role password (required)")
	create.Flags().StringVar(&description, "description", "", "Role description")
	_ = create.MarkFlagRequired("name")
	_ = create.MarkFlagRequired("password")
	cmd.AddCommand(create)

	var deleteYes bool
	var deleteName string
	del := &cobra.Command{
		Use: "delete --name <role-name>", Aliases: []string{"rm"},
		Short: "Delete a PostgreSQL role", Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if strings.TrimSpace(deleteName) == "" {
				return fmt.Errorf("--name is required")
			}
			client, wsID, bid, err := resolve(cmd)
			if err != nil {
				return err
			}
			if !deleteYes {
				if err := confirmDestructiveAction(cmd, deleteName, fmt.Sprintf("Delete role %q? This operation cannot be undone.", deleteName), "delete"); err != nil {
					return err
				}
			}
			if err := client.DeleteDBAccount(cmd.Context(), wsID, bid, deleteName); err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Role %s deleted\n", deleteName)
			return nil
		},
	}
	del.Flags().StringVar(&deleteName, "name", "", "Role name (required)")
	del.Flags().BoolVarP(&deleteYes, "yes", "y", false, "Skip the confirmation prompt")
	cmd.AddCommand(del)

	var resetPassword string
	var resetName string
	reset := &cobra.Command{
		Use:   "reset-password --name <role-name> --password <password>",
		Short: "Reset a PostgreSQL role password", Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if strings.TrimSpace(resetName) == "" {
				return fmt.Errorf("--name is required")
			}
			client, wsID, bid, err := resolve(cmd)
			if err != nil {
				return err
			}
			if err := client.ResetDBAccountPassword(cmd.Context(), wsID, bid, resetName, resetPassword); err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Role %s password reset\n", resetName)
			return nil
		},
	}
	reset.Flags().StringVar(&resetName, "name", "", "Role name (required)")
	reset.Flags().StringVar(&resetPassword, "password", "", "New role password (required)")
	_ = reset.MarkFlagRequired("password")
	cmd.AddCommand(reset)
	for _, child := range cmd.Commands() {
		child.Flags().StringVar(&workspaceID, "workspace-id", "", "Workspace ID (required in non-interactive mode)")
		child.Flags().StringVar(&branchID, "branch-id", "", "Branch ID (defaults to the workspace's default branch)")
	}

	return cmd
}
