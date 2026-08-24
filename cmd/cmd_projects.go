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
	"sort"

	"github.com/spf13/cobra"

	"github.com/volcengine/byted-postgresql-cli/internal/volcengine"
)

var projectFields = []string{"ProjectName", "WorkspaceCount"}

type projectSummary struct {
	ProjectName    string `json:"ProjectName" yaml:"ProjectName"`
	WorkspaceCount int    `json:"WorkspaceCount" yaml:"WorkspaceCount"`
}

func newProjectsCmd(ctx ProviderContext) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "projects",
		Short: "List billing projects visible to the current user",
	}
	list := &cobra.Command{
		Use:   "list",
		Short: "List projects and their workspace counts",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			g := fromCtx(cmd)
			client, err := g.NewVolcClient(cmd.Context())
			if err != nil {
				return err
			}
			result, err := client.ListAllWorkspaces(cmd.Context(), volcengine.ListWorkspacesParams{
				Limit: volcengine.MaxPageLimit,
			})
			if err != nil {
				return err
			}
			summaries := summarizeProjects(result.Workspaces)
			return g.Writer().WriteList(summaries, projectFields)
		},
	}
	cmd.AddCommand(list)
	return cmd
}

func summarizeProjects(workspaces []volcengine.Workspace) []projectSummary {
	counts := make(map[string]int)
	for _, workspace := range workspaces {
		if workspace.ProjectName != "" {
			counts[workspace.ProjectName]++
		}
	}
	names := make([]string, 0, len(counts))
	for name := range counts {
		names = append(names, name)
	}
	sort.Strings(names)
	summaries := make([]projectSummary, 0, len(names))
	for _, name := range names {
		summaries = append(summaries, projectSummary{
			ProjectName:    name,
			WorkspaceCount: counts[name],
		})
	}
	return summaries
}
