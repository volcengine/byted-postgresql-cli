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
	"fmt"
	"os/exec"
	"regexp"
	"strconv"
	"strings"

	"github.com/spf13/cobra"

	"github.com/volcengine/byted-postgresql-cli/internal/volcengine"
)

var postgresVersionPattern = regexp.MustCompile(`\b(\d+)(?:\.\d+)?\b`)

func checkPostgresClientVersion(cmd *cobra.Command, client *volcengine.Client, workspaceID, bin, tool string) error {
	if workspaceID == "" {
		return nil
	}
	workspace, err := client.DescribeWorkspaceDetail(cmd.Context(), workspaceID)
	if err != nil {
		return err
	}
	serverMajor, ok := postgresMajorVersion(workspace.EngineVersion)
	if !ok {
		return nil
	}
	output, err := exec.CommandContext(cmd.Context(), bin, "--version").CombinedOutput()
	if err != nil {
		return fmt.Errorf("check %s version: %w", tool, err)
	}
	clientMajor, ok := postgresMajorVersion(string(output))
	if !ok || clientMajor >= serverMajor {
		return nil
	}
	fmt.Fprintf(cmd.ErrOrStderr(),
		"Warning: detected local %s major version %d, but the server is PostgreSQL %d. "+
			"Upgrade the PostgreSQL client to version %d or newer before using this command. "+
			"Install the matching postgresql-client package (for example, postgresql-client-%d).\n",
		tool, clientMajor, serverMajor, serverMajor, serverMajor)
	return nil
}

func postgresMajorVersion(value string) (int, bool) {
	value = strings.TrimSpace(value)
	if strings.Contains(value, "PostgreSQL_") {
		value = strings.TrimPrefix(value, "PostgreSQL_")
	}
	match := postgresVersionPattern.FindStringSubmatch(value)
	if len(match) != 2 {
		return 0, false
	}
	major, err := strconv.Atoi(match[1])
	return major, err == nil
}
