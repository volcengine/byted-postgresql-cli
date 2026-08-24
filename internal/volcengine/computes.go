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
	"strings"

	"github.com/volcengine/volcengine-go-sdk/service/aidap"
)

// Compute is a branch's compute node. PostgreSQL workspaces run a single
// Database service type (Supabase's second "Baas" compute has no PG analogue).
type Compute struct {
	WorkspaceID           string  `json:"WorkspaceId"`
	BranchID              string  `json:"BranchId"`
	ComputeID             string  `json:"ComputeId"`
	ComputeName           string  `json:"ComputeName"`
	ComputeStatus         string  `json:"ComputeStatus"`
	ComputeRole           string  `json:"ComputeRole"`
	ServiceType           string  `json:"ServiceType"`
	AutoScalingLimitMinCU float64 `json:"AutoScalingLimitMinCU"`
	AutoScalingLimitMaxCU float64 `json:"AutoScalingLimitMaxCU"`
	EnableAnalytics       string  `json:"EnableAnalytics"`
	CreationSource        string  `json:"CreationSource"`
	Disabled              bool    `json:"Disabled"`
	CreateTime            string  `json:"CreateTime"`
	UpdateTime            string  `json:"UpdateTime"`
	LastActiveTime        string  `json:"LastActiveTime"`
	StatusChangedTime     string  `json:"StatusChangedTime"`
	SuspendedTime         string  `json:"SuspendedTime"`
}

type DescribeComputesResult struct {
	Total    int
	Computes []Compute
}

type ModifyComputeSpecParams struct {
	WorkspaceID           string
	ComputeID             string
	AutoScalingLimitMinCU float64
	AutoScalingLimitMaxCU float64
}

type ModifyComputeNameParams struct {
	WorkspaceID string
	ComputeID   string
	ComputeName string
}

type CreateComputeParams struct {
	WorkspaceID           string
	BranchID              string
	ComputeName           string
	ComputeRole           string
	AutoScalingLimitMinCU float64
	AutoScalingLimitMaxCU float64
}

// DescribeComputes lists a branch's computes. When serviceType is empty it
// defaults to the Database service type PostgreSQL workspaces use.
func (c *Client) DescribeComputes(ctx context.Context, workspaceID, branchID, serviceType string) (DescribeComputesResult, error) {
	if strings.TrimSpace(serviceType) == "" {
		serviceType = ServiceTypeDatabase
	}
	req := (&aidap.DescribeComputesInput{}).
		SetWorkspaceId(workspaceID).
		SetBranchId(branchID).
		SetServiceType(serviceType)
	resp, err := c.aidap.DescribeComputesWithContext(ctx, req)
	if err != nil {
		return DescribeComputesResult{}, fmt.Errorf("failed to describe computes: %w", err)
	}
	result := DescribeComputesResult{Total: intValue(resp.Total)}
	for _, item := range resp.Computes {
		if item == nil {
			continue
		}
		result.Computes = append(result.Computes, Compute{
			WorkspaceID:           stringValue(item.WorkspaceId),
			BranchID:              stringValue(item.BranchId),
			ComputeID:             stringValue(item.ComputeId),
			ComputeName:           stringValue(item.ComputeName),
			ComputeStatus:         stringValue(item.ComputeStatus),
			ComputeRole:           stringValue(item.ComputeRole),
			ServiceType:           stringValue(item.ServiceType),
			AutoScalingLimitMinCU: floatValue(item.AutoScalingLimitMinCU),
			AutoScalingLimitMaxCU: floatValue(item.AutoScalingLimitMaxCU),
			EnableAnalytics:       stringValue(item.EnableAnalytic),
			CreationSource:        stringValue(item.CreationSource),
			Disabled:              boolValue(item.Disabled),
			CreateTime:            stringValue(item.CreateTime),
			UpdateTime:            stringValue(item.UpdateTime),
			LastActiveTime:        stringValue(item.LastActiveTime),
			StatusChangedTime:     stringValue(item.StatusChangedTime),
			SuspendedTime:         stringValue(item.SuspendedTime),
		})
	}
	return result, nil
}

func (c *Client) DescribeComputeDetail(ctx context.Context, workspaceID, computeID string) (Compute, error) {
	req := (&aidap.DescribeComputeDetailInput{}).
		SetWorkspaceId(workspaceID).
		SetComputeId(computeID)
	resp, err := c.aidap.DescribeComputeDetailWithContext(ctx, req)
	if err != nil {
		return Compute{}, fmt.Errorf("failed to describe compute detail: %w", err)
	}
	item := resp.Compute
	if item == nil {
		return Compute{}, nil
	}
	return Compute{
		WorkspaceID:           stringValue(item.WorkspaceId),
		BranchID:              stringValue(item.BranchId),
		ComputeID:             stringValue(item.ComputeId),
		ComputeName:           stringValue(item.ComputeName),
		ComputeStatus:         stringValue(item.ComputeStatus),
		ComputeRole:           stringValue(item.ComputeRole),
		ServiceType:           stringValue(item.ServiceType),
		AutoScalingLimitMinCU: floatValue(item.AutoScalingLimitMinCU),
		AutoScalingLimitMaxCU: floatValue(item.AutoScalingLimitMaxCU),
		EnableAnalytics:       stringValue(item.EnableAnalytic),
		CreationSource:        stringValue(item.CreationSource),
		Disabled:              boolValue(item.Disabled),
		CreateTime:            stringValue(item.CreateTime),
		UpdateTime:            stringValue(item.UpdateTime),
		LastActiveTime:        stringValue(item.LastActiveTime),
		StatusChangedTime:     stringValue(item.StatusChangedTime),
		SuspendedTime:         stringValue(item.SuspendedTime),
	}, nil
}

func (c *Client) ModifyComputeSpec(ctx context.Context, params ModifyComputeSpecParams) (Compute, error) {
	req := (&aidap.ModifyComputeSpecInput{}).
		SetWorkspaceId(params.WorkspaceID).
		SetComputeId(params.ComputeID).
		SetAutoScalingLimitMinCU(params.AutoScalingLimitMinCU).
		SetAutoScalingLimitMaxCU(params.AutoScalingLimitMaxCU)
	resp, err := c.aidap.ModifyComputeSpecWithContext(ctx, req)
	if err != nil {
		return Compute{}, fmt.Errorf("failed to modify compute spec: %w", err)
	}
	item := resp.Compute
	if item == nil {
		return Compute{
			WorkspaceID: stringValue(resp.WorkspaceId),
			ComputeID:   stringValue(resp.ComputeId),
		}, nil
	}
	return Compute{
		WorkspaceID:           stringValue(item.WorkspaceId),
		BranchID:              stringValue(item.BranchId),
		ComputeID:             stringValue(item.ComputeId),
		ComputeName:           stringValue(item.ComputeName),
		ComputeStatus:         stringValue(item.ComputeStatus),
		ComputeRole:           stringValue(item.ComputeRole),
		ServiceType:           stringValue(item.ServiceType),
		AutoScalingLimitMinCU: floatValue(item.AutoScalingLimitMinCU),
		AutoScalingLimitMaxCU: floatValue(item.AutoScalingLimitMaxCU),
		EnableAnalytics:       stringValue(item.EnableAnalytic),
		CreationSource:        stringValue(item.CreationSource),
		Disabled:              boolValue(item.Disabled),
		CreateTime:            stringValue(item.CreateTime),
		UpdateTime:            stringValue(item.UpdateTime),
		LastActiveTime:        stringValue(item.LastActiveTime),
		StatusChangedTime:     stringValue(item.StatusChangedTime),
		SuspendedTime:         stringValue(item.SuspendedTime),
	}, nil
}

func (c *Client) ModifyComputeName(ctx context.Context, params ModifyComputeNameParams) error {
	req := (&aidap.ModifyComputeNameInput{}).
		SetWorkspaceId(params.WorkspaceID).
		SetComputeId(params.ComputeID).
		SetComputeName(params.ComputeName)
	if _, err := c.aidap.ModifyComputeNameWithContext(ctx, req); err != nil {
		return fmt.Errorf("failed to modify compute name: %w", err)
	}
	return nil
}

func (c *Client) CreateCompute(ctx context.Context, params CreateComputeParams) (Compute, error) {
	settings := &aidap.ComputeSettingsForCreateComputeInput{}
	if params.ComputeRole != "" {
		settings.SetComputeRole(params.ComputeRole)
	}
	if params.AutoScalingLimitMinCU > 0 {
		settings.SetAutoScalingLimitMinCU(params.AutoScalingLimitMinCU)
	}
	if params.AutoScalingLimitMaxCU > 0 {
		settings.SetAutoScalingLimitMaxCU(params.AutoScalingLimitMaxCU)
	}
	req := (&aidap.CreateComputeInput{}).
		SetWorkspaceId(params.WorkspaceID).
		SetBranchId(params.BranchID).
		SetComputeSettings(settings)
	if params.ComputeName != "" {
		req.SetComputeName(params.ComputeName)
	}
	resp, err := c.aidap.CreateComputeWithContext(ctx, req)
	if err != nil {
		return Compute{}, fmt.Errorf("failed to create compute: %w", err)
	}
	item := resp.Compute
	if item == nil {
		return Compute{
			WorkspaceID: stringValue(resp.WorkspaceId),
			BranchID:    stringValue(resp.BranchId),
			ComputeID:   stringValue(resp.ComputeId),
		}, nil
	}
	return Compute{
		WorkspaceID:           stringValue(item.WorkspaceId),
		BranchID:              stringValue(item.BranchId),
		ComputeID:             stringValue(item.ComputeId),
		ComputeName:           stringValue(item.ComputeName),
		ComputeStatus:         stringValue(item.ComputeStatus),
		ComputeRole:           stringValue(item.ComputeRole),
		ServiceType:           stringValue(item.ServiceType),
		AutoScalingLimitMinCU: floatValue(item.AutoScalingLimitMinCU),
		AutoScalingLimitMaxCU: floatValue(item.AutoScalingLimitMaxCU),
		EnableAnalytics:       stringValue(item.EnableAnalytic),
		CreationSource:        stringValue(item.CreationSource),
		Disabled:              boolValue(item.Disabled),
		CreateTime:            stringValue(item.CreateTime),
		UpdateTime:            stringValue(item.UpdateTime),
		LastActiveTime:        stringValue(item.LastActiveTime),
		StatusChangedTime:     stringValue(item.StatusChangedTime),
		SuspendedTime:         stringValue(item.SuspendedTime),
	}, nil
}

func (c *Client) DeleteCompute(ctx context.Context, workspaceID, computeID string) error {
	req := (&aidap.DeleteComputeInput{}).
		SetWorkspaceId(workspaceID).
		SetComputeId(computeID)
	if _, err := c.aidap.DeleteComputeWithContext(ctx, req); err != nil {
		return fmt.Errorf("failed to delete compute: %w", err)
	}
	return nil
}

func (c *Client) ModifyComputeAnalyticPolicy(ctx context.Context, workspaceID, computeID string) error {
	return c.SetComputeAnalyticPolicy(ctx, workspaceID, computeID, true)
}

func (c *Client) SetComputeAnalyticPolicy(ctx context.Context, workspaceID, computeID string, enabled bool) error {
	policy := aidap.EnumOfEnableAnalyticForModifyComputeAnalyticPolicyInputDisabled
	if enabled {
		policy = aidap.EnumOfEnableAnalyticForModifyComputeAnalyticPolicyInputEnabled
	}
	req := (&aidap.ModifyComputeAnalyticPolicyInput{}).
		SetWorkspaceId(workspaceID).
		SetComputeId(computeID).
		SetEnableAnalytic(policy)
	if _, err := c.aidap.ModifyComputeAnalyticPolicyWithContext(ctx, req); err != nil {
		return fmt.Errorf("failed to modify compute analytic policy: %w", err)
	}
	return nil
}

// ResolvePrimaryDatabaseComputeID resolves the primary Database compute of a
// branch. When branchID is empty the workspace's default branch is used. It is
// the single source of truth for the "resolve branch → find primary database
// compute" path shared by the CLI connection-string/psql helpers.
func (c *Client) ResolvePrimaryDatabaseComputeID(ctx context.Context, workspaceID, branchID string) (resolvedBranchID, computeID string, err error) {
	branchID, err = c.ResolveDefaultBranchID(ctx, workspaceID, branchID)
	if err != nil {
		return "", "", err
	}
	computes, err := c.DescribeComputes(ctx, workspaceID, branchID, ServiceTypeDatabase)
	if err != nil {
		return "", "", err
	}
	for _, compute := range computes.Computes {
		if strings.EqualFold(compute.ServiceType, ServiceTypeDatabase) &&
			strings.EqualFold(compute.ComputeRole, ComputeRolePrimary) {
			return branchID, compute.ComputeID, nil
		}
	}
	// Fall back to the first compute the branch exposes so callers still get a
	// usable connection when the role labelling is absent.
	if len(computes.Computes) > 0 {
		return branchID, computes.Computes[0].ComputeID, nil
	}
	return "", "", fmt.Errorf("primary database compute not found for branch %s", branchID)
}
