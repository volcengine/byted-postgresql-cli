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
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/volcengine/byted-postgresql-cli/internal/pagination"
	"github.com/volcengine/byted-postgresql-cli/internal/volcengine"
)

const (
	maxResultBytes = 256 * 1024
)

type Options struct {
	WorkspaceID string
	ReadOnly    bool
	Provider    volcengine.Provider
}

type server struct {
	opts     Options
	provider volcengine.Provider
}

func Serve(ctx context.Context, provider volcengine.Provider, opts Options) error {
	if provider == "" {
		provider = volcengine.ProviderVolcengine
	}
	opts.Provider = provider
	spec, err := volcengine.ProviderSpecFor(provider)
	if err != nil {
		return err
	}
	s := mcp.NewServer(&mcp.Implementation{
		Name: "byted-postgresql-cli", Title: spec.DisplayName + " PostgreSQL CLI", Version: "dev",
	}, &mcp.ServerOptions{
		Instructions: fmt.Sprintf("Manage %s AIDAP PostgreSQL workspaces, branches, computes, databases, roles, and operations.", spec.DisplayName),
	})
	state := &server{opts: opts, provider: provider}
	registerCommonTools(s, state)
	err = s.Run(ctx, newStdioTransport())
	if isNormalStdioClose(err) || errors.Is(err, context.Canceled) {
		return nil
	}
	return err
}

func isNormalStdioClose(err error) bool {
	return errors.Is(err, io.EOF) || err != nil && err.Error() == "server is closing: EOF"
}

const (
	stdioEOFGracePeriod = 500 * time.Millisecond
	stdioEOFQuietPeriod = 25 * time.Millisecond
)

type stdioTransport struct {
	reader *gracefulEOFReader
	writer *writeSignalWriter
}

func newStdioTransport() *mcp.IOTransport {
	writes := make(chan struct{}, 1)
	reader := &gracefulEOFReader{
		reader: os.Stdin,
		writes: writes,
		closed: make(chan struct{}),
	}
	writer := &writeSignalWriter{writer: os.Stdout, writes: writes}
	return &mcp.IOTransport{Reader: reader, Writer: writer}
}

type gracefulEOFReader struct {
	reader io.Reader
	writes <-chan struct{}
	closed chan struct{}

	closeOnce sync.Once
}

func (r *gracefulEOFReader) Read(p []byte) (int, error) {
	n, err := r.reader.Read(p)
	if !errors.Is(err, io.EOF) {
		return n, err
	}
	if n > 0 {
		return n, nil
	}

	grace := time.NewTimer(stdioEOFGracePeriod)
	defer grace.Stop()
	quiet := (*time.Timer)(nil)
	var quietC <-chan time.Time
	for {
		select {
		case <-r.closed:
			return 0, io.EOF
		case <-r.writes:
			if quiet == nil {
				quiet = time.NewTimer(stdioEOFQuietPeriod)
			} else {
				if !quiet.Stop() {
					select {
					case <-quiet.C:
					default:
					}
				}
				quiet.Reset(stdioEOFQuietPeriod)
			}
			quietC = quiet.C
		case <-quietC:
			return 0, io.EOF
		case <-grace.C:
			return 0, io.EOF
		}
	}
}

func (r *gracefulEOFReader) Close() error {
	r.closeOnce.Do(func() {
		close(r.closed)
	})
	return nil
}

type writeSignalWriter struct {
	writer io.Writer
	writes chan<- struct{}
}

func (w *writeSignalWriter) Write(p []byte) (int, error) {
	n, err := w.writer.Write(p)
	if err == nil {
		select {
		case w.writes <- struct{}{}:
		default:
		}
	}
	return n, err
}

func (w *writeSignalWriter) Close() error {
	return nil
}

// client builds one AIDAP client from the resolved AK/SK credentials + region.
func (s *server) client(ctx context.Context) (*volcengine.Client, error) {
	cfg, err := volcengine.ResolveConfig(ctx)
	if err != nil {
		return nil, err
	}
	return volcengine.NewClient(cfg)
}

func (s *server) workspaceID(input string) (string, error) {
	id := firstNonEmpty(input, s.opts.WorkspaceID)
	if id == "" {
		return "", fmt.Errorf("workspace_id is required")
	}
	return id, nil
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

type listPage = pagination.Page

func normalizeListInput(input listInput) (listPage, error) {
	limit := 0
	if input.Limit != nil {
		limit = *input.Limit
		if limit == 0 {
			limit = pagination.MaxLimit
		}
	}
	return pagination.DefaultPolicy.Normalize(limit, input.Offset)
}

type workspacesPage struct {
	Workspaces []volcengine.Workspace `json:"workspaces"`
	Total      int                    `json:"total"`
	Limit      int                    `json:"limit"`
	Offset     int                    `json:"offset"`
	HasMore    bool                   `json:"has_more"`
	NextOffset *int                   `json:"next_offset,omitempty"`
}

func newWorkspacesPage(result volcengine.ListWorkspacesResult, page listPage) workspacesPage {
	metadata := page.Metadata(len(result.Workspaces), result.Total)
	return workspacesPage{
		Workspaces: result.Workspaces,
		Total:      result.Total,
		Limit:      page.Limit,
		Offset:     page.Offset,
		HasMore:    metadata.HasMore,
		NextOffset: metadata.NextOffset,
	}
}

type branchesPage struct {
	WorkspaceName string              `json:"workspace_name,omitempty"`
	Branches      []volcengine.Branch `json:"branches"`
	Total         int                 `json:"total"`
	Limit         int                 `json:"limit"`
	Offset        int                 `json:"offset"`
	HasMore       bool                `json:"has_more"`
	NextOffset    *int                `json:"next_offset,omitempty"`
}

func newBranchesPage(result volcengine.DescribeBranchesResult, page listPage) branchesPage {
	metadata := page.Metadata(len(result.Branches), result.Total)
	return branchesPage{
		WorkspaceName: result.WorkspaceName,
		Branches:      result.Branches,
		Total:         result.Total,
		Limit:         page.Limit,
		Offset:        page.Offset,
		HasMore:       metadata.HasMore,
		NextOffset:    metadata.NextOffset,
	}
}

func resultJSON(value any) (*mcp.CallToolResult, any, error) {
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return nil, nil, err
	}
	if len(data) > maxResultBytes {
		return toolError(fmt.Errorf("result exceeds the %d-byte MCP response limit; reduce limit or use search and pagination", maxResultBytes))
	}
	return &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: string(data)}}}, nil, nil
}

func toolError(err error) (*mcp.CallToolResult, any, error) {
	return &mcp.CallToolResult{IsError: true, Content: []mcp.Content{&mcp.TextContent{Text: err.Error()}}}, nil, nil
}
