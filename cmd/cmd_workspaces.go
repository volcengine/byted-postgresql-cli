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
	"bufio"
	"fmt"
	"strings"
	"unicode"

	"github.com/spf13/cobra"

	"github.com/volcengine/byted-postgresql-cli/internal/volcengine"
)

var workspaceFields = []string{
	"WorkspaceName", "WorkspaceId", "ProjectName", "EngineType", "EngineVersion",
	"WorkspaceStatus", "DeletionProtectionStatus", "CreateTime", "UpdateTime",
}

// Bounds documented by the Volcengine AIDAP request models. Checking them here turns
// an opaque gateway rejection into an actionable message; anything the gateway
// alone knows (quota, CU step values, and permissions) still comes back
// unchanged.
const (
	minWorkspaceNameRunes = 2
	maxWorkspaceNameRunes = 64
	minSuspendTimeout     = 300
	maxSuspendTimeout     = 604800
	suspendTimeoutNever   = -1
	minCUValue            = 0.25
	maxCUValue            = 32.0
	maxCURatio            = 8
	minRetentionHours     = 1
	maxRetentionHours     = 720
)

// workspaceIDFromArgs reads the positional workspace id, falling back to the
// interactive picker when it is omitted — the behaviour promised by the root
// command's help and already used by `branches`.
func workspaceIDFromArgs(cmd *cobra.Command, args []string) (string, error) {
	flag := cmd.Flags().Lookup("workspace-id")
	if flag != nil && flag.Changed {
		workspaceID, err := cmd.Flags().GetString("workspace-id")
		if err != nil {
			return "", err
		}
		workspaceID = strings.TrimSpace(workspaceID)
		if workspaceID == "" {
			return "", fmt.Errorf("workspace id cannot be empty")
		}
		if len(args) > 0 {
			return "", fmt.Errorf("workspace id given twice: as %q and as --workspace-id %q", args[0], workspaceID)
		}
		return workspaceID, nil
	}
	if len(args) > 0 {
		if id := strings.TrimSpace(args[0]); id != "" {
			return id, nil
		}
		return "", fmt.Errorf("workspace id cannot be empty")
	}
	g := fromCtx(cmd)
	client, err := g.NewVolcClient(cmd.Context())
	if err != nil {
		return "", err
	}
	return resolveWorkspace(cmd.Context(), g, client, "")
}

// validateWorkspaceName mirrors the naming rule documented on
// CreateWorkspaceRequest.WorkspaceName: 2–64 characters, Chinese characters or
// letters/digits/-/_, starting with a Chinese character or a letter.
func validateWorkspaceName(name string) error {
	runes := []rune(name)
	if len(runes) < minWorkspaceNameRunes || len(runes) > maxWorkspaceNameRunes {
		return fmt.Errorf("workspace name %q must be %d-%d characters", name, minWorkspaceNameRunes, maxWorkspaceNameRunes)
	}
	if !unicode.IsLetter(runes[0]) {
		return fmt.Errorf("workspace name %q must start with a letter or a Chinese character", name)
	}
	for _, r := range runes {
		if unicode.IsLetter(r) || unicode.IsDigit(r) || r == '-' || r == '_' {
			continue
		}
		return fmt.Errorf("workspace name %q may only contain letters, digits, Chinese characters, '-' and '_'", name)
	}
	return nil
}

func validateSuspendTimeout(seconds int) error {
	if seconds == suspendTimeoutNever {
		return nil
	}
	if seconds < minSuspendTimeout || seconds > maxSuspendTimeout {
		return fmt.Errorf("--suspend-timeout must be %d (never suspend) or between %d and %d seconds, got %d",
			suspendTimeoutNever, minSuspendTimeout, maxSuspendTimeout, seconds)
	}
	return nil
}

func validateComputeUnits(minCU, maxCU float64) error {
	if minCU < minCUValue || minCU > maxCUValue {
		return fmt.Errorf("--min-cu must be between %g and %g compute units, got %g", minCUValue, maxCUValue, minCU)
	}
	if maxCU < minCU {
		return fmt.Errorf("--max-cu (%g) must be greater than or equal to --min-cu (%g)", maxCU, minCU)
	}
	if maxCU < minCUValue || maxCU > maxCUValue {
		return fmt.Errorf("--max-cu must be between %g and %g compute units, got %g", minCUValue, maxCUValue, maxCU)
	}
	if maxCU > minCU*maxCURatio {
		return fmt.Errorf("--max-cu (%g) may not exceed %d× --min-cu (%g)", maxCU, maxCURatio, minCU)
	}
	return nil
}

// confirmWorkspaceDeletion asks before an irreversible delete. Non-interactive
// runs must pass --yes so scripts never block on a prompt.
func confirmWorkspaceDeletion(cmd *cobra.Command, workspaceID, summary string) error {
	return confirmDestructiveAction(cmd, workspaceID, summary, "delete")
}

func confirmDestructiveAction(cmd *cobra.Command, target, summary, action string) error {
	if !interactiveTarget() {
		return fmt.Errorf("refusing to %s %s without confirmation; pass --yes in non-interactive mode", action, target)
	}
	fmt.Fprintf(cmd.OutOrStdout(), "%s\n\nType %s to confirm: ", summary, target)
	scanner := bufio.NewScanner(cmd.InOrStdin())
	if !scanner.Scan() {
		if err := scanner.Err(); err != nil {
			return fmt.Errorf("failed to read confirmation: %w", err)
		}
		return fmt.Errorf("aborted: no confirmation entered")
	}
	if strings.TrimSpace(scanner.Text()) != target {
		return fmt.Errorf("aborted: confirmation did not match %s", target)
	}
	return nil
}

func workspaceDeletionSummary(workspaceID string, workspace volcengine.Workspace, region string) string {
	name := workspace.WorkspaceName
	if name == "" {
		name = "unknown"
	}
	project := workspace.ProjectName
	if project == "" {
		project = "unknown"
	}
	return fmt.Sprintf(
		"Delete workspace %q (%s)?\nProject: %s\nRegion: %s\nThis operation cannot be undone. It also deletes the workspace's branches, computes, databases, and endpoints.\nResource counts are not available from the preview API.\nFor automation, use --yes only after verifying the workspace ID and cascade impact.",
		name, workspaceID, project, region,
	)
}

// newWorkspacesCmd manages PostgreSQL workspaces. A workspace is a serverless
// PostgreSQL instance and the top-level resource on the Volcengine AIDAP platform;
// every listing here is filtered to the PostgreSQL engine.
func newWorkspacesCmd(ctx ProviderContext) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "workspaces",
		Aliases: []string{"workspace"},
		Short:   "Manage PostgreSQL workspaces",
	}

	cmd.AddCommand(
		newWorkspacesListCmd(),
		newWorkspacesGetCmd(),
		newWorkspacesCreateCmd(),
		newWorkspacesDeleteCmd(),
		newWorkspacesStartCmd(),
		newWorkspacesStopCmd(),
		newWorkspacesRenameCmd(),
		newWorkspacesDeletionProtectionCmd(),
		newWorkspacesComputeSettingsCmd(),
		newWorkspacesSettingsCmd(),
		newWorkspacesOverviewCmd(),
	)
	return cmd
}

func newWorkspacesListCmd() *cobra.Command {
	var (
		search      string
		projectName string
		limit       int
		offset      int
		all         bool
	)
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List PostgreSQL workspaces",
		RunE: func(cmd *cobra.Command, args []string) error {
			g := fromCtx(cmd)
			client, err := g.NewVolcClient(cmd.Context())
			if err != nil {
				return err
			}
			params := volcengine.ListWorkspacesParams{
				Search:      search,
				ProjectName: projectName,
				Limit:       requestedListLimit(cmd, limit),
				Offset:      offset,
			}
			var result volcengine.ListWorkspacesResult
			if all {
				result, err = client.ListAllWorkspaces(cmd.Context(), params)
			} else {
				result, err = client.ListWorkspaces(cmd.Context(), params)
			}
			if err != nil {
				return err
			}
			return writeListPage(cmd, g, result.Workspaces, workspaceFields, result.Total, limit, offset, "workspaces")
		},
	}
	cmd.Flags().StringVar(&search, "search", "", "Filter workspaces by name")
	cmd.Flags().StringVar(&projectName, "project-name", "", "Filter workspaces by the project name they belong to")
	cmd.Flags().IntVar(&limit, "limit", 0, "Maximum number of workspaces to return (default 10, 0=100, max 100)")
	cmd.Flags().IntVar(&offset, "offset", 0, "Number of workspaces to skip")
	cmd.Flags().BoolVar(&all, "all", false, "List every workspace (paginates through all pages)")
	// --all paginates to the end, so a page window alongside it is a
	// contradiction rather than a refinement.
	cmd.MarkFlagsMutuallyExclusive("all", "limit")
	cmd.MarkFlagsMutuallyExclusive("all", "offset")
	return cmd
}

func newWorkspacesGetCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "get [workspace-id]",
		Short: "Get a workspace by id",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			g := fromCtx(cmd)
			workspaceID, err := workspaceIDFromArgs(cmd, args)
			if err != nil {
				return err
			}
			client, err := g.NewVolcClient(cmd.Context())
			if err != nil {
				return err
			}
			ws, err := client.DescribeWorkspaceDetail(cmd.Context(), workspaceID)
			if err != nil {
				return err
			}
			return g.Writer().WriteItem(ws, workspaceFields)
		},
	}
	cmd.Flags().String("workspace-id", "", "Workspace ID (or pass it positionally)")
	return cmd
}

func newWorkspacesCreateCmd() *cobra.Command {
	var (
		name               string
		projectName        string
		suspendTimeout     int
		minCU              float64
		maxCU              float64
		retention          int
		sharedPublic       bool
		vpcID              string
		subnetID           string
		internetProtocol   string
		deletionProtection string
		tags               []string
	)
	cmd := &cobra.Command{
		Use:   "create [name]",
		Short: "Create a PostgreSQL workspace",
		Long: "Create a PostgreSQL workspace.\n\n" +
			"The name may be given positionally or with --name; these forms are equivalent. " +
			"Pass exactly one form: using both is an error. Whitespace-only values are rejected.",
		Example: "byted-postgresql-cli workspaces create my-pg",
		Args:    cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			g := fromCtx(cmd)
			// A positional name is the shape people reach for first; accept it
			// rather than silently dropping it and reporting a missing flag.
			if len(args) > 0 {
				if cmd.Flags().Changed("name") {
					return fmt.Errorf("workspace name given twice: as %q and as --name %q", args[0], name)
				}
				name = args[0]
			}
			name = strings.TrimSpace(name)
			if name == "" {
				return fmt.Errorf("workspace name is required; pass it positionally or with --name")
			}
			if err := validateWorkspaceName(name); err != nil {
				return err
			}
			if cmd.Flags().Changed("suspend-timeout") {
				if err := validateSuspendTimeout(suspendTimeout); err != nil {
					return err
				}
			}
			if cmd.Flags().Changed("min-cu") || cmd.Flags().Changed("max-cu") {
				if !cmd.Flags().Changed("min-cu") || !cmd.Flags().Changed("max-cu") {
					return fmt.Errorf("--min-cu and --max-cu must be provided together for workspace creation")
				}
				if err := validateComputeUnits(minCU, maxCU); err != nil {
					return err
				}
			}
			if cmd.Flags().Changed("retention-hours") &&
				(retention < minRetentionHours || retention > maxRetentionHours) {
				return fmt.Errorf("--retention-hours must be between %d and %d", minRetentionHours, maxRetentionHours)
			}
			if deletionProtection != "" && deletionProtection != "Enabled" && deletionProtection != "Disabled" {
				return fmt.Errorf("--deletion-protection must be Enabled or Disabled")
			}
			client, err := g.NewVolcClient(cmd.Context())
			if err != nil {
				return err
			}
			params := volcengine.CreateWorkspaceParams{
				WorkspaceName: name,
				ProjectName:   projectName,
			}
			if cmd.Flags().Changed("suspend-timeout") {
				params.SuspendTimeoutSeconds = &suspendTimeout
			}
			if cmd.Flags().Changed("min-cu") {
				params.MinCU = &minCU
				params.MaxCU = &maxCU
			}
			if cmd.Flags().Changed("retention-hours") {
				params.HistoryRetentionHours = &retention
			}
			if cmd.Flags().Changed("shared-public") {
				params.SharedPublicNetwork = &sharedPublic
			}
			params.VPCID = vpcID
			params.SubnetID = subnetID
			params.InternetProtocol = internetProtocol
			params.DeletionProtection = deletionProtection
			for _, raw := range tags {
				parts := strings.SplitN(raw, "=", 2)
				if len(parts) != 2 || strings.TrimSpace(parts[0]) == "" {
					return fmt.Errorf("--tag must use key=value, got %q", raw)
				}
				params.Tags = append(params.Tags, volcengine.WorkspaceTag{
					Key: strings.TrimSpace(parts[0]), Value: parts[1],
				})
			}
			result, err := client.CreateWorkspace(cmd.Context(), params)
			if err != nil {
				return err
			}
			return g.Writer().WriteItem(result.Workspace, workspaceFields)
		},
	}
	cmd.Flags().StringVar(&name, "name", "", "Workspace name (required)")
	cmd.Flags().StringVar(&projectName, "project-name", "", "Project name for billing allocation")
	cmd.Flags().IntVar(&suspendTimeout, "suspend-timeout", 0, "Auto-suspend timeout in seconds (-1 never suspends)")
	cmd.Flags().Float64Var(&minCU, "min-cu", 0, "Initial minimum compute units (0.25-32)")
	cmd.Flags().Float64Var(&maxCU, "max-cu", 0, "Initial maximum compute units (0.25-32, at most 8x --min-cu)")
	cmd.Flags().IntVar(&retention, "retention-hours", 0, "Initial point-in-time history retention in hours (1-720)")
	cmd.Flags().BoolVar(&sharedPublic, "shared-public", false, "Enable shared public network")
	cmd.Flags().StringVar(&vpcID, "vpc-id", "", "VPC ID")
	cmd.Flags().StringVar(&subnetID, "subnet-id", "", "Subnet ID")
	cmd.Flags().StringVar(&internetProtocol, "internet-protocol", "", "Internet protocol (IPv4 or DualStack)")
	cmd.Flags().StringVar(&deletionProtection, "deletion-protection", "", "Deletion protection (Enabled or Disabled)")
	cmd.Flags().StringSliceVar(&tags, "tag", nil, "Workspace tag key=value (repeatable)")
	return cmd
}

func newWorkspacesDeleteCmd() *cobra.Command {
	var yes bool
	cmd := &cobra.Command{
		Use:     "delete <workspace-id>",
		Aliases: []string{"rm"},
		Short:   "Delete a workspace",
		// Unlike the other subcommands this one does not fall back to the
		// interactive picker: a destructive call must name its target.
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			g := fromCtx(cmd)
			workspaceID := strings.TrimSpace(args[0])
			if workspaceID == "" {
				return fmt.Errorf("workspace id cannot be empty")
			}
			if !yes {
				summary := fmt.Sprintf("Delete workspace %q? This operation cannot be undone.", workspaceID)
				if interactiveTarget() {
					client, err := g.NewVolcClient(cmd.Context())
					if err != nil {
						return err
					}
					workspace, err := client.DescribeWorkspaceDetail(cmd.Context(), workspaceID)
					if err != nil {
						return err
					}
					summary = workspaceDeletionSummary(workspaceID, workspace, g.Region)
				}
				if err := confirmWorkspaceDeletion(cmd, workspaceID, summary); err != nil {
					return err
				}
			}
			client, err := g.NewVolcClient(cmd.Context())
			if err != nil {
				return err
			}
			if _, err := client.DeleteWorkspace(cmd.Context(), workspaceID); err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Workspace %s deleted\n", workspaceID)
			return nil
		},
	}
	cmd.Flags().BoolVarP(&yes, "yes", "y", false, "Skip the confirmation prompt")
	return cmd
}

func newWorkspacesStartCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "start [workspace-id]",
		Short: "Start a stopped workspace",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			g := fromCtx(cmd)
			workspaceID, err := workspaceIDFromArgs(cmd, args)
			if err != nil {
				return err
			}
			client, err := g.NewVolcClient(cmd.Context())
			if err != nil {
				return err
			}
			if err := client.StartWorkspace(cmd.Context(), workspaceID); err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Workspace %s starting\n", workspaceID)
			return nil
		},
	}
	return cmd
}

func newWorkspacesStopCmd() *cobra.Command {
	var yes bool
	cmd := &cobra.Command{
		Use:   "stop [workspace-id]",
		Short: "Stop a running workspace",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			g := fromCtx(cmd)
			workspaceID, err := workspaceIDFromArgs(cmd, args)
			if err != nil {
				return err
			}
			if !yes {
				summary := fmt.Sprintf("Stop workspace %q? The workspace can be started again, but active connections will be interrupted.", workspaceID)
				if err := confirmDestructiveAction(cmd, workspaceID, summary, "stop"); err != nil {
					return err
				}
			}
			client, err := g.NewVolcClient(cmd.Context())
			if err != nil {
				return err
			}
			if err := client.StopWorkspace(cmd.Context(), workspaceID); err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Workspace %s stopping\n", workspaceID)
			return nil
		},
	}
	cmd.Flags().BoolVarP(&yes, "yes", "y", false, "Skip the confirmation prompt")
	return cmd
}

func newWorkspacesRenameCmd() *cobra.Command {
	var name string
	cmd := &cobra.Command{
		Use:   "rename [workspace-id] --name <new-name>",
		Short: "Rename a workspace",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			g := fromCtx(cmd)
			name = strings.TrimSpace(name)
			if name == "" {
				return fmt.Errorf("--name is required")
			}
			if err := validateWorkspaceName(name); err != nil {
				return err
			}
			workspaceID, err := workspaceIDFromArgs(cmd, args)
			if err != nil {
				return err
			}
			client, err := g.NewVolcClient(cmd.Context())
			if err != nil {
				return err
			}
			if err := client.ModifyWorkspaceName(cmd.Context(), workspaceID, name); err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Workspace %s renamed to %s\n", workspaceID, name)
			return nil
		},
	}
	cmd.Flags().StringVar(&name, "name", "", "New workspace name")
	return cmd
}

func newWorkspacesDeletionProtectionCmd() *cobra.Command {
	var (
		enable  bool
		disable bool
	)
	cmd := &cobra.Command{
		Use:   "deletion-protection [workspace-id]",
		Short: "Enable or disable deletion protection",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			g := fromCtx(cmd)
			if enable == disable {
				return fmt.Errorf("specify exactly one of --enable or --disable")
			}
			workspaceID, err := workspaceIDFromArgs(cmd, args)
			if err != nil {
				return err
			}
			client, err := g.NewVolcClient(cmd.Context())
			if err != nil {
				return err
			}
			if err := client.ModifyWorkspaceDeletionProtectionPolicy(cmd.Context(), workspaceID, enable); err != nil {
				return err
			}
			state := "disabled"
			if enable {
				state = "enabled"
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Deletion protection %s for workspace %s\n", state, workspaceID)
			return nil
		},
	}
	cmd.Flags().BoolVar(&enable, "enable", false, "Enable deletion protection")
	cmd.Flags().BoolVar(&disable, "disable", false, "Disable deletion protection")
	cmd.Flags().String("workspace-id", "", "Workspace ID (or pass it positionally)")
	cmd.MarkFlagsMutuallyExclusive("enable", "disable")
	return cmd
}

func newWorkspacesComputeSettingsCmd() *cobra.Command {
	var (
		minCU          float64
		maxCU          float64
		suspendTimeout int
		serviceType    string
	)
	cmd := &cobra.Command{
		Use:   "compute-settings [workspace-id]",
		Short: "Modify workspace autoscaling and suspend settings",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			g := fromCtx(cmd)
			changedMin := cmd.Flags().Changed("min-cu")
			changedMax := cmd.Flags().Changed("max-cu")
			changedTimeout := cmd.Flags().Changed("suspend-timeout")
			if !changedMin && !changedMax && !changedTimeout {
				return fmt.Errorf("nothing to change; pass --min-cu/--max-cu and/or --suspend-timeout")
			}
			if changedTimeout {
				if err := validateSuspendTimeout(suspendTimeout); err != nil {
					return err
				}
			}
			workspaceID, err := workspaceIDFromArgs(cmd, args)
			if err != nil {
				return err
			}
			client, err := g.NewVolcClient(cmd.Context())
			if err != nil {
				return err
			}
			// ModifyComputeSettings always carries both CU bounds, so an
			// unspecified one has to be read back rather than sent as 0 —
			// otherwise changing only the suspend timeout would silently
			// collapse the workspace's autoscaling range.
			if !changedMin || !changedMax {
				current, detailErr := client.DescribeWorkspaceDetail(cmd.Context(), workspaceID)
				if detailErr != nil {
					return detailErr
				}
				if !changedMin {
					minCU = current.ComputeSettings.AutoScalingLimitMinCU
				}
				if !changedMax {
					maxCU = current.ComputeSettings.AutoScalingLimitMaxCU
				}
			}
			if err := validateComputeUnits(minCU, maxCU); err != nil {
				return err
			}
			params := volcengine.ModifyComputeSettingsParams{
				AutoScalingLimitMinCU: minCU,
				AutoScalingLimitMaxCU: maxCU,
				ServiceType:           serviceType,
			}
			if changedTimeout {
				params.SuspendTimeoutSeconds = &suspendTimeout
			}
			ws, err := client.ModifyComputeSettings(cmd.Context(), workspaceID, params)
			if err != nil {
				return err
			}
			return g.Writer().WriteItem(ws, workspaceFields)
		},
	}
	cmd.Flags().Float64Var(&minCU, "min-cu", 0, "Minimum compute units (0.25-32, unchanged when omitted)")
	cmd.Flags().Float64Var(&maxCU, "max-cu", 0, "Maximum compute units (0.25-32, at most 8x --min-cu, unchanged when omitted)")
	cmd.Flags().IntVar(&suspendTimeout, "suspend-timeout", 0, "Auto-suspend timeout in seconds (-1 never suspends)")
	cmd.Flags().StringVar(&serviceType, "service-type", "", "Compute service type")
	return cmd
}

func newWorkspacesSettingsCmd() *cobra.Command {
	var retentionHours int
	cmd := &cobra.Command{
		Use:   "settings [workspace-id]",
		Short: "Modify workspace-level settings",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			g := fromCtx(cmd)
			// The only setting here is required by the request model, so an
			// omitted flag would post a retention of 0 rather than no-op.
			if !cmd.Flags().Changed("history-retention-hours") {
				return fmt.Errorf("nothing to change; pass --history-retention-hours")
			}
			if retentionHours < minRetentionHours || retentionHours > maxRetentionHours {
				return fmt.Errorf("--history-retention-hours must be between %d and %d, got %d",
					minRetentionHours, maxRetentionHours, retentionHours)
			}
			workspaceID, err := workspaceIDFromArgs(cmd, args)
			if err != nil {
				return err
			}
			client, err := g.NewVolcClient(cmd.Context())
			if err != nil {
				return err
			}
			ws, err := client.ModifyWorkspaceSettings(cmd.Context(), workspaceID, volcengine.ModifyWorkspaceSettingsParams{
				HistoryRetentionHours: retentionHours,
			})
			if err != nil {
				return err
			}
			return g.Writer().WriteItem(ws, workspaceFields)
		},
	}
	cmd.Flags().IntVar(&retentionHours, "history-retention-hours", 0, "Point-in-time history retention in hours (1-720)")
	return cmd
}

func newWorkspacesOverviewCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "overview",
		Short: "Show PostgreSQL workspace status counts",
		RunE: func(cmd *cobra.Command, args []string) error {
			g := fromCtx(cmd)
			client, err := g.NewVolcClient(cmd.Context())
			if err != nil {
				return err
			}
			result, err := client.DescribeWorkspaceOverview(cmd.Context(), volcengine.WorkspaceOverviewParams{})
			if err != nil {
				return err
			}
			return g.Writer().WriteList(result.Overviews, []string{
				"EngineType", "WorkspaceTotal", "RunningTotal", "CreatingTotal",
				"UpdatingTotal", "StoppedTotal", "SuspendedTotal", "ClosedTotal", "ErrorTotal",
			})
		},
	}
	return cmd
}
