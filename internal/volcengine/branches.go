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

type DescribeBranchesParams struct {
	WorkspaceID string
	Search      string
	Limit       int
	Offset      int
}

type DescribeChildBranchesParams struct {
	WorkspaceID string
	ParentID    string
	Limit       int
	Offset      int
}

type DescribeBranchesResult struct {
	Total         int
	WorkspaceName string
	Branches      []Branch
}

type DescribeBranchDetailResult struct {
	WorkspaceName string       `json:"WorkspaceName"`
	Branch        BranchDetail `json:"Branch"`
}

type DescribeDefaultBranchResult struct {
	Branch Branch
}

type CreateBranchParams struct {
	WorkspaceID string
	Name        string
	ParentID    string
	ParentTime  string
}

type CreateBranchResult struct {
	WorkspaceID string       `json:"WorkspaceId"`
	BranchID    string       `json:"BranchId"`
	Branch      BranchDetail `json:"Branch"`
}

// NormalizedBranch returns the result's branch detail with its branch/workspace
// ids backfilled from the top-level fields when the API leaves the nested Branch
// sparse.
func (r CreateBranchResult) NormalizedBranch() BranchDetail {
	branch := r.Branch
	if branch.BranchID == "" {
		branch.BranchID = r.BranchID
	}
	if branch.WorkspaceID == "" {
		branch.WorkspaceID = r.WorkspaceID
	}
	return branch
}

type UpdateBranchParams struct {
	WorkspaceID string
	BranchID    string
	Name        *string
	Protected   *bool
}

type UpdateBranchResult struct {
	Branch BranchDetail `json:"Branch"`
}

type DeleteBranchResult struct {
	WorkspaceID string `json:"WorkspaceId"`
	BranchID    string `json:"BranchId"`
}

type RestartBranchParams struct {
	WorkspaceID string
	BranchID    string
	ComputeIDs  []string
}

type RestartBranchResult struct {
	WorkspaceID string   `json:"WorkspaceId"`
	BranchID    string   `json:"BranchId"`
	ComputeIDs  []string `json:"ComputeIds,omitempty"`
}

type SetAsDefaultBranchResult struct {
	Branch BranchDetail `json:"Branch"`
}

type RestoreWindow struct {
	WorkspaceID        string `json:"WorkspaceId,omitempty"`
	BranchID           string `json:"BranchId"`
	WindowSizeSeconds  int64  `json:"WindowSizeSeconds"`
	BranchCreationTime string `json:"BranchCreateTime"`
	StartTime          string `json:"StartTime"`
	EndTime            string `json:"EndTime"`
}

type DescribeRestorableBranchesParams struct {
	WorkspaceID string
	Time        string
	Search      string
	Limit       int
	Offset      int
}

type DescribeRestorableBranchesResult struct {
	Total          int             `json:"Total"`
	WorkspaceName  string          `json:"WorkspaceName"`
	Branches       []Branch        `json:"Branches"`
	RestoreWindows []RestoreWindow `json:"RestoreWindows"`
}

type BranchRestoreParams struct {
	WorkspaceID    string
	BranchID       string
	Time           string
	SourceBranchID string
}

type BranchRestoreResult struct {
	WorkspaceID    string `json:"WorkspaceId"`
	BranchID       string `json:"BranchId"`
	SourceBranchID string `json:"SourceBranchId,omitempty"`
	Time           string `json:"Time,omitempty"`
	BackupBranchID string `json:"BackupBranchId,omitempty"`
}

type Branch struct {
	WorkspaceID  string `json:"WorkspaceId"`
	BranchID     string `json:"BranchId"`
	BranchName   string `json:"BranchName"`
	BranchStatus string `json:"BranchStatus"`
	Default      bool   `json:"Default"`
	Protected    bool   `json:"Protected"`
	Archived     bool   `json:"Archived"`
	InitSource   string `json:"InitSource"`
	CreateTime   string `json:"CreateTime"`
	UpdateTime   string `json:"UpdateTime"`
}

type BranchDetail struct {
	WorkspaceID       string       `json:"WorkspaceId"`
	BranchID          string       `json:"BranchId"`
	BranchName        string       `json:"BranchName"`
	BranchStatus      string       `json:"BranchStatus"`
	Default           bool         `json:"Default"`
	Protected         bool         `json:"Protected"`
	Archived          bool         `json:"Archived"`
	InitSource        string       `json:"InitSource"`
	CreationSource    string       `json:"CreationSource"`
	CreateTime        string       `json:"CreateTime"`
	UpdateTime        string       `json:"UpdateTime"`
	LastResetTime     string       `json:"LastResetTime"`
	StartParentLSN    string       `json:"StartParentLSN"`
	StartParentTime   string       `json:"StartParentTime"`
	StatusChangedTime string       `json:"StatusChangedTime"`
	ParentBranch      ParentBranch `json:"ParentBranch"`
	BranchUsage       BranchUsage  `json:"BranchUsage"`
}

type ParentBranch struct {
	WorkspaceID  string `json:"WorkspaceId"`
	BranchID     string `json:"BranchId"`
	BranchName   string `json:"BranchName"`
	BranchStatus string `json:"BranchStatus"`
}

type BranchUsage struct {
	WorkspaceID          string `json:"WorkspaceId"`
	BranchID             string `json:"BranchId"`
	ComputeTimeSeconds   int64  `json:"ComputeTimeSeconds"`
	DataSizeTotalBytes   int64  `json:"DataSizeTotalBytes"`
	DataSizeUsedBytes    int64  `json:"DataSizeUsedBytes"`
	FunctionCallNum      int64  `json:"FunctionCallNum"`
	LastRunningTime      string `json:"LastRunningTime"`
	ServiceTimeSeconds   int64  `json:"ServiceTimeSeconds"`
	StatTime             string `json:"StatTime"`
	StorageSizeUsedBytes int64  `json:"StorageSizeUsedBytes"`
}

func (c *Client) DescribeBranches(ctx context.Context, params DescribeBranchesParams) (DescribeBranchesResult, error) {
	limit := params.Limit
	if limit <= 0 {
		limit = DefaultListLimit
	}
	req := (&aidap.DescribeBranchesInput{}).
		SetWorkspaceId(params.WorkspaceID).
		SetSortOrder(sortOrderDesc).
		SetLimit(int32(limit)).
		SetOffset(int32(params.Offset))
	if params.Search != "" {
		req.SetSearch(params.Search)
	}
	resp, err := c.aidap.DescribeBranchesWithContext(ctx, req)
	if err != nil {
		return DescribeBranchesResult{}, fmt.Errorf("failed to describe branches: %w", err)
	}
	result := DescribeBranchesResult{
		Total:         intValue(resp.Total),
		WorkspaceName: stringValue(resp.WorkspaceName),
	}
	for _, branch := range resp.Branches {
		result.Branches = append(result.Branches, mapBranchFromDescribeBranches(branch))
	}
	return result, nil
}

func (c *Client) DescribeAllBranches(ctx context.Context, params DescribeBranchesParams) (DescribeBranchesResult, error) {
	limit := params.Limit
	if limit == 0 || limit > maxWorkspaceLimit {
		limit = maxWorkspaceLimit
	}
	params.Limit = limit

	var result DescribeBranchesResult
	for {
		page, err := c.DescribeBranches(ctx, params)
		if err != nil {
			return DescribeBranchesResult{}, err
		}
		result.Total = page.Total
		if result.WorkspaceName == "" {
			result.WorkspaceName = page.WorkspaceName
		}
		result.Branches = append(result.Branches, page.Branches...)
		if len(result.Branches) >= page.Total || len(page.Branches) == 0 {
			return result, nil
		}
		params.Offset += limit
	}
}

func (c *Client) DescribeChildBranches(ctx context.Context, params DescribeChildBranchesParams) (DescribeBranchesResult, error) {
	limit := params.Limit
	if limit == 0 {
		limit = maxWorkspaceLimit
	}
	req := (&aidap.DescribeChildBranchesInput{}).
		SetWorkspaceId(params.WorkspaceID).
		SetParentBranchId(params.ParentID).
		SetSortOrder(sortOrderDesc).
		SetLimit(int32(limit)).
		SetOffset(int32(params.Offset))
	resp, err := c.aidap.DescribeChildBranchesWithContext(ctx, req)
	if err != nil {
		return DescribeBranchesResult{}, fmt.Errorf("failed to describe child branches: %w", err)
	}
	result := DescribeBranchesResult{
		Total:         intValue(resp.Total),
		WorkspaceName: stringValue(resp.WorkspaceName),
	}
	for _, branch := range resp.Branches {
		result.Branches = append(result.Branches, mapBranchFromDescribeChildBranches(branch))
	}
	return result, nil
}

func (c *Client) DescribeBranchDetail(ctx context.Context, workspaceID, branchID string) (DescribeBranchDetailResult, error) {
	req := (&aidap.DescribeBranchDetailInput{}).SetWorkspaceId(workspaceID).SetBranchId(branchID)
	resp, err := c.aidap.DescribeBranchDetailWithContext(ctx, req)
	if err != nil {
		return DescribeBranchDetailResult{}, fmt.Errorf("failed to describe branch detail: %w", err)
	}
	return DescribeBranchDetailResult{
		WorkspaceName: stringValue(resp.WorkspaceName),
		Branch:        mapBranchDetailFromDescribeBranchDetail(resp.Branch),
	}, nil
}

func (c *Client) DescribeDefaultBranch(ctx context.Context, workspaceID string) (DescribeDefaultBranchResult, error) {
	req := (&aidap.DescribeDefaultBranchInput{}).SetWorkspaceId(workspaceID)
	resp, err := c.aidap.DescribeDefaultBranchWithContext(ctx, req)
	if err != nil {
		return DescribeDefaultBranchResult{}, fmt.Errorf("failed to describe default branch: %w", err)
	}
	return DescribeDefaultBranchResult{Branch: mapBranchFromDescribeDefaultBranch(resp.Branch)}, nil
}

// ResolveDefaultBranchID returns branchID unchanged when non-empty, otherwise it
// resolves the workspace's default branch id.
func (c *Client) ResolveDefaultBranchID(ctx context.Context, workspaceID, branchID string) (string, error) {
	branchID = strings.TrimSpace(branchID)
	if branchID != "" {
		return branchID, nil
	}
	defaultBranch, err := c.DescribeDefaultBranch(ctx, workspaceID)
	if err != nil {
		return "", err
	}
	branchID = strings.TrimSpace(defaultBranch.Branch.BranchID)
	if branchID == "" {
		return "", fmt.Errorf("failed to resolve default branch for workspace %s", workspaceID)
	}
	return branchID, nil
}

func (c *Client) CreateBranch(ctx context.Context, params CreateBranchParams) (CreateBranchResult, error) {
	settings := (&aidap.BranchSettingsForCreateBranchInput{}).
		SetName(params.Name).
		SetInitSource(aidap.EnumOfInitSourceForCreateBranchInputParentData)
	if params.ParentID != "" {
		settings.SetParentId(params.ParentID)
	}
	if params.ParentTime != "" {
		settings.SetParentTime(params.ParentTime)
	}
	req := (&aidap.CreateBranchInput{}).
		SetWorkspaceId(params.WorkspaceID).
		SetBranchSettings(settings)
	resp, err := c.aidap.CreateBranchWithContext(ctx, req)
	if err != nil {
		return CreateBranchResult{}, fmt.Errorf("failed to create branch: %w", err)
	}
	return CreateBranchResult{
		WorkspaceID: stringValue(resp.WorkspaceId),
		BranchID:    stringValue(resp.BranchId),
		Branch:      mapBranchDetailFromCreateBranch(resp.Branch),
	}, nil
}

func (c *Client) UpdateBranch(ctx context.Context, params UpdateBranchParams) (UpdateBranchResult, error) {
	req := (&aidap.UpdateBranchInput{}).
		SetWorkspaceId(params.WorkspaceID).
		SetBranchId(params.BranchID)
	if params.Name != nil {
		req.SetName(*params.Name)
	}
	if params.Protected != nil {
		req.SetProtected(*params.Protected)
	}
	resp, err := c.aidap.UpdateBranchWithContext(ctx, req)
	if err != nil {
		return UpdateBranchResult{}, fmt.Errorf("failed to update branch: %w", err)
	}
	return UpdateBranchResult{Branch: mapBranchDetailFromUpdateBranch(resp.Branch)}, nil
}

func (c *Client) DeleteBranch(ctx context.Context, workspaceID, branchID string) (DeleteBranchResult, error) {
	req := (&aidap.DeleteBranchInput{}).SetWorkspaceId(workspaceID).SetBranchId(branchID)
	resp, err := c.aidap.DeleteBranchWithContext(ctx, req)
	if err != nil {
		return DeleteBranchResult{}, fmt.Errorf("failed to delete branch: %w", err)
	}
	return DeleteBranchResult{
		WorkspaceID: stringValue(resp.WorkspaceId),
		BranchID:    stringValue(resp.BranchId),
	}, nil
}

func (c *Client) RestartBranch(ctx context.Context, params RestartBranchParams) (RestartBranchResult, error) {
	req := (&aidap.RestartBranchInput{}).
		SetWorkspaceId(params.WorkspaceID).
		SetBranchId(params.BranchID)
	if len(params.ComputeIDs) > 0 {
		req.SetComputeIds(stringSlicePointers(params.ComputeIDs))
	}
	if _, err := c.aidap.RestartBranchWithContext(ctx, req); err != nil {
		return RestartBranchResult{}, fmt.Errorf("failed to restart branch: %w", err)
	}
	return RestartBranchResult{
		WorkspaceID: params.WorkspaceID,
		BranchID:    params.BranchID,
		ComputeIDs:  append([]string(nil), params.ComputeIDs...),
	}, nil
}

func (c *Client) SetAsDefaultBranch(ctx context.Context, workspaceID, branchID string) (SetAsDefaultBranchResult, error) {
	req := (&aidap.SetAsDefaultBranchInput{}).SetWorkspaceId(workspaceID).SetBranchId(branchID)
	resp, err := c.aidap.SetAsDefaultBranchWithContext(ctx, req)
	if err != nil {
		return SetAsDefaultBranchResult{}, fmt.Errorf("failed to set default branch: %w", err)
	}
	return SetAsDefaultBranchResult{Branch: mapBranchDetailFromSetAsDefaultBranch(resp.Branch)}, nil
}

func (c *Client) GetRestoreWindow(ctx context.Context, workspaceID, branchID string) (RestoreWindow, error) {
	req := (&aidap.GetRestoreWindowInput{}).SetWorkspaceId(workspaceID).SetBranchId(branchID)
	resp, err := c.aidap.GetRestoreWindowWithContext(ctx, req)
	if err != nil {
		return RestoreWindow{}, fmt.Errorf("failed to get restore window: %w", err)
	}
	return RestoreWindow{
		WorkspaceID:        stringValue(resp.WorkspaceId),
		BranchID:           stringValue(resp.BranchId),
		WindowSizeSeconds:  int64Value(resp.WindowSizeSeconds),
		BranchCreationTime: stringValue(resp.BranchCreateTime),
		StartTime:          stringValue(resp.StartTime),
		EndTime:            stringValue(resp.EndTime),
	}, nil
}

func (c *Client) DescribeRestorableBranches(ctx context.Context, params DescribeRestorableBranchesParams) (DescribeRestorableBranchesResult, error) {
	limit := params.Limit
	if limit == 0 {
		limit = maxWorkspaceLimit
	}
	req := (&aidap.DescribeRestorableBranchesInput{}).
		SetWorkspaceId(params.WorkspaceID).
		SetTime(params.Time).
		SetLimit(int32(limit)).
		SetOffset(int32(params.Offset))
	if params.Search != "" {
		req.SetSearch(params.Search)
	}
	resp, err := c.aidap.DescribeRestorableBranchesWithContext(ctx, req)
	if err != nil {
		return DescribeRestorableBranchesResult{}, fmt.Errorf("failed to describe restorable branches: %w", err)
	}
	result := DescribeRestorableBranchesResult{
		Total:         intValue(resp.Total),
		WorkspaceName: stringValue(resp.WorkspaceName),
	}
	for _, branch := range resp.Branches {
		result.Branches = append(result.Branches, mapBranchFromDescribeRestorableBranches(branch))
	}
	for _, window := range resp.RestoreWindow {
		if window == nil {
			continue
		}
		result.RestoreWindows = append(result.RestoreWindows, RestoreWindow{
			BranchID:           stringValue(window.BranchId),
			WindowSizeSeconds:  int64Value(window.WindowSizeSeconds),
			BranchCreationTime: stringValue(window.BranchCreateTime),
			StartTime:          stringValue(window.StartTime),
			EndTime:            stringValue(window.EndTime),
		})
	}
	return result, nil
}

func (c *Client) BranchRestore(ctx context.Context, params BranchRestoreParams) (BranchRestoreResult, error) {
	settings := &aidap.RestoreSettingsForBranchRestoreInput{}
	if params.Time != "" {
		settings.SetTime(params.Time)
	}
	if params.SourceBranchID != "" {
		settings.SetSourceBranchId(params.SourceBranchID)
	}
	req := (&aidap.BranchRestoreInput{}).
		SetWorkspaceId(params.WorkspaceID).
		SetBranchId(params.BranchID).
		SetRestoreSettings(settings)
	resp, err := c.aidap.BranchRestoreWithContext(ctx, req)
	if err != nil {
		return BranchRestoreResult{}, fmt.Errorf("failed to restore branch: %w", err)
	}
	return BranchRestoreResult{
		WorkspaceID:    params.WorkspaceID,
		BranchID:       params.BranchID,
		SourceBranchID: params.SourceBranchID,
		Time:           params.Time,
		BackupBranchID: stringValue(resp.BackupBranchID),
	}, nil
}

func mapBranchFromDescribeBranches(branch *aidap.BranchForDescribeBranchesOutput) Branch {
	if branch == nil {
		return Branch{}
	}
	return Branch{
		WorkspaceID:  stringValue(branch.WorkspaceId),
		BranchID:     stringValue(branch.BranchId),
		BranchName:   stringValue(branch.BranchName),
		BranchStatus: stringValue(branch.BranchStatus),
		Default:      boolValue(branch.Default),
		Protected:    boolValue(branch.Protected),
		Archived:     boolValue(branch.Archived),
		InitSource:   stringValue(branch.InitSource),
		CreateTime:   stringValue(branch.CreateTime),
		UpdateTime:   stringValue(branch.UpdateTime),
	}
}

func mapBranchFromDescribeChildBranches(branch *aidap.BranchForDescribeChildBranchesOutput) Branch {
	if branch == nil {
		return Branch{}
	}
	return Branch{
		WorkspaceID:  stringValue(branch.WorkspaceId),
		BranchID:     stringValue(branch.BranchId),
		BranchName:   stringValue(branch.BranchName),
		BranchStatus: stringValue(branch.BranchStatus),
		Default:      boolValue(branch.Default),
		Protected:    boolValue(branch.Protected),
		Archived:     boolValue(branch.Archived),
		InitSource:   stringValue(branch.InitSource),
		CreateTime:   stringValue(branch.CreateTime),
		UpdateTime:   stringValue(branch.UpdateTime),
	}
}

func mapBranchFromDescribeDefaultBranch(branch *aidap.BranchForDescribeDefaultBranchOutput) Branch {
	if branch == nil {
		return Branch{}
	}
	return Branch{
		WorkspaceID:  stringValue(branch.WorkspaceId),
		BranchID:     stringValue(branch.BranchId),
		BranchName:   stringValue(branch.BranchName),
		BranchStatus: stringValue(branch.BranchStatus),
		Default:      boolValue(branch.Default),
		Protected:    boolValue(branch.Protected),
		Archived:     boolValue(branch.Archived),
		InitSource:   stringValue(branch.InitSource),
		CreateTime:   stringValue(branch.CreateTime),
		UpdateTime:   stringValue(branch.UpdateTime),
	}
}

func mapBranchFromDescribeRestorableBranches(branch *aidap.BranchForDescribeRestorableBranchesOutput) Branch {
	if branch == nil {
		return Branch{}
	}
	return Branch{
		WorkspaceID:  stringValue(branch.WorkspaceId),
		BranchID:     stringValue(branch.BranchId),
		BranchName:   stringValue(branch.BranchName),
		BranchStatus: stringValue(branch.BranchStatus),
		Default:      boolValue(branch.Default),
		Protected:    boolValue(branch.Protected),
		Archived:     boolValue(branch.Archived),
		InitSource:   stringValue(branch.InitSource),
		CreateTime:   stringValue(branch.CreateTime),
		UpdateTime:   stringValue(branch.UpdateTime),
	}
}

func mapBranchDetailFromDescribeBranchDetail(branch *aidap.BranchForDescribeBranchDetailOutput) BranchDetail {
	if branch == nil {
		return BranchDetail{}
	}
	detail := BranchDetail{
		WorkspaceID:       stringValue(branch.WorkspaceId),
		BranchID:          stringValue(branch.BranchId),
		BranchName:        stringValue(branch.BranchName),
		BranchStatus:      stringValue(branch.BranchStatus),
		Default:           boolValue(branch.Default),
		Protected:         boolValue(branch.Protected),
		Archived:          boolValue(branch.Archived),
		InitSource:        stringValue(branch.InitSource),
		CreationSource:    stringValue(branch.CreationSource),
		CreateTime:        stringValue(branch.CreateTime),
		UpdateTime:        stringValue(branch.UpdateTime),
		LastResetTime:     stringValue(branch.LastResetTime),
		StartParentLSN:    stringValue(branch.StartParentLSN),
		StartParentTime:   stringValue(branch.StartParentTime),
		StatusChangedTime: stringValue(branch.StatusChangedTime),
	}
	if branch.ParentBranch != nil {
		detail.ParentBranch = ParentBranch{
			WorkspaceID:  stringValue(branch.ParentBranch.WorkspaceId),
			BranchID:     stringValue(branch.ParentBranch.BranchId),
			BranchName:   stringValue(branch.ParentBranch.BranchName),
			BranchStatus: stringValue(branch.ParentBranch.BranchStatus),
		}
	}
	if branch.BranchUsage != nil {
		detail.BranchUsage = BranchUsage{
			WorkspaceID:          stringValue(branch.BranchUsage.WorkspaceId),
			BranchID:             stringValue(branch.BranchUsage.BranchId),
			ComputeTimeSeconds:   int64Value(branch.BranchUsage.ComputeTimeSeconds),
			DataSizeTotalBytes:   int64Value(branch.BranchUsage.DataSizeTotalBytes),
			DataSizeUsedBytes:    int64Value(branch.BranchUsage.DataSizeUsedBytes),
			FunctionCallNum:      int64Value(branch.BranchUsage.FunctionCallNum),
			LastRunningTime:      stringValue(branch.BranchUsage.LastRunningTime),
			ServiceTimeSeconds:   int64Value(branch.BranchUsage.ServiceTimeSeconds),
			StatTime:             stringValue(branch.BranchUsage.StatTime),
			StorageSizeUsedBytes: int64Value(branch.BranchUsage.StorageSizeUsedBytes),
		}
	}
	return detail
}

func mapBranchDetailFromCreateBranch(branch *aidap.BranchForCreateBranchOutput) BranchDetail {
	if branch == nil {
		return BranchDetail{}
	}
	detail := BranchDetail{
		WorkspaceID:       stringValue(branch.WorkspaceId),
		BranchID:          stringValue(branch.BranchId),
		BranchName:        stringValue(branch.BranchName),
		BranchStatus:      stringValue(branch.BranchStatus),
		Default:           boolValue(branch.Default),
		Protected:         boolValue(branch.Protected),
		Archived:          boolValue(branch.Archived),
		InitSource:        stringValue(branch.InitSource),
		CreationSource:    stringValue(branch.CreationSource),
		CreateTime:        stringValue(branch.CreateTime),
		UpdateTime:        stringValue(branch.UpdateTime),
		LastResetTime:     stringValue(branch.LastResetTime),
		StartParentLSN:    stringValue(branch.StartParentLSN),
		StartParentTime:   stringValue(branch.StartParentTime),
		StatusChangedTime: stringValue(branch.StatusChangedTime),
	}
	if branch.ParentBranch != nil {
		detail.ParentBranch = ParentBranch{
			WorkspaceID:  stringValue(branch.ParentBranch.WorkspaceId),
			BranchID:     stringValue(branch.ParentBranch.BranchId),
			BranchName:   stringValue(branch.ParentBranch.BranchName),
			BranchStatus: stringValue(branch.ParentBranch.BranchStatus),
		}
	}
	if branch.BranchUsage != nil {
		detail.BranchUsage = BranchUsage{
			WorkspaceID:          stringValue(branch.BranchUsage.WorkspaceId),
			BranchID:             stringValue(branch.BranchUsage.BranchId),
			ComputeTimeSeconds:   int64Value(branch.BranchUsage.ComputeTimeSeconds),
			DataSizeTotalBytes:   int64Value(branch.BranchUsage.DataSizeTotalBytes),
			DataSizeUsedBytes:    int64Value(branch.BranchUsage.DataSizeUsedBytes),
			FunctionCallNum:      int64Value(branch.BranchUsage.FunctionCallNum),
			LastRunningTime:      stringValue(branch.BranchUsage.LastRunningTime),
			ServiceTimeSeconds:   int64Value(branch.BranchUsage.ServiceTimeSeconds),
			StatTime:             stringValue(branch.BranchUsage.StatTime),
			StorageSizeUsedBytes: int64Value(branch.BranchUsage.StorageSizeUsedBytes),
		}
	}
	return detail
}

func mapBranchDetailFromUpdateBranch(branch *aidap.BranchForUpdateBranchOutput) BranchDetail {
	if branch == nil {
		return BranchDetail{}
	}
	detail := BranchDetail{
		WorkspaceID:       stringValue(branch.WorkspaceId),
		BranchID:          stringValue(branch.BranchId),
		BranchName:        stringValue(branch.BranchName),
		BranchStatus:      stringValue(branch.BranchStatus),
		Default:           boolValue(branch.Default),
		Protected:         boolValue(branch.Protected),
		Archived:          boolValue(branch.Archived),
		InitSource:        stringValue(branch.InitSource),
		CreationSource:    stringValue(branch.CreationSource),
		CreateTime:        stringValue(branch.CreateTime),
		UpdateTime:        stringValue(branch.UpdateTime),
		LastResetTime:     stringValue(branch.LastResetTime),
		StartParentLSN:    stringValue(branch.StartParentLSN),
		StartParentTime:   stringValue(branch.StartParentTime),
		StatusChangedTime: stringValue(branch.StatusChangedTime),
	}
	if branch.ParentBranch != nil {
		detail.ParentBranch = ParentBranch{
			WorkspaceID:  stringValue(branch.ParentBranch.WorkspaceId),
			BranchID:     stringValue(branch.ParentBranch.BranchId),
			BranchName:   stringValue(branch.ParentBranch.BranchName),
			BranchStatus: stringValue(branch.ParentBranch.BranchStatus),
		}
	}
	if branch.BranchUsage != nil {
		detail.BranchUsage = BranchUsage{
			WorkspaceID:          stringValue(branch.BranchUsage.WorkspaceId),
			BranchID:             stringValue(branch.BranchUsage.BranchId),
			ComputeTimeSeconds:   int64Value(branch.BranchUsage.ComputeTimeSeconds),
			DataSizeTotalBytes:   int64Value(branch.BranchUsage.DataSizeTotalBytes),
			DataSizeUsedBytes:    int64Value(branch.BranchUsage.DataSizeUsedBytes),
			FunctionCallNum:      int64Value(branch.BranchUsage.FunctionCallNum),
			LastRunningTime:      stringValue(branch.BranchUsage.LastRunningTime),
			ServiceTimeSeconds:   int64Value(branch.BranchUsage.ServiceTimeSeconds),
			StatTime:             stringValue(branch.BranchUsage.StatTime),
			StorageSizeUsedBytes: int64Value(branch.BranchUsage.StorageSizeUsedBytes),
		}
	}
	return detail
}

func mapBranchDetailFromSetAsDefaultBranch(branch *aidap.BranchForSetAsDefaultBranchOutput) BranchDetail {
	if branch == nil {
		return BranchDetail{}
	}
	detail := BranchDetail{
		WorkspaceID:       stringValue(branch.WorkspaceId),
		BranchID:          stringValue(branch.BranchId),
		BranchName:        stringValue(branch.BranchName),
		BranchStatus:      stringValue(branch.BranchStatus),
		Default:           boolValue(branch.Default),
		Protected:         boolValue(branch.Protected),
		Archived:          boolValue(branch.Archived),
		InitSource:        stringValue(branch.InitSource),
		CreationSource:    stringValue(branch.CreationSource),
		CreateTime:        stringValue(branch.CreateTime),
		UpdateTime:        stringValue(branch.UpdateTime),
		LastResetTime:     stringValue(branch.LastResetTime),
		StartParentLSN:    stringValue(branch.StartParentLSN),
		StartParentTime:   stringValue(branch.StartParentTime),
		StatusChangedTime: stringValue(branch.StatusChangedTime),
	}
	if branch.ParentBranch != nil {
		detail.ParentBranch = ParentBranch{
			WorkspaceID:  stringValue(branch.ParentBranch.WorkspaceId),
			BranchID:     stringValue(branch.ParentBranch.BranchId),
			BranchName:   stringValue(branch.ParentBranch.BranchName),
			BranchStatus: stringValue(branch.ParentBranch.BranchStatus),
		}
	}
	if branch.BranchUsage != nil {
		detail.BranchUsage = BranchUsage{
			WorkspaceID:          stringValue(branch.BranchUsage.WorkspaceId),
			BranchID:             stringValue(branch.BranchUsage.BranchId),
			ComputeTimeSeconds:   int64Value(branch.BranchUsage.ComputeTimeSeconds),
			DataSizeTotalBytes:   int64Value(branch.BranchUsage.DataSizeTotalBytes),
			DataSizeUsedBytes:    int64Value(branch.BranchUsage.DataSizeUsedBytes),
			FunctionCallNum:      int64Value(branch.BranchUsage.FunctionCallNum),
			LastRunningTime:      stringValue(branch.BranchUsage.LastRunningTime),
			ServiceTimeSeconds:   int64Value(branch.BranchUsage.ServiceTimeSeconds),
			StatTime:             stringValue(branch.BranchUsage.StatTime),
			StorageSizeUsedBytes: int64Value(branch.BranchUsage.StorageSizeUsedBytes),
		}
	}
	return detail
}
