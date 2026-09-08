// Copyright (c) 2026 Beijing Volcano Engine Technology Co., Ltd.
// SPDX-License-Identifier: MIT
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in all
// copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
// SOFTWARE.

package cli

import (
	"strings"
	"testing"

	"github.com/volcengine/byted-postgresql-cli/internal/volcengine"
)

func TestConfigHelpHidesAPIHostCommand(t *testing.T) {
	cmd := newRootCmd()
	cmd.SetArgs([]string{"config", "--help"})
	var output strings.Builder
	cmd.SetOut(&output)

	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(output.String(), "apihost") || strings.Contains(output.String(), "api-host") {
		t.Fatalf("config help contains hidden API host command: %q", output.String())
	}
}

func TestConfigRejectsUnknownCommand(t *testing.T) {
	cmd := newRootCmd()
	cmd.SetArgs([]string{"config", "apihost"})

	err := cmd.Execute()
	if err == nil || !strings.Contains(err.Error(), `unknown command "apihost"`) {
		t.Fatalf("config apihost error = %v, want unknown command error", err)
	}
}

func TestConfigureIncludesAgentPlanCommand(t *testing.T) {
	cmd := newRootCmd()
	found, _, err := cmd.Find([]string{"configure", "agent-plan"})
	if err != nil || found == nil {
		t.Fatalf("configure agent-plan is not registered: %v", err)
	}
	if found.LocalNonPersistentFlags().Lookup("profile") != nil {
		t.Fatal("configure agent-plan must use the root persistent --profile flag")
	}
}

func TestUpdateAgentPlanProfileDisablesAndClearsSeat(t *testing.T) {
	profile := &volcengine.Profile{
		IsAgentPlan:     true,
		AgentPlanSeatID: "seat-old",
	}
	if err := updateAgentPlanProfile(profile, true, false, false, ""); err != nil {
		t.Fatal(err)
	}
	if profile.IsAgentPlan || profile.AgentPlanSeatID != "" {
		t.Fatalf("profile = %+v, want Agent Plan disabled with empty seat ID", profile)
	}
}

func TestUpdateAgentPlanProfileRejectsDisabledNonEmptySeat(t *testing.T) {
	profile := &volcengine.Profile{IsAgentPlan: true, AgentPlanSeatID: "seat-old"}
	err := updateAgentPlanProfile(profile, true, false, true, "seat-new")
	if err == nil || !strings.Contains(err.Error(), "--is-agent-plan=false") {
		t.Fatalf("error = %v, want disabled Agent Plan conflict", err)
	}
}

func TestUpdateAgentPlanProfileEnablesSeat(t *testing.T) {
	profile := &volcengine.Profile{}
	if err := updateAgentPlanProfile(profile, false, false, true, "seat-new"); err != nil {
		t.Fatal(err)
	}
	if !profile.IsAgentPlan || profile.AgentPlanSeatID != "seat-new" {
		t.Fatalf("profile = %+v, want Agent Plan enabled with seat-new", profile)
	}
}
