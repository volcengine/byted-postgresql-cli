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
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	"github.com/volcengine/byted-postgresql-cli/internal/config"
	"github.com/volcengine/byted-postgresql-cli/internal/log"
	"github.com/volcengine/byted-postgresql-cli/internal/pagination"
	"github.com/volcengine/byted-postgresql-cli/internal/volcengine"
	"github.com/volcengine/byted-postgresql-cli/internal/writer"
)

// Version is stamped at build time via -ldflags "-X ...cli.Version=...".
var Version = "dev"

const (
	commandGroupAuthentication = "authentication"
	commandGroupDiscovery      = "discovery"
	commandGroupDatabaseAccess = "database-access"
	commandGroupResources      = "database-resources"
	commandGroupSettings       = "workspace-settings"
	commandGroupTools          = "tools"
)

// Globals holds the resolved global flags for the current invocation. They are
// populated in PersistentPreRunE and read by subcommand handlers via the
// context helper below.
type Globals struct {
	ConfigDir string
	Output    string
	Region    string
	Profile   string
	Provider  volcengine.Provider
	Debug     bool
}

type ctxKey struct{}

func withGlobals(ctx context.Context, g *Globals) context.Context {
	return context.WithValue(ctx, ctxKey{}, g)
}

func fromCtx(cmd *cobra.Command) *Globals {
	if cmd == nil || cmd.Context() == nil {
		return &Globals{}
	}
	g, _ := cmd.Context().Value(ctxKey{}).(*Globals)
	if g == nil {
		return &Globals{}
	}
	return g
}

func (g *Globals) Writer() *writer.Writer { return writer.New(g.Output) }

// NewVolcClient builds a Volcengine AIDAP control-plane client (workspaces /
// branches) for the PostgreSQL engine from the resolved AK/SK credentials +
// region. Errors are passed through so callers receive the gateway error
// directly.
func (g *Globals) NewVolcClient(ctx context.Context) (*volcengine.Client, error) {
	cfg, err := volcengine.ResolveConfig(ctx)
	if err != nil {
		return nil, err
	}
	return volcengine.NewClient(cfg)
}

func (g *Globals) ResolveWorkspace(explicit string) string {
	return explicit
}

func (g *Globals) ResolveBranch(explicit string) string {
	return explicit
}

// Execute wires up the root command and runs the CLI.
func Execute() error {
	root := NewRootCommand(rootProviderContext())
	executed, err := root.ExecuteC()
	if err == nil {
		ctx := root.Context()
		if executed != nil {
			ctx = executed.Context()
		}
		checkForUpgrade(ctx, executed)
	}
	return err
}

func newRootCmd() *cobra.Command {
	return NewRootCommand(defaultProviderContext())
}

func NewRootCommand(providerContext ProviderContext) *cobra.Command {
	g := &Globals{}
	displayName := "Volcengine"
	longDescription := fmt.Sprintf(`%s PostgreSQL CLI %s — manage %s Serverless Postgres from the terminal.`, displayName, versionLabel(Version), displayName)

	cmd := &cobra.Command{
		Use:           "byted-postgresql-cli",
		Short:         fmt.Sprintf("%s PostgreSQL CLI %s", displayName, versionLabel(Version)),
		Long:          longDescription,
		SilenceUsage:  true,
		SilenceErrors: true,
		Version:       Version,
	}
	cmd.SetVersionTemplate("{{.Version}}\n")
	cmd.SetHelpCommandGroupID(commandGroupTools)
	cmd.SetCompletionCommandGroupID(commandGroupTools)

	// Global flags.
	cmd.PersistentFlags().StringVar(&g.ConfigDir, "config-dir", config.DefaultDir(), "Path to config directory")
	_ = cmd.PersistentFlags().MarkHidden("config-dir")
	cmd.PersistentFlags().StringVar(&g.Region, "region", "", fmt.Sprintf("Volcengine region (defaults to $%s or the provider default)", volcengine.EnvRegion))
	cmd.PersistentFlags().StringVar(&g.Profile, "profile", "", "Credential profile in the Volcengine configuration directory")
	cmd.PersistentFlags().StringVarP(&g.Output, "output", "o", "table", "Set output format (table|json|yaml|csv|tsv)")
	cmd.PersistentFlags().BoolVar(&g.Debug, "debug", false, "Enable debug logging")

	cmd.PersistentPreRunE = func(cmd *cobra.Command, args []string) error {
		if err := writer.ValidateFormat(g.Output); err != nil {
			return err
		}
		if region := strings.TrimSpace(g.Region); region != "" {
			if err := volcengine.ValidateRegion(region); err != nil {
				return err
			}
		}
		if profile := strings.TrimSpace(g.Profile); profile != "" {
			if err := volcengine.ValidateProfileForProvider(volcengine.ProviderVolcengine, profile); err != nil {
				return err
			}
		}
		if err := validatePaginationFlags(cmd); err != nil {
			return err
		}
		if err := validateRequiredStringFlags(cmd); err != nil {
			return err
		}
		log.SetDebug(g.Debug)
		if err := config.EnsureDir(g.ConfigDir); err != nil {
			return fmt.Errorf("prepare config dir: %w", err)
		}
		// Propagate --region/--profile into the Volcengine credential resolver so
		// every client built this invocation honors them.
		if strings.TrimSpace(g.Region) != "" {
			volcengine.SetRegionOverride(g.Region)
		}
		if strings.TrimSpace(g.Profile) != "" {
			volcengine.SetProfileOverride(g.Profile)
		}
		g.Provider = volcengine.ProviderVolcengine
		cmd.SetContext(withGlobals(cmd.Context(), g))
		return nil
	}

	tree := NewProviderCommandTree(providerContext)
	for _, group := range tree.Groups {
		cmd.AddGroup(group.Group)
		for _, factory := range group.Factories {
			command := factory(providerContext)
			groupID := group.Group.ID
			addRootCommand(cmd, groupID, command)
		}
	}

	return cmd
}

func addRootCommand(root *cobra.Command, groupID string, commands ...*cobra.Command) {
	for _, command := range commands {
		command.GroupID = groupID
		if len(command.Commands()) > 0 && command.RunE == nil && command.Run == nil {
			command.RunE = func(cmd *cobra.Command, args []string) error {
				if len(args) > 0 {
					return fmt.Errorf("unknown command %q for %q", args[0], cmd.CommandPath())
				}
				return cmd.Help()
			}
		}
	}
	root.AddCommand(commands...)
}

func validatePaginationFlags(cmd *cobra.Command) error {
	for _, name := range []string{"limit", "offset"} {
		flag := cmd.Flags().Lookup(name)
		if flag == nil {
			continue
		}
		value, err := cmd.Flags().GetInt(name)
		if err != nil {
			return err
		}
		if name == "limit" {
			if _, err := pagination.DefaultPolicy.Normalize(value, 0); err != nil {
				if value < 0 {
					return fmt.Errorf("--limit must be greater than or equal to 0")
				}
				return fmt.Errorf("--limit must be less than or equal to %d, got %d", pagination.DefaultPolicy.MaxLimit, value)
			}
			continue
		}
		if _, err := pagination.DefaultPolicy.Normalize(0, value); err != nil {
			return fmt.Errorf("--offset must be between 0 and %d, got %d", pagination.DefaultPolicy.MaxOffset, value)
		}
	}
	return nil
}

func validateRequiredStringFlags(cmd *cobra.Command) error {
	var validationErr error
	cmd.Flags().VisitAll(func(flag *pflag.Flag) {
		if validationErr != nil || flag.Value.Type() != "string" {
			return
		}
		required := false
		if values, ok := flag.Annotations[cobra.BashCompOneRequiredFlag]; ok {
			required = len(values) > 0 && values[0] == "true"
		}
		if required && flag.Changed {
			value, err := cmd.Flags().GetString(flag.Name)
			if err != nil {
				validationErr = err
				return
			}
			if strings.TrimSpace(value) == "" {
				validationErr = fmt.Errorf("--%s is required", flag.Name)
			}
		}
	})
	return validationErr
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}

func versionLabel(version string) string {
	version = strings.TrimSpace(strings.TrimPrefix(version, "v"))
	if version == "" {
		return "dev"
	}
	return version
}
