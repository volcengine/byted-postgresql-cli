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
)

func newWorkspaceNetworkCmd(ctx ProviderContext) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "network",
		Short: "Manage workspace network settings",
	}
	var workspaceID, branchID string
	cmd.PersistentFlags().StringVar(&workspaceID, "workspace-id", "", "Workspace ID")
	cmd.PersistentFlags().StringVar(&branchID, "branch-id", "", "Branch ID (required for branch-scoped workspaces)")
	get := &cobra.Command{
		Use:   "get",
		Short: "Get workspace network settings",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if workspaceID == "" {
				return fmt.Errorf("--workspace-id is required")
			}
			g := fromCtx(cmd)
			client, err := g.NewVolcClient(cmd.Context())
			if err != nil {
				return err
			}
			workspace, err := client.DescribeWorkspaceDetail(cmd.Context(), workspaceID)
			if err != nil {
				return err
			}
			return g.Writer().WriteItem(networkSettings{
				WorkspaceID:      workspace.WorkspaceID,
				BranchID:         branchID,
				VPCID:            workspace.VpcID,
				SubnetID:         workspace.SubnetID,
				InternetProtocol: workspace.InternetProtocol,
			}, nil)
		},
	}
	cmd.AddCommand(get)
	var vpcID, subnetID, internetProtocol string
	update := &cobra.Command{
		Use:   "update",
		Short: "Update workspace network settings",
		RunE: func(cmd *cobra.Command, args []string) error {
			if vpcID == "" || subnetID == "" || internetProtocol == "" {
				return fmt.Errorf("--vpc-id, --subnet-id, and --internet-protocol are required")
			}
			client, err := fromCtx(cmd).NewVolcClient(cmd.Context())
			if err != nil {
				return err
			}
			if err := client.ModifyWorkspaceVpcSettings(cmd.Context(), workspaceID, branchID, vpcID, subnetID, internetProtocol); err != nil {
				return err
			}
			fmt.Fprintln(cmd.OutOrStdout(), "Workspace network settings updated")
			return nil
		},
	}
	update.Flags().StringVar(&vpcID, "vpc-id", "", "VPC ID (required)")
	update.Flags().StringVar(&subnetID, "subnet-id", "", "Subnet ID (required)")
	update.Flags().StringVar(&internetProtocol, "internet-protocol", "", "Internet protocol (IPv4 or DualStack, required)")
	cmd.AddCommand(update)
	return cmd
}

type networkSettings struct {
	WorkspaceID      string `json:"workspace_id" yaml:"workspace_id"`
	BranchID         string `json:"branch_id,omitempty" yaml:"branch_id,omitempty"`
	VPCID            string `json:"vpc_id" yaml:"vpc_id"`
	SubnetID         string `json:"subnet_id" yaml:"subnet_id"`
	InternetProtocol string `json:"internet_protocol" yaml:"internet_protocol"`
}
