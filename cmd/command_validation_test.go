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
	"errors"
	"io"
	"strings"
	"testing"
)

func TestRootCommandGroupsRejectUnknownSubcommands(t *testing.T) {
	for _, name := range []string{
		"projects",
		"workspaces",
		"branches",
		"operations",
		"configure",
		"db",
		"inspect",
		"databases",
		"roles",
		"computes",
		"endpoints",
		"schema-diff",
		"network",
		"tags",
		"mcp",
	} {
		t.Run(name, func(t *testing.T) {
			root := newRootCmd()
			root.SetOut(io.Discard)
			root.SetErr(io.Discard)
			root.SetArgs([]string{name, "typo"})
			err := root.Execute()
			if err == nil {
				t.Fatalf("%s accepted an unknown subcommand", name)
			}
			if !strings.Contains(err.Error(), "unknown command") {
				t.Fatalf("%s error = %v, want unknown command error", name, err)
			}
		})
	}
}

func TestFormatErrorForUpdateRegistryFailureIsEnglish(t *testing.T) {
	message := FormatError(errors.New("fetch latest release: status 404 Not Found"))
	if message == "" {
		t.Fatal("FormatError() returned an empty message")
	}
	for _, r := range message {
		if r >= '\u4e00' && r <= '\u9fff' {
			t.Fatalf("update error contains Chinese characters: %q", message)
		}
	}
	for _, want := range []string{"Unable to check for CLI updates", "public npm registry", "@byted-postgresql/cli"} {
		if !strings.Contains(message, want) {
			t.Fatalf("FormatError() = %q, want %q", message, want)
		}
	}
}
