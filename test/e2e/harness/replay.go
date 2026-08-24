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

package harness

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
)

type ReplayServer struct {
	server   *httptest.Server
	mu       sync.Mutex
	requests []string
}

func NewReplayServer(t *testing.T) *ReplayServer {
	t.Helper()
	replay := &ReplayServer{}
	replay.server = httptest.NewServer(http.HandlerFunc(replay.handle))
	t.Cleanup(replay.server.Close)
	return replay
}

func (r *ReplayServer) URL() string {
	return r.server.URL
}

func (r *ReplayServer) Requests() []string {
	r.mu.Lock()
	defer r.mu.Unlock()
	return append([]string(nil), r.requests...)
}

func (r *ReplayServer) handle(w http.ResponseWriter, request *http.Request) {
	body, _ := io.ReadAll(request.Body)
	r.mu.Lock()
	r.requests = append(r.requests, string(body))
	r.mu.Unlock()

	action := request.URL.Query().Get("Action")
	if action == "" {
		var payload map[string]any
		_ = json.Unmarshal(body, &payload)
		if value, ok := payload["Action"].(string); ok {
			action = value
		}
	}

	response := `{"Result":{}}`
	switch {
	case strings.Contains(action, "DescribeWorkspaces"):
		response = `{"Result":{"Total":1,"Workspaces":[{"WorkspaceId":"ws-e2e-1","WorkspaceName":"e2e-workspace","RegionId":"cn-beijing","Status":"Running","EngineType":"PostgreSQL","EngineVersion":"PostgreSQL_17"}]}}`
	case strings.Contains(action, "DescribeBranches"):
		response = `{"Result":{"Total":1,"WorkspaceName":"e2e-workspace","Branches":[{"BranchId":"br-e2e-1","BranchName":"main","BranchStatus":"Running","Default":true,"Protected":false}]}}`
	case strings.Contains(action, "DescribeDatabases"):
		response = `{"Result":{"Total":1,"Databases":[{"WorkspaceId":"ws-e2e-1","BranchId":"br-e2e-1","DatabaseName":"postgres","DatabaseOwner":"postgres","DatabaseDesc":"default"}]}}`
	case strings.Contains(action, "DescribeOperations"):
		response = `{"Result":{"Total":0,"Operations":[]}}`
	default:
		http.Error(w, `{"Message":"replay fixture missing for action `+action+`"}`, http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_, _ = io.WriteString(w, response)
}

func RequireRequestContains(t *testing.T, server *ReplayServer, fragment string) {
	t.Helper()
	for _, request := range server.Requests() {
		if strings.Contains(request, fragment) {
			return
		}
	}
	t.Fatalf("replay requests do not contain %q: %v", fragment, server.Requests())
}
