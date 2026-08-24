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
	"strings"
	"testing"
)

func TestPgDumpBinaryDryRunDoesNotRequirePATH(t *testing.T) {
	t.Setenv("PATH", t.TempDir())

	binary, err := pgDumpBinary(true)
	if err != nil {
		t.Fatalf("pgDumpBinary(true) error = %v", err)
	}
	if binary != "pg_dump" {
		t.Fatalf("pgDumpBinary(true) = %q, want pg_dump", binary)
	}

	if _, err := pgDumpBinary(false); err == nil || !strings.Contains(err.Error(), "pg_dump not found in PATH") {
		t.Fatalf("pgDumpBinary(false) error = %v, want missing binary error", err)
	}
}

func TestShellQuoteOnlyQuotesArgumentsThatNeedIt(t *testing.T) {
	tests := []struct {
		value string
		want  string
	}{
		{value: "/opt/homebrew/bin/pg_dump", want: "/opt/homebrew/bin/pg_dump"},
		{value: "--no-password", want: "--no-password"},
		{value: "--dbname", want: "--dbname"},
		{value: "postgresql://user:*****@host/db?sslmode=require&channel_binding=require", want: "'postgresql://user:*****@host/db?sslmode=require&channel_binding=require'"},
		{value: "backup file.sql", want: "'backup file.sql'"},
	}
	for _, test := range tests {
		if got := shellQuote(test.value); got != test.want {
			t.Errorf("shellQuote(%q) = %q, want %q", test.value, got, test.want)
		}
	}
}

func TestDBDumpDryRunMasksPasswordByDefault(t *testing.T) {
	cmd := newDBDumpCmd()
	var stdout, stderr bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)
	cmd.SetArgs([]string{
		"--db-url", "postgresql://dump_user:s3cr3t%21@db.example/aidb",
		"--dry-run",
	})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("db dump --dry-run error = %v", err)
	}
	if strings.Contains(stdout.String(), "s3cr3t") {
		t.Fatalf("dry-run output leaked password: %q", stdout.String())
	}
	if !strings.Contains(stdout.String(), "dump_user:*****@db.example") {
		t.Fatalf("dry-run output = %q, want masked connection URL", stdout.String())
	}
	if !strings.Contains(stdout.String(), "'postgresql://dump_user:*****@db.example/aidb'") {
		t.Fatalf("dry-run output = %q, want shell-quoted connection URL", stdout.String())
	}
	if !strings.Contains(stderr.String(), "replace it with the real password") {
		t.Fatalf("masked dry-run warning = %q", stderr.String())
	}
}

func TestDBDumpDryRunCanRevealPasswordExplicitly(t *testing.T) {
	cmd := newDBDumpCmd()
	var stdout, stderr bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)
	cmd.SetArgs([]string{
		"--db-url", "postgresql://dump_user:s3cr3t%21@db.example/aidb",
		"--dry-run",
		"--reveal",
	})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("db dump --dry-run --reveal error = %v", err)
	}
	if !strings.Contains(stdout.String(), "dump_user:s3cr3t%21@db.example") {
		t.Fatalf("dry-run reveal output = %q, want plaintext password", stdout.String())
	}
	if !strings.Contains(stderr.String(), "contains a password") {
		t.Fatalf("dry-run reveal warning = %q", stderr.String())
	}
}

func TestDBDumpDryRunRejectsRevealAndMaskedTogether(t *testing.T) {
	cmd := newDBDumpCmd()
	cmd.SetArgs([]string{
		"--db-url", "postgresql://dump_user:s3cr3t@db.example/aidb",
		"--dry-run",
		"--reveal",
		"--masked",
	})

	err := cmd.Execute()
	if err == nil || !strings.Contains(err.Error(), "--reveal cannot be combined with --masked") {
		t.Fatalf("error = %v, want reveal/masked conflict", err)
	}
}

func TestDBDumpRejectsOutputModeFlagsWithoutDryRun(t *testing.T) {
	for _, flag := range []string{"--reveal", "--masked"} {
		t.Run(flag, func(t *testing.T) {
			cmd := newDBDumpCmd()
			cmd.SetArgs([]string{
				"--db-url", "postgresql://dump_user:s3cr3t@db.example/aidb",
				flag,
			})
			err := cmd.Execute()
			if err == nil || !strings.Contains(err.Error(), "can only be used with --dry-run") {
				t.Fatalf("error = %v, want dry-run usage error", err)
			}
		})
	}
}
