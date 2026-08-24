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

package installstate

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSaveLoadRoundTrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config", FileName)
	want := State{
		Prefix:   "/home/x/.volcengine-postgresql",
		Bin:      "/home/x/.volcengine-postgresql/bin/byted-postgresql-cli",
		Version:  "0.0.2",
		Registry: "https://registry.npmjs.org",
		Source:   "installer",
	}
	if err := SaveTo(path, want); err != nil {
		t.Fatal(err)
	}
	got, ok, err := LoadFrom(path)
	if err != nil || !ok {
		t.Fatalf("LoadFrom() ok=%v err=%v", ok, err)
	}
	if got.Prefix != want.Prefix || got.Bin != want.Bin || got.Version != want.Version || got.Registry != want.Registry || got.Source != want.Source {
		t.Fatalf("round trip mismatch: got %#v want %#v", got, want)
	}
	if got.UpdatedAt == "" {
		t.Fatal("SaveTo should stamp UpdatedAt")
	}
	// File must be private.
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if perm := info.Mode().Perm(); perm != 0o600 {
		t.Fatalf("state file perm = %o, want 600", perm)
	}
}

func TestLoadMissingReturnsNotOK(t *testing.T) {
	_, ok, err := LoadFrom(filepath.Join(t.TempDir(), "absent.json"))
	if err != nil {
		t.Fatalf("missing file should not error, got %v", err)
	}
	if ok {
		t.Fatal("ok should be false for a missing file")
	}
}

func TestLoadCorruptReturnsError(t *testing.T) {
	path := filepath.Join(t.TempDir(), FileName)
	if err := os.WriteFile(path, []byte("{not json"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, _, err := LoadFrom(path); err == nil {
		t.Fatal("corrupt JSON should return an error")
	}
}

func TestPathIsUnderConfigSubdir(t *testing.T) {
	got := Path("/root")
	want := filepath.Join("/root", "config", FileName)
	if got != want {
		t.Fatalf("Path() = %q, want %q", got, want)
	}
}
