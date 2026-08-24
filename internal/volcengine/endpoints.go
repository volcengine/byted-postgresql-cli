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

package volcengine

import (
	"context"
	"fmt"

	"github.com/volcengine/volcengine-go-sdk/service/aidap"
)

type DescribeWorkspaceEndpointsParams struct {
	WorkspaceID string
	BranchID    string
	ComputeID   string
}

type DescribeWorkspaceEndpointsResult struct {
	WorkspaceID string     `json:"WorkspaceId"`
	BranchID    string     `json:"BranchId"`
	Endpoints   []Endpoint `json:"Endpoints"`
}

type Endpoint struct {
	EndpointID   string            `json:"EndpointId"`
	EndpointName string            `json:"EndpointName"`
	EndpointType string            `json:"EndpointType"`
	Addresses    []EndpointAddress `json:"Addresses"`
}

type EndpointAddress struct {
	AddressID     string `json:"AddressId"`
	AddressType   string `json:"AddressType"`
	AddressDomain string `json:"AddressDomain"`
	AddressPort   int    `json:"AddressPort"`
	IPAddress     string `json:"IPAddress"`
	IPv6Address   string `json:"IPv6Address"`
}

func (c *Client) DescribeWorkspaceEndpoints(ctx context.Context, params DescribeWorkspaceEndpointsParams) (DescribeWorkspaceEndpointsResult, error) {
	req := (&aidap.DescribeWorkspaceEndpointInput{}).SetWorkspaceId(params.WorkspaceID)
	if params.BranchID != "" {
		req.SetBranchId(params.BranchID)
	}
	if params.ComputeID != "" {
		req.SetComputeId(params.ComputeID)
	}
	resp, err := c.aidap.DescribeWorkspaceEndpointWithContext(ctx, req)
	if err != nil {
		return DescribeWorkspaceEndpointsResult{}, fmt.Errorf("failed to describe workspace endpoint: %w", err)
	}
	return mapDescribeWorkspaceEndpointsResult(resp), nil
}

func mapDescribeWorkspaceEndpointsResult(resp *aidap.DescribeWorkspaceEndpointOutput) DescribeWorkspaceEndpointsResult {
	if resp == nil {
		return DescribeWorkspaceEndpointsResult{}
	}
	result := DescribeWorkspaceEndpointsResult{
		WorkspaceID: stringValue(resp.WorkspaceId),
		BranchID:    stringValue(resp.BranchId),
	}
	for _, endpoint := range resp.Endpoints {
		if endpoint == nil {
			continue
		}
		mapped := Endpoint{
			EndpointID:   stringValue(endpoint.EndpointId),
			EndpointName: stringValue(endpoint.EndpointName),
			EndpointType: stringValue(endpoint.EndpointType),
		}
		for _, address := range endpoint.Addresses {
			if address == nil {
				continue
			}
			mapped.Addresses = append(mapped.Addresses, EndpointAddress{
				AddressID:     stringValue(address.AddressId),
				AddressType:   stringValue(address.AddressType),
				AddressDomain: stringValue(address.AddressDomain),
				AddressPort:   intValue(address.AddressPort),
				IPAddress:     stringValue(address.IPAddress),
				IPv6Address:   stringValue(address.IPv6Address),
			})
		}
		result.Endpoints = append(result.Endpoints, mapped)
	}
	return result
}
