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

type ListWorkspacesParams struct {
	Search      string
	ProjectName string
	Limit       int
	Offset      int
}

type WorkspaceOverviewParams struct {
}

type CreateWorkspaceParams struct {
	WorkspaceName         string
	ProjectName           string
	IsAgentPlan           *bool
	AgentPlanSeatID       string
	MinCU                 *float64
	MaxCU                 *float64
	SuspendTimeoutSeconds *int
	HistoryRetentionHours *int
	SharedPublicNetwork   *bool
	VPCID                 string
	SubnetID              string
	InternetProtocol      string
	DeletionProtection    string
	Tags                  []WorkspaceTag
}

type WorkspaceTag struct {
	Key   string `json:"key" yaml:"key"`
	Value string `json:"value" yaml:"value"`
}

type ModifyComputeSettingsParams struct {
	AutoScalingLimitMinCU float64
	AutoScalingLimitMaxCU float64
	SuspendTimeoutSeconds *int
	ServiceType           string
}

type ModifyWorkspaceSettingsParams struct {
	HistoryRetentionHours int
}

type ListWorkspacesResult struct {
	Total      int
	Workspaces []Workspace `json:"Workspaces"`
}

type CreateWorkspaceResult struct {
	WorkspaceID string    `json:"WorkspaceId"`
	Workspace   Workspace `json:"Workspace"`
}

type DeleteWorkspaceResult struct {
	WorkspaceID string `json:"WorkspaceId"`
}

type WorkspaceOverviewResult struct {
	Overviews []WorkspaceOverview `json:"WorkspaceOverviews"`
}

type WorkspaceOverview struct {
	EngineType     string `json:"EngineType"`
	WorkspaceTotal int    `json:"WorkspaceTotal"`
	RunningTotal   int    `json:"RunningTotal"`
	CreatingTotal  int    `json:"CreatingTotal"`
	UpdatingTotal  int    `json:"UpdatingTotal"`
	StoppedTotal   int    `json:"StoppedTotal"`
	SuspendedTotal int    `json:"SuspendedTotal"`
	ClosedTotal    int    `json:"ClosedTotal"`
	ErrorTotal     int    `json:"ErrorTotal"`
}

type Workspace struct {
	WorkspaceID              string                   `json:"WorkspaceId"`
	WorkspaceName            string                   `json:"WorkspaceName"`
	ProjectName              string                   `json:"ProjectName"`
	AccountID                string                   `json:"AccountId"`
	EngineType               string                   `json:"EngineType"`
	EngineVersion            string                   `json:"EngineVersion"`
	WorkspaceStatus          string                   `json:"WorkspaceStatus"`
	CreateTime               string                   `json:"CreateTime"`
	UpdateTime               string                   `json:"UpdateTime"`
	CreationSource           string                   `json:"CreationSource"`
	DeletionProtectionStatus string                   `json:"DeletionProtectionStatus"`
	InternetProtocol         string                   `json:"InternetProtocol"`
	DNSVisibility            bool                     `json:"DNSVisibility"`
	SharedPrivateNetwork     bool                     `json:"SharedPrivateNetwork"`
	StorageSize              int                      `json:"StorageSize"`
	StorageType              string                   `json:"StorageType"`
	VpcID                    string                   `json:"VpcId"`
	SubnetID                 string                   `json:"SubnetId"`
	WorkspaceSetting         WorkspaceSetting         `json:"WorkspaceSetting"`
	ComputeSettings          WorkspaceComputeSettings `json:"ComputeSettings"`
	WorkspaceUsage           WorkspaceUsage           `json:"WorkspaceUsage"`
	Tags                     []WorkspaceTag           `json:"Tags,omitempty"`
}

type WorkspaceSetting struct {
	DeletionProtection    string `json:"DeletionProtection"`
	HistoryRetentionHours int    `json:"HistoryRetentionHours"`
	PublicConnection      string `json:"PublicConnection"`
}

type WorkspaceComputeSettings struct {
	AutoScalingLimitMinCU float64 `json:"AutoScalingLimitMinCU"`
	AutoScalingLimitMaxCU float64 `json:"AutoScalingLimitMaxCU"`
	EnableAnalytics       string  `json:"EnableAnalytics"`
	SuspendTimeoutSeconds int     `json:"SuspendTimeoutSeconds"`
}

type WorkspaceUsage struct {
	DataSizeTotalBytes   int64  `json:"DataSizeTotalBytes"`
	DataSizeUsedBytes    int64  `json:"DataSizeUsedBytes"`
	StorageSizeUsedBytes int64  `json:"StorageSizeUsedBytes"`
	BranchCreatedNum     int64  `json:"BranchCreatedNum"`
	FunctionCallNum      int64  `json:"FunctionCallNum"`
	StatTime             string `json:"StatTime"`
}

// ListWorkspaces lists PostgreSQL workspaces, filtering out every other engine
// hosted on the same AIDAP platform.
//
// DescribeWorkspaces is region-scoped by the signed SDK endpoint: one call
// returns every workspace the caller can see in the selected region.
func (c *Client) ListWorkspaces(ctx context.Context, params ListWorkspacesParams) (ListWorkspacesResult, error) {
	limit, err := normalizePageLimit(params.Limit)
	if err != nil {
		return ListWorkspacesResult{}, err
	}
	if err := validatePageOffset(params.Offset); err != nil {
		return ListWorkspacesResult{}, err
	}
	req := (&aidap.DescribeWorkspacesInput{}).
		SetSortBy("update_time").
		SetSortOrder(sortOrderDesc).
		SetLimit(int32(limit)).
		SetOffset(int32(params.Offset)).
		SetFilters([]*aidap.FilterForDescribeWorkspacesInput{
			(&aidap.FilterForDescribeWorkspacesInput{}).
				SetName(filterNameDBEngineVersion).
				SetValue(engineVersionPostgreSQL17),
		})
	if params.Search != "" {
		req.SetSearch(params.Search)
	}
	if params.ProjectName != "" {
		req.SetProjectName(params.ProjectName)
	}
	resp, err := c.aidap.DescribeWorkspacesWithContext(ctx, req)
	if err != nil {
		return ListWorkspacesResult{}, fmt.Errorf("failed to describe workspaces: %w", err)
	}
	return mapListWorkspacesResult(resp), nil
}

// ListAllWorkspaces pages through every PostgreSQL workspace.
func (c *Client) ListAllWorkspaces(ctx context.Context, params ListWorkspacesParams) (ListWorkspacesResult, error) {
	limit := params.Limit
	if limit == 0 || limit > maxWorkspaceLimit {
		limit = maxWorkspaceLimit
	}
	params.Limit = limit

	var result ListWorkspacesResult
	for pageNumber := 0; ; pageNumber++ {
		if pageNumber >= MaxAllPages {
			return ListWorkspacesResult{}, fmt.Errorf("workspace pagination exceeded maximum of %d pages", MaxAllPages)
		}
		page, err := c.ListWorkspaces(ctx, params)
		if err != nil {
			return ListWorkspacesResult{}, err
		}
		result.Total = page.Total
		result.Workspaces = append(result.Workspaces, page.Workspaces...)
		if len(result.Workspaces) >= page.Total || len(page.Workspaces) == 0 {
			return result, nil
		}
		nextOffset := params.Offset + limit
		if nextOffset <= params.Offset {
			return ListWorkspacesResult{}, fmt.Errorf("workspace pagination offset did not advance")
		}
		params.Offset = nextOffset
	}
}

// CreateWorkspace creates a PostgreSQL_17 workspace.
func (c *Client) CreateWorkspace(ctx context.Context, params CreateWorkspaceParams) (CreateWorkspaceResult, error) {
	req := (&aidap.CreateWorkspaceInput{}).
		SetEngineType(aidap.EnumOfEngineTypeForCreateWorkspaceInputPostgreSql).
		SetEngineVersion(aidap.EnumOfEngineVersionForCreateWorkspaceInputPostgreSql17)
	if params.WorkspaceName != "" {
		req.SetWorkspaceName(params.WorkspaceName)
	}
	if params.ProjectName != "" {
		req.SetProjectName(params.ProjectName)
	}
	if params.IsAgentPlan != nil || strings.TrimSpace(params.AgentPlanSeatID) != "" {
		agentPlan := &aidap.AgentPlanSettingsForCreateWorkspaceInput{}
		if params.IsAgentPlan != nil {
			agentPlan.SetIsAgentPlan(*params.IsAgentPlan)
		}
		if seatID := strings.TrimSpace(params.AgentPlanSeatID); seatID != "" {
			agentPlan.SetAgentPlanSeatId(seatID)
		}
		req.SetAgentPlanSettings(agentPlan)
	}
	if params.SuspendTimeoutSeconds != nil {
		settings := (&aidap.ComputeSettingsForCreateWorkspaceInput{}).
			SetSuspendTimeoutSeconds(int32(*params.SuspendTimeoutSeconds))
		if params.MinCU != nil {
			settings.SetAutoScalingLimitMinCU(*params.MinCU)
		}
		if params.MaxCU != nil {
			settings.SetAutoScalingLimitMaxCU(*params.MaxCU)
		}
		req.SetComputeSettings(settings)
	} else if params.MinCU != nil || params.MaxCU != nil {
		settings := &aidap.ComputeSettingsForCreateWorkspaceInput{}
		if params.MinCU != nil {
			settings.SetAutoScalingLimitMinCU(*params.MinCU)
		}
		if params.MaxCU != nil {
			settings.SetAutoScalingLimitMaxCU(*params.MaxCU)
		}
		req.SetComputeSettings(settings)
	}
	if params.HistoryRetentionHours != nil || params.DeletionProtection != "" {
		settings := &aidap.WorkspaceSettingsForCreateWorkspaceInput{}
		if params.HistoryRetentionHours != nil {
			settings.SetHistoryRetentionHours(int32(*params.HistoryRetentionHours))
		}
		if params.DeletionProtection != "" {
			settings.SetDeletionProtection(params.DeletionProtection)
		}
		req.SetWorkspaceSettings(settings)
	}
	if params.SharedPublicNetwork != nil || params.VPCID != "" || params.SubnetID != "" || params.InternetProtocol != "" {
		settings := (&aidap.NetworkSettingsForCreateWorkspaceInput{}).
			SetVpcId(params.VPCID).
			SetSubnetId(params.SubnetID).
			SetInternetProtocol(params.InternetProtocol)
		if params.SharedPublicNetwork != nil {
			settings.SetSharedPublicNetwork(*params.SharedPublicNetwork)
		}
		req.SetNetworkSettings(settings)
	}
	if len(params.Tags) > 0 {
		tags := make([]*aidap.WorkspaceTagForCreateWorkspaceInput, 0, len(params.Tags))
		for _, tag := range params.Tags {
			tags = append(tags, (&aidap.WorkspaceTagForCreateWorkspaceInput{}).SetKey(tag.Key).SetValue(tag.Value))
		}
		req.SetWorkspaceTags(tags)
	}
	resp, err := c.aidap.CreateWorkspaceWithContext(ctx, req)
	if err != nil {
		return CreateWorkspaceResult{}, fmt.Errorf("failed to create workspace: %w", err)
	}
	return mapCreateWorkspaceResult(resp), nil
}

func (c *Client) ModifyWorkspaceVpcSettings(ctx context.Context, workspaceID, branchID, vpcID, subnetID, internetProtocol string) error {
	req := (&aidap.ModifyVpcSettingsInput{}).
		SetWorkspaceId(workspaceID).SetBranchId(branchID).
		SetVpcId(vpcID).SetSubnetId(subnetID).SetInternetProtocol(internetProtocol)
	if _, err := c.aidap.ModifyVpcSettingsWithContext(ctx, req); err != nil {
		return fmt.Errorf("failed to modify workspace network settings: %w", err)
	}
	return nil
}

func (c *Client) AddWorkspaceTags(ctx context.Context, workspaceID string, tags []WorkspaceTag) error {
	tagInputs := make([]*aidap.TagForAddTagsToWorkspacesInput, 0, len(tags))
	for _, tag := range tags {
		tagInputs = append(tagInputs, (&aidap.TagForAddTagsToWorkspacesInput{}).SetKey(tag.Key).SetValue(tag.Value))
	}
	if _, err := c.aidap.AddTagsToWorkspacesWithContext(ctx,
		(&aidap.AddTagsToWorkspacesInput{}).
			SetWorkspaceIds([]*string{&workspaceID}).SetTags(tagInputs)); err != nil {
		return fmt.Errorf("failed to add workspace tags: %w", err)
	}
	return nil
}

func (c *Client) RemoveWorkspaceTags(ctx context.Context, workspaceID string, keys []string) error {
	if _, err := c.aidap.RemoveTagsFromWorkspacesWithContext(ctx,
		(&aidap.RemoveTagsFromWorkspacesInput{}).
			SetWorkspaceIds([]*string{&workspaceID}).SetTagKeys(stringSlicePointers(keys))); err != nil {
		return fmt.Errorf("failed to remove workspace tags: %w", err)
	}
	return nil
}

func (c *Client) DeleteWorkspace(ctx context.Context, workspaceID string) (DeleteWorkspaceResult, error) {
	req := (&aidap.DeleteWorkspaceInput{}).SetWorkspaceId(workspaceID)
	resp, err := c.aidap.DeleteWorkspaceWithContext(ctx, req)
	if err != nil {
		return DeleteWorkspaceResult{}, fmt.Errorf("failed to delete workspace: %w", err)
	}
	return DeleteWorkspaceResult{WorkspaceID: stringValue(resp.WorkspaceId)}, nil
}

func (c *Client) StartWorkspace(ctx context.Context, workspaceID string) error {
	req := (&aidap.StartWorkspaceInput{}).SetWorkspaceId(workspaceID)
	if _, err := c.aidap.StartWorkspaceWithContext(ctx, req); err != nil {
		return fmt.Errorf("failed to start workspace: %w", err)
	}
	return nil
}

func (c *Client) StopWorkspace(ctx context.Context, workspaceID string) error {
	req := (&aidap.StopWorkspaceInput{}).SetWorkspaceId(workspaceID)
	if _, err := c.aidap.StopWorkspaceWithContext(ctx, req); err != nil {
		return fmt.Errorf("failed to stop workspace: %w", err)
	}
	return nil
}

func (c *Client) ModifyWorkspaceName(ctx context.Context, workspaceID, workspaceName string) error {
	req := (&aidap.ModifyWorkspaceNameInput{}).SetWorkspaceId(workspaceID).SetWorkspaceName(workspaceName)
	if _, err := c.aidap.ModifyWorkspaceNameWithContext(ctx, req); err != nil {
		return fmt.Errorf("failed to modify workspace name: %w", err)
	}
	return nil
}

func (c *Client) ModifyWorkspaceDeletionProtectionPolicy(ctx context.Context, workspaceID string, enabled bool) error {
	deletionProtection := deletionProtectionDisabled
	if enabled {
		deletionProtection = deletionProtectionEnabled
	}
	req := (&aidap.ModifyWorkspaceDeletionProtectionPolicyInput{}).
		SetWorkspaceId(workspaceID).
		SetDeletionProtection(deletionProtection)
	if _, err := c.aidap.ModifyWorkspaceDeletionProtectionPolicyWithContext(ctx, req); err != nil {
		return fmt.Errorf("failed to modify workspace deletion protection policy: %w", err)
	}
	return nil
}

func (c *Client) ModifyComputeSettings(ctx context.Context, workspaceID string, params ModifyComputeSettingsParams) (Workspace, error) {
	req := (&aidap.ModifyComputeSettingsInput{}).
		SetWorkspaceId(workspaceID).
		SetAutoScalingLimitMinCU(params.AutoScalingLimitMinCU).
		SetAutoScalingLimitMaxCU(params.AutoScalingLimitMaxCU)
	if params.ServiceType != "" {
		req.SetServiceType(params.ServiceType)
	}
	if params.SuspendTimeoutSeconds != nil {
		req.SetSuspendTimeoutSeconds(int32(*params.SuspendTimeoutSeconds))
	}
	resp, err := c.aidap.ModifyComputeSettingsWithContext(ctx, req)
	if err != nil {
		return Workspace{}, fmt.Errorf("failed to modify compute settings: %w", err)
	}
	return mapWorkspaceFromModifyComputeSettings(resp.Workspace), nil
}

func (c *Client) ModifyWorkspaceSettings(ctx context.Context, workspaceID string, params ModifyWorkspaceSettingsParams) (Workspace, error) {
	req := (&aidap.ModifyWorkspaceSettingsInput{}).
		SetWorkspaceId(workspaceID).
		SetWorkspaceSettings((&aidap.WorkspaceSettingsForModifyWorkspaceSettingsInput{}).
			SetHistoryRetentionHours(int32(params.HistoryRetentionHours)))
	resp, err := c.aidap.ModifyWorkspaceSettingsWithContext(ctx, req)
	if err != nil {
		return Workspace{}, fmt.Errorf("failed to modify workspace settings: %w", err)
	}
	return mapWorkspaceFromModifyWorkspaceSettings(resp.Workspace), nil
}

func (c *Client) DescribeWorkspaceOverview(ctx context.Context, params WorkspaceOverviewParams) (WorkspaceOverviewResult, error) {
	req := (&aidap.DescribeWorkspaceOverviewInput{}).
		SetEngineType(aidap.EnumOfEngineTypeForDescribeWorkspaceOverviewInputPostgreSql)
	resp, err := c.aidap.DescribeWorkspaceOverviewWithContext(ctx, req)
	if err != nil {
		return WorkspaceOverviewResult{}, fmt.Errorf("failed to describe workspace overview: %w", err)
	}
	return mapWorkspaceOverviewResult(resp), nil
}

// DescribeWorkspaceDetail returns a single workspace by id.
func (c *Client) DescribeWorkspaceDetail(ctx context.Context, workspaceID string) (Workspace, error) {
	req := (&aidap.DescribeWorkspaceDetailInput{}).SetWorkspaceId(workspaceID)
	resp, err := c.aidap.DescribeWorkspaceDetailWithContext(ctx, req)
	if err != nil {
		return Workspace{}, fmt.Errorf("failed to describe workspace detail: %w", err)
	}
	return mapWorkspaceFromDetail(resp.Workspace), nil
}

func mapListWorkspacesResult(resp *aidap.DescribeWorkspacesOutput) ListWorkspacesResult {
	if resp == nil {
		return ListWorkspacesResult{}
	}
	result := ListWorkspacesResult{Total: intValue(resp.Total)}
	for _, workspace := range resp.Workspaces {
		result.Workspaces = append(result.Workspaces, mapWorkspaceFromList(workspace))
	}
	return result
}

func mapCreateWorkspaceResult(resp *aidap.CreateWorkspaceOutput) CreateWorkspaceResult {
	if resp == nil {
		return CreateWorkspaceResult{}
	}
	return CreateWorkspaceResult{
		WorkspaceID: stringValue(resp.WorkspaceId),
		Workspace:   mapWorkspaceFromCreate(resp.Workspace),
	}
}

func mapWorkspaceOverviewResult(resp *aidap.DescribeWorkspaceOverviewOutput) WorkspaceOverviewResult {
	if resp == nil {
		return WorkspaceOverviewResult{}
	}
	var result WorkspaceOverviewResult
	for _, overview := range resp.WorkspaceOverviews {
		if overview == nil {
			continue
		}
		result.Overviews = append(result.Overviews, WorkspaceOverview{
			EngineType:     stringValue(overview.EngineType),
			WorkspaceTotal: intValue(overview.WorkspaceTotal),
			RunningTotal:   intValue(overview.RunningTotal),
			CreatingTotal:  intValue(overview.CreatingTotal),
			UpdatingTotal:  intValue(overview.UpdatingTotal),
			StoppedTotal:   intValue(overview.StoppedTotal),
			SuspendedTotal: intValue(overview.SuspendedTotal),
			ClosedTotal:    intValue(overview.ClosedTotal),
			ErrorTotal:     intValue(overview.ErrorTotal),
		})
	}
	return result
}

func mapWorkspaceFromList(workspace *aidap.WorkspaceForDescribeWorkspacesOutput) Workspace {
	if workspace == nil {
		return Workspace{}
	}
	result := Workspace{
		WorkspaceID:              stringValue(workspace.WorkspaceId),
		WorkspaceName:            stringValue(workspace.WorkspaceName),
		ProjectName:              stringValue(workspace.ProjectName),
		AccountID:                stringValue(workspace.AccountId),
		EngineType:               stringValue(workspace.EngineType),
		EngineVersion:            stringValue(workspace.EngineVersion),
		WorkspaceStatus:          stringValue(workspace.WorkspaceStatus),
		CreateTime:               stringValue(workspace.CreateTime),
		UpdateTime:               stringValue(workspace.UpdateTime),
		CreationSource:           stringValue(workspace.CreationSource),
		DeletionProtectionStatus: stringValue(workspace.DeletionProtectionStatus),
		InternetProtocol:         stringValue(workspace.InternetProtocol),
		DNSVisibility:            boolValue(workspace.DNSVisibility),
		SharedPrivateNetwork:     boolValue(workspace.SharedPrivateNetwork),
		VpcID:                    stringValue(workspace.VpcId),
		SubnetID:                 stringValue(workspace.SubnetId),
	}
	if workspace.WorkspaceSetting != nil {
		result.WorkspaceSetting = WorkspaceSetting{
			DeletionProtection:    stringValue(workspace.WorkspaceSetting.DeletionProtection),
			HistoryRetentionHours: intValue(workspace.WorkspaceSetting.HistoryRetentionHours),
			PublicConnection:      stringValue(workspace.WorkspaceSetting.PublicConnection),
		}
	}
	if workspace.ComputeSettings != nil {
		result.ComputeSettings = WorkspaceComputeSettings{
			AutoScalingLimitMinCU: floatValue(workspace.ComputeSettings.AutoScalingLimitMinCU),
			AutoScalingLimitMaxCU: floatValue(workspace.ComputeSettings.AutoScalingLimitMaxCU),
			EnableAnalytics:       stringValue(workspace.ComputeSettings.EnableAnalytic),
			SuspendTimeoutSeconds: intValue(workspace.ComputeSettings.SuspendTimeoutSeconds),
		}
	}
	if workspace.WorkspaceUsage != nil {
		result.WorkspaceUsage = WorkspaceUsage{
			DataSizeTotalBytes:   int64Value(workspace.WorkspaceUsage.DataSizeTotalBytes),
			DataSizeUsedBytes:    int64Value(workspace.WorkspaceUsage.DataSizeUsedBytes),
			StorageSizeUsedBytes: int64Value(workspace.WorkspaceUsage.StorageSizeUsedBytes),
			BranchCreatedNum:     int64Value(workspace.WorkspaceUsage.BranchCreatedNum),
			FunctionCallNum:      int64Value(workspace.WorkspaceUsage.FunctionCallNum),
			StatTime:             stringValue(workspace.WorkspaceUsage.StatTime),
		}
	}
	return result
}

func mapWorkspaceFromCreate(workspace *aidap.WorkspaceForCreateWorkspaceOutput) Workspace {
	if workspace == nil {
		return Workspace{}
	}
	result := Workspace{
		WorkspaceID:              stringValue(workspace.WorkspaceId),
		WorkspaceName:            stringValue(workspace.WorkspaceName),
		ProjectName:              stringValue(workspace.ProjectName),
		AccountID:                stringValue(workspace.AccountId),
		EngineType:               stringValue(workspace.EngineType),
		EngineVersion:            stringValue(workspace.EngineVersion),
		WorkspaceStatus:          stringValue(workspace.WorkspaceStatus),
		CreateTime:               stringValue(workspace.CreateTime),
		UpdateTime:               stringValue(workspace.UpdateTime),
		CreationSource:           stringValue(workspace.CreationSource),
		DeletionProtectionStatus: stringValue(workspace.DeletionProtectionStatus),
		InternetProtocol:         stringValue(workspace.InternetProtocol),
		DNSVisibility:            boolValue(workspace.DNSVisibility),
		SharedPrivateNetwork:     boolValue(workspace.SharedPrivateNetwork),
		VpcID:                    stringValue(workspace.VpcId),
		SubnetID:                 stringValue(workspace.SubnetId),
	}
	if workspace.WorkspaceSetting != nil {
		result.WorkspaceSetting = WorkspaceSetting{
			DeletionProtection:    stringValue(workspace.WorkspaceSetting.DeletionProtection),
			HistoryRetentionHours: intValue(workspace.WorkspaceSetting.HistoryRetentionHours),
			PublicConnection:      stringValue(workspace.WorkspaceSetting.PublicConnection),
		}
	}
	if workspace.ComputeSettings != nil {
		result.ComputeSettings = WorkspaceComputeSettings{
			AutoScalingLimitMinCU: floatValue(workspace.ComputeSettings.AutoScalingLimitMinCU),
			AutoScalingLimitMaxCU: floatValue(workspace.ComputeSettings.AutoScalingLimitMaxCU),
			EnableAnalytics:       stringValue(workspace.ComputeSettings.EnableAnalytic),
			SuspendTimeoutSeconds: intValue(workspace.ComputeSettings.SuspendTimeoutSeconds),
		}
	}
	return result
}

func mapWorkspaceFromDetail(workspace *aidap.WorkspaceForDescribeWorkspaceDetailOutput) Workspace {
	if workspace == nil {
		return Workspace{}
	}
	result := Workspace{
		WorkspaceID:              stringValue(workspace.WorkspaceId),
		WorkspaceName:            stringValue(workspace.WorkspaceName),
		ProjectName:              stringValue(workspace.ProjectName),
		AccountID:                stringValue(workspace.AccountId),
		EngineType:               stringValue(workspace.EngineType),
		EngineVersion:            stringValue(workspace.EngineVersion),
		WorkspaceStatus:          stringValue(workspace.WorkspaceStatus),
		CreateTime:               stringValue(workspace.CreateTime),
		UpdateTime:               stringValue(workspace.UpdateTime),
		CreationSource:           stringValue(workspace.CreationSource),
		DeletionProtectionStatus: stringValue(workspace.DeletionProtectionStatus),
		InternetProtocol:         stringValue(workspace.InternetProtocol),
		DNSVisibility:            boolValue(workspace.DNSVisibility),
		SharedPrivateNetwork:     boolValue(workspace.SharedPrivateNetwork),
		VpcID:                    stringValue(workspace.VpcId),
		SubnetID:                 stringValue(workspace.SubnetId),
	}
	for _, tag := range workspace.WorkspaceTags {
		if tag != nil {
			result.Tags = append(result.Tags, WorkspaceTag{Key: stringValue(tag.Key), Value: stringValue(tag.Value)})
		}
	}
	if workspace.WorkspaceSetting != nil {
		result.WorkspaceSetting = WorkspaceSetting{
			DeletionProtection:    stringValue(workspace.WorkspaceSetting.DeletionProtection),
			HistoryRetentionHours: intValue(workspace.WorkspaceSetting.HistoryRetentionHours),
			PublicConnection:      stringValue(workspace.WorkspaceSetting.PublicConnection),
		}
	}
	if workspace.ComputeSettings != nil {
		result.ComputeSettings = WorkspaceComputeSettings{
			AutoScalingLimitMinCU: floatValue(workspace.ComputeSettings.AutoScalingLimitMinCU),
			AutoScalingLimitMaxCU: floatValue(workspace.ComputeSettings.AutoScalingLimitMaxCU),
			EnableAnalytics:       stringValue(workspace.ComputeSettings.EnableAnalytic),
			SuspendTimeoutSeconds: intValue(workspace.ComputeSettings.SuspendTimeoutSeconds),
		}
	}
	if workspace.WorkspaceUsage != nil {
		result.WorkspaceUsage = WorkspaceUsage{
			DataSizeTotalBytes:   int64Value(workspace.WorkspaceUsage.DataSizeTotalBytes),
			DataSizeUsedBytes:    int64Value(workspace.WorkspaceUsage.DataSizeUsedBytes),
			StorageSizeUsedBytes: int64Value(workspace.WorkspaceUsage.StorageSizeUsedBytes),
			BranchCreatedNum:     int64Value(workspace.WorkspaceUsage.BranchCreatedNum),
			FunctionCallNum:      int64Value(workspace.WorkspaceUsage.FunctionCallNum),
			StatTime:             stringValue(workspace.WorkspaceUsage.StatTime),
		}
	}
	return result
}

func mapWorkspaceFromModifyComputeSettings(workspace *aidap.WorkspaceForModifyComputeSettingsOutput) Workspace {
	if workspace == nil {
		return Workspace{}
	}
	result := Workspace{
		WorkspaceID:              stringValue(workspace.WorkspaceId),
		WorkspaceName:            stringValue(workspace.WorkspaceName),
		ProjectName:              stringValue(workspace.ProjectName),
		AccountID:                stringValue(workspace.AccountId),
		EngineType:               stringValue(workspace.EngineType),
		EngineVersion:            stringValue(workspace.EngineVersion),
		WorkspaceStatus:          stringValue(workspace.WorkspaceStatus),
		CreateTime:               stringValue(workspace.CreateTime),
		UpdateTime:               stringValue(workspace.UpdateTime),
		CreationSource:           stringValue(workspace.CreationSource),
		DeletionProtectionStatus: stringValue(workspace.DeletionProtectionStatus),
		InternetProtocol:         stringValue(workspace.InternetProtocol),
		DNSVisibility:            boolValue(workspace.DNSVisibility),
		SharedPrivateNetwork:     boolValue(workspace.SharedPrivateNetwork),
		VpcID:                    stringValue(workspace.VpcId),
		SubnetID:                 stringValue(workspace.SubnetId),
	}
	if workspace.WorkspaceSetting != nil {
		result.WorkspaceSetting = WorkspaceSetting{
			DeletionProtection:    stringValue(workspace.WorkspaceSetting.DeletionProtection),
			HistoryRetentionHours: intValue(workspace.WorkspaceSetting.HistoryRetentionHours),
			PublicConnection:      stringValue(workspace.WorkspaceSetting.PublicConnection),
		}
	}
	if workspace.ComputeSettings != nil {
		result.ComputeSettings = WorkspaceComputeSettings{
			AutoScalingLimitMinCU: floatValue(workspace.ComputeSettings.AutoScalingLimitMinCU),
			AutoScalingLimitMaxCU: floatValue(workspace.ComputeSettings.AutoScalingLimitMaxCU),
			EnableAnalytics:       stringValue(workspace.ComputeSettings.EnableAnalytic),
			SuspendTimeoutSeconds: intValue(workspace.ComputeSettings.SuspendTimeoutSeconds),
		}
	}
	return result
}

func mapWorkspaceFromModifyWorkspaceSettings(workspace *aidap.WorkspaceForModifyWorkspaceSettingsOutput) Workspace {
	if workspace == nil {
		return Workspace{}
	}
	result := Workspace{
		WorkspaceID:              stringValue(workspace.WorkspaceId),
		WorkspaceName:            stringValue(workspace.WorkspaceName),
		ProjectName:              stringValue(workspace.ProjectName),
		AccountID:                stringValue(workspace.AccountId),
		EngineType:               stringValue(workspace.EngineType),
		EngineVersion:            stringValue(workspace.EngineVersion),
		WorkspaceStatus:          stringValue(workspace.WorkspaceStatus),
		CreateTime:               stringValue(workspace.CreateTime),
		UpdateTime:               stringValue(workspace.UpdateTime),
		CreationSource:           stringValue(workspace.CreationSource),
		DeletionProtectionStatus: stringValue(workspace.DeletionProtectionStatus),
		InternetProtocol:         stringValue(workspace.InternetProtocol),
		DNSVisibility:            boolValue(workspace.DNSVisibility),
		SharedPrivateNetwork:     boolValue(workspace.SharedPrivateNetwork),
		VpcID:                    stringValue(workspace.VpcId),
		SubnetID:                 stringValue(workspace.SubnetId),
	}
	if workspace.WorkspaceSetting != nil {
		result.WorkspaceSetting = WorkspaceSetting{
			DeletionProtection:    stringValue(workspace.WorkspaceSetting.DeletionProtection),
			HistoryRetentionHours: intValue(workspace.WorkspaceSetting.HistoryRetentionHours),
			PublicConnection:      stringValue(workspace.WorkspaceSetting.PublicConnection),
		}
	}
	if workspace.ComputeSettings != nil {
		result.ComputeSettings = WorkspaceComputeSettings{
			AutoScalingLimitMinCU: floatValue(workspace.ComputeSettings.AutoScalingLimitMinCU),
			AutoScalingLimitMaxCU: floatValue(workspace.ComputeSettings.AutoScalingLimitMaxCU),
			EnableAnalytics:       stringValue(workspace.ComputeSettings.EnableAnalytic),
			SuspendTimeoutSeconds: intValue(workspace.ComputeSettings.SuspendTimeoutSeconds),
		}
	}
	return result
}
