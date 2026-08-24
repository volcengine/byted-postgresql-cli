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

func newLifecycleRecordingClient(t *testing.T, response string) (*Client, *string) {
	t.Helper()
	var requestBody string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		requestBody = string(body)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"ResponseMetadata":{"RequestId":"test-request"},"Result":` + response + `}`))
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
	return client, &requestBody
}

func assertRequestFields(t *testing.T, body string, fields map[string]string) {
	t.Helper()
	var request map[string]any
	if err := json.Unmarshal([]byte(body), &request); err != nil {
		t.Fatalf("decode request %q: %v", body, err)
	}
	for key, want := range fields {
		if got := request[key]; got != want {
			t.Errorf("%s = %v, want %q in request %s", key, got, want, body)
		}
	}
}

func TestModifyWorkspaceDeletionProtectionSendsEnabledState(t *testing.T) {
	client, body := newLifecycleRecordingClient(t, `{}`)
	if err := client.ModifyWorkspaceDeletionProtectionPolicy(context.Background(), "ws-1", true); err != nil {
		t.Fatal(err)
	}
	assertRequestFields(t, *body, map[string]string{
		"WorkspaceId":        "ws-1",
		"DeletionProtection": "Enabled",
	})
}

func TestModifyWorkspaceDeletionProtectionSendsDisabledState(t *testing.T) {
	client, body := newLifecycleRecordingClient(t, `{}`)
	if err := client.ModifyWorkspaceDeletionProtectionPolicy(context.Background(), "ws-1", false); err != nil {
		t.Fatal(err)
	}
	assertRequestFields(t, *body, map[string]string{
		"WorkspaceId":        "ws-1",
		"DeletionProtection": "Disabled",
	})
}

func TestGetRestoreWindowSendsWorkspaceAndBranch(t *testing.T) {
	client, body := newLifecycleRecordingClient(t, `{"WorkspaceId":"ws-1","BranchId":"br-1","WindowSizeSeconds":3600}`)
	window, err := client.GetRestoreWindow(context.Background(), "ws-1", "br-1")
	if err != nil {
		t.Fatal(err)
	}
	assertRequestFields(t, *body, map[string]string{
		"WorkspaceId": "ws-1",
		"BranchId":    "br-1",
	})
	if window.WorkspaceID != "ws-1" || window.BranchID != "br-1" || window.WindowSizeSeconds != 3600 {
		t.Fatalf("window = %+v, want mapped restore window", window)
	}
}

func TestResetDBAccountPasswordSendsAllCredentials(t *testing.T) {
	client, body := newLifecycleRecordingClient(t, `{"Success":true}`)
	if err := client.ResetDBAccountPassword(context.Background(), "ws-1", "br-1", "user_admin", "new-secret"); err != nil {
		t.Fatal(err)
	}
	assertRequestFields(t, *body, map[string]string{
		"WorkspaceId":     "ws-1",
		"BranchId":        "br-1",
		"AccountName":     "user_admin",
		"AccountPassword": "new-secret",
	})
	if strings.Contains(*body, "ProjectName") {
		t.Fatalf("password reset request unexpectedly contains project scope: %s", *body)
	}
}
