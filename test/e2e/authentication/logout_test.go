// Copyright (c) 2026 Beijing Volcano Engine Technology Co., Ltd.
// SPDX-License-Identifier: MIT

package authentication

import (
	"strings"
	"testing"

	"github.com/volcengine/byted-postgresql-cli/test/e2e/harness"
)

func TestLogoutHelp(t *testing.T) {
	config := harness.New(t)
	if config.Mode != harness.ModeReplay {
		t.Skip("replay E2E suite is disabled in live mode")
	}
	result := config.Run(t, "logout", "--help")
	config.RequireSuccess(t, result)
	if !strings.Contains(result.Stdout, "Log out") {
		t.Fatalf("logout help = %q", result.Stdout)
	}
}
