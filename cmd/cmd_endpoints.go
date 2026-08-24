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

var endpointFields = []string{"EndpointId", "EndpointName", "EndpointType", "Addresses"}

func newEndpointsBaseCmd(ctx ProviderContext) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "endpoints",
		Aliases: []string{"endpoint"},
		Short:   "Manage PostgreSQL workspace endpoints on " + ctx.DisplayName,
	}
	var workspaceID, branchID string
	cmd.PersistentFlags().StringVar(&workspaceID, "workspace-id", "", "Workspace ID")
	cmd.PersistentFlags().StringVar(&branchID, "branch-id", "", "Branch ID (defaults to the workspace's default branch)")
	return cmd
}

func newEndpointsListCmd(ctx ProviderContext) *cobra.Command {
	resolve := func(cmd *cobra.Command) (*volcengine.Client, string, string, error) {
		g := fromCtx(cmd)
		client, err := g.NewVolcClient(cmd.Context())
		if err != nil {
			return nil, "", "", err
		}
		workspaceID, err := cmd.Flags().GetString("workspace-id")
		if err != nil {
			return nil, "", "", err
		}
		branchID, err := cmd.Flags().GetString("branch-id")
		if err != nil {
			return nil, "", "", err
		}
		workspace, err := resolveWorkspace(cmd.Context(), g, client, workspaceID)
		if err != nil {
			return nil, "", "", err
		}
		branch, err := resolveBranch(cmd.Context(), g, client, workspace, branchID)
		if err != nil {
			return nil, "", "", err
		}
		return client, workspace, branch, nil
	}

	list := &cobra.Command{
		Use:   "list",
		Short: "List workspace endpoints",
		RunE: func(cmd *cobra.Command, args []string) error {
			g := fromCtx(cmd)
			client, wsID, bid, err := resolve(cmd)
			if err != nil {
				return err
			}
			resolvedBranch, computeID, err := client.ResolvePrimaryDatabaseComputeID(cmd.Context(), wsID, bid)
			if err != nil {
				return err
			}
			result, err := client.DescribeWorkspaceEndpoints(cmd.Context(), volcengine.DescribeWorkspaceEndpointsParams{
				WorkspaceID: wsID,
				BranchID:    resolvedBranch,
				ComputeID:   computeID,
			})
			if err != nil {
				return err
			}
			if g.Writer().Format == "table" {
				fmt.Fprintf(cmd.OutOrStdout(), "Showing %d endpoints.\n", len(result.Endpoints))
				return g.Writer().WriteList(endpointTableRows(result.Endpoints), endpointFields)
			}
			return writeListSummary(cmd, g, result.Endpoints, endpointFields, nil, "endpoints")
		},
	}
	return list
}

type endpointTableRow struct {
	EndpointID   string
	EndpointName string
	EndpointType string
	Addresses    string
}

func endpointTableRows(endpoints []volcengine.Endpoint) []endpointTableRow {
	rows := make([]endpointTableRow, 0, len(endpoints))
	for _, endpoint := range endpoints {
		addresses := make([]string, 0, len(endpoint.Addresses))
		for _, address := range endpoint.Addresses {
			host := address.AddressDomain
			if host == "" {
				host = address.IPAddress
			}
			port := ""
			if address.AddressPort > 0 {
				port = fmt.Sprintf(":%d", address.AddressPort)
			}
			ip := address.IPAddress
			if ip == "" {
				ip = address.IPv6Address
			}
			detail := fmt.Sprintf("%s %s%s", address.AddressType, host, port)
			if ip != "" && ip != host {
				detail += " (" + ip + ")"
			}
			addresses = append(addresses, strings.TrimSpace(detail))
		}
		rows = append(rows, endpointTableRow{
			EndpointID:   endpoint.EndpointID,
			EndpointName: endpoint.EndpointName,
			EndpointType: endpoint.EndpointType,
			Addresses:    strings.Join(addresses, "\n"),
		})
	}
	return rows
}

func newVolcengineEndpointsCmd(ctx ProviderContext) *cobra.Command {
	cmd := newEndpointsBaseCmd(ctx)
	cmd.AddCommand(newEndpointsListCmd(ctx))
	return cmd
}

func newEndpointsCmd() *cobra.Command {
	return newVolcengineEndpointsCmd(providerContext(volcengine.DefaultRegion))
}
