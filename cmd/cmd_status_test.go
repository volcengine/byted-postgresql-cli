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

package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/volcengine/byted-postgresql-cli/internal/volcengine"
	"gopkg.in/yaml.v3"
)

func TestFormatStatusShowsResolvedVolcengineConfiguration(t *testing.T) {
	got := formatStatus(statusSnapshot{Authentication: statusAuthentication{
		Status:         "configured",
		Profile:        "default",
		AccessKey:      "test*******-key",
		Region:         "cn-beijing",
		CredentialFrom: "profile (~/.volcengine/config.json)",
	}})
	for _, want := range []string{
		"Authentication: configured",
		"Profile: default",
		"Access key: test*******-key",
		"Region: cn-beijing",
		"Credentials: profile (~/.volcengine/config.json)",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("formatStatus() = %q, want substring %q", got, want)
		}
	}
}

func TestFormatStatusWithoutCredentialsIsActionable(t *testing.T) {
	got := formatStatus(statusSnapshot{Authentication: statusAuthentication{
		Status:         "unauthenticated",
		Region:         volcengine.DefaultRegion,
		CredentialFrom: "none",
	}})
	for _, want := range []string{
		"Authentication: unauthenticated",
		"Region: " + volcengine.DefaultRegion,
		"configure set --access-key <key> --secret-key <secret> --region <region>",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("formatStatus() = %q, want substring %q", got, want)
		}
	}
}

func TestBuildStatusSnapshotUsesEnvironmentCredentials(t *testing.T) {
	t.Setenv(volcengine.EnvAccessKeyID, "test-access-key")
	t.Setenv(volcengine.EnvSecretAccessKey, "test-secret-key")
	t.Setenv(volcengine.EnvRegion, "cn-shanghai")

	snapshot := buildStatusSnapshot()
	auth := snapshot.Authentication
	if auth.Status != "configured" || auth.Profile != "(env)" ||
		auth.AccessKey != "test*******-key" || auth.Region != "cn-shanghai" ||
		auth.CredentialFrom != "environment" {
		t.Fatalf("unexpected environment status: %+v", auth)
	}
}

func TestStatusOmitsConfigurationFields(t *testing.T) {
	cmd := newRootCmd()
	cmd.SetArgs([]string{"status"})
	var output strings.Builder
	cmd.SetOut(&output)

	_ = cmd.Execute()
	got := output.String()
	if strings.Contains(got, "config-dir") || strings.Contains(got, "api-host") {
		t.Fatalf("status output contains hidden configuration fields: %q", got)
	}
}

func TestWriteStructuredStatusJSON(t *testing.T) {
	snapshot := statusSnapshot{
		Authentication: statusAuthentication{
			Status:         "configured",
			Profile:        "production",
			AccessKey:      "test*******-key",
			Region:         "cn-beijing",
			Endpoint:       "https://aidap.cn-beijing.volcengineapi.com",
			CredentialFrom: "profile (~/.volcengine/config.json)",
		},
	}
	var output bytes.Buffer
	if err := writeStructuredStatus(&output, "json", snapshot); err != nil {
		t.Fatalf("writeStructuredStatus() error = %v", err)
	}

	var got map[string]any
	if err := json.Unmarshal(output.Bytes(), &got); err != nil {
		t.Fatalf("JSON output is invalid: %v\n%s", err, output.String())
	}
	auth, ok := got["authentication"].(map[string]any)
	if !ok {
		t.Fatalf("authentication = %T, want object", got["authentication"])
	}
	if auth["status"] != "configured" || auth["profile"] != "production" ||
		auth["access_key"] != "test*******-key" || auth["region"] != "cn-beijing" ||
		auth["endpoint"] != "https://aidap.cn-beijing.volcengineapi.com" ||
		auth["credential_from"] != "profile (~/.volcengine/config.json)" {
		t.Fatalf("unexpected JSON status: %s", output.String())
	}
	if _, ok := got["context"]; ok {
		t.Fatalf("status JSON still contains unsupported context: %s", output.String())
	}
}

func TestWriteStructuredStatusYAMLOmitsEmptyEndpoint(t *testing.T) {
	snapshot := statusSnapshot{
		Authentication: statusAuthentication{
			Status:         "unauthenticated",
			Region:         volcengine.DefaultRegion,
			CredentialFrom: "none",
		},
	}
	var output bytes.Buffer
	if err := writeStructuredStatus(&output, "yaml", snapshot); err != nil {
		t.Fatalf("writeStructuredStatus() error = %v", err)
	}

	var got map[string]any
	if err := yaml.Unmarshal(output.Bytes(), &got); err != nil {
		t.Fatalf("YAML output is invalid: %v\n%s", err, output.String())
	}
	auth, ok := got["authentication"].(map[string]any)
	if !ok {
		t.Fatalf("authentication = %T, want object", got["authentication"])
	}
	if auth["status"] != "unauthenticated" || auth["region"] != volcengine.DefaultRegion ||
		auth["credential_from"] != "none" {
		t.Fatalf("unexpected YAML status: %s", output.String())
	}
	if _, ok := auth["endpoint"]; ok {
		t.Fatalf("empty endpoint must be omitted: %s", output.String())
	}
	if _, ok := got["context"]; ok {
		t.Fatalf("status YAML still contains unsupported context: %s", output.String())
	}
}

func TestBuildStatusSnapshotUsesSelectedProfile(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv(volcengine.EnvAccessKeyID, "")
	t.Setenv(volcengine.EnvSecretAccessKey, "")
	t.Setenv(volcengine.EnvRegion, "")
	t.Setenv(volcengine.EnvLongRegion, "")
	t.Setenv(volcengine.EnvProfile, "")

	path := filepath.Join(home, ".volcengine", "config.json")
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatal(err)
	}
	data := `{"current":"prod","profiles":{"prod":{"name":"prod","mode":"ak","access-key":"test-access-key","secret-key":"test-secret-key","region":"cn-shanghai"}}}`
	if err := os.WriteFile(path, []byte(data), 0o600); err != nil {
		t.Fatal(err)
	}

	snapshot := buildStatusSnapshot()
	auth := snapshot.Authentication
	if auth.Status != "configured" || auth.Profile != "prod" ||
		auth.AccessKey != "test*******-key" || auth.Region != "cn-shanghai" {
		t.Fatalf("unexpected profile status: %+v", auth)
	}
}
