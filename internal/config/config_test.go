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

package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestInstallRootMatchesLauncherDir(t *testing.T) {
	got := InstallRoot()
	if !strings.HasSuffix(got, ".volcengine-postgresql") {
		t.Fatalf("InstallRoot() = %q, want it to end with .volcengine-postgresql (matching bin/paths.js)", got)
	}
	if home, err := os.UserHomeDir(); err == nil {
		if want := filepath.Join(home, ".volcengine-postgresql"); got != want {
			t.Fatalf("InstallRoot() = %q, want %q", got, want)
		}
	}
}

func TestEnsureDirCreatesPrivateConfigDirectory(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "nested", "config")
	if err := EnsureDir(dir); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(dir)
	if err != nil {
		t.Fatal(err)
	}
	if !info.IsDir() {
		t.Fatalf("EnsureDir() created %q as a non-directory", dir)
	}
	if got := info.Mode().Perm(); got != 0o700 {
		t.Fatalf("config directory permissions = %o, want 700", got)
	}
}
