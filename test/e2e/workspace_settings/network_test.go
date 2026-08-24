// Copyright (c) 2026 Beijing Volcano Engine Technology Co., Ltd.
// SPDX-License-Identifier: MIT

package workspacesettings

import (
	"testing"

	"github.com/volcengine/byted-postgresql-cli/test/e2e/harness"
)

func TestNetworkHelp(t *testing.T) {
	config := harness.RequireReplay(t)
	config.RequireSuccess(t, config.Run(t, "network", "--help"))
}
