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

func TestValidatePaginationFlags(t *testing.T) {
	for _, value := range []string{"-1", "-10"} {
		t.Run(value, func(t *testing.T) {
			cmd := &cobra.Command{}
			cmd.Flags().Int("limit", 0, "")
			if err := cmd.Flags().Set("limit", value); err != nil {
				t.Fatal(err)
			}
			err := validatePaginationFlags(cmd)
			if err == nil || err.Error() != "--limit must be greater than or equal to 0" {
				t.Fatalf("validatePaginationFlags() error = %v", err)
			}
		})
	}
}

func TestValidatePaginationFlagsRejectsExcessiveLimit(t *testing.T) {
	cmd := &cobra.Command{}
	cmd.Flags().Int("limit", 0, "")
	if err := cmd.Flags().Set("limit", "101"); err != nil {
		t.Fatal(err)
	}
	err := validatePaginationFlags(cmd)
	if err == nil || !strings.Contains(err.Error(), "--limit must be less than or equal to 100") {
		t.Fatalf("validatePaginationFlags() error = %v, want upper-bound error", err)
	}
}

func TestValidateRequiredStringFlagsRejectsEmptyValue(t *testing.T) {
	cmd := &cobra.Command{}
	cmd.Flags().String("name", "", "")
	if err := cmd.MarkFlagRequired("name"); err != nil {
		t.Fatal(err)
	}
	if err := cmd.Flags().Set("name", "  "); err != nil {
		t.Fatal(err)
	}

	err := validateRequiredStringFlags(cmd)
	if err == nil || err.Error() != "--name is required" {
		t.Fatalf("validateRequiredStringFlags() error = %v", err)
	}
}

func TestValidateRequiredStringFlagsAllowsNonEmptyValue(t *testing.T) {
	cmd := &cobra.Command{}
	cmd.Flags().String("name", "", "")
	if err := cmd.MarkFlagRequired("name"); err != nil {
		t.Fatal(err)
	}
	if err := cmd.Flags().Set("name", "compute-a"); err != nil {
		t.Fatal(err)
	}

	if err := validateRequiredStringFlags(cmd); err != nil {
		t.Fatalf("validateRequiredStringFlags() error = %v", err)
	}
}

func TestNewRootCmdRejectsInvalidOutputBeforeRun(t *testing.T) {
	cmd := newRootCmd()
	cmd.SetArgs([]string{"--output", "xml", "workspaces", "list"})
	err := cmd.Execute()
	if err == nil || !strings.Contains(err.Error(), `unsupported output format "xml"`) {
		t.Fatalf("newRootCmd().Execute() error = %v", err)
	}
}
