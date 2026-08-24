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

var operationFields = []string{"OperationId", "ActionName", "ActionStatus", "WorkspaceId", "BranchId", "CreateTime", "FinishTime"}

var operationStatuses = []string{"Start", "Running", "Success", "Failed"}

func normalizeOperationStatus(status string) (string, error) {
	status = strings.TrimSpace(status)
	if status == "" {
		return "", nil
	}
	for _, allowed := range operationStatuses {
		if strings.EqualFold(status, allowed) {
			return allowed, nil
		}
	}
	return "", fmt.Errorf("--status must be one of %s, got %q", strings.Join(operationStatuses, ", "), status)
}

// newOperationsCmd lists Volcengine AIDAP operations. Operations are filtered by
// workspace/branch and status/action; pass --workspace-id (and optionally
// --branch-id) to narrow the results.
func newOperationsCmd(ctx ProviderContext) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "operations",
		Aliases: []string{"operation", "ops"},
		Short:   "List operations",
	}

	var (
		workspaceID string
		branchID    string
		status      string
		action      string
		limit       int
		offset      int
	)

	list := &cobra.Command{
		Use:   "list",
		Short: "List operations",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			g := fromCtx(cmd)
			normalizedStatus, err := normalizeOperationStatus(status)
			if err != nil {
				return err
			}
			client, err := g.NewVolcClient(cmd.Context())
			if err != nil {
				return err
			}
			result, err := client.DescribeOperations(cmd.Context(), volcengine.DescribeOperationsParams{
				WorkspaceID: workspaceID,
				BranchID:    branchID,
				Status:      normalizedStatus,
				ActionName:  action,
				Limit:       requestedListLimit(cmd, limit),
				Offset:      offset,
			})
			if err != nil {
				return err
			}
			return writeListPage(cmd, g, result.Operations, operationFields, result.Total, limit, offset, "operations")
		},
	}
	list.Flags().StringVar(&workspaceID, "workspace-id", "", "Filter by workspace ID")
	list.Flags().StringVar(&branchID, "branch-id", "", "Filter by branch ID")
	list.Flags().StringVar(&status, "status", "", "Filter by operation status")
	list.Flags().StringVar(&action, "action", "", "Filter by action name")
	list.Flags().IntVar(&limit, "limit", 0, "Maximum number of operations to return (default 10, 0=100, max 100)")
	list.Flags().IntVar(&offset, "offset", 0, "Number of operations to skip")
	cmd.AddCommand(list)

	return cmd
}
