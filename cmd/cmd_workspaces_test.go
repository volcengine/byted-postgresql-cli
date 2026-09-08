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
	"io"
	"strings"
	"testing"

	"github.com/volcengine/byted-postgresql-cli/internal/volcengine"
)

func TestValidateWorkspaceName(t *testing.T) {
	valid := []string{"e2e-d", "ab", "my_pg17", "测试库", strings.Repeat("a", 64)}
	for _, name := range valid {
		if err := validateWorkspaceName(name); err != nil {
			t.Fatalf("validateWorkspaceName(%q) = %v, want nil", name, err)
		}
	}
	invalid := []string{"a", "", "1abc", "-abc", "_abc", "has space", "bad!", strings.Repeat("a", 65)}
	for _, name := range invalid {
		if err := validateWorkspaceName(name); err == nil {
			t.Fatalf("validateWorkspaceName(%q) = nil, want error", name)
		}
	}
}

func TestValidateBranchNameRejectsTooLongName(t *testing.T) {
	if err := validateBranchName(strings.Repeat("a", 65)); err == nil {
		t.Fatal("validateBranchName() accepted a 65-character branch name")
	}
}

func TestValidateSuspendTimeout(t *testing.T) {
	for _, ok := range []int{-1, 300, 3600, 604800} {
		if err := validateSuspendTimeout(ok); err != nil {
			t.Fatalf("validateSuspendTimeout(%d) = %v, want nil", ok, err)
		}
	}
	for _, bad := range []int{0, 1, 299, 604801, -2} {
		if err := validateSuspendTimeout(bad); err == nil {
			t.Fatalf("validateSuspendTimeout(%d) = nil, want error", bad)
		}
	}
}

func TestValidateComputeUnits(t *testing.T) {
	cases := []struct {
		minCU, maxCU float64
		wantErr      bool
	}{
		{1, 8, false},    // ratio exactly 8
		{2, 2, false},    // fixed size
		{0.25, 2, false}, // ratio 8 at the low end
		{1, 9, true},     // ratio over 8
		{4, 2, true},     // max below min
		{0.5, 0.1, true}, // max below the minimum and below min
		{0, 4, true},     // below the minimum CU
		{2, 64, true},    // above the maximum CU
	}
	for _, c := range cases {
		err := validateComputeUnits(c.minCU, c.maxCU)
		if (err != nil) != c.wantErr {
			t.Fatalf("validateComputeUnits(%g, %g) = %v, wantErr %v", c.minCU, c.maxCU, err, c.wantErr)
		}
	}
}

// runWorkspacesCmd drives the real root command far enough to hit argument and
// flag validation. Every case here must fail before a gateway client is built,
// so no network or credentials are involved.
func runWorkspacesCmd(t *testing.T, args ...string) error {
	t.Helper()
	cmd := newRootCmd()
	cmd.SetOut(io.Discard)
	cmd.SetErr(io.Discard)
	cmd.SetIn(strings.NewReader(""))
	cmd.SetArgs(append([]string{"--config-dir", t.TempDir()}, args...))
	return cmd.Execute()
}

func TestWorkspacesFlagValidation(t *testing.T) {
	cases := []struct {
		name    string
		args    []string
		wantErr string
	}{
		{
			name:    "create rejects a positional name",
			args:    []string{"workspaces", "create", "foo"},
			wantErr: "unknown command",
		},
		{
			name:    "create rejects an invalid name",
			args:    []string{"workspaces", "create", "--name", "1bad name"},
			wantErr: "must start with a letter",
		},
		{
			name:    "create rejects a missing name",
			args:    []string{"workspaces", "create"},
			wantErr: `required flag(s) "name" not set`,
		},
		{
			name:    "create rejects a blank name",
			args:    []string{"workspaces", "create", "--name", "  "},
			wantErr: "--name is required",
		},
		{
			name:    "create rejects an out-of-range suspend timeout",
			args:    []string{"workspaces", "create", "--name", "pg-demo", "--suspend-timeout", "5"},
			wantErr: "--suspend-timeout must be",
		},
		{
			name:    "list rejects --all with --limit",
			args:    []string{"workspaces", "list", "--all", "--limit", "5"},
			wantErr: "if any flags in the group",
		},
		{
			name:    "list rejects an excessive limit",
			args:    []string{"workspaces", "list", "--limit", "101"},
			wantErr: "--limit must be less than or equal to 100",
		},
		{
			name:    "delete refuses without confirmation",
			args:    []string{"workspaces", "delete", "--workspace-id", "ws-1"},
			wantErr: "pass --yes in non-interactive mode",
		},
		{
			name:    "delete rejects an empty id",
			args:    []string{"workspaces", "delete", "--workspace-id", "  "},
			wantErr: "workspace id cannot be empty",
		},
		{
			name:    "rename requires a new name",
			args:    []string{"workspaces", "rename", "--workspace-id", "ws-1"},
			wantErr: "--name is required",
		},
		{
			name:    "rename rejects an invalid new name",
			args:    []string{"workspaces", "rename", "--workspace-id", "ws-1", "--name", "bad name"},
			wantErr: "may only contain",
		},
		{
			name:    "deletion-protection requires a direction",
			args:    []string{"workspaces", "deletion-protection", "--workspace-id", "ws-1"},
			wantErr: "exactly one of --enable or --disable",
		},
		{
			name:    "compute-settings requires something to change",
			args:    []string{"workspaces", "compute-settings", "--workspace-id", "ws-1"},
			wantErr: "nothing to change",
		},
		{
			name:    "compute-settings validates the suspend timeout",
			args:    []string{"workspaces", "compute-settings", "--workspace-id", "ws-1", "--suspend-timeout", "10"},
			wantErr: "--suspend-timeout must be",
		},
		{
			name:    "settings requires something to change",
			args:    []string{"workspaces", "settings", "--workspace-id", "ws-1"},
			wantErr: "nothing to change",
		},
		{
			name:    "settings validates the retention window",
			args:    []string{"workspaces", "settings", "--workspace-id", "ws-1", "--history-retention-hours", "0"},
			wantErr: "--history-retention-hours must be between",
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			err := runWorkspacesCmd(t, c.args...)
			if err == nil || !strings.Contains(err.Error(), c.wantErr) {
				t.Fatalf("error = %v, want it to contain %q", err, c.wantErr)
			}
		})
	}
}

func TestWorkspacesCreateExposesResourceProject(t *testing.T) {
	cmd := newWorkspacesCreateCmd()
	if cmd.Flags().Lookup("resource-project") == nil {
		t.Fatal("workspaces create should expose --resource-project")
	}
}

func TestWorkspaceDeletionSummaryIncludesCascadeWarning(t *testing.T) {
	summary := workspaceDeletionSummary("ws-1", volcengine.Workspace{
		WorkspaceName: "demo",
		ProjectName:   "project-1",
	}, "cn-beijing")
	for _, want := range []string{
		`Delete workspace "demo" (ws-1)?`,
		"Project: project-1",
		"Region: cn-beijing",
		"cannot be undone",
		"branches, computes, databases, and endpoints",
		"--yes",
	} {
		if !strings.Contains(summary, want) {
			t.Fatalf("workspaceDeletionSummary() missing %q: %s", want, summary)
		}
	}
}

func TestWorkspacesCreateRequiresExplicitNameFlag(t *testing.T) {
	cmd := newWorkspacesCreateCmd()
	if err := cmd.Args(cmd, []string{"e2e-d"}); err == nil {
		t.Fatal("create must reject a positional workspace name")
	}
	if flag := cmd.Flags().Lookup("name"); flag == nil {
		t.Fatal("create must expose --name")
	}
}

func TestWorkspacesCreateHelpUsesNameFlag(t *testing.T) {
	cmd := newWorkspacesCreateCmd()
	if cmd.Flags().Lookup("name") == nil {
		t.Fatal("create should expose --name")
	}
}

func TestWorkspaceIDFromFlagAcceptsFlag(t *testing.T) {
	cmd := newWorkspacesGetCmd()
	if err := cmd.Flags().Set("workspace-id", "ws-1"); err != nil {
		t.Fatalf("set workspace-id: %v", err)
	}
	got, err := workspaceIDFromFlag(cmd)
	if err != nil {
		t.Fatalf("workspaceIDFromFlag() error = %v", err)
	}
	if got != "ws-1" {
		t.Fatalf("workspaceIDFromFlag() = %q, want %q", got, "ws-1")
	}
}

func TestWorkspaceIDFlagRejectsPositionalTarget(t *testing.T) {
	cmd := newWorkspacesGetCmd()
	cmd.SetArgs([]string{"ws-positional"})
	err := cmd.Execute()
	if err == nil || !strings.Contains(err.Error(), "unknown command") && !strings.Contains(err.Error(), "accepts 0 arg") {
		t.Fatalf("positional workspace id error = %v, want positional argument rejection", err)
	}
}

func TestWorkspaceIDFromFlagRejectsMissingIDBeforeCredentialResolution(t *testing.T) {
	cmd := newWorkspacesGetCmd()
	_, err := workspaceIDFromFlag(cmd)
	if err == nil || !strings.Contains(err.Error(), "no workspace selected") {
		t.Fatalf("workspaceIDFromFlag() error = %v, want non-TTY workspace selection error", err)
	}
	if strings.Contains(err.Error(), "credentials") {
		t.Fatalf("workspaceIDFromFlag() resolved credentials before reporting missing workspace: %v", err)
	}
}
