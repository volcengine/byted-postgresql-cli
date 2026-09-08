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
	"net/url"
	"os"
	"os/exec"
	"strings"

	"github.com/spf13/cobra"

	"github.com/volcengine/byted-postgresql-cli/internal/volcengine"
)

func newPsqlCmd(ctx ProviderContext) *cobra.Command {
	var (
		workspaceID, branchID, database, account string
	)
	cmd := &cobra.Command{
		Use:   "psql [--branch-id <branch-id>] [-- psql-args...]",
		Short: "Open a psql session for a branch",
		Long: "Open a psql session for a branch.\n\n" +
			"Database and role defaults are used only when the branch has a single unambiguous candidate. " +
			"For multiple databases or roles, pass --database-name and --role-name; inspect candidates with " +
			"`databases list` and `roles list`. The workspace's default branch is used when --branch-id is omitted. " +
			"psql connects with the role password returned by the control plane.",
		Args: func(cmd *cobra.Command, args []string) error {
			if dashIdx := cmd.ArgsLenAtDash(); dashIdx >= 0 {
				if dashIdx > 0 {
					return fmt.Errorf("psql accepts no positional arguments; pass psql arguments after --")
				}
				return nil
			}
			if len(args) > 0 {
				return fmt.Errorf("psql accepts no positional arguments; pass psql arguments after --")
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			g := fromCtx(cmd)
			bid := branchID
			var passThrough []string
			if dashIdx := cmd.ArgsLenAtDash(); dashIdx >= 0 {
				passThrough = args[dashIdx:]
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
			bin, err := exec.LookPath("psql")
			if err != nil {
				return fmt.Errorf("psql not found in PATH: %w", err)
			}
			if err := checkPostgresClientVersion(cmd, client, workspaceID, bin, "psql"); err != nil {
				return err
			}
			psqlArgs := append([]string{connectionURL}, passThrough...)
			c := exec.Command(bin, psqlArgs...)
			c.Stdin = os.Stdin
			c.Stdout = os.Stdout
			c.Stderr = os.Stderr
			return c.Run()
		},
	}
	cmd.Flags().StringVar(&workspaceID, "workspace-id", "", "Workspace ID")
	cmd.Flags().StringVar(&branchID, "branch-id", "", "Branch ID (defaults to the workspace's default branch)")
	cmd.Flags().StringVar(&database, "database-name", "", "Database name (defaults to the branch default)")
	cmd.Flags().StringVar(&account, "role-name", "", "Role/account name (defaults to the branch default)")
	return cmd
}

// psqlConnectionURL builds a ready-to-use libpq URL from a DBAccountConnection.
// The Volcengine control plane returns the role password separately from the
// connection URL, so the password is injected into the URL's userinfo. When the
// gateway already returns a complete URL with credentials, it is used as-is.
func psqlConnectionURL(conn volcengine.DBAccountConnection) (string, error) {
	connectionURL := extractPostgresURL(conn.ConnectionURL)
	if connectionURL == "" {
		return "", fmt.Errorf("gateway returned an invalid connection URL")
	}
	return injectPassword(connectionURL, conn.AccountName, conn.AccountPassword)
}

// injectPassword sets the userinfo (user:password) on a libpq URL. It fills in
// the account name/password only when they are missing from the URL, so a
// gateway-provided complete URL is left untouched.
func injectPassword(rawURL, account, password string) (string, error) {
	u, err := url.Parse(rawURL)
	if err != nil {
		return "", fmt.Errorf("parse connection URL: %w", err)
	}
	user := account
	if u.User != nil && u.User.Username() != "" {
		user = u.User.Username()
	}
	// Preserve an existing password when the URL already carries one.
	if u.User != nil {
		if existing, ok := u.User.Password(); ok && existing != "" {
			password = existing
		}
	}
	if user == "" {
		// Nothing to set; return the URL unchanged.
		return rawURL, nil
	}
	if password != "" {
		u.User = url.UserPassword(user, password)
	} else {
		u.User = url.User(user)
	}
	return u.String(), nil
}

func extractPostgresURL(value string) string {
	start := strings.Index(value, "postgresql://")
	if start < 0 {
		start = strings.Index(value, "postgres://")
	}
	if start < 0 {
		// The gateway may return a bare host/URL without a scheme; treat the
		// whole trimmed value as the URL in that case.
		return strings.TrimSpace(value)
	}
	value = value[start:]
	if end := strings.IndexAny(value, "\"'\r\n\t "); end >= 0 {
		value = value[:end]
	}
	return value
}
