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
	"os"

	"golang.org/x/term"

	"github.com/volcengine/byted-postgresql-cli/internal/volcengine"
)

const (
	nextPageChoice = "__next_page__"
	prevPageChoice = "__prev_page__"
)

func interactiveTarget() bool {
	return term.IsTerminal(int(os.Stdin.Fd()))
}

func resolveWorkspace(ctx context.Context, g *Globals, client *volcengine.Client, explicit string) (string, error) {
	if workspace := g.ResolveWorkspace(explicit); workspace != "" {
		return workspace, nil
	}
	if !interactiveTarget() {
		return "", fmt.Errorf("no workspace selected\n\nRun:\n  byted-postgresql-cli workspaces list\n\nThen retry with:\n  --workspace-id <workspace-id>")
	}
	offset := 0
	for {
		result, err := client.ListWorkspaces(ctx, volcengine.ListWorkspacesParams{
			Limit:  volcengine.InteractiveListLimit,
			Offset: offset,
		})
		if err != nil {
			return "", err
		}
		items := make([]promptItem, 0, len(result.Workspaces)+2)
		for _, workspace := range result.Workspaces {
			items = append(items, promptItem{
				summary: workspace.WorkspaceID,
				details: fmt.Sprintf("%s, %s", workspace.WorkspaceName, workspace.WorkspaceStatus),
			})
		}
		addPagingItems(&items, "workspaces", offset, len(result.Workspaces), result.Total)
		title := pagedPromptTitle("Select a PostgreSQL workspace", "workspaces", offset, len(result.Workspaces), result.Total)
		choice, err := promptList(ctx, title, items)
		if err != nil {
			return "", err
		}
		switch choice {
		case nextPageChoice:
			offset += len(result.Workspaces)
		case prevPageChoice:
			offset -= volcengine.InteractiveListLimit
			if offset < 0 {
				offset = 0
			}
		default:
			return choice, nil
		}
	}
}

func resolveBranch(ctx context.Context, g *Globals, client *volcengine.Client, workspace, explicit string) (string, error) {
	if branch := g.ResolveBranch(explicit); branch != "" {
		return branch, nil
	}
	if branch, err := client.ResolveDefaultBranchID(ctx, workspace, ""); err == nil {
		return branch, nil
	} else if !interactiveTarget() {
		return "", err
	}
	if !interactiveTarget() {
		return "", fmt.Errorf("no branch selected\n\nRun:\n  byted-postgresql-cli branches list --workspace-id <workspace-id>\n\nThen retry with:\n  --branch-id <branch-id>")
	}
	offset := 0
	for {
		result, err := client.DescribeBranches(ctx, volcengine.DescribeBranchesParams{
			WorkspaceID: workspace,
			Limit:       volcengine.InteractiveListLimit,
			Offset:      offset,
		})
		if err != nil {
			return "", err
		}
		items := make([]promptItem, 0, len(result.Branches)+2)
		for _, branch := range result.Branches {
			details := branch.BranchStatus
			if branch.Default {
				details += ", default"
			}
			items = append(items, promptItem{summary: branch.BranchID, details: branch.BranchName + ", " + details})
		}
		addPagingItems(&items, "branches", offset, len(result.Branches), result.Total)
		title := pagedPromptTitle("Select a branch", "branches", offset, len(result.Branches), result.Total)
		choice, err := promptList(ctx, title, items)
		if err != nil {
			return "", err
		}
		switch choice {
		case nextPageChoice:
			offset += len(result.Branches)
		case prevPageChoice:
			offset -= volcengine.InteractiveListLimit
			if offset < 0 {
				offset = 0
			}
		default:
			return choice, nil
		}
	}
}

func addPagingItems(items *[]promptItem, noun string, offset, count, total int) {
	if offset > 0 {
		*items = append(*items, promptItem{
			summary: "Previous page",
			details: "Load the previous page of " + noun,
			value:   prevPageChoice,
		})
	}
	if count > 0 && offset+count < total {
		*items = append(*items, promptItem{
			summary: "Next page",
			details: "Load the next page of " + noun,
			value:   nextPageChoice,
		})
	}
}

func pagedPromptTitle(verb, noun string, offset, count, total int) string {
	if total == 0 {
		return fmt.Sprintf("No %s found:", noun)
	}
	return fmt.Sprintf("%s (showing %d-%d of %d; choose Next/Previous page to load more):",
		verb, offset+1, offset+count, total)
}
