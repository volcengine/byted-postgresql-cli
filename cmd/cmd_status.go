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
	"io"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/volcengine/byted-postgresql-cli/internal/volcengine"
	"github.com/volcengine/byted-postgresql-cli/internal/writer"
)

type statusSnapshot struct {
	Authentication statusAuthentication `json:"authentication" yaml:"authentication"`
}

type statusAuthentication struct {
	Status         string `json:"status" yaml:"status"`
	Profile        string `json:"profile" yaml:"profile"`
	Mode           string `json:"mode,omitempty" yaml:"mode,omitempty"`
	Identity       string `json:"identity,omitempty" yaml:"identity,omitempty"`
	AccessKey      string `json:"access_key" yaml:"access_key"`
	Region         string `json:"region" yaml:"region"`
	Endpoint       string `json:"endpoint,omitempty" yaml:"endpoint,omitempty"`
	CredentialFrom string `json:"credential_from" yaml:"credential_from"`
}

func newStatusCmd(ctx ProviderContext) *cobra.Command {
	provider := ctx.Provider
	return &cobra.Command{
		Use:   "status",
		Short: "Show the resolved CLI configuration and Volcengine credentials",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			out := cmd.OutOrStdout()
			g := fromCtx(cmd)
			snapshot := buildStatusSnapshot(provider)
			if g.Output != string(writer.FormatTable) {
				return writeStructuredStatus(out, g.Output, snapshot)
			}
			_, err := io.WriteString(out, formatStatus(snapshot)+"\n")
			return err
		},
	}
}

// buildStatusSnapshot resolves the effective AK/SK credentials the same way an
// API call would, so `status` reports the truth the next command will use.
func buildStatusSnapshot(providers ...volcengine.Provider) statusSnapshot {
	provider := providerForCommandTree()
	if len(providers) > 0 {
		provider = providers[0]
	}
	spec, _ := volcengine.ProviderSpecFor(provider)
	auth := statusAuthentication{
		Status:         "unauthenticated",
		Region:         volcengine.RegionSetting(),
		CredentialFrom: "none",
	}
	if auth.Region == "" {
		auth.Region = spec.DefaultRegion
	}

	// Environment credentials take precedence over the profile file.
	if envAK := strings.TrimSpace(os.Getenv(volcengine.AccessKeyEnvironment(provider))); envAK != "" {
		auth.Status = "configured"
		auth.Mode = "environment"
		auth.AccessKey = maskSecret(envAK)
		auth.CredentialFrom = "environment"
		auth.Profile = "(env)"
		return statusSnapshot{Authentication: auth}
	}

	cfg, _, profile, err := volcengine.LoadSelectedProfileFor(provider)
	if err == nil && profile != nil {
		auth.Status = "configured"
		auth.Profile = profile.Name
		if auth.Profile == "" {
			auth.Profile = cfg.Current
		}
		mode := strings.ToLower(strings.TrimSpace(profile.Mode))
		if mode == "" {
			mode = volcengine.ModeAK
		}
		auth.Mode = mode
		switch mode {
		case volcengine.ModeConsoleLogin:
			// Console-login profiles hold no static AK/SK; surface the browser
			// login identity (login_session) instead of an empty access key.
			auth.Identity = strings.TrimSpace(profile.LoginSession)
		default:
			auth.AccessKey = maskSecret(profile.AccessKey)
		}
		if auth.Region == spec.DefaultRegion && strings.TrimSpace(profile.Region) != "" &&
			!volcengine.HasRegionOverride() &&
			strings.TrimSpace(os.Getenv(volcengine.RegionEnvironment(provider))) == "" {
			auth.Region = strings.TrimSpace(profile.Region)
		}
		auth.Endpoint = strings.TrimSpace(profile.Endpoint)
		path, _ := volcengine.ConfigFilePath(provider)
		auth.CredentialFrom = "profile (" + path + ")"
	}
	return statusSnapshot{Authentication: auth}
}

func formatStatus(s statusSnapshot) string {
	a := s.Authentication
	lines := []string{
		"Authentication: " + a.Status,
		"Profile: " + orNone(a.Profile),
	}
	if a.Mode != "" {
		lines = append(lines, "Mode: "+a.Mode)
	}
	if a.Identity != "" {
		lines = append(lines, "Identity: "+a.Identity)
	}
	if a.Mode != volcengine.ModeConsoleLogin {
		lines = append(lines, "Access key: "+orNone(a.AccessKey))
	}
	lines = append(lines, "Region: "+orNone(a.Region))
	if a.Endpoint != "" {
		lines = append(lines, "Endpoint: "+a.Endpoint)
	}
	lines = append(lines, "Credentials: "+a.CredentialFrom)
	if a.Status != "configured" {
		lines = append(lines,
			"Run `byted-postgresql-cli login` to authenticate in your browser,",
			"or `byted-postgresql-cli configure set --access-key <key> --secret-key <secret> --region <region>`.")
	}
	return strings.Join(lines, "\n")
}

func writeStructuredStatus(out io.Writer, format string, snapshot statusSnapshot) error {
	output := writer.New(format)
	output.Out = out
	return output.WriteItem(snapshot, nil)
}

func orNone(v string) string {
	if strings.TrimSpace(v) == "" {
		return "none"
	}
	return v
}
