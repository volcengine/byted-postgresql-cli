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

// Database is a logical database inside a branch.
type Database struct {
	WorkspaceID   string
	BranchID      string
	DatabaseName  string
	DatabaseOwner string
	DatabaseDesc  string
	CreateTime    string
	UpdateTime    string
}

type DescribeDatabasesParams struct {
	WorkspaceID string
	BranchID    string
	Search      string
	Limit       int
	Offset      int
}

type DescribeDatabasesResult struct {
	Databases []Database
	Total     int
}

type CreateDatabaseParams struct {
	WorkspaceID  string
	BranchID     string
	DatabaseName string
	Owner        string
	Description  string
}

func (c *Client) DescribeDatabases(ctx context.Context, params DescribeDatabasesParams) (DescribeDatabasesResult, error) {
	// The backend defaults to 10 per page when Limit is not supplied; pass
	// Limit/Offset explicitly and fall back to DefaultListLimit when the caller
	// omits count (Limit<=0) to stay consistent with that convention.
	limit := params.Limit
	if limit <= 0 {
		limit = DefaultListLimit
	}
	req := (&aidap.DescribeDatabasesInput{}).
		SetWorkspaceId(params.WorkspaceID).
		SetBranchId(params.BranchID).
		SetLimit(int32(limit)).
		SetOffset(int32(params.Offset))
	if params.Search != "" {
		req.SetSearch(params.Search)
	}
	resp, err := c.aidap.DescribeDatabasesWithContext(ctx, req)
	if err != nil {
		return DescribeDatabasesResult{}, fmt.Errorf("failed to describe databases: %w", err)
	}
	result := DescribeDatabasesResult{Total: intValue(resp.Total)}
	for _, db := range resp.Databases {
		if db == nil {
			continue
		}
		result.Databases = append(result.Databases, Database{
			WorkspaceID:   stringValue(db.WorkspaceId),
			BranchID:      stringValue(db.BranchId),
			DatabaseName:  stringValue(db.DatabaseName),
			DatabaseOwner: stringValue(db.DatabaseOwner),
			DatabaseDesc:  stringValue(db.DatabaseDesc),
			CreateTime:    stringValue(db.CreateTime),
			UpdateTime:    stringValue(db.UpdateTime),
		})
	}
	return result, nil
}

func (c *Client) CreateDatabase(ctx context.Context, params CreateDatabaseParams) (Database, error) {
	req := (&aidap.CreateDatabaseInput{}).
		SetWorkspaceId(params.WorkspaceID).
		SetBranchId(params.BranchID).
		SetDatabaseName(params.DatabaseName)
	if params.Owner != "" {
		req.SetDatabaseOwner(params.Owner)
	}
	if params.Description != "" {
		req.SetDatabaseDesc(params.Description)
	}
	resp, err := c.aidap.CreateDatabaseWithContext(ctx, req)
	if err != nil {
		return Database{}, fmt.Errorf("failed to create database: %w", err)
	}
	if resp.Database == nil || stringValue(resp.Database.DatabaseName) == "" {
		return Database{WorkspaceID: params.WorkspaceID, BranchID: params.BranchID, DatabaseName: params.DatabaseName}, nil
	}
	return Database{
		WorkspaceID:   stringValue(resp.Database.WorkspaceId),
		BranchID:      stringValue(resp.Database.BranchId),
		DatabaseName:  stringValue(resp.Database.DatabaseName),
		DatabaseOwner: stringValue(resp.Database.DatabaseOwner),
		DatabaseDesc:  stringValue(resp.Database.DatabaseDesc),
		CreateTime:    stringValue(resp.Database.CreateTime),
		UpdateTime:    stringValue(resp.Database.UpdateTime),
	}, nil
}

// DropDatabase deletes a logical database from a branch.
func (c *Client) DropDatabase(ctx context.Context, workspaceID, branchID, databaseName string) error {
	req := (&aidap.DropDatabaseInput{}).
		SetWorkspaceId(workspaceID).
		SetBranchId(branchID).
		SetDatabaseName(databaseName)
	if _, err := c.aidap.DropDatabaseWithContext(ctx, req); err != nil {
		return fmt.Errorf("failed to drop database: %w", err)
	}
	return nil
}

// DBAccount is a Postgres role/account inside a branch.
type DBAccount struct {
	WorkspaceID string
	BranchID    string
	AccountName string
	AccountDesc string
	CreateTime  string
	UpdateTime  string
}

type CreateDBAccountParams struct {
	WorkspaceID string
	BranchID    string
	AccountName string
	Password    string
	Description string
}

func (c *Client) CreateDBAccount(ctx context.Context, params CreateDBAccountParams) (DBAccount, error) {
	req := (&aidap.CreateDBAccountInput{}).
		SetWorkspaceId(params.WorkspaceID).
		SetBranchId(params.BranchID).
		SetAccountName(params.AccountName).
		SetAccountPassword(params.Password)
	if params.Description != "" {
		req.SetAccountDesc(params.Description)
	}
	resp, err := c.aidap.CreateDBAccountWithContext(ctx, req)
	if err != nil {
		return DBAccount{}, fmt.Errorf("failed to create db account: %w", err)
	}
	account := DBAccount{WorkspaceID: params.WorkspaceID, BranchID: params.BranchID, AccountName: params.AccountName, AccountDesc: params.Description}
	if resp != nil && resp.Account != nil {
		account.WorkspaceID = stringValue(resp.Account.WorkspaceId)
		account.BranchID = stringValue(resp.Account.BranchId)
		account.AccountName = stringValue(resp.Account.AccountName)
		account.AccountDesc = stringValue(resp.Account.AccountDesc)
		account.CreateTime = stringValue(resp.Account.CreateTime)
		account.UpdateTime = stringValue(resp.Account.UpdateTime)
	}
	return account, nil
}

func (c *Client) DeleteDBAccount(ctx context.Context, workspaceID, branchID, accountName string) error {
	req := (&aidap.DeleteDBAccountInput{}).
		SetWorkspaceId(workspaceID).
		SetBranchId(branchID).
		SetAccountName(accountName)
	resp, err := c.aidap.DeleteDBAccountWithContext(ctx, req)
	if err != nil {
		return fmt.Errorf("failed to delete db account: %w", err)
	}
	if resp != nil && !boolValue(resp.Success) {
		return fmt.Errorf("failed to delete db account: backend returned success=false")
	}
	return nil
}

func (c *Client) ResetDBAccountPassword(ctx context.Context, workspaceID, branchID, accountName, password string) error {
	req := (&aidap.ResetDBAccountPasswordInput{}).
		SetWorkspaceId(workspaceID).
		SetBranchId(branchID).
		SetAccountName(accountName).
		SetAccountPassword(password)
	resp, err := c.aidap.ResetDBAccountPasswordWithContext(ctx, req)
	if err != nil {
		return fmt.Errorf("failed to reset db account password: %w", err)
	}
	if resp != nil && !boolValue(resp.Success) {
		return fmt.Errorf("failed to reset db account password: backend returned success=false")
	}
	return nil
}

type DescribeDBAccountsParams struct {
	WorkspaceID string
	BranchID    string
	Search      string
	Limit       int
	Offset      int
}

type DescribeDBAccountsResult struct {
	Accounts []DBAccount
	Total    int
}

func (c *Client) DescribeDBAccounts(ctx context.Context, params DescribeDBAccountsParams) (DescribeDBAccountsResult, error) {
	limit := params.Limit
	if limit <= 0 {
		limit = DefaultListLimit
	}
	req := (&aidap.DescribeDBAccountsInput{}).
		SetWorkspaceId(params.WorkspaceID).
		SetBranchId(params.BranchID).
		SetLimit(int32(limit)).
		SetOffset(int32(params.Offset))
	if params.Search != "" {
		req.SetSearch(params.Search)
	}
	resp, err := c.aidap.DescribeDBAccountsWithContext(ctx, req)
	if err != nil {
		return DescribeDBAccountsResult{}, fmt.Errorf("failed to describe dbaccounts: %w", err)
	}
	result := DescribeDBAccountsResult{Total: intValue(resp.Total)}
	for _, acct := range resp.Accounts {
		if acct == nil {
			continue
		}
		result.Accounts = append(result.Accounts, DBAccount{
			WorkspaceID: stringValue(acct.WorkspaceId),
			BranchID:    stringValue(acct.BranchId),
			AccountName: stringValue(acct.AccountName),
			AccountDesc: stringValue(acct.AccountDesc),
			CreateTime:  stringValue(acct.CreateTime),
			UpdateTime:  stringValue(acct.UpdateTime),
		})
	}
	return result, nil
}

type DescribeDBAccountConnectionParams struct {
	WorkspaceID  string
	BranchID     string
	ComputeID    string
	AccountName  string
	DatabaseName string
	ProjectName  string
	AddressID    string
}

type DBAccountConnection struct {
	WorkspaceID            string `json:"WorkspaceId"`
	BranchID               string `json:"BranchId"`
	ComputeID              string `json:"ComputeId"`
	AccountName            string `json:"AccountName"`
	AccountPassword        string `json:"AccountPassword"`
	AllowHost              string `json:"AllowHost"`
	DatabaseName           string `json:"DatabaseName"`
	ConnectionURL          string `json:"ConnectionUrl"`
	ConnectionExampleCount int    `json:"-"`
}

// DescribeDBAccountConnection returns a ready-to-use libpq connection URL for a
// branch's Database compute. This is the native PostgreSQL access path (there is
// no pg-meta/PostgREST layer for the PG engine).
func (c *Client) DescribeDBAccountConnection(ctx context.Context, params DescribeDBAccountConnectionParams) (DBAccountConnection, error) {
	requestInput := map[string]interface{}{
		"WorkspaceId":  params.WorkspaceID,
		"BranchId":     params.BranchID,
		"ComputeId":    params.ComputeID,
		"AccountName":  params.AccountName,
		"DatabaseName": params.DatabaseName,
	}
	if params.ProjectName != "" {
		requestInput["ProjectName"] = params.ProjectName
	}
	if params.AddressID != "" {
		requestInput["AddressId"] = params.AddressID
	}
	resp, err := c.aidap.DescribeDBAccountConnectionCommonWithContext(ctx, &requestInput)
	if err != nil {
		return DBAccountConnection{}, fmt.Errorf("failed to describe dbaccount connection: %w", err)
	}
	var result DBAccountConnection
	if err := remarshal(*resp, &result); err != nil {
		return DBAccountConnection{}, fmt.Errorf("decode dbaccount connection: %w", err)
	}
	return result, nil
}
