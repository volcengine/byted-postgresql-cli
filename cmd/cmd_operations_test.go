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
)

func TestNormalizeOperationStatus(t *testing.T) {
	for _, status := range []string{"Start", "start", "Running", "SUCCESS", "failed"} {
		if normalized, err := normalizeOperationStatus(status); err != nil || normalized == "" {
			t.Fatalf("normalizeOperationStatus(%q) = %q, %v; want valid status", status, normalized, err)
		}
	}
	if normalized, err := normalizeOperationStatus(""); err != nil || normalized != "" {
		t.Fatalf("normalizeOperationStatus(\"\") = %q, %v; want empty status", normalized, err)
	}
	if _, err := normalizeOperationStatus("Invalid"); err == nil || !strings.Contains(err.Error(), "must be one of") {
		t.Fatalf("normalizeOperationStatus(Invalid) = %v, want enum validation error", err)
	}
}

func TestOperationsListRejectsInvalidStatusAndExtraArguments(t *testing.T) {
	for _, args := range [][]string{
		{"operations", "list", "--status", "Invalid"},
		{"operations", "list", "extra"},
		{"operations", "list", "--limit", "100000"},
	} {
		cmd := newRootCmd()
		cmd.SetArgs(args)
		err := cmd.Execute()
		if err == nil {
			t.Fatalf("args %v unexpectedly succeeded", args)
		}
	}
}

func TestStatusRejectsExtraArguments(t *testing.T) {
	cmd := newRootCmd()
	cmd.SetArgs([]string{"status", "extra"})
	err := cmd.Execute()
	if err == nil || !strings.Contains(err.Error(), `unknown command "extra"`) && !strings.Contains(err.Error(), "accepts 0 arg") {
		t.Fatalf("error = %v, want extra argument validation error", err)
	}
}
