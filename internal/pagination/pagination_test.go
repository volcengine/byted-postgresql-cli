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

package pagination

import (
	"strings"
	"testing"
)

func TestDefaultPolicyNormalize(t *testing.T) {
	tests := []struct {
		name        string
		limit       int
		offset      int
		want        Page
		wantErrPart string
	}{
		{name: "defaults", want: Page{Limit: DefaultLimit}},
		{name: "custom", limit: 50, offset: 100, want: Page{Limit: 50, Offset: 100}},
		{name: "negative limit", limit: -1, wantErrPart: "limit must be between"},
		{name: "excessive limit", limit: MaxLimit + 1, wantErrPart: "limit must be between"},
		{name: "negative offset", offset: -1, wantErrPart: "offset must be between"},
		{name: "excessive offset", offset: MaxOffset + 1, wantErrPart: "offset must be between"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := DefaultPolicy.Normalize(test.limit, test.offset)
			if test.wantErrPart != "" {
				if err == nil || !strings.Contains(err.Error(), test.wantErrPart) {
					t.Fatalf("Normalize() error = %v, want %q", err, test.wantErrPart)
				}
				return
			}
			if err != nil {
				t.Fatalf("Normalize() error = %v", err)
			}
			if got != test.want {
				t.Fatalf("Normalize() = %+v, want %+v", got, test.want)
			}
		})
	}
}

func TestMetadata(t *testing.T) {
	page := Page{Limit: 10, Offset: 10}
	if got := page.Metadata(10, 20); got.HasMore || got.NextOffset != nil {
		t.Fatalf("Metadata() at end = %+v", got)
	}
	got := page.Metadata(10, 30)
	if !got.HasMore || got.NextOffset == nil || *got.NextOffset != 20 {
		t.Fatalf("Metadata() with more results = %+v", got)
	}
}
