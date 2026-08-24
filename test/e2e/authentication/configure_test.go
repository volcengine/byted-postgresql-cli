// Copyright (c) 2026 Beijing Volcano Engine Technology Co., Ltd.
// SPDX-License-Identifier: MIT

package authentication

import (
	"strings"
	"testing"

	"github.com/volcengine/byted-postgresql-cli/test/e2e/harness"
)

func TestConfigureHelp(t *testing.T) {
	config := harness.New(t)
	if config.Mode != harness.ModeReplay {
		t.Skip("replay E2E suite is disabled in live mode")
	}
	result := config.Run(t, "configure", "--help")
	config.RequireSuccess(t, result)
	if !strings.Contains(result.Stdout, "Manage credential profiles") {
		t.Fatalf("configure help = %q", result.Stdout)
	}
}
