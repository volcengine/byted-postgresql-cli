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

package update

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestIsNewer(t *testing.T) {
	tests := []struct {
		latest, current string
		want            bool
	}{
		{"v0.0.2", "0.0.1", true},
		{"0.0.1", "v0.0.1", false},
		{"v0.0.10", "v0.0.2", true},
		{"v0.0.1", "v0.0.1-4-gabc123", false},
		{"latest", "v0.0.1", false},
		{"v0.0.2", "dev", true},
	}
	for _, tt := range tests {
		if got := IsNewer(tt.latest, tt.current); got != tt.want {
			t.Errorf("IsNewer(%q, %q) = %v, want %v", tt.latest, tt.current, got, tt.want)
		}
	}
}

func TestLatestRelease(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/@byted-postgresql/cli/latest" {
			t.Fatalf("path = %q", r.URL.Path)
		}
		_, _ = io.WriteString(w, `{"version":"0.0.2"}`)
	}))
	defer server.Close()

	originalRegistry := NpmRegistry
	originalClient := httpClient
	NpmRegistry = server.URL
	httpClient = server.Client()
	t.Cleanup(func() {
		NpmRegistry = originalRegistry
		httpClient = originalClient
	})
	got, err := LatestRelease(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if got != "v0.0.2" {
		t.Fatalf("LatestRelease() = %q, want v0.0.2", got)
	}
}

// updateHarness stubs the network, npm, and the installed binary so Run can be
// exercised end-to-end offline. It also isolates HOME for path-related checks.
type updateHarness struct {
	npmArgs    []string
	reportVer  string // what the fake installed binary reports for --version
	installErr error
	warmCalls  []string
	registry   string
}

func newUpdateHarness(t *testing.T, latest string) *updateHarness {
	t.Helper()
	h := &updateHarness{reportVer: strings.TrimPrefix(latest, "v")}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/@byted-postgresql/cli/latest" {
			t.Errorf("path = %q", r.URL.Path)
		}
		_, _ = io.WriteString(w, `{"version":"`+strings.TrimPrefix(latest, "v")+`"}`)
	}))
	t.Cleanup(server.Close)

	origRegistry, origClient := NpmRegistry, httpClient
	origLook, origRun, origVer, origWarm, origPrefix := lookPath, runCommand, binaryVersion, warmCache, npmPrefix
	NpmRegistry = server.URL
	httpClient = server.Client()
	h.registry = server.URL
	lookPath = func(string) (string, error) { return "npm", nil }
	npmPrefix = func(string) (string, error) { return "/npm/global", nil }
	runCommand = func(_ context.Context, _, _ io.Writer, _ string, args ...string) error {
		h.npmArgs = args
		return h.installErr
	}
	binaryVersion = func(context.Context, string) (string, error) { return h.reportVer, nil }
	warmCache = func(_ context.Context, bin string) error {
		h.warmCalls = append(h.warmCalls, bin)
		return nil
	}
	t.Cleanup(func() {
		NpmRegistry, httpClient = origRegistry, origClient
		lookPath, runCommand, binaryVersion, warmCache, npmPrefix = origLook, origRun, origVer, origWarm, origPrefix
	})
	// Isolate HOME so no-state fallback + state writes stay in a temp dir.
	t.Setenv("HOME", t.TempDir())
	return h
}

func TestRunInstallsToNpmGlobalPrefix(t *testing.T) {
	h := newUpdateHarness(t, "v0.0.3")

	var out, errOut strings.Builder
	if err := Run(context.Background(), "0.0.2", false, false, &out, &errOut); err != nil {
		t.Fatalf("Run() error: %v", err)
	}
	joined := strings.Join(h.npmArgs, " ")
	if strings.Contains(joined, "--prefix") {
		t.Fatalf("npm args should use npm's global prefix: %q", joined)
	}
	if !strings.Contains(out.String(), "Successfully updated") {
		t.Fatalf("expected success, got out=%q err=%q", out.String(), errOut.String())
	}
	if len(h.warmCalls) != 1 || h.warmCalls[0] != "/npm/global/bin/byted-postgresql-cli" {
		t.Fatalf("warm-cache calls = %#v", h.warmCalls)
	}
}

func TestRunDoesNotRequireInstallState(t *testing.T) {
	h := newUpdateHarness(t, "v0.0.3")
	var out, errOut strings.Builder
	if err := Run(context.Background(), "0.0.2", false, false, &out, &errOut); err != nil {
		t.Fatalf("Run() should not error without state: %v", err)
	}
	joined := strings.Join(h.npmArgs, " ")
	if strings.Contains(joined, "--prefix") {
		t.Fatalf("global update should not pass --prefix: %q", joined)
	}
}

func TestRunCheckDoesNotRequireNpm(t *testing.T) {
	newUpdateHarness(t, "v0.0.3")
	lookPath = func(string) (string, error) {
		return "", errors.New("npm is unavailable")
	}
	npmPrefix = func(string) (string, error) {
		return "", errors.New("npm prefix failed")
	}

	var out, errOut strings.Builder
	if err := Run(context.Background(), "0.0.2", true, false, &out, &errOut); err != nil {
		t.Fatalf("check should not require npm: %v", err)
	}
	if !strings.Contains(out.String(), "Update available") {
		t.Fatalf("expected update availability output, got %q", out.String())
	}
}

func TestRunFailsVerificationOnStaleBinary(t *testing.T) {
	h := newUpdateHarness(t, "v0.0.3")
	h.reportVer = "0.0.2" // npm "succeeds" but the on-PATH binary is unchanged

	var out, errOut strings.Builder
	err := Run(context.Background(), "0.0.2", false, false, &out, &errOut)
	if err == nil {
		t.Fatal("expected verification failure, got nil")
	}
	msg := err.Error()
	if !strings.Contains(msg, "verification failed") || !strings.Contains(msg, "expected v0.0.3") {
		t.Fatalf("error missing verification detail: %q", msg)
	}
	if strings.Contains(msg, "--prefix") {
		t.Fatalf("recovery command should use global npm prefix: %q", msg)
	}
}

func TestRunContinuesWhenWarmCacheFails(t *testing.T) {
	h := newUpdateHarness(t, "v0.0.3")
	h.warmCalls = nil
	warmCache = func(context.Context, string) error {
		return context.DeadlineExceeded
	}
	t.Cleanup(func() {
		h.warmCalls = nil
	})
	var out, errOut strings.Builder
	if err := Run(context.Background(), "0.0.2", false, false, &out, &errOut); err != nil {
		t.Fatalf("warm-cache failure should not fail update: %v", err)
	}
	if !strings.Contains(errOut.String(), "warm skipped") {
		t.Fatalf("missing warm-cache warning: %q", errOut.String())
	}
}

func TestRunNpmFailurePrintsPrefixAwareRecovery(t *testing.T) {
	h := newUpdateHarness(t, "v0.0.3")
	h.installErr = context.DeadlineExceeded // any non-nil install error
	var out, errOut strings.Builder
	err := Run(context.Background(), "0.0.2", false, false, &out, &errOut)
	if err == nil {
		t.Fatal("expected npm install failure")
	}
	if strings.Contains(err.Error(), "--prefix") {
		t.Fatalf("npm failure recovery should use global npm prefix: %q", err.Error())
	}
}

func TestRunDoesNotWriteInstallState(t *testing.T) {
	newUpdateHarness(t, "v0.0.3")
	var out, errOut strings.Builder
	if err := Run(context.Background(), "0.0.2", false, false, &out, &errOut); err != nil {
		t.Fatal(err)
	}
	statePath := filepath.Join(os.Getenv("HOME"), ".volcengine-postgresql", "config", "install-state.json")
	if _, err := os.Stat(statePath); !os.IsNotExist(err) {
		t.Fatalf("update should not write install-state.json, stat error = %v", err)
	}
}
