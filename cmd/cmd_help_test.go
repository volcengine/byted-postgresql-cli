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

func TestComputeHelpExplainsUnitsAndAP(t *testing.T) {
	computes := newComputesCmd(defaultProviderContext())
	var output strings.Builder
	computes.SetOut(&output)
	computes.SetArgs([]string{"create", "--help"})
	if err := computes.Execute(); err != nil {
		t.Fatal(err)
	}
	help := output.String()
	for _, want := range []string{"compute units (0.25-32)", "at most 8x --min-cu"} {
		if !strings.Contains(help, want) {
			t.Fatalf("compute help missing %q:\n%s", want, help)
		}
	}
	if strings.Contains(help, "CU") {
		t.Fatalf("compute help should spell out compute units instead of using CU:\n%s", help)
	}

	ap := newComputesCmd(defaultProviderContext())
	output.Reset()
	ap.SetOut(&output)
	ap.SetArgs([]string{"enable-ap", "--help"})
	if err := ap.Execute(); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(output.String(), "analytics policy") {
		t.Fatalf("AP help missing analytics policy explanation:\n%s", output.String())
	}
}
