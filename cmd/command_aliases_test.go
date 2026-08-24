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

import "testing"

func TestResourceCommandsHaveSingularAliases(t *testing.T) {
	root := newRootCmd()
	tests := []struct {
		plural   string
		singular string
	}{
		{plural: "workspaces", singular: "workspace"},
		{plural: "branches", singular: "branch"},
		{plural: "computes", singular: "compute"},
		{plural: "databases", singular: "database"},
		{plural: "roles", singular: "role"},
		{plural: "endpoints", singular: "endpoint"},
	}

	for _, test := range tests {
		t.Run(test.singular, func(t *testing.T) {
			plural, _, err := root.Find([]string{test.plural})
			if err != nil {
				t.Fatalf("find %q: %v", test.plural, err)
			}
			singular, _, err := root.Find([]string{test.singular})
			if err != nil {
				t.Fatalf("find %q: %v", test.singular, err)
			}
			if plural != singular {
				t.Fatalf("%q and %q resolve to different commands", test.plural, test.singular)
			}
		})
	}
}
