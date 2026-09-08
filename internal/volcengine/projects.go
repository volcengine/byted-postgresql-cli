// Copyright (c) 2026 Beijing Volcano Engine Technology Co., Ltd.
// SPDX-License-Identifier: MIT

package volcengine

import (
	"context"
	"fmt"

	"github.com/volcengine/volcengine-go-sdk/service/iam20210801"
)

type ResourceProject struct {
	AccountID         int64  `json:"AccountID"`
	ProjectName       string `json:"ProjectName"`
	ParentProjectName string `json:"ParentProjectName"`
	Path              string `json:"Path"`
	DisplayName       string `json:"DisplayName"`
	Description       string `json:"Description"`
	CreateDate        string `json:"CreateDate"`
	UpdateDate        string `json:"UpdateDate"`
	Status            string `json:"Status"`
	HasPermission     bool   `json:"HasPermission"`
}

type ListResourceProjectsResult struct {
	Total    int               `json:"Total"`
	Projects []ResourceProject `json:"Projects"`
}

func mapResourceProject(project *iam20210801.ProjectForListProjectsOutput) (ResourceProject, bool) {
	if project == nil || project.ProjectName == nil || *project.ProjectName == "" {
		return ResourceProject{}, false
	}
	result := ResourceProject{ProjectName: *project.ProjectName}
	if project.AccountID != nil {
		result.AccountID = *project.AccountID
	}
	if project.ParentProjectName != nil {
		result.ParentProjectName = *project.ParentProjectName
	}
	if project.Path != nil {
		result.Path = *project.Path
	}
	if project.DisplayName != nil {
		result.DisplayName = *project.DisplayName
	}
	if project.Description != nil {
		result.Description = *project.Description
	}
	if project.Status != nil {
		result.Status = *project.Status
	}
	if project.CreateDate != nil {
		result.CreateDate = *project.CreateDate
	}
	if project.UpdateDate != nil {
		result.UpdateDate = *project.UpdateDate
	}
	if project.HasPermission != nil {
		result.HasPermission = *project.HasPermission
	}
	return result, true
}

// ListResourceProjects lists IAM resource projects, not AIDAP workspaces.
func (c *Client) ListResourceProjects(ctx context.Context) (ListResourceProjectsResult, error) {
	const limit int32 = MaxPageLimit
	var result ListResourceProjectsResult
	for offset := int32(0); ; offset += limit {
		resp, err := c.iam.ListProjectsWithContext(ctx, (&iam20210801.ListProjectsInput{}).
			SetLimit(limit).
			SetOffset(offset))
		if err != nil {
			return ListResourceProjectsResult{}, fmt.Errorf("failed to list resource projects: %w", err)
		}
		if resp == nil {
			return ListResourceProjectsResult{}, fmt.Errorf("failed to list resource projects: empty response")
		}
		if resp.Total != nil {
			result.Total = int(*resp.Total)
		}
		for _, project := range resp.Projects {
			item, ok := mapResourceProject(project)
			if !ok {
				continue
			}
			result.Projects = append(result.Projects, item)
		}
		if len(resp.Projects) == 0 || int(offset)+len(resp.Projects) >= result.Total {
			return result, nil
		}
	}
}
