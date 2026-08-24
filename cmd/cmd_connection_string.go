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
	"context"
	"fmt"
	"net/url"
	"strings"

	"github.com/spf13/cobra"

	"github.com/volcengine/byted-postgresql-cli/internal/volcengine"
)

// resolveConnection resolves a branch's primary Database compute and asks the
// gateway for a ready-to-use libpq connection URL for the given account and
// database. When branchID is empty the workspace's default branch is used.
// Volcengine AIDAP requires explicit database/account values, so omitted values are
// resolved locally from the branch before requesting the connection.
func resolveConnection(ctx context.Context, client *volcengine.Client, workspaceID, branchID, database, account string) (volcengine.DBAccountConnection, error) {
	resolvedBranch, computeID, err := client.ResolvePrimaryDatabaseComputeID(ctx, workspaceID, branchID)
	if err != nil {
		return volcengine.DBAccountConnection{}, err
	}
	workspace, err := client.DescribeWorkspaceDetail(ctx, workspaceID)
	if err != nil {
		return volcengine.DBAccountConnection{}, err
	}
	endpoints, err := client.DescribeWorkspaceEndpoints(ctx, volcengine.DescribeWorkspaceEndpointsParams{
		WorkspaceID: workspaceID,
		BranchID:    resolvedBranch,
		ComputeID:   computeID,
	})
	if err != nil {
		return volcengine.DBAccountConnection{}, err
	}
	addressID := publicEndpointAddressID(endpoints.Endpoints)
	if addressID == "" {
		return volcengine.DBAccountConnection{}, fmt.Errorf("branch has no public endpoint address")
	}
	if database == "" || account == "" {
		databases, err := client.DescribeDatabases(ctx, volcengine.DescribeDatabasesParams{
			WorkspaceID: workspaceID,
			BranchID:    resolvedBranch,
		})
		if err != nil {
			return volcengine.DBAccountConnection{}, err
		}
		database, account, err = resolveDatabaseAndAccount(databases.Databases, database, account)
		if err != nil {
			return volcengine.DBAccountConnection{}, err
		}
	}
	if account == "" {
		accounts, err := client.DescribeDBAccounts(ctx, volcengine.DescribeDBAccountsParams{
			WorkspaceID: workspaceID,
			BranchID:    resolvedBranch,
		})
		if err != nil {
			return volcengine.DBAccountConnection{}, err
		}
		account, err = resolveSingleAccount(accounts.Accounts)
		if err != nil {
			return volcengine.DBAccountConnection{}, err
		}
	}
	return client.DescribeDBAccountConnection(ctx, volcengine.DescribeDBAccountConnectionParams{
		WorkspaceID:  workspaceID,
		BranchID:     resolvedBranch,
		ComputeID:    computeID,
		AccountName:  account,
		DatabaseName: database,
		ProjectName:  workspace.ProjectName,
		AddressID:    addressID,
	})
}

func publicEndpointAddressID(endpoints []volcengine.Endpoint) string {
	for _, endpoint := range endpoints {
		for _, address := range endpoint.Addresses {
			if strings.EqualFold(strings.TrimSpace(address.AddressType), "Public") && strings.TrimSpace(address.AddressID) != "" {
				return address.AddressID
			}
		}
	}
	return ""
}

func resolveDatabaseAndAccount(databases []volcengine.Database, database, account string) (string, string, error) {
	if database == "" {
		switch len(databases) {
		case 0:
			return "", "", fmt.Errorf("branch has no databases")
		case 1:
			database = databases[0].DatabaseName
			if account == "" {
				account = databases[0].DatabaseOwner
			}
		default:
			return "", "", fmt.Errorf(
				"branch has multiple databases (%s); pass --database-name",
				joinDatabaseNames(databases),
			)
		}
		return database, account, nil
	}

	if account != "" {
		return database, account, nil
	}
	for _, candidate := range databases {
		if candidate.DatabaseName == database {
			return database, candidate.DatabaseOwner, nil
		}
	}
	return "", "", fmt.Errorf("database %q not found on branch; available: %s", database, joinDatabaseNames(databases))
}

func resolveSingleAccount(accounts []volcengine.DBAccount) (string, error) {
	switch len(accounts) {
	case 0:
		return "", fmt.Errorf("branch has no roles; run `byted-postgresql-cli roles list --workspace-id <workspace-id> --branch-id <branch-id>` after creating one, then retry with --role-name <role-name>")
	case 1:
		return accounts[0].AccountName, nil
	default:
		names := make([]string, 0, len(accounts))
		for _, account := range accounts {
			names = append(names, account.AccountName)
		}
		return "", fmt.Errorf("branch has multiple roles (%s); run `byted-postgresql-cli roles list --workspace-id <workspace-id> --branch-id <branch-id>` and retry with --role-name <role-name> (or pass --role-name to databases create)", strings.Join(names, ", "))
	}
}

func joinDatabaseNames(databases []volcengine.Database) string {
	names := make([]string, 0, len(databases))
	for _, database := range databases {
		names = append(names, database.DatabaseName)
	}
	if len(names) == 0 {
		return "none"
	}
	return strings.Join(names, ", ")
}

func newConnectionStringCmd(ctx ProviderContext) *cobra.Command {
	var (
		workspaceID, branchID, database, account string
		reveal, masked                           bool
	)
	cmd := &cobra.Command{
		Use:     "connection-string [branch-id]",
		Aliases: []string{"cs"},
		Short:   "Print a Postgres connection string for a branch",
		Long: "Print a Postgres connection string for a branch.\n\n" +
			"When the branch has one database and one role, their defaults are used. " +
			"When there are multiple databases or roles, pass --database-name and/or --role-name explicitly; " +
			"inspect candidates with `databases list` and `roles list`. In non-interactive mode, provide " +
			"--workspace-id; the workspace's default branch is used when --branch-id is omitted. " +
			"The output contains connection credentials; protect it like a secret.",
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			g := fromCtx(cmd)
			bid := branchID
			if bid == "" && len(args) == 1 {
				bid = args[0]
			}
			client, err := g.NewVolcClient(cmd.Context())
			if err != nil {
				return err
			}
			workspaceID, err = resolveWorkspace(cmd.Context(), g, client, workspaceID)
			if err != nil {
				return err
			}
			bid, err = resolveBranch(cmd.Context(), g, client, workspaceID, bid)
			if err != nil {
				return err
			}
			conn, err := resolveConnection(cmd.Context(), client, workspaceID, bid, database, account)
			if err != nil {
				return err
			}
			if conn.ConnectionURL == "" {
				return fmt.Errorf("gateway returned no connection URL for branch %s", bid)
			}
			connectionURL, err := psqlConnectionURL(conn)
			if err != nil {
				return err
			}
			if reveal && masked {
				return fmt.Errorf("--reveal cannot be combined with --masked")
			}
			if reveal {
				fmt.Fprintln(cmd.ErrOrStderr(), "Warning: connection string contains a password; protect it like a secret.")
			} else {
				connectionURL = maskConnectionPassword(connectionURL)
			}
			fmt.Fprintln(cmd.OutOrStdout(), connectionURL)
			return nil
		},
	}
	cmd.Flags().StringVar(&workspaceID, "workspace-id", "", "Workspace ID")
	cmd.Flags().StringVar(&branchID, "branch-id", "", "Branch ID (defaults to the workspace's default branch)")
	cmd.Flags().StringVar(&database, "database-name", "", "Database name (defaults to the branch default)")
	cmd.Flags().StringVar(&account, "role-name", "", "Role/account name (defaults to the branch default)")
	cmd.Flags().BoolVar(&reveal, "reveal", false, "Print the connection password in plain text (sensitive)")
	cmd.Flags().BoolVar(&masked, "masked", false, "Print the connection string with the password masked (default)")
	return cmd
}

func maskConnectionPassword(rawURL string) string {
	u, err := url.Parse(rawURL)
	if err != nil || u.User == nil {
		return rawURL
	}
	username := u.User.Username()
	if username == "" {
		return rawURL
	}
	u.User = url.User(username)
	maskedURL := u.String()
	schemeEnd := strings.Index(maskedURL, "://")
	if schemeEnd < 0 {
		return maskedURL
	}
	authorityStart := schemeEnd + len("://")
	at := strings.IndexByte(maskedURL[authorityStart:], '@')
	if at < 0 {
		return maskedURL
	}
	at += authorityStart
	return maskedURL[:at] + ":*****" + maskedURL[at:]
}
