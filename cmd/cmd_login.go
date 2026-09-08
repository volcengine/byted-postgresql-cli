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
	"github.com/spf13/cobra"

	"github.com/volcengine/byted-postgresql-cli/internal/volcengine"
)

// newLoginCmd authenticates the CLI with Volcengine using the browser Console
// Login (OAuth 2.0 + PKCE) flow, caching temporary STS credentials locally.
func newLoginCmd(ctx ProviderContext) *cobra.Command {
	var (
		profile        string
		region         string
		remote         bool
		endpointURL    string
		skipRegion     bool
		credentialFile string
		assumeYes      bool
	)
	cmd := &cobra.Command{
		Use:   "login",
		Short: "Authenticate with the selected cloud provider in your browser",
		Long: `Authenticate with Volcengine using OAuth 2.0 + PKCE.
Opens a browser for authentication and caches temporary credentials locally.

Supports three modes:
  - Local (default): opens the browser on this device
  - Remote (--remote): for headless environments, prints a URL and reads the code
  - Credential file (--credential-file): imports an existing login cache

The region is used as the default region for subsequent API calls.
Use --skip-region to authenticate without saving a profile region.`,
		Example: `  byted-postgresql-cli login
  byted-postgresql-cli login --region cn-beijing
  byted-postgresql-cli login --remote
  byted-postgresql-cli login --credential-file /path/to/cache.json`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return volcengine.RunConsoleLogin(cmd.Context(), volcengine.ConsoleLoginParams{
				Profile:        profile,
				Region:         region,
				Remote:         remote,
				EndpointURL:    endpointURL,
				SkipRegion:     skipRegion,
				CredentialFile: credentialFile,
				AssumeYes:      assumeYes,
			}, cmd.InOrStdin(), cmd.OutOrStdout())
		},
	}
	flags := cmd.Flags()
	flags.StringVarP(&profile, "profile", "p", "default", "Credential profile name")
	flags.StringVarP(&region, "region", "r", "", "Cloud region (prompts when omitted; selects the provider)")
	flags.BoolVar(&remote, "remote", false, "Enable cross-device remote login mode")
	flags.StringVar(&endpointURL, "endpoint-url", "", "Override the provider signin service endpoint URL")
	flags.BoolVar(&skipRegion, "skip-region", false, "Authenticate without prompting for or saving a default region")
	flags.StringVar(&credentialFile, "credential-file", "", "Import a login cache file into this profile")
	flags.BoolVarP(&assumeYes, "yes", "y", false, "Replace an existing login session without prompting")
	return cmd
}

// newLogoutCmd removes locally cached Console Login credentials and deletes
// console-login profiles. It never touches AK/SK profiles.
func newLogoutCmd(ctx ProviderContext) *cobra.Command {
	var (
		profile string
		all     bool
	)
	provider := ctx.Provider
	cmd := &cobra.Command{
		Use:   "logout",
		Short: "Log out from Volcengine",
		Long: `Remove locally cached login credentials and delete console-login profiles.

This is a purely local operation. It deletes cached credential files from disk
and deletes console-login profiles from the CLI configuration. It does not
delete AK/SK profiles.`,
		Example: `  byted-postgresql-cli logout
  byted-postgresql-cli logout --profile dev
  byted-postgresql-cli logout --all`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return volcengine.RunConsoleLogout(volcengine.ConsoleLogoutParams{
				Provider: provider,
				Profile:  profile,
				All:      all,
			}, cmd.OutOrStdout())
		},
	}
	flags := cmd.Flags()
	flags.StringVarP(&profile, "profile", "p", "default", "Credential profile name")
	flags.BoolVar(&all, "all", false, "Log out all console-login profiles and remove all cached credentials")
	return cmd
}
