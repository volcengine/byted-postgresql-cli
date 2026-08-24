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

func (c *Client) DescribeBranchSchemaNames(ctx context.Context, workspaceID, branchID, database string) ([]string, error) {
	resp, err := c.aidap.DescribeBranchSchemaNamesWithContext(ctx, (&aidap.DescribeBranchSchemaNamesInput{}).
		SetWorkspaceId(workspaceID).SetBranchId(branchID).SetDatabase(database))
	if err != nil {
		return nil, fmt.Errorf("failed to describe branch schemas: %w", err)
	}
	var names []string
	if resp != nil {
		for _, name := range resp.SchemaNames {
			if name != nil && *name != "" {
				names = append(names, *name)
			}
		}
	}
	return names, nil
}

type CreateSchemaDiffParams struct {
	WorkspaceID        string
	SourceBranchID     string
	SourceDatabaseName string
	SourceSchemaName   string
	TargetBranchID     string
	TargetDatabaseName string
	TargetSchemaName   string
	ForceReCompare     bool
}

type SchemaDiffJobStatus struct {
	CreatedAt    string `json:"CreatedAt"`
	ErrorMessage string `json:"ErrorMessage,omitempty"`
	Exists       bool   `json:"Exists"`
	FinishedAt   string `json:"FinishedAt,omitempty"`
	Status       string `json:"Status"`
}

type SchemaDiffMeta struct {
	JobBeginTime string `json:"JobBeginTime,omitempty"`
	JobEndTime   string `json:"JobEndTime,omitempty"`
	Status       string `json:"Status"`
	Total        int    `json:"Total"`
}

type SchemaDiffResult struct {
	CompareResultSQL       string `json:"CompareResultSQL,omitempty"`
	SourceObjectDefinition string `json:"SourceObjectDefinition,omitempty"`
	TargetObjectDefinition string `json:"TargetObjectDefinition,omitempty"`
}

func (c *Client) CreateSchemaDiff(ctx context.Context, params CreateSchemaDiffParams) (string, error) {
	req := (&aidap.CreateSchemaDiffInput{}).
		SetWorkspaceId(params.WorkspaceID).
		SetSourceBranchId(params.SourceBranchID).
		SetSourceDatabaseName(params.SourceDatabaseName).
		SetSourceSchemaName(params.SourceSchemaName).
		SetTargetBranchId(params.TargetBranchID).
		SetTargetDatabaseName(params.TargetDatabaseName).
		SetTargetSchemaName(params.TargetSchemaName).
		SetForceReCompare(params.ForceReCompare)
	resp, err := c.aidap.CreateSchemaDiffWithContext(ctx, req)
	if err != nil {
		return "", fmt.Errorf("failed to create schema diff: %w", err)
	}
	if resp == nil || stringValue(resp.SchemaDiffJobId) == "" {
		return "", fmt.Errorf("create schema diff returned no job id")
	}
	return stringValue(resp.SchemaDiffJobId), nil
}

func (c *Client) DescribeSchemaDiffJobStatus(ctx context.Context, workspaceID, jobID string) (SchemaDiffJobStatus, error) {
	resp, err := c.aidap.DescribeSchemaDiffJobStatusWithContext(ctx, (&aidap.DescribeSchemaDiffJobStatusInput{}).
		SetWorkspaceId(workspaceID).SetSchemaDiffJobId(jobID))
	if err != nil {
		return SchemaDiffJobStatus{}, fmt.Errorf("failed to describe schema diff status: %w", err)
	}
	return SchemaDiffJobStatus{
		CreatedAt: stringValue(resp.CreatedAt), ErrorMessage: stringValue(resp.ErrorMessage),
		Exists: boolValue(resp.Exists), FinishedAt: stringValue(resp.FinishedAt), Status: stringValue(resp.Status),
	}, nil
}

func (c *Client) DescribeSchemaDiffResultSQLAll(ctx context.Context, workspaceID, jobID string) (string, error) {
	resp, err := c.aidap.DescribeSchemaDiffResultSQLTextAllWithContext(ctx, (&aidap.DescribeSchemaDiffResultSQLTextAllInput{}).
		SetWorkspaceId(workspaceID).SetSchemaDiffJobId(jobID))
	if err != nil {
		return "", fmt.Errorf("failed to describe schema diff SQL: %w", err)
	}
	return stringValue(resp.MigrationSQL), nil
}

func (c *Client) DescribeSchemaDiffResultMeta(ctx context.Context, workspaceID, jobID string) (SchemaDiffMeta, error) {
	resp, err := c.aidap.DescribeSchemaDiffResultMetaWithContext(ctx, (&aidap.DescribeSchemaDiffResultMetaInput{}).
		SetWorkspaceId(workspaceID).SetSchemaDiffJobId(jobID))
	if err != nil {
		return SchemaDiffMeta{}, fmt.Errorf("failed to describe schema diff metadata: %w", err)
	}
	return SchemaDiffMeta{
		JobBeginTime: stringValue(resp.JobBeginTime),
		JobEndTime:   stringValue(resp.JobEndTime),
		Status:       stringValue(resp.Status),
		Total:        int(intValue(resp.Total)),
	}, nil
}

func (c *Client) GetSchemaDiffDownloadLink(ctx context.Context, workspaceID, jobID string) (string, error) {
	resp, err := c.aidap.GetSchemaDiffResultDownloadLinkWithContext(ctx, (&aidap.GetSchemaDiffResultDownloadLinkInput{}).
		SetWorkspaceId(workspaceID).SetSchemaDiffJobId(jobID))
	if err != nil {
		return "", fmt.Errorf("failed to get schema diff download link: %w", err)
	}
	return stringValue(resp.DownloadLink), nil
}
