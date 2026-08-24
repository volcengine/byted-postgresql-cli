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

package mcp

import (
	"context"
	"fmt"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/volcengine/byted-postgresql-cli/internal/volcengine"
)

type emptyInput struct{}
type listInput struct {
	Search string `json:"search,omitempty" jsonschema:"Filter results by name or ID"`
	Limit  *int   `json:"limit,omitempty" jsonschema:"Number of results to return (default 10, 0=100, maximum 100)"`
	Offset int    `json:"offset,omitempty" jsonschema:"Number of results to skip (default 0)"`
}
type workspaceListInput struct {
	listInput
	ProjectName string `json:"project_name,omitempty" jsonschema:"Filter workspaces by the project name they belong to"`
}
type workspaceInput struct {
	WorkspaceID string `json:"workspace_id,omitempty"`
}
type branchListInput struct {
	listInput
	WorkspaceID string `json:"workspace_id,omitempty" jsonschema:"Workspace ID"`
}
type branchInput struct {
	WorkspaceID string `json:"workspace_id,omitempty"`
	BranchID    string `json:"branch_id,omitempty"`
}

func registerCommonTools(s *mcp.Server, state *server) {
	mcp.AddTool(s, &mcp.Tool{Name: "workspaces_list", Description: "List one page of PostgreSQL workspaces. Use search and pagination to continue the results."},
		func(ctx context.Context, _ *mcp.CallToolRequest, in workspaceListInput) (*mcp.CallToolResult, any, error) {
			page, err := normalizeListInput(in.listInput)
			if err != nil {
				return toolError(err)
			}
			client, err := state.client(ctx)
			if err != nil {
				return toolError(err)
			}
			result, err := client.ListWorkspaces(ctx, volcengine.ListWorkspacesParams{
				ProjectName: in.ProjectName,
				Search:      in.Search,
				Limit:       page.Limit,
				Offset:      page.Offset,
			})
			if err != nil {
				return toolError(err)
			}
			return resultJSON(newWorkspacesPage(result, page))
		})
	mcp.AddTool(s, &mcp.Tool{Name: "workspace_get", Description: "Get PostgreSQL workspace details."},
		func(ctx context.Context, _ *mcp.CallToolRequest, in workspaceInput) (*mcp.CallToolResult, any, error) {
			id, err := state.workspaceID(in.WorkspaceID)
			if err != nil {
				return toolError(err)
			}
			client, err := state.client(ctx)
			if err != nil {
				return toolError(err)
			}
			result, err := client.DescribeWorkspaceDetail(ctx, id)
			if err != nil {
				return toolError(err)
			}
			return resultJSON(result)
		})
	mcp.AddTool(s, &mcp.Tool{Name: "branches_list", Description: "List one page of branches in a PostgreSQL workspace. Use search and pagination to continue the results."},
		func(ctx context.Context, _ *mcp.CallToolRequest, in branchListInput) (*mcp.CallToolResult, any, error) {
			id, err := state.workspaceID(in.WorkspaceID)
			if err != nil {
				return toolError(err)
			}
			page, err := normalizeListInput(in.listInput)
			if err != nil {
				return toolError(err)
			}
			client, err := state.client(ctx)
			if err != nil {
				return toolError(err)
			}
			result, err := client.DescribeBranches(ctx, volcengine.DescribeBranchesParams{
				WorkspaceID: id,
				Search:      in.Search,
				Limit:       page.Limit,
				Offset:      page.Offset,
			})
			if err != nil {
				return toolError(err)
			}
			return resultJSON(newBranchesPage(result, page))
		})
	mcp.AddTool(s, &mcp.Tool{Name: "branch_get", Description: "Get PostgreSQL branch details."},
		func(ctx context.Context, _ *mcp.CallToolRequest, in branchInput) (*mcp.CallToolResult, any, error) {
			ws, err := state.workspaceID(in.WorkspaceID)
			if err != nil {
				return toolError(err)
			}
			if in.BranchID == "" {
				return toolError(fmt.Errorf("branch_id is required"))
			}
			client, err := state.client(ctx)
			if err != nil {
				return toolError(err)
			}
			result, err := client.DescribeBranchDetail(ctx, ws, in.BranchID)
			if err != nil {
				return toolError(err)
			}
			return resultJSON(result)
		})
	mcp.AddTool(s, &mcp.Tool{Name: "computes_list", Description: "List PostgreSQL branch computes."},
		func(ctx context.Context, _ *mcp.CallToolRequest, in branchInput) (*mcp.CallToolResult, any, error) {
			ws, err := state.workspaceID(in.WorkspaceID)
			if err != nil {
				return toolError(err)
			}
			client, err := state.client(ctx)
			if err != nil {
				return toolError(err)
			}
			branch, err := client.ResolveDefaultBranchID(ctx, ws, in.BranchID)
			if err != nil {
				return toolError(err)
			}
			result, err := client.DescribeComputes(ctx, ws, branch, volcengine.ServiceTypeDatabase)
			if err != nil {
				return toolError(err)
			}
			return resultJSON(result)
		})
	mcp.AddTool(s, &mcp.Tool{Name: "databases_list", Description: "List databases on a PostgreSQL branch."},
		func(ctx context.Context, _ *mcp.CallToolRequest, in branchInput) (*mcp.CallToolResult, any, error) {
			ws, err := state.workspaceID(in.WorkspaceID)
			if err != nil {
				return toolError(err)
			}
			client, err := state.client(ctx)
			if err != nil {
				return toolError(err)
			}
			branch, err := client.ResolveDefaultBranchID(ctx, ws, in.BranchID)
			if err != nil {
				return toolError(err)
			}
			result, err := client.DescribeDatabases(ctx, volcengine.DescribeDatabasesParams{WorkspaceID: ws, BranchID: branch})
			if err != nil {
				return toolError(err)
			}
			return resultJSON(result)
		})
	mcp.AddTool(s, &mcp.Tool{Name: "roles_list", Description: "List database roles on a PostgreSQL branch."},
		func(ctx context.Context, _ *mcp.CallToolRequest, in branchInput) (*mcp.CallToolResult, any, error) {
			ws, err := state.workspaceID(in.WorkspaceID)
			if err != nil {
				return toolError(err)
			}
			client, err := state.client(ctx)
			if err != nil {
				return toolError(err)
			}
			branch, err := client.ResolveDefaultBranchID(ctx, ws, in.BranchID)
			if err != nil {
				return toolError(err)
			}
			result, err := client.DescribeDBAccounts(ctx, volcengine.DescribeDBAccountsParams{WorkspaceID: ws, BranchID: branch})
			if err != nil {
				return toolError(err)
			}
			return resultJSON(result)
		})
	mcp.AddTool(s, &mcp.Tool{Name: "operations_list", Description: "List PostgreSQL workspace operations."},
		func(ctx context.Context, _ *mcp.CallToolRequest, in workspaceInput) (*mcp.CallToolResult, any, error) {
			client, err := state.client(ctx)
			if err != nil {
				return toolError(err)
			}
			result, err := client.DescribeOperations(ctx, volcengine.DescribeOperationsParams{WorkspaceID: firstNonEmpty(in.WorkspaceID, state.opts.WorkspaceID)})
			if err != nil {
				return toolError(err)
			}
			return resultJSON(result)
		})
}
