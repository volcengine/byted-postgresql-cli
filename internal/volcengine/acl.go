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
	"errors"
	"fmt"
	"net/http"

	"github.com/volcengine/volcengine-go-sdk/service/aidap"
)

// AccessControlList is a workspace-level network allow-list. It is the
// PostgreSQL engine's IP allow-list mechanism (there is no per-project
// AllowedIPs setting on Volcengine AIDAP).
type AccessControlList struct {
	Name    string   `json:"Name"`
	AclType string   `json:"AclType"`
	IPList  []string `json:"IPList"`
}

type AccessControlListParams struct {
	WorkspaceID string
	Name        string
	AclType     string
	IPList      []string
}

type ModifyAccessControlListParams struct {
	WorkspaceID string
	Name        string
	ModifyMode  string
	IPList      []string
}

type DescribeAccessControlListResult struct {
	Total              int                 `json:"Total"`
	AccessControlLists []AccessControlList `json:"AccessControlLists"`
}

func (c *Client) CreateAccessControlList(ctx context.Context, params AccessControlListParams) error {
	aclType := params.AclType
	if aclType == "" {
		aclType = ACLTypeAllow
	}
	req := (&aidap.CreateAccessControlListInput{}).
		SetWorkspaceId(params.WorkspaceID).
		SetAccessControlListName(params.Name).
		SetAclType(aclType).
		SetIPList(stringSlicePointers(params.IPList))
	if _, err := c.aidap.CreateAccessControlListWithContext(ctx, req); err != nil {
		return fmt.Errorf("failed to create access control list: %w", err)
	}
	return nil
}

func (c *Client) DescribeAccessControlList(ctx context.Context, workspaceID, name string) (DescribeAccessControlListResult, error) {
	req := (&aidap.DescribeAccessControlListInput{}).SetWorkspaceId(workspaceID)
	if name != "" {
		req.SetAccessControlListName(name)
	}
	resp, err := c.aidap.DescribeAccessControlListWithContext(ctx, req)
	if err != nil {
		// A workspace that has never had an allow-list returns 404 for the named
		// list. That is not an error for callers: an absent list means "allow
		// all", and `set` treats Total == 0 as "create on first write". Represent
		// it as an empty result so both read and write paths behave sanely.
		var reqFailure interface{ StatusCode() int }
		if errors.As(err, &reqFailure) && reqFailure.StatusCode() == http.StatusNotFound {
			return DescribeAccessControlListResult{}, nil
		}
		return DescribeAccessControlListResult{}, fmt.Errorf("failed to describe access control list: %w", err)
	}
	result := DescribeAccessControlListResult{Total: intValue(resp.Total)}
	for _, data := range resp.Datas {
		if data == nil {
			continue
		}
		result.AccessControlLists = append(result.AccessControlLists, AccessControlList{
			Name:    stringValue(data.AccessControlListName),
			AclType: stringValue(data.AclType),
			IPList:  stringPointersValue(data.IPList),
		})
	}
	return result, nil
}

func (c *Client) ModifyAccessControlList(ctx context.Context, params ModifyAccessControlListParams) error {
	req := (&aidap.ModifyAccessControlListInput{}).
		SetWorkspaceId(params.WorkspaceID).
		SetAccessControlListName(params.Name).
		SetModifyMode(params.ModifyMode).
		SetIPList(stringSlicePointers(params.IPList))
	if _, err := c.aidap.ModifyAccessControlListWithContext(ctx, req); err != nil {
		return fmt.Errorf("failed to modify access control list: %w", err)
	}
	return nil
}

func (c *Client) DeleteAccessControlList(ctx context.Context, workspaceID, name string) error {
	req := (&aidap.DeleteAccessControlListInput{}).
		SetWorkspaceId(workspaceID).
		SetAccessControlListName(name)
	if _, err := c.aidap.DeleteAccessControlListWithContext(ctx, req); err != nil {
		return fmt.Errorf("failed to delete access control list: %w", err)
	}
	return nil
}
