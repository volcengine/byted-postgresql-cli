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

func TestResolveDatabaseAndAccount(t *testing.T) {
	databases := []volcengine.Database{
		{DatabaseName: "aidb", DatabaseOwner: "byte_user_admin"},
	}

	database, account, err := resolveDatabaseAndAccount(databases, "", "")
	if err != nil {
		t.Fatal(err)
	}
	if database != "aidb" || account != "byte_user_admin" {
		t.Fatalf("got database=%q account=%q", database, account)
	}

	database, account, err = resolveDatabaseAndAccount(databases, "aidb", "")
	if err != nil {
		t.Fatal(err)
	}
	if database != "aidb" || account != "byte_user_admin" {
		t.Fatalf("explicit database got database=%q account=%q", database, account)
	}

	database, account, err = resolveDatabaseAndAccount(databases, "aidb", "custom")
	if err != nil {
		t.Fatal(err)
	}
	if database != "aidb" || account != "custom" {
		t.Fatalf("explicit values got database=%q account=%q", database, account)
	}
}

func TestResolveDatabaseAndAccountRejectsAmbiguousDatabase(t *testing.T) {
	databases := []volcengine.Database{
		{DatabaseName: "aidb", DatabaseOwner: "admin"},
		{DatabaseName: "analytics", DatabaseOwner: "analyst"},
	}

	_, _, err := resolveDatabaseAndAccount(databases, "", "")
	if err == nil || !strings.Contains(err.Error(), "--database-name") {
		t.Fatalf("expected actionable ambiguity error, got %v", err)
	}

	_, _, err = resolveDatabaseAndAccount(databases, "missing", "")
	if err == nil || !strings.Contains(err.Error(), "available: aidb, analytics") {
		t.Fatalf("expected available database names, got %v", err)
	}
}

func TestResolveDatabaseAndAccountDefersEmptyOwner(t *testing.T) {
	database, account, err := resolveDatabaseAndAccount(
		[]volcengine.Database{{DatabaseName: "aidb"}},
		"",
		"",
	)
	if err != nil {
		t.Fatal(err)
	}
	if database != "aidb" || account != "" {
		t.Fatalf("got database=%q account=%q", database, account)
	}
}

func TestResolveSingleAccount(t *testing.T) {
	account, err := resolveSingleAccount([]volcengine.DBAccount{{AccountName: "admin"}})
	if err != nil || account != "admin" {
		t.Fatalf("got account=%q err=%v", account, err)
	}

	if _, err := resolveSingleAccount(nil); err == nil || !strings.Contains(err.Error(), "no roles") {
		t.Fatalf("expected no-role error, got %v", err)
	}

	_, err = resolveSingleAccount([]volcengine.DBAccount{
		{AccountName: "admin"},
		{AccountName: "app"},
	})
	if err == nil || !strings.Contains(err.Error(), "--role-name") {
		t.Fatalf("expected actionable ambiguity error, got %v", err)
	}
}

func TestConnectionCommandsExplainSelectionAndTokenRules(t *testing.T) {
	connection := newConnectionStringCmd(defaultProviderContext())
	for _, want := range []string{"multiple databases", "--database-name", "--role-name", "databases list", "roles list", "non-interactive"} {
		if !strings.Contains(connection.Long, want) {
			t.Fatalf("connection-string help missing %q: %s", want, connection.Long)
		}
	}

	psql := newPsqlCmd(defaultProviderContext())
	for _, want := range []string{"multiple databases", "--database-name", "--role-name", "role password returned by the control plane"} {
		if !strings.Contains(psql.Long, want) {
			t.Fatalf("psql help missing %q: %s", want, psql.Long)
		}
	}
	for _, want := range []string{"--reveal", "--masked", "password"} {
		if want == "password" {
			if !strings.Contains(connection.Long, want) &&
				!strings.Contains(connection.Flags().Lookup("reveal").Usage, want) {
				t.Fatalf("connection-string help missing %q", want)
			}
			continue
		}
		if connection.Flags().Lookup(strings.TrimPrefix(want, "--")) == nil {
			t.Fatalf("connection-string help missing %q", want)
		}
	}
}

func TestPsqlRejectsPositionalArgumentsBeforeDash(t *testing.T) {
	psql := newPsqlCmd(defaultProviderContext())
	psql.SetArgs([]string{"branch-positional", "--", "-c", "SELECT 1"})

	err := psql.Execute()
	if err == nil || !strings.Contains(err.Error(), "psql accepts no positional arguments") {
		t.Fatalf("psql positional argument error = %v, want explicit rejection", err)
	}
}

func TestMaskConnectionPassword(t *testing.T) {
	raw := "postgresql://user_admin:66-Jd627mqsjFahR9%23@host/aidb?sslmode=require"
	got := maskConnectionPassword(raw)
	if strings.Contains(got, "66-Jd627mqsjFahR9") {
		t.Fatalf("masked URL leaked password: %q", got)
	}
	if !strings.Contains(got, "user_admin:*****@host") {
		t.Fatalf("masked URL = %q, want masked userinfo", got)
	}
}
