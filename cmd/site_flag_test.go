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

func TestRegionAndProfileAreGlobalAuthenticationFlags(t *testing.T) {
	root := newRootCmd()

	for _, name := range []string{"region", "profile"} {
		if root.PersistentFlags().Lookup(name) == nil {
			t.Fatalf("root must expose persistent --%s", name)
		}
	}

	for _, path := range [][]string{
		{"status"},
		{"workspaces", "list"},
		{"branches", "list"},
	} {
		command, _, err := root.Find(path)
		if err != nil {
			t.Fatalf("find %v: %v", path, err)
		}
		if command.Flags().Lookup("site") != nil {
			t.Fatalf("%v must not expose --site", path)
		}
	}

	if root.PersistentFlags().Lookup("site") != nil {
		t.Fatal("root must not expose a persistent --site")
	}

	// The Volcengine edition authenticates via browser Console Login (which
	// mints temporary STS credentials) in addition to static AK/SK. The login
	// and logout commands must be mounted.
	for _, name := range []string{"login", "logout"} {
		if _, _, err := root.Find([]string{name}); err != nil {
			t.Fatalf("root must expose the %q command: %v", name, err)
		}
	}
}
