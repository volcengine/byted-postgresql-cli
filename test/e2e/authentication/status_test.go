// Copyright (c) 2026 Beijing Volcano Engine Technology Co., Ltd.
// SPDX-License-Identifier: MIT

package authentication

import (
	"testing"

	"github.com/volcengine/byted-postgresql-cli/test/e2e/harness"
)

func TestStatusUsesIsolatedReplayCredentials(t *testing.T) {
	config := harness.New(t)
	if config.Mode != harness.ModeReplay {
		t.Skip("replay E2E suite is disabled in live mode")
	}
	payload := config.RequireJSON(t, config.Run(t, "--config-dir", config.Dir, "status", "--output", "json"))
	authentication, ok := payload["authentication"].(map[string]any)
	if !ok || authentication["credential_from"] != "environment" {
		t.Fatalf("status authentication = %#v", payload["authentication"])
	}
}
