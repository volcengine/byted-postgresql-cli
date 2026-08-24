// Copyright (c) 2026 Beijing Volcano Engine Technology Co., Ltd.
// SPDX-License-Identifier: MIT

package databaseaccess

import (
	"testing"

	"github.com/volcengine/byted-postgresql-cli/test/e2e/harness"
)

func TestPullHelp(t *testing.T) {
	config := harness.RequireReplay(t)
	config.RequireSuccess(t, config.Run(t, "db", "pull", "--help"))
}
