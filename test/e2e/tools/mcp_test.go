// Copyright (c) 2026 Beijing Volcano Engine Technology Co., Ltd.
// SPDX-License-Identifier: MIT

package tools

import (
	"testing"

	"github.com/volcengine/byted-postgresql-cli/test/e2e/harness"
)

func TestMCPHelp(t *testing.T) {
	config := harness.RequireReplay(t)
	config.RequireSuccess(t, config.Run(t, "mcp", "--help"))
	config.RequireSuccess(t, config.Run(t, "mcp", "serve", "--help"))
}
