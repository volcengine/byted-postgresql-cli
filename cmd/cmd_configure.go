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
		Example: "byted-postgresql-cli configure set --profile default " +
			"--access-key <access-key> --secret-key <secret-key> --region cn-beijing",
	}
	cmd.AddCommand(
		newConfigureSetCmd(provider),
		newConfigureGetCmd(provider),
		newConfigureListCmd(provider),
		newConfigureProfileCmd(provider),
		newConfigureRegionCmd(provider),
		newConfigureAgentPlanCmd(provider),
		newConfigureDeleteCmd(provider),
	)
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

func newConfigureProfileCmd(providers ...volcengine.Provider) *cobra.Command {
	provider := providerForCommandTree()
	if len(providers) > 0 {
		provider = providers[0]
	}
	cmd := &cobra.Command{
		Use:   "profile <name>",
		Short: "Switch the current Volcengine profile",
		Args:  configureRequiredArg("profile", "name"),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := strings.TrimSpace(args[0])
			cfg, err := volcengine.LoadFileConfigFor(provider)
			if err != nil {
				return err
			}
			if cfg.Profiles[name] == nil {
				return fmt.Errorf("Volcengine profile %q not found", name)
			}
			cfg.Current = name
			if err := volcengine.SaveFileConfigFor(provider, cfg); err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Switched to Volcengine profile %q.\n", name)
			return nil
		},
	}
	return cmd
}

func newConfigureRegionCmd(providers ...volcengine.Provider) *cobra.Command {
	var profileName string
	provider := providerForCommandTree()
	if len(providers) > 0 {
		provider = providers[0]
	}
	cmd := &cobra.Command{
		Use:   "region <region>",
		Short: "Update the default region of a Volcengine profile",
		Args:  configureRequiredArg("region", "region"),
		RunE: func(cmd *cobra.Command, args []string) error {
			region := strings.TrimSpace(args[0])
			if err := volcengine.ValidateProviderRegion(provider, region); err != nil {
				return err
			}
			cfg, err := volcengine.LoadFileConfigFor(provider)
			if err != nil {
				return err
			}
			name := profileName
			if name == "" {
				name = cfg.Current
			}
			profile := cfg.Profiles[name]
			if profile == nil {
				return fmt.Errorf("Volcengine profile %q not found", name)
			}
			profile.Region = region
			if err := volcengine.SaveFileConfigFor(provider, cfg); err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Updated Volcengine profile %q region to %q.\n", name, region)
			return nil
		},
	}
	cmd.Flags().StringVar(&profileName, "profile", "", "Profile name (defaults to the current profile)")
	return cmd
}

func configureRequiredArg(commandName, argumentName string) cobra.PositionalArgs {
	return func(cmd *cobra.Command, args []string) error {
		usage := fmt.Sprintf("use `%s <%s>`", cmd.CommandPath(), argumentName)
		switch len(args) {
		case 0:
			return fmt.Errorf("%s is required; %s", argumentName, usage)
		case 1:
			if strings.TrimSpace(args[0]) == "" {
				return fmt.Errorf("%s cannot be empty; %s", argumentName, usage)
			}
			return nil
		default:
			return fmt.Errorf("%s accepts exactly one %s; %s", commandName, argumentName, usage)
		}
	}
}

func newConfigureAgentPlanCmd(providers ...volcengine.Provider) *cobra.Command {
	var (
		isAgentPlan bool
		seatID      string
	)
	provider := providerForCommandTree()
	if len(providers) > 0 {
		provider = providers[0]
	}
	cmd := &cobra.Command{
		Use:   "agent-plan",
		Short: "Update Agent Plan defaults for a Volcengine profile",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if !cmd.Flags().Changed("is-agent-plan") && !cmd.Flags().Changed("agent-plan-seat-id") {
				return fmt.Errorf("supply --is-agent-plan or --agent-plan-seat-id")
			}
			seatID = strings.TrimSpace(seatID)
			cfg, err := volcengine.LoadFileConfigFor(provider)
			if err != nil {
				return err
			}
			name := fromCtx(cmd).Profile
			if name == "" {
				name = cfg.Current
			}
			profile := cfg.Profiles[name]
			if profile == nil {
				return fmt.Errorf("Volcengine profile %q not found", name)
			}
			if err := updateAgentPlanProfile(
				profile,
				cmd.Flags().Changed("is-agent-plan"),
				isAgentPlan,
				cmd.Flags().Changed("agent-plan-seat-id"),
				seatID,
			); err != nil {
				return err
			}
			if err := volcengine.SaveFileConfigFor(provider, cfg); err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Updated Agent Plan defaults for Volcengine profile %q.\n", name)
			return nil
		},
	}
	cmd.Flags().BoolVar(&isAgentPlan, "is-agent-plan", false, "Use Agent Plan by default for this profile")
	cmd.Flags().StringVar(&seatID, "agent-plan-seat-id", "", "Default Agent Plan seat ID for enterprise edition")
	return cmd
}

func updateAgentPlanProfile(
	profile *volcengine.Profile,
	isAgentPlanChanged bool,
	isAgentPlan bool,
	seatIDChanged bool,
	seatID string,
) error {
	if seatID != "" && isAgentPlanChanged && !isAgentPlan {
		return fmt.Errorf("--is-agent-plan=false cannot be combined with a non-empty --agent-plan-seat-id")
	}
	if isAgentPlanChanged {
		profile.IsAgentPlan = isAgentPlan
		if !isAgentPlan && !seatIDChanged {
			profile.AgentPlanSeatID = ""
		}
	}
	if seatIDChanged {
		profile.AgentPlanSeatID = seatID
		if seatID != "" {
			profile.IsAgentPlan = true
		}
	}
	return nil
}

func newConfigureDeleteCmd(providers ...volcengine.Provider) *cobra.Command {
	var profileName string
	var assumeYes bool
	provider := providerForCommandTree()
	if len(providers) > 0 {
		provider = providers[0]
	}
	cmd := &cobra.Command{
		Use:   "delete [name]",
		Short: "Delete a Volcengine profile",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := volcengine.LoadFileConfigFor(provider)
			if err != nil {
				return err
			}
			name := profileName
			if len(args) == 1 {
				name = strings.TrimSpace(args[0])
			}
			if name == "" {
				name = cfg.Current
			}
			if cfg.Profiles[name] == nil {
				return fmt.Errorf("Volcengine profile %q not found", name)
			}
			if !assumeYes {
				return fmt.Errorf("pass --yes to delete Volcengine profile %q", name)
			}
			delete(cfg.Profiles, name)
			if cfg.Current == name {
				cfg.Current = "default"
				names := make([]string, 0, len(cfg.Profiles))
				for candidate := range cfg.Profiles {
					names = append(names, candidate)
				}
				sort.Strings(names)
				if len(names) > 0 {
					cfg.Current = names[0]
				}
			}
			if err := volcengine.SaveFileConfigFor(provider, cfg); err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Deleted Volcengine profile %q.\n", name)
			return nil
		},
	}
	cmd.Flags().StringVar(&profileName, "profile", "", "Profile name (defaults to the current profile)")
	cmd.Flags().BoolVar(&assumeYes, "yes", false, "Confirm profile deletion")
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
			return g.Writer().WriteList(items, []string{"current", "profile", "mode", "access_key", "region", "endpoint", "is_agent_plan", "agent_plan_seat_id"})
		},
	}
}

func configureProfileFields(profile *volcengine.Profile, isCurrent bool) (map[string]string, []string) {
	item := map[string]string{
		"profile":            profile.Name,
		"access_key":         maskSecret(profile.AccessKey),
		"region":             profile.Region,
		"endpoint":           profile.Endpoint,
		"mode":               profile.Mode,
		"is_agent_plan":      boolLabel(profile.IsAgentPlan),
		"agent_plan_seat_id": maskSecret(profile.AgentPlanSeatID),
		"current":            boolLabel(isCurrent),
	}
	fields := []string{"profile", "current", "mode", "access_key", "region", "endpoint", "is_agent_plan", "agent_plan_seat_id"}
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
