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

	"github.com/volcengine/byted-postgresql-cli/internal/volcengine"
)

func TestDatabasesCreateUsesDefaultOwnerOnly(t *testing.T) {
	cmd := newDatabasesCmd(defaultProviderContext())
	create, _, err := cmd.Find([]string{"create"})
	if err != nil {
		t.Fatal(err)
	}
	if create.Flags().Lookup("owner-name") != nil {
		t.Fatal("databases create must not expose unsupported --owner-name")
	}
	if create.Flags().Lookup("role-name") == nil {
		t.Fatal("databases create should expose --role-name for selecting the owner")
	}
	if !strings.Contains(create.Long, "--role-name") {
		t.Fatalf("create help does not describe the default-owner behavior: %q", create.Long)
	}
}

func TestResolveDatabaseOwner(t *testing.T) {
	owner, err := resolveDatabaseOwner([]volcengine.DBAccount{{AccountName: "user_admin"}})
	if err != nil || owner != "user_admin" {
		t.Fatalf("resolveDatabaseOwner() = %q, %v; want user_admin", owner, err)
	}

	if _, err := resolveDatabaseOwner(nil); err == nil || !strings.Contains(err.Error(), "no roles") {
		t.Fatalf("resolveDatabaseOwner(nil) error = %v, want no-role guidance", err)
	}

	if _, err := resolveDatabaseOwner([]volcengine.DBAccount{
		{AccountName: "user_admin"},
		{AccountName: "reporting"},
	}); err == nil || !strings.Contains(err.Error(), "--role-name") {
		t.Fatalf("resolveDatabaseOwner(multiple) error = %v, want role guidance", err)
	}
}

func TestSelectDatabaseOwnerRejectsUnknownExplicitRole(t *testing.T) {
	_, err := selectDatabaseOwner(
		&cobra.Command{},
		[]volcengine.DBAccount{{AccountName: "user_admin"}},
		"missing",
	)
	if err == nil || !strings.Contains(err.Error(), `role "missing" not found`) {
		t.Fatalf("selectDatabaseOwner() error = %v, want unknown role error", err)
	}
}
