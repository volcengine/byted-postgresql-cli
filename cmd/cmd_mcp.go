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
	"github.com/spf13/cobra"

	"github.com/volcengine/byted-postgresql-cli/internal/mcp"
)

func newMCPCmd(ctx ProviderContext) *cobra.Command {
	var workspace string
	var readOnly bool
	serve := &cobra.Command{
		Use:   "serve",
		Short: "Start the PostgreSQL MCP server over stdio",
		RunE: func(cmd *cobra.Command, args []string) error {
			return mcp.Serve(cmd.Context(), ctx.Provider, mcp.Options{
				WorkspaceID: firstNonEmpty(workspace, ""),
				ReadOnly:    readOnly,
			})
		},
	}
	serve.Flags().StringVar(&workspace, "workspace-id", "", "Hard-scope tools to one workspace")
	serve.Flags().BoolVar(&readOnly, "read-only", false, "Expose only read-only tools")
	cmd := &cobra.Command{Use: "mcp", Short: "Model Context Protocol server for PostgreSQL"}
	cmd.AddCommand(serve)
	return cmd
}
