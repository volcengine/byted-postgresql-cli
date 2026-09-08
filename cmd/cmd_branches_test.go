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

func TestBranchesChildrenListUsesExplicitRequiredParentBranchFlag(t *testing.T) {
	cmd := newBranchesChildrenCmd(func(*cobra.Command) (string, error) {
		t.Fatal("workspace resolver should not run without the required parent branch")
		return "", nil
	})
	list, _, err := cmd.Find([]string{"list"})
	if err != nil {
		t.Fatal(err)
	}
	if list == nil {
		t.Fatal("children should define a list subcommand")
	}

	if flag := list.Flags().Lookup("parent-branch-id"); flag == nil {
		t.Fatal("children list should expose --parent-branch-id")
	} else if flag.Usage != "Parent branch ID (required)" {
		t.Fatalf("parent-branch-id usage = %q", flag.Usage)
	}

	err = list.RunE(list, nil)
	if err == nil || !strings.Contains(err.Error(), "--parent-branch-id is required") {
		t.Fatalf("missing parent branch error = %v", err)
	}
}

func TestBranchesChildrenHelpUsesListSubcommand(t *testing.T) {
	cmd := newBranchesChildrenCmd(func(*cobra.Command) (string, error) {
		return "", nil
	})
	var output strings.Builder
	cmd.SetOut(&output)
	cmd.SetArgs([]string{"--help"})
	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}
	help := output.String()
	if !strings.Contains(help, "Available Commands:") ||
		!strings.Contains(help, "list") {
		t.Fatalf("children help does not show the list subcommand:\n%s", help)
	}
	list, _, err := cmd.Find([]string{"list"})
	if err != nil {
		t.Fatal(err)
	}
	if list == nil || list.Use != "list --parent-branch-id <branch-id>" {
		t.Fatalf("children list command = %#v, want explicit list subcommand", list)
	}
	var listOutput strings.Builder
	list.SetOut(&listOutput)
	if err := list.Help(); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(listOutput.String(), "--parent-branch-id string") {
		t.Fatalf("children list help does not explain the required flag:\n%s", listOutput.String())
	}
}
