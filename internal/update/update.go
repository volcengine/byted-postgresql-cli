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

package update

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

var (
	NpmRegistry = "https://registry.npmjs.org"
	NpmPackage  = "@byted-postgresql/cli"
)

var httpClient = http.DefaultClient

// lookPath, runCommand, and binaryVersion are indirections so tests can inject
// a fake npm and a fake installed binary without shelling out to the real
// toolchain.
var lookPath = exec.LookPath

var npmPrefix = func(npm string) (string, error) {
	out, err := exec.Command(npm, "prefix", "-g").Output()
	return strings.TrimSpace(string(out)), err
}

var runCommand = func(ctx context.Context, out, errOut io.Writer, name string, args ...string) error {
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Stdout = out
	cmd.Stderr = errOut
	return cmd.Run()
}

// binaryVersion runs `<bin> --version` and returns the reported version. The
// native CLI prints just the version (see cli.SetVersionTemplate), so the raw
// trimmed output is the version string.
var binaryVersion = func(ctx context.Context, bin string) (string, error) {
	outBytes, err := exec.CommandContext(ctx, bin, "--version").Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(outBytes)), nil
}

var warmCache = func(ctx context.Context, bin string) error {
	return runCommand(ctx, io.Discard, io.Discard, bin, "__warm-cache")
}

// LatestRelease returns the latest version published under the npm latest tag.
// The returned version always has a leading "v", which makes it convenient to
// compare with the version stamped into the native binary.
func LatestRelease(ctx context.Context) (string, error) {
	return LatestReleaseFrom(ctx, NpmRegistry)
}

// LatestReleaseFrom queries the latest tag from an explicit npm registry.
func LatestReleaseFrom(ctx context.Context, registry string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, strings.TrimRight(registry, "/")+"/"+NpmPackage+"/latest", nil)
	if err != nil {
		return "", fmt.Errorf("create release request: %w", err)
	}
	resp, err := httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("fetch latest release: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("fetch latest release: status %s", resp.Status)
	}
	var manifest struct {
		Version string `json:"version"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&manifest); err != nil {
		return "", fmt.Errorf("parse latest release: %w", err)
	}
	if strings.TrimSpace(manifest.Version) == "" {
		return "", nil
	}
	return "v" + strings.TrimPrefix(strings.TrimSpace(manifest.Version), "v"), nil
}

// IsNewer reports whether latest is a valid release strictly newer than
// current. Development builds and malformed versions are never prompted.
func IsNewer(latest, current string) bool {
	l, lok := parseVersion(latest)
	c, cok := parseVersion(current)
	if !lok {
		return false
	}
	if !cok {
		return true
	}
	for i := range l {
		if l[i] != c[i] {
			return l[i] > c[i]
		}
	}
	return false
}

func parseVersion(version string) ([3]int, bool) {
	var result [3]int
	version = strings.TrimPrefix(strings.TrimSpace(version), "v")
	if version == "" {
		return result, false
	}
	parts := strings.SplitN(version, ".", 4)
	if len(parts) != 3 {
		return result, false
	}
	for i, part := range parts {
		if j := strings.IndexByte(part, '-'); j >= 0 {
			part = part[:j]
		}
		if part == "" {
			return result, false
		}
		n := 0
		for _, ch := range part {
			if ch < '0' || ch > '9' {
				return result, false
			}
			n = n*10 + int(ch-'0')
		}
		result[i] = n
	}
	return result, true
}

// Run installs the latest package from the configured public npm registry.
// --check is represented by checkOnly.
//
// It targets npm's default global prefix, matching the installer and the
// upstream Supabase CLI. After installing it verifies the on-disk binary
// actually reports the target version.
func Run(ctx context.Context, current string, checkOnly, force bool, out, errOut io.Writer) error {
	registry := NpmRegistry
	var prefix, bin string
	if !checkOnly {
		var err error
		prefix, registry, bin, err = resolveTarget()
		if err != nil {
			return fmt.Errorf("%w; reinstall manually: %s", err, installCommand("latest", NpmRegistry))
		}
	}

	latest, err := LatestReleaseFrom(ctx, registry)
	if err != nil {
		return err
	}
	current = strings.TrimSpace(current)
	if latest == "" {
		fmt.Fprintln(out, "No published version was found.")
		return nil
	}
	if !force && !IsNewer(latest, current) {
		fmt.Fprintf(out, "byted-postgresql-cli %s is already up to date.\n", displayVersion(current))
		return nil
	}
	if checkOnly {
		fmt.Fprintf(out, "Update available: %s -> %s\n", displayVersion(current), latest)
		fmt.Fprintf(out, "Run `byted-postgresql-cli update` to install.\n")
		return nil
	}

	npm, err := lookPath("npm")
	if err != nil {
		return fmt.Errorf("npm not found in PATH; install manually with: %s", installCommand(latest, registry))
	}
	fmt.Fprintf(errOut, "Updating byted-postgresql-cli %s -> %s via npm (prefix %s) ...\n", displayVersion(current), latest, prefix)
	args := []string{"install", "-g", NpmPackage + "@" + strings.TrimPrefix(latest, "v"), "--registry", registry}
	if err := runCommand(ctx, errOut, errOut, npm, args...); err != nil {
		return fmt.Errorf("npm install failed: %w\nreinstall manually: %s", err, installCommand(latest, registry))
	}

	// Verify the installed binary reports the target version. This catches the
	// class of failure where npm succeeds but the on-PATH binary is unchanged
	// (e.g. it was installed into a different prefix).
	if bin != "" {
		got, err := binaryVersion(ctx, bin)
		if err != nil {
			return fmt.Errorf(
				"update verification failed: could not execute %s --version: %w\nautomatic rollback is unavailable; reinstall manually: %s",
				bin, err, installCommand(latest, registry),
			)
		}
		if !sameVersion(got, latest) {
			return fmt.Errorf(
				"update verification failed: expected %s, got %s\nautomatic rollback is unavailable; reinstall manually: %s",
				latest, displayVersion(got), installCommand(latest, registry),
			)
		}
	}

	if bin != "" {
		if err := warmCache(ctx, bin); err != nil {
			fmt.Fprintf(errOut, "Warning: native binary cache warm skipped: %v\n", err)
		}
	}

	fmt.Fprintf(out, "Successfully updated byted-postgresql-cli to %s.\n", latest)
	return nil
}

// resolveTarget returns the install prefix, registry, and binary path to use.
// It resolves npm's global prefix so update follows the same installation
// location as `npm install -g` and the JavaScript installer.
func resolveTarget() (prefix, registry, bin string, err error) {
	npm, lookErr := lookPath("npm")
	if lookErr != nil {
		return "", "", "", lookErr
	}
	prefix, prefixErr := npmPrefix(npm)
	if prefixErr != nil {
		return "", "", "", fmt.Errorf("npm prefix -g failed: %w", prefixErr)
	}
	if prefix == "" {
		return "", "", "", fmt.Errorf("npm prefix -g returned an empty path")
	}
	registry = NpmRegistry
	bin = filepath.Join(prefix, "bin", "byted-postgresql-cli")
	if runtime.GOOS == "windows" {
		bin = filepath.Join(prefix, "byted-postgresql-cli.exe")
	}
	return prefix, registry, bin, nil
}

// sameVersion compares two version strings ignoring a leading "v".
func sameVersion(a, b string) bool {
	return strings.TrimPrefix(strings.TrimSpace(a), "v") == strings.TrimPrefix(strings.TrimSpace(b), "v")
}

// installCommand returns the global npm reinstall command.
func installCommand(version, registry string) string {
	return fmt.Sprintf("npm install -g %s@%s --registry %s",
		NpmPackage, strings.TrimPrefix(version, "v"), registry)
}

func displayVersion(version string) string {
	if version == "" || version == "dev" {
		return "(dev)"
	}
	return "v" + strings.TrimPrefix(version, "v")
}
