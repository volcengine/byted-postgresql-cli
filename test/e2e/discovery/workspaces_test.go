// Copyright (c) 2026 Beijing Volcano Engine Technology Co., Ltd.
// SPDX-License-Identifier: MIT

package discovery

import (
	"encoding/json"
	"os"
	"strings"
	"testing"

	"github.com/volcengine/byted-postgresql-cli/test/e2e/harness"
)

func TestWorkspacesList(t *testing.T) {
	config := harness.RequireReplay(t)
	payload := config.RequireJSON(t, config.Run(t, "--output", "json", "workspaces", "list"))
	if payload["total"] != float64(1) {
		t.Fatalf("workspace payload = %#v, want total 1", payload)
	}
}

func TestLiveReadOnlyWorkflows(t *testing.T) {
	config := harness.New(t)
	config.RequireLive(t)
	workspaceID := strings.TrimSpace(os.Getenv("VOLCENGINE_E2E_WORKSPACE_ID"))
	branchID := strings.TrimSpace(os.Getenv("VOLCENGINE_E2E_BRANCH_ID"))
	if workspaceID == "" || branchID == "" {
		t.Fatal("live E2E requires VOLCENGINE_E2E_WORKSPACE_ID and VOLCENGINE_E2E_BRANCH_ID")
	}
	args := []string{"--region", os.Getenv("VOLCENGINE_REGION"), "--output", "json"}
	tests := []struct {
		name     string
		command  []string
		validate func(*testing.T, any)
	}{
		{
			name:    "projects_list",
			command: []string{"projects", "list"},
			validate: func(t *testing.T, value any) {
				if _, ok := value.([]any); !ok {
					t.Fatalf("projects list JSON = %T, want array", value)
				}
			},
		},
		{
			name:     "workspaces_list",
			command:  []string{"workspaces", "list"},
			validate: validateObject,
		},
		{
			name:    "workspaces_get",
			command: []string{"workspaces", "get", workspaceID},
			validate: func(t *testing.T, value any) {
				object := requireObject(t, value)
				if object["workspace_id"] != workspaceID {
					t.Fatalf("workspace get workspace_id = %v, want %s", object["workspace_id"], workspaceID)
				}
			},
		},
		{
			name:     "branches_list",
			command:  []string{"branches", "list", "--workspace-id", workspaceID},
			validate: validateObject,
		},
		{
			name:     "databases_list",
			command:  []string{"databases", "list", "--workspace-id", workspaceID, "--branch-id", branchID},
			validate: validateObject,
		},
		{
			name:     "computes_list",
			command:  []string{"computes", "list", "--workspace-id", workspaceID, "--branch-id", branchID},
			validate: validateObject,
		},
		{
			name:     "roles_list",
			command:  []string{"roles", "list", "--workspace-id", workspaceID, "--branch-id", branchID},
			validate: validateObject,
		},
		{
			name:     "endpoints_list",
			command:  []string{"endpoints", "list", "--workspace-id", workspaceID, "--branch-id", branchID},
			validate: validateObject,
		},
		{
			name:     "operations_list",
			command:  []string{"operations", "list", "--workspace-id", workspaceID, "--branch-id", branchID},
			validate: validateObject,
		},
		{
			name:    "network_get",
			command: []string{"network", "get", "--workspace-id", workspaceID, "--branch-id", branchID},
			validate: func(t *testing.T, value any) {
				object := requireObject(t, value)
				if object["workspace_id"] != workspaceID {
					t.Fatalf("network get workspace_id = %v, want %s", object["workspace_id"], workspaceID)
				}
			},
		},
		{
			name:    "tags_list",
			command: []string{"tags", "list", "--workspace-id", workspaceID},
			validate: func(t *testing.T, value any) {
				if _, ok := value.([]any); !ok {
					t.Fatalf("tags list JSON = %T, want array", value)
				}
			},
		},
	}
	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			result := config.Run(t, append(args, test.command...)...)
			config.RequireSuccess(t, result)
			var value any
			if err := json.Unmarshal([]byte(result.Stdout), &value); err != nil {
				t.Fatalf("parse JSON stdout: %v\nstdout:\n%s", err, result.Stdout)
			}
			test.validate(t, value)
		})
	}
}

func requireObject(t *testing.T, value any) map[string]any {
	t.Helper()
	object, ok := value.(map[string]any)
	if !ok {
		t.Fatalf("JSON value = %T, want object", value)
	}
	return object
}

func validateObject(t *testing.T, value any) {
	t.Helper()
	requireObject(t, value)
}
