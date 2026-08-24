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

	"github.com/spf13/cobra"
)

func TestLifecycleCommandsAreRegistered(t *testing.T) {
	root := newRootCmd()

	workspaces, _, err := root.Find([]string{"workspaces"})
	if err != nil || workspaces == nil {
		t.Fatalf("workspaces command is not registered: command=%v err=%v", workspaces, err)
	}
	if _, _, err := workspaces.Find([]string{"deletion-protection"}); err != nil {
		t.Fatalf("workspaces deletion-protection is not registered: %v", err)
	}

	branches, _, err := root.Find([]string{"branches"})
	if err != nil || branches == nil {
		t.Fatalf("branches command is not registered: command=%v err=%v", branches, err)
	}
	if _, _, err := branches.Find([]string{"restore-window"}); err != nil {
		t.Fatalf("branches restore-window is not registered: %v", err)
	}

	roles, _, err := root.Find([]string{"roles"})
	if err != nil || roles == nil {
		t.Fatalf("roles command is not registered: command=%v err=%v", roles, err)
	}
	if _, _, err := roles.Find([]string{"reset-password"}); err != nil {
		t.Fatalf("roles reset-password is not registered: %v", err)
	}
}

func TestDeletionProtectionCommandRequiresExactlyOneDirection(t *testing.T) {
	cmd := newWorkspacesDeletionProtectionCmd()
	cmd.SetArgs([]string{"ws-1"})
	err := cmd.Execute()
	if err == nil || !strings.Contains(err.Error(), "exactly one of --enable or --disable") {
		t.Fatalf("missing direction error = %v", err)
	}

	cmd = newWorkspacesDeletionProtectionCmd()
	cmd.SetArgs([]string{"ws-1", "--enable", "--disable"})
	err = cmd.Execute()
	if err == nil {
		t.Fatal("expected mutually exclusive direction flags to fail")
	}
}

func TestRestoreWindowCommandRequiresBranchID(t *testing.T) {
	cmd := newBranchesRestoreWindowCmd(func(*cobra.Command) (string, error) {
		t.Fatal("workspace resolver should not run before branch argument validation")
		return "", nil
	})
	cmd.SetArgs(nil)
	err := cmd.Execute()
	if err == nil || !strings.Contains(err.Error(), "accepts 1 arg(s), received 0") {
		t.Fatalf("missing branch id error = %v", err)
	}
}

func TestResetPasswordCommandRequiresRoleAndPassword(t *testing.T) {
	cmd := newRolesCmd(defaultProviderContext())
	cmd.SetArgs([]string{"reset-password", "user_admin"})
	err := cmd.Execute()
	if err == nil || !strings.Contains(err.Error(), "required flag") {
		t.Fatalf("missing password error = %v", err)
	}

	cmd = newRolesCmd(defaultProviderContext())
	cmd.SetArgs([]string{"reset-password", "--password", "secret"})
	err = cmd.Execute()
	if err == nil || !strings.Contains(err.Error(), "accepts 1 arg(s), received 0") {
		t.Fatalf("missing role error = %v", err)
	}
}
