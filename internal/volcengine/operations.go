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

type Operation struct {
	OperationID  string `json:"OperationId"`
	WorkspaceID  string `json:"WorkspaceId"`
	BranchID     string `json:"BranchId"`
	ComputeID    string `json:"ComputeId"`
	ActionName   string `json:"ActionName"`
	ActionStatus string `json:"ActionStatus"`
	CreateTime   string `json:"CreateTime"`
	FinishTime   string `json:"FinishTime,omitempty"`
	DurationTime string `json:"DurationTime,omitempty"`
}

type DescribeOperationsParams struct {
	WorkspaceID     string
	BranchID        string
	ComputeID       string
	ActionName      string
	Status          string
	CreateTimeStart string
	CreateTimeEnd   string
	Limit           int
	Offset          int
}

type DescribeOperationsResult struct {
	Total      int         `json:"Total"`
	Operations []Operation `json:"Operations"`
}

func (c *Client) DescribeOperations(ctx context.Context, params DescribeOperationsParams) (DescribeOperationsResult, error) {
	limit := params.Limit
	if limit == 0 {
		limit = DefaultListLimit
	}
	req := (&aidap.DescribeOperationsInput{}).
		SetLimit(int32(limit)).
		SetOffset(int32(params.Offset))
	if params.CreateTimeStart != "" {
		req.SetCreateTimeStart(params.CreateTimeStart)
	}
	if params.CreateTimeEnd != "" {
		req.SetCreateTimeEnd(params.CreateTimeEnd)
	}
	filters := make([]*aidap.FilterForDescribeOperationsInput, 0, 5)
	for _, filter := range []struct {
		name  string
		value string
	}{
		{name: "WorkspaceId", value: params.WorkspaceID},
		{name: "BranchId", value: params.BranchID},
		{name: "ComputeId", value: params.ComputeID},
		{name: "ActionName", value: params.ActionName},
		{name: "Status", value: params.Status},
	} {
		if filter.value != "" {
			filters = append(filters, (&aidap.FilterForDescribeOperationsInput{}).SetName(filter.name).SetValue(filter.value))
		}
	}
	if len(filters) > 0 {
		req.SetFilters(filters)
	}
	resp, err := c.aidap.DescribeOperationsWithContext(ctx, req)
	if err != nil {
		return DescribeOperationsResult{}, fmt.Errorf("failed to describe operations: %w", err)
	}
	return mapDescribeOperationsResult(resp), nil
}

func mapDescribeOperationsResult(resp *aidap.DescribeOperationsOutput) DescribeOperationsResult {
	if resp == nil {
		return DescribeOperationsResult{}
	}
	result := DescribeOperationsResult{Total: intValue(resp.Total)}
	for _, operation := range resp.Operations {
		if operation == nil {
			continue
		}
		result.Operations = append(result.Operations, Operation{
			OperationID:  stringValue(operation.OperationId),
			WorkspaceID:  stringValue(operation.WorkspaceId),
			BranchID:     stringValue(operation.BranchId),
			ComputeID:    stringValue(operation.ComputeId),
			ActionName:   stringValue(operation.ActionName),
			ActionStatus: stringValue(operation.ActionStatus),
			CreateTime:   stringValue(operation.CreateTime),
			FinishTime:   stringValue(operation.FinishTime),
			DurationTime: stringValue(operation.DurationTime),
		})
	}
	return result
}
