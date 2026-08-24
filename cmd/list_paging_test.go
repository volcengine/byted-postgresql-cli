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
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/volcengine/byted-postgresql-cli/internal/volcengine"
)

func TestListPageOutputContainsPaginationMetadata(t *testing.T) {
	next := 10
	output := listPageOutput{
		Items:      []volcengine.Workspace{{WorkspaceID: "ws-1"}},
		Count:      1,
		Total:      32,
		Limit:      10,
		Offset:     0,
		NextOffset: &next,
	}
	data, err := json.Marshal(output)
	if err != nil {
		t.Fatal(err)
	}
	var got map[string]any
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatal(err)
	}
	for key, want := range map[string]any{
		"count": float64(1), "total": float64(32), "limit": float64(10),
		"offset": float64(0), "next_offset": float64(10),
	} {
		if got[key] != want {
			t.Fatalf("%s = %v, want %v", key, got[key], want)
		}
	}
}

func TestWriteListPageTableSummaryDoesNotDuplicateShowingLine(t *testing.T) {
	var out bytes.Buffer
	cmd := newWorkspacesCmd(defaultProviderContext())
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	g := &Globals{Output: "table"}
	if err := writeListPage(cmd, g, []volcengine.Workspace{
		{WorkspaceID: "ws-1", WorkspaceName: "demo"},
	}, workspaceFields, 2, 1, 0, "workspaces"); err != nil {
		t.Fatal(err)
	}
	text := out.String()
	if strings.Count(text, "Showing 1 of 2 workspaces") != 1 {
		t.Fatalf("summary was duplicated: %q", text)
	}
	if !strings.Contains(text, "Showing 1 of 2 workspaces (offset=0, limit=1). 1 remaining; use --offset 1 --limit 1 to fetch the next page.") {
		t.Fatalf("next-page guidance missing: %q", text)
	}
}

func TestWriteListPageDelimitedOutputHasNoSummary(t *testing.T) {
	for _, format := range []string{"csv", "tsv"} {
		t.Run(format, func(t *testing.T) {
			var out bytes.Buffer
			cmd := newWorkspacesCmd(defaultProviderContext())
			cmd.SetOut(&out)
			g := &Globals{Output: format}
			if err := writeListPage(cmd, g, []volcengine.Workspace{
				{WorkspaceID: "ws-1", WorkspaceName: "demo"},
			}, workspaceFields, 2, 1, 0, "workspaces"); err != nil {
				t.Fatal(err)
			}
			if strings.Contains(out.String(), "Showing ") {
				t.Fatalf("delimited output contains pagination prose: %q", out.String())
			}
		})
	}
}

func TestWriteListSummaryDelimitedOutputHasNoSummary(t *testing.T) {
	for _, format := range []string{"csv", "tsv"} {
		t.Run(format, func(t *testing.T) {
			var out bytes.Buffer
			cmd := newWorkspacesCmd(defaultProviderContext())
			cmd.SetOut(&out)
			g := &Globals{Output: format}
			if err := writeListSummary(cmd, g, []volcengine.Workspace{
				{WorkspaceID: "ws-1", WorkspaceName: "demo"},
			}, workspaceFields, nil, "workspaces"); err != nil {
				t.Fatal(err)
			}
			if strings.Contains(out.String(), "Showing ") {
				t.Fatalf("delimited output contains pagination prose: %q", out.String())
			}
		})
	}
}
