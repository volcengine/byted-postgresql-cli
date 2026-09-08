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
)

func newSchemaDiffCmd(ctx ProviderContext) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "schema-diff",
		Short:   "Inspect PostgreSQL schema diff jobs",
		Long:    "Check schema diff jobs with `status`, retrieve SQL with `result`, or get a download URL with `download`.",
		Example: "byted-postgresql-cli schema-diff status --workspace-id ws-xxx --job-id job-xxx --json",
	}

	status := &cobra.Command{
		Use: "status --job-id <job-id>", Short: "Show schema diff job status", Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			workspaceID, err := cmd.Flags().GetString("workspace-id")
			if err != nil {
				return err
			}
			jobID, err := cmd.Flags().GetString("job-id")
			if err != nil || jobID == "" {
				if err != nil {
					return err
				}
				return fmt.Errorf("--job-id is required")
			}
			client, err := fromCtx(cmd).NewVolcClient(cmd.Context())
			if err != nil {
				return err
			}
			result, err := client.DescribeSchemaDiffJobStatus(cmd.Context(), workspaceID, jobID)
			if err != nil {
				return err
			}
			return fromCtx(cmd).Writer().WriteItem(result, nil)
		},
	}
	status.Flags().String("job-id", "", "Schema diff job ID (required)")
	result := &cobra.Command{
		Use: "result --job-id <job-id>", Short: "Show schema diff migration SQL", Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			workspaceID, err := cmd.Flags().GetString("workspace-id")
			if err != nil {
				return err
			}
			jobID, err := cmd.Flags().GetString("job-id")
			if err != nil || jobID == "" {
				if err != nil {
					return err
				}
				return fmt.Errorf("--job-id is required")
			}
			client, err := fromCtx(cmd).NewVolcClient(cmd.Context())
			if err != nil {
				return err
			}
			sqlText, err := client.DescribeSchemaDiffResultSQLAll(cmd.Context(), workspaceID, jobID)
			if err != nil {
				return err
			}
			fmt.Fprintln(cmd.OutOrStdout(), sqlText)
			return nil
		},
	}
	result.Flags().String("job-id", "", "Schema diff job ID (required)")
	download := &cobra.Command{
		Use: "download --job-id <job-id>", Short: "Get schema diff result download URL", Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			workspaceID, err := cmd.Flags().GetString("workspace-id")
			if err != nil {
				return err
			}
			jobID, err := cmd.Flags().GetString("job-id")
			if err != nil || jobID == "" {
				if err != nil {
					return err
				}
				return fmt.Errorf("--job-id is required")
			}
			client, err := fromCtx(cmd).NewVolcClient(cmd.Context())
			if err != nil {
				return err
			}
			link, err := client.GetSchemaDiffDownloadLink(cmd.Context(), workspaceID, jobID)
			if err != nil {
				return err
			}
			fmt.Fprintln(cmd.OutOrStdout(), link)
			return nil
		},
	}
	download.Flags().String("job-id", "", "Schema diff job ID (required)")
	for _, child := range []*cobra.Command{status, result, download} {
		child.Flags().String("workspace-id", "", "Workspace ID (required)")
	}
	cmd.AddCommand(status, result, download)
	return cmd
}
