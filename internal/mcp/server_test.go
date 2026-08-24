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
	"bytes"
	"errors"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/stretchr/testify/require"

	"github.com/volcengine/byted-postgresql-cli/internal/pagination"
	"github.com/volcengine/byted-postgresql-cli/internal/volcengine"
)

func TestNormalizeListInput(t *testing.T) {
	zero := 0
	tests := []struct {
		name    string
		input   listInput
		want    listPage
		wantErr string
	}{
		{name: "defaults", want: listPage{Limit: pagination.DefaultLimit}},
		{name: "custom page", input: listInput{Limit: intPtr(50), Offset: 100}, want: listPage{Limit: 50, Offset: 100}},
		{name: "explicit zero is maximum", input: listInput{Limit: &zero}, want: listPage{Limit: pagination.MaxLimit}},
		{name: "limit too small", input: listInput{Limit: intPtr(-1)}, wantErr: "limit must be between"},
		{name: "limit too large", input: listInput{Limit: intPtr(pagination.MaxLimit + 1)}, wantErr: "limit must be between"},
		{name: "offset negative", input: listInput{Offset: -1}, wantErr: "offset must be between"},
		{name: "offset too large", input: listInput{Offset: pagination.MaxOffset + 1}, wantErr: "offset must be between"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := normalizeListInput(test.input)
			if test.wantErr != "" {
				require.ErrorContains(t, err, test.wantErr)
				return
			}
			require.NoError(t, err)
			require.Equal(t, test.want, got)
		})
	}
}

func intPtr(value int) *int {
	return &value
}

func TestResultJSONRejectsOversizedResponse(t *testing.T) {
	result, _, err := resultJSON(strings.Repeat("x", maxResultBytes))
	require.NoError(t, err)
	require.True(t, result.IsError)
	require.Contains(t, result.Content[0].(*mcp.TextContent).Text, "response limit")
}

func TestIsNormalStdioClose(t *testing.T) {
	require.True(t, isNormalStdioClose(io.EOF))
	require.True(t, isNormalStdioClose(errors.New("server is closing: EOF")))
	require.False(t, isNormalStdioClose(errors.New("server is closing: broken pipe")))
	require.False(t, isNormalStdioClose(errors.New("invalid JSON")))
}

func TestGracefulEOFReaderWaitsForResponseWrite(t *testing.T) {
	writes := make(chan struct{}, 1)
	reader := &gracefulEOFReader{
		reader: bytes.NewReader(nil),
		writes: writes,
		closed: make(chan struct{}),
	}
	writerOutput := new(bytes.Buffer)
	writer := &writeSignalWriter{writer: writerOutput, writes: writes}

	result := make(chan error, 1)
	go func() {
		_, err := reader.Read(make([]byte, 1))
		result <- err
	}()

	select {
	case err := <-result:
		t.Fatalf("reader returned before the response was written: %v", err)
	case <-time.After(stdioEOFQuietPeriod / 2):
	}

	_, err := writer.Write([]byte(`{"jsonrpc":"2.0","id":1}`))
	require.NoError(t, err)
	require.ErrorIs(t, <-result, io.EOF)
	require.Equal(t, `{"jsonrpc":"2.0","id":1}`, writerOutput.String())
}

func TestWorkspacesPageIncludesPaginationMetadata(t *testing.T) {
	page := newWorkspacesPage(volcengine.ListWorkspacesResult{
		Total:      3,
		Workspaces: []volcengine.Workspace{{WorkspaceID: "ws-1"}, {WorkspaceID: "ws-2"}},
	}, listPage{Limit: 2, Offset: 0})

	require.Equal(t, 3, page.Total)
	require.True(t, page.HasMore)
	require.NotNil(t, page.NextOffset)
	require.Equal(t, 2, *page.NextOffset)
}

func TestRegisterToolsDoesNotPanic(t *testing.T) {
	mcpServer := mcp.NewServer(&mcp.Implementation{
		Name:    "test",
		Version: "test",
	}, nil)

	require.NotPanics(t, func() {
		registerCommonTools(mcpServer, &server{
			opts:     Options{WorkspaceID: "ws-test", Provider: volcengine.ProviderVolcengine},
			provider: volcengine.ProviderVolcengine,
		})
	})
}
