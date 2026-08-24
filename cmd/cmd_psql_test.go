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

func TestExtractPostgresURL(t *testing.T) {
	const url = "postgresql://[::1]:5432/aidb?options=test"
	cases := []string{
		url,
		"psql \"" + url + "\"",
		"export Token=<Your Access Token>\npsql \"" + url + "\"\n",
	}
	for _, input := range cases {
		if got := extractPostgresURL(input); got != url {
			t.Fatalf("extractPostgresURL(%q) = %q, want %q", input, got, url)
		}
	}
	if got := extractPostgresURL(" db.internal.example:5432/aidb "); got != "db.internal.example:5432/aidb" {
		t.Fatalf("bare connection returned %q", got)
	}
}

func TestPsqlConnectionURL(t *testing.T) {
	conn := volcengine.DBAccountConnection{
		ConnectionURL:   "postgresql://host/aidb?sslmode=require",
		AccountName:     "app_user",
		AccountPassword: "p@ss word",
	}
	got, err := psqlConnectionURL(conn)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(got, "app_user:p%40ss%20word@host") {
		t.Fatalf("role credentials were not injected: %q", got)
	}
}

func TestPsqlConnectionURLLeavesCompleteURLUnchanged(t *testing.T) {
	conn := volcengine.DBAccountConnection{
		ConnectionURL:   "postgresql://existing:password@host/aidb?options=complete",
		AccountName:     "replacement",
		AccountPassword: "replacement",
	}
	got, err := psqlConnectionURL(conn)
	if err != nil {
		t.Fatal(err)
	}
	if got != conn.ConnectionURL {
		t.Fatalf("complete URL changed: %q", got)
	}
}

func TestPsqlConnectionURLAllowsURLWithoutRoleCredentials(t *testing.T) {
	const connectionURL = "postgresql://host/aidb?sslmode=require"
	got, err := psqlConnectionURL(volcengine.DBAccountConnection{ConnectionURL: connectionURL})
	if err != nil {
		t.Fatal(err)
	}
	if got != connectionURL {
		t.Fatalf("connection URL changed: %q", got)
	}
}
