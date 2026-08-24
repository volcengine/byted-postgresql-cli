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

func TestDescribeOperationsFiltersByWorkspace(t *testing.T) {
	var requests []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		requests = append(requests, string(body))
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"Result":{"Total":0,"Operations":[]}}`))
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

	if _, err := client.DescribeOperations(context.Background(), DescribeOperationsParams{
		WorkspaceID: "ws-1",
		Limit:       5,
	}); err != nil {
		t.Fatal(err)
	}
	if len(requests) != 1 {
		t.Fatalf("got %d requests, want one operations request", len(requests))
	}

	var operationsRequest map[string]any
	if err := json.Unmarshal([]byte(requests[0]), &operationsRequest); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(requests[0], "ProjectName") {
		t.Fatalf("operations request must not include project scope: %s", requests[0])
	}
	if operationsRequest["Filters"] == nil {
		t.Fatalf("operations request missing workspace filter: %s", requests[0])
	}
}
