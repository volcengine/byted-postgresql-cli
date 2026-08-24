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

	"github.com/spf13/cobra"

	"github.com/volcengine/byted-postgresql-cli/internal/volcengine"
)

func newSchemaDiffCmd(ctx ProviderContext) *cobra.Command {
	var workspaceID string
	cmd := &cobra.Command{Use: "schema-diff", Short: "Inspect PostgreSQL schema diff jobs"}
	cmd.PersistentFlags().StringVar(&workspaceID, "workspace-id", "", "Workspace ID")

	resolveClient := func(cmd *cobra.Command) (*volcengine.Client, error) {
		return fromCtx(cmd).NewVolcClient(cmd.Context())
	}

	status := &cobra.Command{
		Use: "status <job-id>", Short: "Show schema diff job status", Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := resolveClient(cmd)
			if err != nil {
				return err
			}
			result, err := client.DescribeSchemaDiffJobStatus(cmd.Context(), workspaceID, args[0])
			if err != nil {
				return err
			}
			return fromCtx(cmd).Writer().WriteItem(result, nil)
		},
	}
	result := &cobra.Command{
		Use: "result <job-id>", Short: "Show schema diff migration SQL", Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := resolveClient(cmd)
			if err != nil {
				return err
			}
			sqlText, err := client.DescribeSchemaDiffResultSQLAll(cmd.Context(), workspaceID, args[0])
			if err != nil {
				return err
			}
			fmt.Fprintln(cmd.OutOrStdout(), sqlText)
			return nil
		},
	}
	download := &cobra.Command{
		Use: "download <job-id>", Short: "Get schema diff result download URL", Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := resolveClient(cmd)
			if err != nil {
				return err
			}
			link, err := client.GetSchemaDiffDownloadLink(cmd.Context(), workspaceID, args[0])
			if err != nil {
				return err
			}
			fmt.Fprintln(cmd.OutOrStdout(), link)
			return nil
		},
	}
	cmd.AddCommand(status, result, download)
	return cmd
}
