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

// Package installstate reads and writes the install-state manifest that records
// where and what the npm installer placed the CLI. `update` reads it so it
// targets the same install prefix the user's `install` wrote to (and the same
// binary the PATH points at), instead of npm's default global prefix — the two
// are different directories, which silently leaves the on-PATH binary stale.
//
// The manifest is written by bin/install.js (see writeInstallState there); this
// package is its Go-side reader/updater. Keep the schema in sync with that file.
package installstate

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/volcengine/byted-postgresql-cli/internal/config"
)

// FileName is the manifest name; it lives in the config/ subdirectory of the
// install root, a sibling of the npm-managed bin/, lib/, and <version>/ dirs
// that install/--force never touch.
const FileName = "install-state.json"

// State mirrors the JSON written by bin/install.js.
type State struct {
	Prefix    string `json:"prefix"`
	Bin       string `json:"bin"`
	Version   string `json:"version"`
	Registry  string `json:"registry"`
	Source    string `json:"source"`
	UpdatedAt string `json:"updated_at"`
}

// Path returns the manifest path under the given install root.
func Path(installRoot string) string {
	return filepath.Join(installRoot, "config", FileName)
}

// DefaultPath returns the manifest path under config.InstallRoot().
func DefaultPath() string {
	return Path(config.InstallRoot())
}

// Load reads the manifest at DefaultPath(). ok is false when the file is absent
// (an older install predating this manifest, or a manual install), letting
// callers fall back gracefully rather than failing.
func Load() (state State, ok bool, err error) {
	return LoadFrom(DefaultPath())
}

// LoadFrom reads the manifest at an explicit path.
func LoadFrom(path string) (state State, ok bool, err error) {
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return State{}, false, nil
	}
	if err != nil {
		return State{}, false, err
	}
	if err := json.Unmarshal(data, &state); err != nil {
		return State{}, false, err
	}
	if err := state.Validate(); err != nil {
		return State{}, false, err
	}
	return state, true, nil
}

// Validate checks the fields required to target and verify an installation.
func (s State) Validate() error {
	if s.Prefix == "" {
		return fmt.Errorf("prefix is missing")
	}
	if s.Bin == "" {
		return fmt.Errorf("bin is missing")
	}
	if s.Registry == "" {
		return fmt.Errorf("registry is missing")
	}
	return nil
}

// Save writes the manifest to DefaultPath(), stamping UpdatedAt.
func Save(state State) error {
	return SaveTo(DefaultPath(), state)
}

// SaveTo writes the manifest to an explicit path (0600, dir 0700).
func SaveTo(path string, state State) error {
	if state.UpdatedAt == "" {
		state.UpdatedAt = time.Now().UTC().Format(time.RFC3339)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	return os.WriteFile(path, data, 0o600)
}
