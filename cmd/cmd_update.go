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
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/volcengine/byted-postgresql-cli/internal/update"
)

const releaseCheckInterval = 10 * time.Hour

func newUpdateCmd(ctx ProviderContext) *cobra.Command {
	var checkOnly, force bool
	cmd := &cobra.Command{
		Use:   "update",
		Short: "Update byted-postgresql-cli to the latest npm version",
		Long: "Check and update the globally installed `@byted-postgresql/cli` npm package from the public npm registry.\n\n" +
			"Prerequisites:\n" +
			"- npm 8 or newer must be installed and available in PATH;\n" +
			"- the public npm registry must be reachable.\n\n" +
			"`--check` only queries the latest version and never installs. Without `--check`, update installs into npm's default global prefix, matching `install` and the upstream Supabase CLI. It does not rebuild the current source tree or replace the repository's `./bin/byted-postgresql-cli`. " +
			"After installing, update verifies the on-disk binary reports the target version and fails loudly with a prefix-aware reinstall command if it does not. " +
			"Offline environments cannot check or update. If installation fails, npm reports the error and the existing installed version remains in place; retry after restoring registry access or run the printed npm install command manually.",
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return update.Run(cmd.Context(), Version, checkOnly, force, cmd.OutOrStdout(), cmd.ErrOrStderr())
		},
	}
	cmd.Flags().BoolVar(&checkOnly, "check", false, "Only check for updates, do not install.")
	cmd.Flags().BoolVar(&force, "force", false, "Reinstall even if already up to date.")
	return cmd
}

func checkForUpgrade(ctx context.Context, executed *cobra.Command) {
	if executed != nil && (executed.Name() == "update" || executed.Name() == "version") {
		return
	}
	if !isReleaseVersion(Version) {
		return
	}
	path, err := releaseCachePath()
	if err != nil {
		return
	}
	latest, err := readOrFetchLatest(ctx, path)
	if err != nil || !update.IsNewer(latest, Version) {
		return
	}
	fmt.Fprintf(os.Stderr, "\nA new version of Volcengine PostgreSQL CLI is available: %s (currently installed v%s)\nUpdate with: %s update\n",
		latest, strings.TrimPrefix(Version, "v"), binaryName())
}

func isReleaseVersion(version string) bool {
	parts := strings.Split(strings.TrimPrefix(strings.TrimSpace(version), "v"), ".")
	return len(parts) == 3 && version != "" && version != "dev" && !strings.Contains(version, "-")
}

func releaseCachePath() (string, error) {
	cache, err := os.UserCacheDir()
	if err != nil {
		return "", err
	}
	dir := filepath.Join(cache, "byted-postgresql-cli")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return "", err
	}
	return filepath.Join(dir, "latest-version.json"), nil
}

func readOrFetchLatest(ctx context.Context, path string) (string, error) {
	if info, err := os.Stat(path); err == nil && time.Since(info.ModTime()) < releaseCheckInterval {
		var cached struct {
			Version string `json:"version"`
		}
		if data, err := os.ReadFile(path); err == nil && json.Unmarshal(data, &cached) == nil {
			return cached.Version, nil
		}
	}
	latest, err := update.LatestRelease(ctx)
	if err != nil {
		return "", err
	}
	data, err := json.Marshal(struct {
		Version string `json:"version"`
	}{latest})
	if err == nil {
		_ = os.WriteFile(path, data, 0o600)
	}
	return latest, nil
}

func binaryName() string {
	return "byted-postgresql-cli"
}
