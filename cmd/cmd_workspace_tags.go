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

func newWorkspaceTagsCmd(ctx ProviderContext) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "tags",
		Short: "Manage workspace tags",
	}

	list := &cobra.Command{
		Use: "list", Short: "List workspace tags",
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			workspaceID, err := cmd.Flags().GetString("workspace-id")
			if err != nil {
				return err
			}
			client, err := fromCtx(cmd).NewVolcClient(cmd.Context())
			if err != nil {
				return err
			}
			workspace, err := client.DescribeWorkspaceDetail(cmd.Context(), workspaceID)
			if err != nil {
				return err
			}
			return fromCtx(cmd).Writer().WriteList(workspace.Tags, []string{"Key", "Value"})
		},
	}
	cmd.AddCommand(list)

	add := &cobra.Command{
		Use: "add --tag key=value", Short: "Add workspace tags",
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			workspaceID, err := cmd.Flags().GetString("workspace-id")
			if err != nil {
				return err
			}
			values, _ := cmd.Flags().GetStringSlice("tag")
			workspaceID = strings.TrimSpace(workspaceID)
			if workspaceID == "" {
				return fmt.Errorf("--workspace-id is required")
			}
			if len(values) == 0 {
				return fmt.Errorf("--tag is required")
			}
			tags, err := parseWorkspaceTags(values)
			if err != nil {
				return err
			}
			client, err := fromCtx(cmd).NewVolcClient(cmd.Context())
			if err != nil {
				return err
			}
			if err := client.AddWorkspaceTags(cmd.Context(), workspaceID, tags); err != nil {
				return err
			}
			fmt.Fprintln(cmd.OutOrStdout(), "Workspace tags added")
			return nil
		},
	}
	add.Flags().StringSlice("tag", nil, "Tag key=value (repeatable)")

	remove := &cobra.Command{
		Use: "remove --key <key>", Short: "Remove workspace tags",
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			workspaceID, err := cmd.Flags().GetString("workspace-id")
			if err != nil {
				return err
			}
			keys, _ := cmd.Flags().GetStringSlice("key")
			if len(keys) == 0 {
				return fmt.Errorf("--key is required")
			}
			client, err := fromCtx(cmd).NewVolcClient(cmd.Context())
			if err != nil {
				return err
			}
			if err := client.RemoveWorkspaceTags(cmd.Context(), workspaceID, keys); err != nil {
				return err
			}
			fmt.Fprintln(cmd.OutOrStdout(), "Workspace tags removed")
			return nil
		},
	}
	remove.Flags().StringSlice("key", nil, "Tag key to remove (repeatable)")
	for _, child := range []*cobra.Command{list, add, remove} {
		child.Flags().String("workspace-id", "", "Workspace ID (required)")
	}
	cmd.AddCommand(add, remove)
	return cmd
}

func parseWorkspaceTags(values []string) ([]volcengine.WorkspaceTag, error) {
	if len(values) == 0 {
		return nil, fmt.Errorf("--tag is required")
	}
	tags := make([]volcengine.WorkspaceTag, 0, len(values))
	for _, raw := range values {
		parts := strings.SplitN(raw, "=", 2)
		if len(parts) != 2 || strings.TrimSpace(parts[0]) == "" {
			return nil, fmt.Errorf("--tag must use key=value, got %q", raw)
		}
		tags = append(tags, volcengine.WorkspaceTag{Key: strings.TrimSpace(parts[0]), Value: parts[1]})
	}
	return tags, nil
}
