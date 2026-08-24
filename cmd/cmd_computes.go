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
	cmd.PersistentFlags().StringVar(&workspaceID, "workspace-id", "", "Workspace ID")
	cmd.PersistentFlags().StringVar(&branchID, "branch-id", "", "Branch ID (defaults to the workspace's default branch)")
	cmd.PersistentFlags().StringVar(&serviceType, "service-type", "", "Compute service type")

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
		Use:   "get <compute-id>",
		Short: "Get a compute",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			g := fromCtx(cmd)
			client, err := g.NewVolcClient(cmd.Context())
			if err != nil {
				return err
			}
			workspace, err := resolveWorkspace(cmd.Context(), g, client, workspaceID)
			if err != nil {
				return err
			}
			compute, err := client.DescribeComputeDetail(cmd.Context(), workspace, args[0])
			if err != nil {
				return err
			}
			return g.Writer().WriteItem(compute, computeFields)
		},
	}
	cmd.AddCommand(get)

	var name, role string
	var minCU, maxCU float64
	create := &cobra.Command{
		Use:   "create",
		Short: "Create a compute",
		RunE: func(cmd *cobra.Command, args []string) error {
			g := fromCtx(cmd)
			_, wsID, bid, err := resolve(cmd)
			if err != nil {
				return err
			}
			client, err := g.NewVolcClient(cmd.Context())
			if err != nil {
				return err
			}
			compute, err := client.CreateCompute(cmd.Context(), volcengine.CreateComputeParams{
				WorkspaceID: wsID, BranchID: bid, ComputeName: name, ComputeRole: role,
				AutoScalingLimitMinCU: minCU, AutoScalingLimitMaxCU: maxCU,
			})
			if err != nil {
				return err
			}
			return g.Writer().WriteItem(compute, computeFields)
		},
	}
	create.Flags().StringVar(&name, "name", "", "Compute name")
	create.Flags().StringVar(&role, "role", "Primary", "Compute role: Primary or ReadOnly")
	create.Flags().Float64Var(&minCU, "min-cu", 0, "Minimum compute units (0.25-32)")
	create.Flags().Float64Var(&maxCU, "max-cu", 0, "Maximum compute units (0.25-32, at most 8x --min-cu)")
	_ = create.MarkFlagRequired("name")
	cmd.AddCommand(create)

	var yes bool
	del := &cobra.Command{
		Use:     "delete <compute-id>",
		Aliases: []string{"rm"},
		Short:   "Delete a compute",
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			g := fromCtx(cmd)
			client, err := g.NewVolcClient(cmd.Context())
			if err != nil {
				return err
			}
			workspace, err := resolveWorkspace(cmd.Context(), g, client, workspaceID)
			if err != nil {
				return err
			}
			if !yes {
				summary := fmt.Sprintf("Delete compute %q? This operation cannot be undone.", args[0])
				if err := confirmDestructiveAction(cmd, args[0], summary, "delete"); err != nil {
					return err
				}
			}
			if err := client.DeleteCompute(cmd.Context(), workspace, args[0]); err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Compute %s deleted\n", args[0])
			return nil
		},
	}
	del.Flags().BoolVarP(&yes, "yes", "y", false, "Skip the confirmation prompt")
	cmd.AddCommand(del)

	restart := &cobra.Command{
		Use:   "restart <compute-id>",
		Short: "Restart the branch containing a compute",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			g := fromCtx(cmd)
			client, wsID, bid, err := resolve(cmd)
			if err != nil {
				return err
			}
			if _, err := client.RestartBranch(cmd.Context(), volcengine.RestartBranchParams{
				WorkspaceID: wsID, BranchID: bid, ComputeIDs: []string{args[0]},
			}); err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Compute %s restarting\n", args[0])
			_ = g
			return nil
		},
	}
	cmd.AddCommand(restart)

	enableAP := &cobra.Command{
		Use:   "enable-ap <compute-id>",
		Short: "Enable AP analytics acceleration for a compute",
		Long:  "Enable AP (analytics acceleration) for a compute. This changes the compute's analytics policy; it does not create a separate compute.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			g := fromCtx(cmd)
			client, err := g.NewVolcClient(cmd.Context())
			if err != nil {
				return err
			}
			workspace, err := resolveWorkspace(cmd.Context(), g, client, workspaceID)
			if err != nil {
				return err
			}
			return client.ModifyComputeAnalyticPolicy(cmd.Context(), workspace, args[0])
		},
	}
	cmd.AddCommand(enableAP)

	var updateName string
	var updateMin, updateMax float64
	update := &cobra.Command{
		Use: "update <compute-id>", Short: "Update compute name or scaling limits",
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
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
				current, err := client.DescribeComputeDetail(cmd.Context(), workspace, args[0])
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
					WorkspaceID: workspace, ComputeID: args[0], AutoScalingLimitMinCU: updateMin, AutoScalingLimitMaxCU: updateMax,
				}); err != nil {
					return err
				}
			}
			if cmd.Flags().Changed("name") {
				if err := client.ModifyComputeName(cmd.Context(), volcengine.ModifyComputeNameParams{
					WorkspaceID: workspace, ComputeID: args[0], ComputeName: updateName,
				}); err != nil {
					return err
				}
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Compute %s updated\n", args[0])
			return nil
		},
	}
	update.Flags().StringVar(&updateName, "name", "", "New compute name")
	update.Flags().Float64Var(&updateMin, "min-cu", 0, "New minimum compute units")
	update.Flags().Float64Var(&updateMax, "max-cu", 0, "New maximum compute units")
	cmd.AddCommand(update)
	return cmd
}
