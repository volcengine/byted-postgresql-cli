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
	"strings"

	"github.com/spf13/cobra"

	"github.com/volcengine/byted-postgresql-cli/internal/volcengine"
)

var computeFields = []string{
	"ComputeName", "ComputeId", "BranchId", "ComputeStatus", "ComputeRole",
	"ServiceType", "AutoScalingLimitMinCU", "AutoScalingLimitMaxCU",
	"EnableAnalytics", "CreateTime", "UpdateTime",
}

func newComputesCmd(ctx ProviderContext) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "computes",
		Aliases: []string{"compute"},
		Short:   "Manage PostgreSQL computes",
	}
	var workspaceID, branchID, serviceType string

	resolve := func(cmd *cobra.Command) (*volcengine.Client, string, string, error) {
		g := fromCtx(cmd)
		client, err := g.NewVolcClient(cmd.Context())
		if err != nil {
			return nil, "", "", err
		}
		workspace, err := resolveWorkspace(cmd.Context(), g, client, workspaceID)
		if err != nil {
			return nil, "", "", err
		}
		branch, err := resolveBranch(cmd.Context(), g, client, workspace, branchID)
		return client, workspace, branch, err
	}

	list := &cobra.Command{
		Use:   "list",
		Short: "List computes",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			g := fromCtx(cmd)
			client, wsID, bid, err := resolve(cmd)
			if err != nil {
				return err
			}
			result, err := client.DescribeComputes(cmd.Context(), wsID, bid, serviceType)
			if err != nil {
				return err
			}
			total := result.Total
			return writeListSummary(cmd, g, result.Computes, computeFields, &total, "computes")
		},
	}
	cmd.AddCommand(list)

	get := &cobra.Command{
		Use:   "get --compute-id <compute-id>",
		Short: "Get a compute",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			g := fromCtx(cmd)
			computeID, err := cmd.Flags().GetString("compute-id")
			if err != nil {
				return err
			}
			if strings.TrimSpace(computeID) == "" {
				return fmt.Errorf("--compute-id is required")
			}
			client, err := g.NewVolcClient(cmd.Context())
			if err != nil {
				return err
			}
			workspace, err := resolveWorkspace(cmd.Context(), g, client, workspaceID)
			if err != nil {
				return err
			}
			compute, err := client.DescribeComputeDetail(cmd.Context(), workspace, computeID)
			if err != nil {
				return err
			}
			return g.Writer().WriteItem(compute, computeFields)
		},
	}
	get.Flags().String("compute-id", "", "Compute ID (required)")
	cmd.AddCommand(get)

	var name, computeType string
	var minCU, maxCU float64
	create := &cobra.Command{
		Use:   "create",
		Short: "Create a read-only or DuckDB compute",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			g := fromCtx(cmd)
			computeType = strings.TrimSpace(computeType)
			if err := validateComputeType(computeType); err != nil {
				return err
			}
			if err := validateComputeCreateUnits(cmd, minCU, maxCU); err != nil {
				return err
			}
			_, wsID, bid, err := resolve(cmd)
			if err != nil {
				return err
			}
			client, err := g.NewVolcClient(cmd.Context())
			if err != nil {
				return err
			}
			compute, err := client.CreateCompute(cmd.Context(), volcengine.CreateComputeParams{
				WorkspaceID: wsID, BranchID: bid, ComputeName: name, ComputeRole: computeType,
				AutoScalingLimitMinCU: minCU, AutoScalingLimitMaxCU: maxCU,
			})
			if err != nil {
				return err
			}
			return g.Writer().WriteItem(compute, computeFields)
		},
	}
	create.Flags().StringVar(&name, "name", "", "Compute name")
	create.Flags().StringVar(&computeType, "type", "", "Compute type: ReadOnly or Analytic (DuckDB)")
	create.Flags().Float64Var(&minCU, "min-cu", 0, "Minimum compute units (0.25-2)")
	create.Flags().Float64Var(&maxCU, "max-cu", 0, "Maximum compute units (0.25-2)")
	_ = create.MarkFlagRequired("name")
	_ = create.MarkFlagRequired("type")
	cmd.AddCommand(create)

	var yes bool
	del := &cobra.Command{
		Use:     "delete --compute-id <compute-id>",
		Aliases: []string{"rm"},
		Short:   "Delete a compute",
		Args:    cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			g := fromCtx(cmd)
			computeID, err := cmd.Flags().GetString("compute-id")
			if err != nil {
				return err
			}
			if strings.TrimSpace(computeID) == "" {
				return fmt.Errorf("--compute-id is required")
			}
			client, err := g.NewVolcClient(cmd.Context())
			if err != nil {
				return err
			}
			workspace, err := resolveWorkspace(cmd.Context(), g, client, workspaceID)
			if err != nil {
				return err
			}
			if !yes {
				summary := fmt.Sprintf("Delete compute %q? This operation cannot be undone.", computeID)
				if err := confirmDestructiveAction(cmd, computeID, summary, "delete"); err != nil {
					return err
				}
			}
			if err := client.DeleteCompute(cmd.Context(), workspace, computeID); err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Compute %s deleted\n", computeID)
			return nil
		},
	}
	del.Flags().String("compute-id", "", "Compute ID (required)")
	del.Flags().BoolVarP(&yes, "yes", "y", false, "Skip the confirmation prompt")
	cmd.AddCommand(del)

	restart := &cobra.Command{
		Use:   "restart --compute-id <compute-id>",
		Short: "Restart the branch containing a compute",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			g := fromCtx(cmd)
			computeID, err := cmd.Flags().GetString("compute-id")
			if err != nil {
				return err
			}
			if strings.TrimSpace(computeID) == "" {
				return fmt.Errorf("--compute-id is required")
			}
			client, wsID, bid, err := resolve(cmd)
			if err != nil {
				return err
			}
			if _, err := client.RestartBranch(cmd.Context(), volcengine.RestartBranchParams{
				WorkspaceID: wsID, BranchID: bid, ComputeIDs: []string{computeID},
			}); err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Compute %s restarting\n", computeID)
			_ = g
			return nil
		},
	}
	restart.Flags().String("compute-id", "", "Compute ID (required)")
	cmd.AddCommand(restart)

	enableAP := &cobra.Command{
		Use:   "enable-ap --compute-id <compute-id>",
		Short: "Enable AP analytics acceleration for a compute",
		Long:  "This changes the compute's analytics policy; it does not create a separate compute.",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			g := fromCtx(cmd)
			computeID, err := cmd.Flags().GetString("compute-id")
			if err != nil {
				return err
			}
			if strings.TrimSpace(computeID) == "" {
				return fmt.Errorf("--compute-id is required")
			}
			client, err := g.NewVolcClient(cmd.Context())
			if err != nil {
				return err
			}
			workspace, err := resolveWorkspace(cmd.Context(), g, client, workspaceID)
			if err != nil {
				return err
			}
			return client.ModifyComputeAnalyticPolicy(cmd.Context(), workspace, computeID)
		},
	}
	enableAP.Flags().String("compute-id", "", "Compute ID (required)")
	cmd.AddCommand(enableAP)

	var updateName string
	var updateMin, updateMax float64
	update := &cobra.Command{
		Use: "update --compute-id <compute-id>", Short: "Update compute name or scaling limits",
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			computeID, err := cmd.Flags().GetString("compute-id")
			if err != nil {
				return err
			}
			if strings.TrimSpace(computeID) == "" {
				return fmt.Errorf("--compute-id is required")
			}
			if !cmd.Flags().Changed("name") && !cmd.Flags().Changed("min-cu") && !cmd.Flags().Changed("max-cu") {
				return fmt.Errorf("nothing to update; pass --name, --min-cu, or --max-cu")
			}
			g := fromCtx(cmd)
			client, err := g.NewVolcClient(cmd.Context())
			if err != nil {
				return err
			}
			workspace, err := resolveWorkspace(cmd.Context(), g, client, workspaceID)
			if err != nil {
				return err
			}
			if cmd.Flags().Changed("min-cu") || cmd.Flags().Changed("max-cu") {
				current, err := client.DescribeComputeDetail(cmd.Context(), workspace, computeID)
				if err != nil {
					return err
				}
				if !cmd.Flags().Changed("min-cu") {
					updateMin = current.AutoScalingLimitMinCU
				}
				if !cmd.Flags().Changed("max-cu") {
					updateMax = current.AutoScalingLimitMaxCU
				}
				if _, err := client.ModifyComputeSpec(cmd.Context(), volcengine.ModifyComputeSpecParams{
					WorkspaceID: workspace, ComputeID: computeID, AutoScalingLimitMinCU: updateMin, AutoScalingLimitMaxCU: updateMax,
				}); err != nil {
					return err
				}
			}
			if cmd.Flags().Changed("name") {
				if err := client.ModifyComputeName(cmd.Context(), volcengine.ModifyComputeNameParams{
					WorkspaceID: workspace, ComputeID: computeID, ComputeName: updateName,
				}); err != nil {
					return err
				}
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Compute %s updated\n", computeID)
			return nil
		},
	}
	update.Flags().StringVar(&updateName, "name", "", "New compute name")
	update.Flags().Float64Var(&updateMin, "min-cu", 0, "New minimum compute units")
	update.Flags().Float64Var(&updateMax, "max-cu", 0, "New maximum compute units")
	update.Flags().String("compute-id", "", "Compute ID (required)")
	cmd.AddCommand(update)
	for _, child := range cmd.Commands() {
		child.Flags().StringVar(&workspaceID, "workspace-id", "", "Workspace ID (required in non-interactive mode)")
		child.Flags().StringVar(&branchID, "branch-id", "", "Branch ID (defaults to the workspace's default branch)")
		child.Flags().StringVar(&serviceType, "service-type", "", "Compute service type")
	}
	return cmd
}

func validateComputeType(computeType string) error {
	switch strings.TrimSpace(computeType) {
	case volcengine.ComputeRoleReadOnly, volcengine.ComputeRoleAnalytic:
		return nil
	default:
		return fmt.Errorf("--type must be %s (read-only) or %s (DuckDB), got %q",
			volcengine.ComputeRoleReadOnly, volcengine.ComputeRoleAnalytic, computeType)
	}
}

func validateComputeCreateUnits(cmd *cobra.Command, minCU, maxCU float64) error {
	if !cmd.Flags().Changed("min-cu") || !cmd.Flags().Changed("max-cu") {
		return fmt.Errorf("--min-cu and --max-cu are required")
	}
	if minCU < 0.25 || minCU > 2 {
		return fmt.Errorf("--min-cu must be between 0.25 and 2 compute units, got %g", minCU)
	}
	if maxCU < 0.25 || maxCU > 2 {
		return fmt.Errorf("--max-cu must be between 0.25 and 2 compute units, got %g", maxCU)
	}
	if maxCU < minCU {
		return fmt.Errorf("--max-cu (%g) must be greater than or equal to --min-cu (%g)", maxCU, minCU)
	}
	return nil
}
