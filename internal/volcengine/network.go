package volcengine

import (
	"context"
	"fmt"

	"github.com/volcengine/volcengine-go-sdk/service/vpc"
	"github.com/volcengine/volcengine-go-sdk/volcengine"
)

type ListVPCsParams struct {
	VPCName    string
	VPCOwnerID int64
	PageSize   int64
	PageNumber int64
}

type VPC struct {
	ID          string   `json:"VpcId"`
	Name        string   `json:"VpcName"`
	AccountID   string   `json:"AccountId"`
	ProjectName string   `json:"ProjectName"`
	CIDRBlock   string   `json:"CidrBlock"`
	Status      string   `json:"Status"`
	IsDefault   bool     `json:"IsDefault"`
	SubnetIDs   []string `json:"SubnetIds"`
}

type ListVPCsResult struct {
	TotalCount int
	VPCs       []VPC
}

func (c *Client) ListVPCs(ctx context.Context, params ListVPCsParams) (ListVPCsResult, error) {
	pageSize := params.PageSize
	if pageSize <= 0 || pageSize > 100 {
		pageSize = 100
	}
	pageNumber := params.PageNumber
	if pageNumber <= 0 {
		pageNumber = 1
	}
	req := (&vpc.DescribeVpcsInput{}).
		SetPageNumber(pageNumber).
		SetPageSize(pageSize)
	if params.VPCName != "" {
		req.SetVpcName(params.VPCName)
	}
	if params.VPCOwnerID > 0 {
		req.SetVpcOwnerId(params.VPCOwnerID)
	}
	resp, err := c.vpc.DescribeVpcsWithContext(ctx, req)
	if err != nil {
		return ListVPCsResult{}, fmt.Errorf("failed to describe VPCs: %w", err)
	}
	result := ListVPCsResult{TotalCount: int(int64Value(resp.TotalCount))}
	for _, item := range resp.Vpcs {
		if item == nil {
			continue
		}
		result.VPCs = append(result.VPCs, VPC{
			ID: stringValue(item.VpcId), Name: stringValue(item.VpcName),
			AccountID: stringValue(item.AccountId), ProjectName: stringValue(item.ProjectName),
			CIDRBlock: stringValue(item.CidrBlock), Status: stringValue(item.Status),
			IsDefault: boolValue(item.IsDefault),
			SubnetIDs: stringPointersValue(item.SubnetIds),
		})
	}
	return result, nil
}

type ListSubnetsParams struct {
	VPCID      string
	SubnetName string
	PageSize   int64
	PageNumber int64
}

type Subnet struct {
	ID                      string `json:"SubnetId"`
	Name                    string `json:"SubnetName"`
	VPCID                   string `json:"VpcId"`
	AccountID               string `json:"AccountId"`
	ProjectName             string `json:"ProjectName"`
	ZoneID                  string `json:"ZoneId"`
	CIDRBlock               string `json:"CidrBlock"`
	Status                  string `json:"Status"`
	AvailableIPAddressCount int64  `json:"AvailableIpAddressCount"`
	IsDefault               bool   `json:"IsDefault"`
}

type ListSubnetsResult struct {
	TotalCount int
	Subnets    []Subnet
}

func (c *Client) ListSubnets(ctx context.Context, params ListSubnetsParams) (ListSubnetsResult, error) {
	if err := c.validateVPCExists(ctx, params.VPCID); err != nil {
		return ListSubnetsResult{}, err
	}
	pageSize := params.PageSize
	if pageSize <= 0 || pageSize > 100 {
		pageSize = 100
	}
	pageNumber := params.PageNumber
	if pageNumber <= 0 {
		pageNumber = 1
	}
	req := (&vpc.DescribeSubnetsInput{}).
		SetVpcId(params.VPCID).
		SetPageNumber(pageNumber).
		SetPageSize(pageSize)
	if params.SubnetName != "" {
		req.SetSubnetName(params.SubnetName)
	}
	resp, err := c.vpc.DescribeSubnetsWithContext(ctx, req)
	if err != nil {
		return ListSubnetsResult{}, fmt.Errorf("failed to describe subnets: %w", err)
	}
	result := ListSubnetsResult{TotalCount: int(int64Value(resp.TotalCount))}
	for _, item := range resp.Subnets {
		if item == nil {
			continue
		}
		result.Subnets = append(result.Subnets, Subnet{
			ID: stringValue(item.SubnetId), Name: stringValue(item.SubnetName),
			VPCID: stringValue(item.VpcId), AccountID: stringValue(item.AccountId),
			ProjectName: stringValue(item.ProjectName), ZoneID: stringValue(item.ZoneId),
			CIDRBlock: stringValue(item.CidrBlock), Status: stringValue(item.Status),
			AvailableIPAddressCount: int64Value(item.AvailableIpAddressCount),
			IsDefault:               boolValue(item.IsDefault),
		})
	}
	return result, nil
}

func (c *Client) validateVPCExists(ctx context.Context, vpcID string) error {
	resp, err := c.vpc.DescribeVpcsWithContext(ctx, (&vpc.DescribeVpcsInput{}).
		SetVpcIds([]*string{volcengine.String(vpcID)}).
		SetPageNumber(1).
		SetPageSize(1))
	if err != nil {
		return fmt.Errorf("failed to validate VPC %q: %w", vpcID, err)
	}
	if resp == nil || len(resp.Vpcs) == 0 {
		return fmt.Errorf("VPC %q does not exist or is not accessible", vpcID)
	}
	return nil
}
