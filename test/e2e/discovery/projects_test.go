// Copyright (c) 2026 Beijing Volcano Engine Technology Co., Ltd.
// SPDX-License-Identifier: MIT

package discovery

import (
	"testing"

	"github.com/volcengine/byted-postgresql-cli/test/e2e/harness"
)

func TestProjectsListHelp(t *testing.T) {
	config := harness.New(t)
	if config.Mode != harness.ModeReplay {
		t.Skip("replay E2E suite is disabled in live mode")
	}
	config.RequireSuccess(t, config.Run(t, "projects", "list", "--help"))
}
