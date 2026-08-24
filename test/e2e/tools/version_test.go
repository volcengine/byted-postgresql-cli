// Copyright (c) 2026 Beijing Volcano Engine Technology Co., Ltd.
// SPDX-License-Identifier: MIT

package tools

import (
	"strings"
	"testing"

	"github.com/volcengine/byted-postgresql-cli/test/e2e/harness"
)

func TestVersion(t *testing.T) {
	config := harness.RequireReplay(t)
	result := config.Run(t, "--version")
	config.RequireSuccess(t, result)
	if !strings.Contains(result.Stdout, "dev") {
		t.Fatalf("version output = %q, want development version", result.Stdout)
	}
}
