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

package harness

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

const (
	ModeReplay = "replay"
	ModeLive   = "live"

	envE2EAccessKey   = "VOLCENGINE_E2E_ACCESS_KEY"
	envE2ESecretKey   = "VOLCENGINE_E2E_SECRET_KEY"
	envE2ERegion      = "VOLCENGINE_REGION"
	envE2EAuthMode    = "VOLCENGINE_E2E_AUTH_MODE"
	envE2EProfileDir  = "VOLCENGINE_E2E_PROFILE_DIR"
	envE2EProfileName = "VOLCENGINE_E2E_PROFILE"

	authModeAK      = "ak"
	authModeProfile = "profile"
)

type Result struct {
	ExitCode int
	Stdout   string
	Stderr   string
}

type Config struct {
	Binary string
	Mode   string
	Env    []string
	Dir    string
}

func New(t *testing.T) *Config {
	t.Helper()
	mode := strings.TrimSpace(os.Getenv("VOLCENGINE_E2E_MODE"))
	if mode == "" {
		mode = ModeReplay
	}
	if mode != ModeReplay && mode != ModeLive {
		t.Fatalf("VOLCENGINE_E2E_MODE must be %q or %q, got %q", ModeReplay, ModeLive, mode)
	}
	if mode == ModeLive && liveAuthMode() != authModeAK && liveAuthMode() != authModeProfile {
		t.Fatalf("%s must be %q or %q, got %q", envE2EAuthMode, authModeAK, authModeProfile, liveAuthMode())
	}
	dir := t.TempDir()
	env := []string{"HOME=" + dir}
	if runtime.GOOS == "windows" {
		env = append(env, "USERPROFILE="+dir)
	}
	if mode == ModeReplay {
		server := NewReplayServer(t)
		env = append(env,
			"VOLCENGINE_ACCESS_KEY=e2e-access-key",
			"VOLCENGINE_SECRET_KEY=e2e-secret-key",
			"VOLCENGINE_REGION=cn-beijing",
			"VOLCENGINE_ENDPOINT="+server.URL(),
		)
	} else {
		switch liveAuthMode() {
		case authModeAK:
			env = append(env,
				"VOLCENGINE_ACCESS_KEY="+os.Getenv(envE2EAccessKey),
				"VOLCENGINE_SECRET_KEY="+os.Getenv(envE2ESecretKey),
				"VOLCENGINE_REGION="+os.Getenv(envE2ERegion),
			)
		case authModeProfile:
			profileName := strings.TrimSpace(os.Getenv(envE2EProfileName))
			if profileName == "" {
				profileName = "default"
			}
			if err := prepareProfileEnvironment(dir, os.Getenv(envE2EProfileDir)); err != nil {
				t.Fatalf("prepare live Console Login profile: %v", err)
			}
			env = append(env,
				"VOLCENGINE_PROFILE="+profileName,
				"VOLCENGINE_REGION="+os.Getenv(envE2ERegion),
			)
		}
	}
	return &Config{
		Binary: buildBinary(t),
		Mode:   mode,
		Dir:    dir,
		Env:    env,
	}
}

func (c *Config) Run(t *testing.T, args ...string) Result {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, c.Binary, args...)
	cmd.Dir = c.Dir
	cmd.Env = append(cleanEnvironment(c.Mode), c.Env...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	if ctx.Err() != nil {
		t.Fatalf("command timed out: %s %s", c.Binary, strings.Join(args, " "))
	}

	result := Result{Stdout: stdout.String(), Stderr: stderr.String()}
	if err == nil {
		return result
	}
	if exitErr, ok := err.(*exec.ExitError); ok {
		result.ExitCode = exitErr.ExitCode()
		return result
	}
	t.Fatalf("run %s %s: %v", c.Binary, strings.Join(args, " "), err)
	return Result{}
}

func (c *Config) RequireSuccess(t *testing.T, result Result) {
	t.Helper()
	if result.ExitCode != 0 {
		t.Fatalf("expected exit code 0, got %d\nstdout:\n%s\nstderr:\n%s", result.ExitCode, result.Stdout, result.Stderr)
	}
}

func (c *Config) RequireFailure(t *testing.T, result Result) {
	t.Helper()
	if result.ExitCode == 0 {
		t.Fatalf("expected command failure\nstdout:\n%s\nstderr:\n%s", result.Stdout, result.Stderr)
	}
}

func (c *Config) RequireJSON(t *testing.T, result Result) map[string]any {
	t.Helper()
	c.RequireSuccess(t, result)
	var value map[string]any
	if err := json.Unmarshal([]byte(result.Stdout), &value); err != nil {
		t.Fatalf("parse JSON stdout: %v\nstdout:\n%s", err, result.Stdout)
	}
	return value
}

func (c *Config) RequireLive(t *testing.T) {
	t.Helper()
	if c.Mode != ModeLive {
		t.Skip("set VOLCENGINE_E2E_MODE=live to run live E2E tests")
	}
	if mode := liveAuthMode(); mode == authModeAK {
		for _, name := range []string{envE2EAccessKey, envE2ESecretKey, envE2ERegion} {
			if strings.TrimSpace(os.Getenv(name)) == "" {
				t.Fatalf("live E2E with %s auth requires %s", authModeAK, name)
			}
		}
		return
	}
	if strings.TrimSpace(os.Getenv(envE2EProfileDir)) == "" {
		t.Fatalf("live E2E with %s auth requires %s", authModeProfile, envE2EProfileDir)
	}
	if strings.TrimSpace(os.Getenv(envE2ERegion)) == "" {
		t.Fatalf("live E2E requires %s", envE2ERegion)
	}
}

func liveAuthMode() string {
	mode := strings.ToLower(strings.TrimSpace(os.Getenv(envE2EAuthMode)))
	if mode == "" {
		return authModeAK
	}
	return mode
}

func prepareProfileEnvironment(home, source string) error {
	source = strings.TrimSpace(source)
	if source == "" {
		return fmt.Errorf("%s is not set", envE2EProfileDir)
	}
	info, err := os.Stat(source)
	if err != nil {
		return err
	}
	if !info.IsDir() {
		return fmt.Errorf("%s must be a directory", envE2EProfileDir)
	}
	target := filepath.Join(home, ".volcengine")
	return copyDirectory(source, target)
}

func copyDirectory(source, target string) error {
	return filepath.Walk(source, func(path string, info os.FileInfo, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		relative, err := filepath.Rel(source, path)
		if err != nil {
			return err
		}
		destination := filepath.Join(target, relative)
		if info.IsDir() {
			return os.MkdirAll(destination, info.Mode().Perm())
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		if err := os.MkdirAll(filepath.Dir(destination), 0o700); err != nil {
			return err
		}
		return os.WriteFile(destination, data, info.Mode().Perm())
	})
}

func RequireReplay(t *testing.T) *Config {
	t.Helper()
	config := New(t)
	if config.Mode != ModeReplay {
		t.Skip("replay E2E suite is disabled in live mode")
	}
	return config
}

func buildBinary(t *testing.T) string {
	t.Helper()
	output := filepath.Join(t.TempDir(), "byted-postgresql-cli")
	if runtime.GOOS == "windows" {
		output += ".exe"
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	cmd := exec.CommandContext(ctx, "go", "build", "-trimpath", "-o", output, ".")
	cmd.Dir = moduleRoot(t)
	cmd.Env = append(cleanEnvironment(ModeReplay), "CGO_ENABLED=0", "HOME="+os.Getenv("HOME"))
	if outputBytes, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("build E2E binary: %v\n%s", err, outputBytes)
	}
	return output
}

func moduleRoot(t *testing.T) string {
	t.Helper()
	_, sourceFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("resolve harness source path")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(sourceFile), "..", "..", ".."))
}

func cleanEnvironment(mode string) []string {
	env := os.Environ()
	filtered := make([]string, 0, len(env))
	for _, item := range env {
		name := item
		if index := strings.IndexByte(item, '='); index >= 0 {
			name = item[:index]
		}
		switch {
		case name == "VOLCENGINE_ACCESS_KEY",
			name == "VOLCENGINE_SECRET_KEY",
			name == "VOLCENGINE_SESSION_TOKEN",
			name == "VOLCENGINE_REGION",
			name == "VOLCENGINE_PROFILE",
			name == "VOLCENGINE_ENDPOINT",
			name == "VOLCENGINE_LOGIN_CACHE_DIRECTORY",
			name == "HOME",
			name == "USERPROFILE":
			continue
		default:
			filtered = append(filtered, item)
		}
	}
	return filtered
}
