// Copyright (c) 2026 Beijing Volcano Engine Technology Co., Ltd.
// SPDX-License-Identifier: MIT

package volcengine

import (
	"testing"

	"github.com/volcengine/volcengine-go-sdk/service/iam20210801"
)

func TestMapResourceProject(t *testing.T) {
	name := "gongna"
	displayName := "Gongna"
	description := "resource project"
	status := "Active"
	parentProjectName := "parent"
	path := "/parent/gongna/"
	createDate := "20260810T074419Z"
	updateDate := "20260810T074419Z"
	project, ok := mapResourceProject((&iam20210801.ProjectForListProjectsOutput{}).
		SetAccountID(2100365180).
		SetProjectName(name).
		SetParentProjectName(parentProjectName).
		SetPath(path).
		SetDisplayName(displayName).
		SetDescription(description).
		SetStatus(status).
		SetCreateDate(createDate).
		SetUpdateDate(updateDate).
		SetHasPermission(true))
	if !ok {
		t.Fatal("mapResourceProject rejected a valid project")
	}
	if project.AccountID != 2100365180 || project.ProjectName != name ||
		project.ParentProjectName != parentProjectName || project.Path != path ||
		project.DisplayName != displayName || project.Description != description ||
		project.Status != status || project.CreateDate != createDate ||
		project.UpdateDate != updateDate || !project.HasPermission {
		t.Fatalf("mapped project = %+v", project)
	}
}

func TestMapResourceProjectSkipsMissingName(t *testing.T) {
	if _, ok := mapResourceProject(&iam20210801.ProjectForListProjectsOutput{}); ok {
		t.Fatal("mapResourceProject accepted a project without a name")
	}
}
