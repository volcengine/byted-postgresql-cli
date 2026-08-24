// Copyright (c) 2026 Beijing Volcano Engine Technology Co., Ltd.
// SPDX-License-Identifier: MIT

package databaseresources

import (
	"testing"

	"github.com/volcengine/byted-postgresql-cli/test/e2e/harness"
)

func TestDatabasesList(t *testing.T) {
	config := harness.RequireReplay(t)
	payload := config.RequireJSON(t, config.Run(t, "--output", "json", "databases", "list", "--workspace-id", "ws-e2e-1", "--branch-id", "br-e2e-1"))
	if payload["total"] != float64(1) {
		t.Fatalf("database payload = %#v, want total 1", payload)
	}
}
