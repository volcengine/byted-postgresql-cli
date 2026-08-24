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
	"fmt"
	"sort"
	"strings"

	"github.com/spf13/cobra"

	"github.com/volcengine/byted-postgresql-cli/internal/volcengine"
)

// newConfigureCmd manages Volcengine AK/SK credential profiles stored in
// ~/.volcengine/config.json (shared with the official volcengine CLI). It
// uses Volcengine access
// is authenticated with a static Access Key / Secret Key pair.
func newConfigureCmd(ctx ProviderContext) *cobra.Command {
	provider := ctx.Provider
	cmd := &cobra.Command{
		Use:   "configure",
		Short: "Manage cloud provider AK/SK credentials",
		Long: `Manage credential profiles stored in the selected provider configuration directory.

Access uses a static Access Key / Secret Key pair. Configure a
profile once with ` + "`configure set`" + `, or supply credentials per invocation via the
` + volcengine.AccessKeyEnvironment(provider) + ` and ` + volcengine.SecretKeyEnvironment(provider) + ` environment variables.`,
	}
	cmd.AddCommand(newConfigureSetCmd(provider), newConfigureGetCmd(provider), newConfigureListCmd(provider))
	return cmd
}

func newConfigureSetCmd(providers ...volcengine.Provider) *cobra.Command {
	var (
		profileName  string
		accessKey    string
		secretKey    string
		sessionToken string
		region       string
		endpoint     string
	)
	provider := providerForCommandTree()
	if len(providers) > 0 {
		provider = providers[0]
	}
	cmd := &cobra.Command{
		Use:   "set",
		Short: "Set a Volcengine AK/SK profile",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if strings.TrimSpace(accessKey) == "" {
				return fmt.Errorf("--access-key is required")
			}
			if strings.TrimSpace(secretKey) == "" {
				return fmt.Errorf("--secret-key is required")
			}
			if region == "" {
				spec, _ := volcengine.ProviderSpecFor(provider)
				region = spec.DefaultRegion
			}
			if err := volcengine.ValidateRegion(region); err != nil {
				return err
			}
			cfg, err := volcengine.LoadFileConfigFor(provider)
			if err != nil {
				return err
			}
			if cfg.Profiles == nil {
				cfg.Profiles = map[string]*volcengine.Profile{}
			}
			name := profileName
			if name == "" {
				name = cfg.Current
			}
			if name == "" {
				name = "default"
			}
			cfg.Profiles[name] = &volcengine.Profile{
				Name:         name,
				Mode:         volcengine.ModeAK,
				Provider:     provider,
				AccessKey:    strings.TrimSpace(accessKey),
				SecretKey:    strings.TrimSpace(secretKey),
				SessionToken: strings.TrimSpace(sessionToken),
				Region:       strings.TrimSpace(region),
				Endpoint:     strings.TrimSpace(endpoint),
			}
			cfg.Current = name
			if err := volcengine.SaveFileConfigFor(provider, cfg); err != nil {
				return err
			}
			path, err := volcengine.ConfigFilePath(provider)
			if err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Saved %s profile %q to %s.\n", provider, name, path)
			return nil
		},
	}
	cmd.Flags().StringVar(&profileName, "profile", "", "Profile name (defaults to the current profile or \"default\")")
	cmd.Flags().StringVar(&accessKey, "access-key", "", "Cloud provider Access Key ID")
	cmd.Flags().StringVar(&secretKey, "secret-key", "", "Cloud provider Secret Access Key")
	cmd.Flags().StringVar(&sessionToken, "session-token", "", "STS session token (optional)")
	cmd.Flags().StringVar(&region, "region", "", "Cloud provider region")
	cmd.Flags().StringVar(&endpoint, "endpoint", "", "Custom AIDAP endpoint (optional)")
	return cmd
}

func newConfigureGetCmd(providers ...volcengine.Provider) *cobra.Command {
	var profileName string
	provider := providerForCommandTree()
	if len(providers) > 0 {
		provider = providers[0]
	}
	cmd := &cobra.Command{
		Use:   "get",
		Short: "Show a Volcengine profile (secret masked)",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := volcengine.LoadFileConfigFor(provider)
			if err != nil {
				return err
			}
			name := profileName
			if name == "" {
				name = cfg.Current
			}
			if name == "" {
				name = "default"
			}
			profile, ok := cfg.Profiles[name]
			if !ok {
				return fmt.Errorf("Volcengine profile %q not found; run `byted-postgresql-cli configure set --access-key <key> --secret-key <secret>`", name)
			}
			g := fromCtx(cmd)
			item, fields := configureProfileFields(profile, name == cfg.Current)
			return g.Writer().WriteItem(item, fields)
		},
	}
	cmd.Flags().StringVar(&profileName, "profile", "", "Profile name (defaults to the current profile)")
	return cmd
}

func newConfigureListCmd(providers ...volcengine.Provider) *cobra.Command {
	provider := providerForCommandTree()
	if len(providers) > 0 {
		provider = providers[0]
	}
	return &cobra.Command{
		Use:   "list",
		Short: "List Volcengine profiles",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := volcengine.LoadFileConfigFor(provider)
			if err != nil {
				return err
			}
			names := make([]string, 0, len(cfg.Profiles))
			for name := range cfg.Profiles {
				names = append(names, name)
			}
			sort.Strings(names)
			items := make([]map[string]string, 0, len(names))
			for _, name := range names {
				item, _ := configureProfileFields(cfg.Profiles[name], name == cfg.Current)
				items = append(items, item)
			}
			g := fromCtx(cmd)
			return g.Writer().WriteList(items, []string{"current", "profile", "access_key", "region", "endpoint"})
		},
	}
}

func configureProfileFields(profile *volcengine.Profile, isCurrent bool) (map[string]string, []string) {
	item := map[string]string{
		"profile":    profile.Name,
		"access_key": maskSecret(profile.AccessKey),
		"region":     profile.Region,
		"endpoint":   profile.Endpoint,
		"current":    boolLabel(isCurrent),
	}
	fields := []string{"profile", "current", "access_key", "region", "endpoint"}
	return item, fields
}

func maskSecret(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	if len(value) <= 8 {
		return strings.Repeat("*", len(value))
	}
	return value[:4] + strings.Repeat("*", len(value)-8) + value[len(value)-4:]
}

func boolLabel(v bool) string {
	if v {
		return "yes"
	}
	return "no"
}
