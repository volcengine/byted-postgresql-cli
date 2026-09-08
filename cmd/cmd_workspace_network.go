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
	"strconv"

	"github.com/spf13/cobra"

	"github.com/volcengine/byted-postgresql-cli/internal/volcengine"
)

func newWorkspaceNetworkCmd(ctx ProviderContext) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "network",
		Short: "Manage workspace network settings",
	}
	var vpcName, subnetName string
	var vpcOwnerID int64
	var vpcID, subnetID, internetProtocol string
	var pageSize, pageNumber int64
	var workspaceID, branchID string
	vpcs := &cobra.Command{Use: "vpcs", Short: "Manage VPCs"}
	vpcsList := &cobra.Command{
		Use: "list", Short: "List VPCs", Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			g := fromCtx(cmd)
			client, err := g.NewVolcClient(cmd.Context())
			if err != nil {
				return err
			}
			if vpcOwnerID <= 0 {
				accountID, identityErr := client.CallerAccountID(cmd.Context())
				if identityErr == nil {
					parsed, parseErr := strconv.ParseInt(accountID, 10, 64)
					if parseErr == nil && parsed > 0 {
						vpcOwnerID = parsed
					}
				}
			}
			if vpcOwnerID <= 0 && workspaceID != "" {
				workspace, err := client.DescribeWorkspaceDetail(cmd.Context(), workspaceID)
				if err != nil {
					return err
				}
				if workspace.AccountID != "" {
					parsed, parseErr := strconv.ParseInt(workspace.AccountID, 10, 64)
					if parseErr == nil && parsed > 0 {
						vpcOwnerID = parsed
					}
				}
			}
			if vpcOwnerID <= 0 {
				return fmt.Errorf("unable to resolve VPC owner account ID from STS identity or workspace; pass --vpc-owner-id explicitly")
			}
			result, err := client.ListVPCs(cmd.Context(), volcengine.ListVPCsParams{
				VPCName: vpcName, VPCOwnerID: vpcOwnerID, PageSize: pageSize, PageNumber: pageNumber,
			})
			if err != nil {
				return err
			}
			return g.Writer().WriteList(result.VPCs, []string{"ID", "Name", "AccountID", "ProjectName", "CIDRBlock", "Status", "IsDefault", "SubnetIDs"})
		},
	}
	vpcsList.Flags().StringVar(&vpcName, "vpc-name", "", "Filter by VPC name")
	vpcsList.Flags().Int64Var(&vpcOwnerID, "vpc-owner-id", 0, "VPC owner account ID")
	vpcsList.Flags().Int64Var(&pageSize, "page-size", 100, "Page size (1-100)")
	vpcsList.Flags().Int64Var(&pageNumber, "page-number", 1, "Page number")
	vpcsList.Flags().StringVar(&workspaceID, "workspace-id", "", "Workspace ID (used to resolve the VPC owner)")
	vpcs.AddCommand(vpcsList)
	cmd.AddCommand(vpcs)
	subnets := &cobra.Command{Use: "subnets", Short: "Manage VPC subnets"}
	subnetsList := &cobra.Command{
		Use: "list", Short: "List subnets in a VPC", Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if vpcID == "" {
				return fmt.Errorf("--vpc-id is required")
			}
			g := fromCtx(cmd)
			client, err := g.NewVolcClient(cmd.Context())
			if err != nil {
				return err
			}
			result, err := client.ListSubnets(cmd.Context(), volcengine.ListSubnetsParams{
				VPCID: vpcID, SubnetName: subnetName, PageSize: pageSize, PageNumber: pageNumber,
			})
			if err != nil {
				return err
			}
			return g.Writer().WriteList(result.Subnets, []string{"ID", "Name", "VPCID", "AccountID", "ProjectName", "ZoneID", "CIDRBlock", "Status", "AvailableIPAddressCount", "IsDefault"})
		},
	}
	subnetsList.Flags().StringVar(&vpcID, "vpc-id", "", "VPC ID (required)")
	subnetsList.Flags().StringVar(&subnetName, "subnet-name", "", "Filter by subnet name")
	subnetsList.Flags().Int64Var(&pageSize, "page-size", 100, "Page size (1-100)")
	subnetsList.Flags().Int64Var(&pageNumber, "page-number", 1, "Page number")
	subnetsList.Flags().StringVar(&workspaceID, "workspace-id", "", "Workspace ID (used to resolve the VPC owner)")
	subnets.AddCommand(subnetsList)
	cmd.AddCommand(subnets)
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
	get.Flags().StringVar(&workspaceID, "workspace-id", "", "Workspace ID (required)")
	get.Flags().StringVar(&branchID, "branch-id", "", "Branch ID (required for branch-scoped workspaces)")
	cmd.AddCommand(get)
	update := &cobra.Command{
		Use:   "update",
		Short: "Update workspace network settings",
		Args:  cobra.NoArgs,
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
	update.Flags().StringVar(&workspaceID, "workspace-id", "", "Workspace ID (required)")
	update.Flags().StringVar(&branchID, "branch-id", "", "Branch ID (required for branch-scoped workspaces)")
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
