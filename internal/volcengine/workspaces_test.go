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

package volcengine

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// newRecordingClient returns a client pointed at a stub gateway plus a pointer
// to the last request body it received.
func newRecordingClient(t *testing.T, response string) (*Client, *string) {
	t.Helper()
	body := new(string)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		payload, _ := io.ReadAll(r.Body)
		*body = string(payload)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(response))
	}))
	t.Cleanup(server.Close)

	client, err := NewClient(Config{
		AccessKeyID:     "test-ak",
		SecretAccessKey: "test-sk",
		Region:          DefaultRegion,
		Endpoint:        server.URL,
	})
	if err != nil {
		t.Fatal(err)
	}
	return client, body
}

// Region selects the signed AIDAP endpoint. DescribeWorkspaces does not need a
// RegionId request filter because the endpoint is already region-scoped.
func TestListWorkspacesSendsNoRegionFilter(t *testing.T) {
	client, body := newRecordingClient(t, `{"Result":{"Total":0,"Workspaces":[]}}`)

	if _, err := client.ListWorkspaces(context.Background(), ListWorkspacesParams{}); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(*body, "RegionId") {
		t.Fatalf("request body must not filter on RegionId: %s", *body)
	}
	// The engine filter must use the documented DescribeWorkspaces filter Name
	// "DBEngineVersion" (value "PostgreSQL_17") so the gateway filters by engine
	// before paginating and Total counts only PostgreSQL; the undocumented
	// "EngineType" filter Name is silently ignored by the gateway.
	if !strings.Contains(*body, `"Name":"DBEngineVersion"`) || !strings.Contains(*body, `"Value":"PostgreSQL_17"`) {
		t.Fatalf("request body missing DBEngineVersion=PostgreSQL_17 engine filter: %s", *body)
	}
}

// CreateWorkspace mirrors the SDK request: name and engine only. The signed
// endpoint already selects the region and project scope, so neither is part of
// the action body.
func TestCreateWorkspaceSendsNoRegion(t *testing.T) {
	client, body := newRecordingClient(t, `{"Result":{"WorkspaceId":"ws-1"}}`)

	_, err := client.CreateWorkspace(context.Background(), CreateWorkspaceParams{
		WorkspaceName: "PostgreSQL-ctcoq4",
	})
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(*body, "Region") {
		t.Fatalf("create request must not carry a region: %s", *body)
	}
	var req map[string]any
	if err := json.Unmarshal([]byte(*body), &req); err != nil {
		t.Fatalf("decode request body %q: %v", *body, err)
	}
	if _, exists := req["ProjectName"]; exists || req["WorkspaceName"] != "PostgreSQL-ctcoq4" {
		t.Fatalf("unexpected create request: %s", *body)
	}
	if req["EngineVersion"] != "PostgreSQL_17" {
		t.Fatalf("create request must pin PostgreSQL_17: %s", *body)
	}
}

func TestCreateWorkspaceSendsProjectNameWhenSpecified(t *testing.T) {
	client, body := newRecordingClient(t, `{"Result":{"WorkspaceId":"ws-1"}}`)
	_, err := client.CreateWorkspace(context.Background(), CreateWorkspaceParams{
		WorkspaceName: "PostgreSQL-project",
		ProjectName:   "gongna",
	})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(*body, `"ProjectName":"gongna"`) {
		t.Fatalf("request body missing project name: %s", *body)
	}
}

// One region-scoped call means --limit/--offset reach the gateway verbatim.
func TestListWorkspacesPassesPaginationThrough(t *testing.T) {
	client, body := newRecordingClient(t, `{"Result":{"Total":0,"Workspaces":[]}}`)

	if _, err := client.ListWorkspaces(context.Background(), ListWorkspacesParams{Limit: 7, Offset: 21}); err != nil {
		t.Fatal(err)
	}
	var req struct {
		Limit  int `json:"Limit"`
		Offset int `json:"Offset"`
	}
	if err := json.Unmarshal([]byte(*body), &req); err != nil {
		t.Fatalf("decode request body %q: %v", *body, err)
	}
	if req.Limit != 7 || req.Offset != 21 {
		t.Fatalf("got Limit=%d Offset=%d, want 7/21: %s", req.Limit, req.Offset, *body)
	}
}

// One region-scoped call returns every workspace the caller can see; the
// gateway's own RegionId on each item is preserved rather than filtered again.
func TestListWorkspacesKeepsEveryResult(t *testing.T) {
	client, _ := newRecordingClient(t, `{"Result":{"Total":2,"Workspaces":[
		{"WorkspaceId":"ws-1","RegionId":"China-North"},
		{"WorkspaceId":"ws-2","RegionId":"China-BOE"}
	]}}`)

	result, err := client.ListWorkspaces(context.Background(), ListWorkspacesParams{})
	if err != nil {
		t.Fatal(err)
	}
	if result.Total != 2 || len(result.Workspaces) != 2 {
		t.Fatalf("got Total=%d len=%d, want 2/2", result.Total, len(result.Workspaces))
	}
	if result.Workspaces[0].WorkspaceID != "ws-1" || result.Workspaces[1].WorkspaceID != "ws-2" {
		t.Fatalf("workspaces not preserved: %+v", result.Workspaces)
	}
}

func TestResolveDefaultBranchIDUsesExplicitBranchWithoutRequest(t *testing.T) {
	client, body := newRecordingClient(t, `{"Result":{"Branch":{"BranchId":"default-branch"}}}`)

	got, err := client.ResolveDefaultBranchID(context.Background(), "ws-1", "explicit-branch")
	if err != nil {
		t.Fatal(err)
	}
	if got != "explicit-branch" {
		t.Fatalf("got branch %q, want explicit-branch", got)
	}
	if *body != "" {
		t.Fatalf("explicit branch should not make a request, got %s", *body)
	}
}

func TestResolveDefaultBranchIDCallsDescribeDefaultBranch(t *testing.T) {
	client, body := newRecordingClient(t, `{"Result":{"Branch":{"BranchId":"default-branch"}}}`)

	got, err := client.ResolveDefaultBranchID(context.Background(), "ws-1", "")
	if err != nil {
		t.Fatal(err)
	}
	if got != "default-branch" {
		t.Fatalf("got branch %q, want default-branch", got)
	}
	if !strings.Contains(*body, `"WorkspaceId":"ws-1"`) {
		t.Fatalf("request missing workspace id: %s", *body)
	}
}

func TestResolveDefaultBranchIDRejectsEmptyAPIResult(t *testing.T) {
	client, _ := newRecordingClient(t, `{"Result":{"Branch":{}}}`)

	if _, err := client.ResolveDefaultBranchID(context.Background(), "ws-1", ""); err == nil {
		t.Fatal("expected an error when the API returns no default branch id")
	}
}
