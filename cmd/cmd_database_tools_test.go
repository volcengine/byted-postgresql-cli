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
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/spf13/cobra"
)

func TestRootIncludesPostgreSQLDataPlaneCommands(t *testing.T) {
	root := newRootCmd()
	for _, path := range [][]string{
		{"db", "query"},
		{"db", "dump"},
		{"db", "pull"},
		{"db", "advisors"},
		{"inspect", "db", "db-stats"},
		{"inspect", "db", "outliers"},
		{"inspect", "db", "blocking"},
	} {
		cmd, _, err := root.Find(path)
		if err != nil {
			t.Fatalf("find %s: %v", strings.Join(path, " "), err)
		}
		if cmd == nil {
			t.Fatalf("command %s is not registered", strings.Join(path, " "))
		}
	}
	if cmd, _, err := root.Find([]string{"databases"}); err != nil || cmd == nil {
		t.Fatal("databases command is not registered")
	}
	if cmd, _, err := root.Find([]string{"db"}); err != nil || cmd == nil || cmd.Name() != "db" {
		t.Fatal("db data-plane command is not registered")
	}
}

func TestDatabaseTargetFlagsRejectAmbiguousTarget(t *testing.T) {
	cmd := newDBQueryCmd()
	cmd.SetArgs([]string{"--db-url", "postgresql://localhost/db", "--workspace-id", "ws", "select 1"})
	if err := cmd.Execute(); err == nil {
		t.Fatal("expected --db-url and --workspace-id conflict")
	}
}

func TestDatabaseAdvisorsDescribesItsActualScope(t *testing.T) {
	cmd := newDBAdvisorsCmd()
	if cmd.Short != "Check PostgreSQL for common security and performance issues" {
		t.Fatalf("advisors short help = %q", cmd.Short)
	}
	for _, want := range []string{"dead tuples", "missing primary keys", "unused indexes", "long", "pg_stat_statements", "skipped"} {
		if !strings.Contains(cmd.Long, want) {
			t.Fatalf("advisors long help missing %q: %s", want, cmd.Long)
		}
	}
	if len(advisorChecks) < 5 {
		t.Fatalf("advisor checks = %d, want at least 5", len(advisorChecks))
	}
}

func TestInspectDatabaseRejectsExtraArguments(t *testing.T) {
	cmd := newRootCmd()
	cmd.SetArgs([]string{"inspect", "db", "db-stats", "extra"})
	err := cmd.Execute()
	if err == nil || !strings.Contains(err.Error(), "unknown command") && !strings.Contains(err.Error(), "accepts 0 arg") {
		t.Fatalf("error = %v, want extra argument validation error", err)
	}
}

func TestInspectDatabaseRejectsUnknownSubcommand(t *testing.T) {
	cmd := newRootCmd()
	cmd.SetArgs([]string{"inspect", "db", "invalid"})
	err := cmd.Execute()
	if err == nil {
		t.Fatal("inspect db invalid unexpectedly succeeded")
	}
}

func TestQueryText(t *testing.T) {
	if _, err := queryText(nil, ""); err == nil {
		t.Fatal("expected missing SQL error")
	}
	sql, err := queryText([]string{"SELECT 1"}, "")
	if err != nil || sql != "SELECT 1" {
		t.Fatalf("queryText() = %q, %v", sql, err)
	}
}

func TestTranslatePostgreSQLQueryErrorForMissingPgStatStatements(t *testing.T) {
	err := &pgconn.PgError{
		Code:    "42P01",
		Message: `relation "pg_stat_statements" does not exist`,
	}
	got := translatePostgreSQLQueryError(err)
	want := "the pg_stat_statements extension is not enabled; run CREATE EXTENSION pg_stat_statements in postgres or ask a DBA to enable it"
	if got.Error() != want {
		t.Fatalf("translatePostgreSQLQueryError() = %q, want %q", got, want)
	}
}

func TestTranslatePostgreSQLQueryErrorPreservesOtherErrors(t *testing.T) {
	tests := []*pgconn.PgError{
		{Code: "42P01", Message: `relation "other_table" does not exist`},
		{Code: "42501", Message: `relation "pg_stat_statements" does not exist`},
	}
	for _, testErr := range tests {
		got := translatePostgreSQLQueryError(testErr)
		if !errors.Is(got, testErr) {
			t.Fatalf("translatePostgreSQLQueryError(%v) changed an unrelated error to %v", testErr, got)
		}
	}
}

// TestDBQueryHasNoFormatFlag guards the fix: db query must no longer expose a
// second "output format" switch. Output is driven by the global --output/-o.
func TestDBQueryHasNoFormatFlag(t *testing.T) {
	cmd := newDBQueryCmd()
	if f := cmd.Flags().Lookup("format"); f != nil {
		t.Fatal("db query must not define its own --format flag; use global --output/-o")
	}
	if f := cmd.Flags().Lookup("unaligned"); f == nil {
		t.Fatal("db query should offer --unaligned for psql-style output")
	}
}

func TestDatabaseSQLCommandsShareOutputFlags(t *testing.T) {
	advisors := newDBAdvisorsCmd()
	if advisors.Flags().Lookup("unaligned") == nil {
		t.Fatal("db advisors should offer --unaligned")
	}

	inspect := newDatabaseInspectCmd(defaultProviderContext())
	db, _, err := inspect.Find([]string{"db"})
	if err != nil {
		t.Fatal(err)
	}
	child, _, err := db.Find([]string{"db-stats"})
	if err != nil {
		t.Fatal(err)
	}
	if child == nil || child.Flags().Lookup("unaligned") == nil {
		t.Fatal("inspect db children should offer --unaligned")
	}
}

func TestSQLOutputFormatUsesGlobalOutput(t *testing.T) {
	cmd := &cobra.Command{}
	cmd.Flags().String("output", "json", "")
	if err := cmd.Flags().Set("output", "json"); err != nil {
		t.Fatal(err)
	}
	cmd.SetContext(withGlobals(context.Background(), &Globals{Output: "json"}))
	format, err := sqlOutputFormat(cmd, false)
	if err != nil || format != "json" {
		t.Fatalf("sqlOutputFormat() = %q, %v; want json", format, err)
	}
	format, err = sqlOutputFormat(cmd, true)
	if err == nil || !strings.Contains(err.Error(), "cannot be combined") || format != "" {
		t.Fatalf("sqlOutputFormat(unaligned) = %q, %v; want conflict", format, err)
	}
}

func TestWriteSQLResultFormats(t *testing.T) {
	columns := []string{"id", "name"}
	values := [][]any{
		{int64(1), "alice"},
		{int64(2), nil},
	}

	tests := []struct {
		format string
		want   string
	}{
		{"unaligned", "1|alice\n2|<nil>\n"},
		{"csv", "id,name\n1,alice\n2,<nil>\n"},
		{"tsv", "id\tname\n1\talice\n2\t<nil>\n"},
		{"table", "  ID   NAME   \n   1   alice  \n   2   <nil>  \n"},
		{"json", "[\n  {\n    \"id\": 1,\n    \"name\": \"alice\"\n  },\n  {\n    \"id\": 2,\n    \"name\": null\n  }\n]\n"},
		{"yaml", "- id: 1\n  name: alice\n- id: 2\n  name: null\n"},
	}
	for _, tt := range tests {
		t.Run(tt.format, func(t *testing.T) {
			cmd := &cobra.Command{}
			var out bytes.Buffer
			cmd.SetOut(&out)
			if err := writeSQLResult(cmd, columns, values, tt.format); err != nil {
				t.Fatalf("writeSQLResult(%s) error: %v", tt.format, err)
			}
			if out.String() != tt.want {
				t.Fatalf("writeSQLResult(%s):\n got %q\nwant %q", tt.format, out.String(), tt.want)
			}
		})
	}
}

func TestWriteSQLResultsKeepsMultipleResultSets(t *testing.T) {
	cmd := &cobra.Command{}
	var out bytes.Buffer
	cmd.SetOut(&out)
	results := []sqlResult{
		{columns: []string{"version"}, values: [][]any{{"PostgreSQL 17"}}},
		{columns: []string{"database"}, values: [][]any{{"aidb"}}},
	}
	if err := writeSQLResults(cmd, results, "json"); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), `"version"`) || !strings.Contains(out.String(), `"database"`) {
		t.Fatalf("multiple result-set JSON = %q", out.String())
	}
}

func TestWriteSQLTableDoesNotEmitTSV(t *testing.T) {
	cmd := &cobra.Command{}
	var out bytes.Buffer
	cmd.SetOut(&out)
	if err := writeSQLResult(cmd, []string{"a", "b"}, [][]any{{1, "hello"}, {22, "world"}}, "table"); err != nil {
		t.Fatalf("writeSQLResult(table) error: %v", err)
	}
	got := out.String()
	if strings.Contains(got, "\t") {
		t.Fatalf("table output contains tab separators: %q", got)
	}
	if !strings.Contains(got, "A      B") || !strings.Contains(got, "1   hello") {
		t.Fatalf("table output is not aligned ASCII: %q", got)
	}
}

// TestWriteSQLResultJSONPreservesColumnOrder ensures the JSON renderer keeps the
// query's column order rather than sorting keys, which matters for automation.
func TestWriteSQLResultJSONPreservesColumnOrder(t *testing.T) {
	cmd := &cobra.Command{}
	var out bytes.Buffer
	cmd.SetOut(&out)
	if err := writeSQLResult(cmd, []string{"zeta", "alpha"}, [][]any{{"z", "a"}}, "json"); err != nil {
		t.Fatalf("writeSQLResult error: %v", err)
	}
	zeta := strings.Index(out.String(), "zeta")
	alpha := strings.Index(out.String(), "alpha")
	if zeta == -1 || alpha == -1 || zeta > alpha {
		t.Fatalf("column order not preserved: %q", out.String())
	}
}

// TestWriteSQLResultRendersBytesAsString ensures text/bytea columns (scanned as
// []byte) are rendered as readable strings across structured formats.
func TestWriteSQLResultRendersBytesAsString(t *testing.T) {
	cmd := &cobra.Command{}
	var out bytes.Buffer
	cmd.SetOut(&out)
	if err := writeSQLResult(cmd, []string{"txt"}, [][]any{{[]byte("hello")}}, "json"); err != nil {
		t.Fatalf("writeSQLResult error: %v", err)
	}
	if !strings.Contains(out.String(), "\"hello\"") {
		t.Fatalf("[]byte should render as string, got %q", out.String())
	}
}
