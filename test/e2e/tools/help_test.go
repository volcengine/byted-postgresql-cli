// Copyright (c) 2026 Beijing Volcano Engine Technology Co., Ltd.
// SPDX-License-Identifier: MIT

package tools

import (
	"strings"
	"testing"

	"github.com/volcengine/byted-postgresql-cli/test/e2e/harness"
)

func TestRootHelp(t *testing.T) {
	config := harness.RequireReplay(t)
	result := config.Run(t, "--help")
	config.RequireSuccess(t, result)
	for _, expected := range []string{"Authentication:", "Database Access:", "mcp"} {
		if !strings.Contains(result.Stdout, expected) {
			t.Fatalf("help output missing %q:\n%s", expected, result.Stdout)
		}
	}
}

func TestCommandValidation(t *testing.T) {
	config := harness.RequireReplay(t)
	for _, testCase := range []struct {
		name string
		args []string
		want string
	}{
		{"unknown root command", []string{"not-a-command"}, "unknown command"},
		{"unknown child command", []string{"workspaces", "not-a-command"}, "unknown command"},
		{"invalid output format", []string{"--output", "invalid", "version"}, "unsupported output format"},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			result := config.Run(t, testCase.args...)
			config.RequireFailure(t, result)
			if !strings.Contains(result.Stderr, testCase.want) && !strings.Contains(result.Stdout, testCase.want) {
				t.Fatalf("output does not contain %q: stdout=%s stderr=%s", testCase.want, result.Stdout, result.Stderr)
			}
		})
	}
}
