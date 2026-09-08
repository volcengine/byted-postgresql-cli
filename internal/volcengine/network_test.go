// Copyright (c) 2026 Beijing Volcano Engine Technology Co., Ltd.
// SPDX-License-Identifier: MIT

package volcengine

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestListSubnetsAllowsExistingVPCWithoutSubnets(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.RawQuery, "Action=DescribeVpcs") {
			_, _ = w.Write([]byte(`{"Result":{"Vpcs":[{"VpcId":"vpc-1"}],"TotalCount":1}}`))
			return
		}
		if strings.Contains(r.URL.RawQuery, "Action=DescribeSubnets") {
			_, _ = w.Write([]byte(`{"Result":{"Subnets":[],"TotalCount":0}}`))
			return
		}
		http.Error(w, `{"Message":"unexpected action"}`, http.StatusNotFound)
	}))
	t.Cleanup(server.Close)

	client, err := NewClient(Config{
		AccessKeyID:     "test-ak",
		SecretAccessKey: "test-sk",
		Region:          DefaultRegion,
		Endpoint:        server.URL,
	})
	if err != nil {
		t.Fatal(err)
	}
	result, err := client.ListSubnets(context.Background(), ListSubnetsParams{VPCID: "vpc-1"})
	if err != nil {
		t.Fatal(err)
	}
	if result.TotalCount != 0 || len(result.Subnets) != 0 {
		t.Fatalf("result = %+v, want an empty subnet list", result)
	}
}

func TestListSubnetsRejectsMissingVPC(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"Result":{"Vpcs":[],"TotalCount":0}}`))
	}))
	t.Cleanup(server.Close)

	client, err := NewClient(Config{
		AccessKeyID:     "test-ak",
		SecretAccessKey: "test-sk",
		Region:          DefaultRegion,
		Endpoint:        server.URL,
	})
	if err != nil {
		t.Fatal(err)
	}
	_, err = client.ListSubnets(context.Background(), ListSubnetsParams{VPCID: "not-exists"})
	if err == nil || !strings.Contains(err.Error(), "does not exist or is not accessible") {
		t.Fatalf("error = %v, want missing VPC error", err)
	}
}
