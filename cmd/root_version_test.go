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
	"testing"
	"unicode"

	"github.com/stretchr/testify/require"
)

func TestVersionLabel(t *testing.T) {
	tests := []struct {
		name    string
		version string
		want    string
	}{
		{name: "release version", version: "v0.1.0", want: "0.1.0"},
		{name: "version without prefix", version: "0.1.0", want: "0.1.0"},
		{name: "development version", version: "dev", want: "dev"},
		{name: "empty version", version: "", want: "dev"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.want, versionLabel(tt.version))
		})
	}
}

func TestRootVersionOutput(t *testing.T) {
	previous := Version
	Version = "v0.1.0"
	t.Cleanup(func() { Version = previous })

	cmd := newRootCmd()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{"--version"})

	require.NoError(t, cmd.Execute())
	require.Equal(t, "v0.1.0\n", out.String())
}

func TestVersionCommandMatchesVersionFlag(t *testing.T) {
	previous := Version
	Version = "v0.1.0"
	t.Cleanup(func() { Version = previous })

	flagCmd := newRootCmd()
	var flagOut bytes.Buffer
	flagCmd.SetOut(&flagOut)
	flagCmd.SetErr(&flagOut)
	flagCmd.SetArgs([]string{"--version"})
	require.NoError(t, flagCmd.Execute())

	versionCmd := newRootCmd()
	var versionOut bytes.Buffer
	versionCmd.SetOut(&versionOut)
	versionCmd.SetErr(&versionOut)
	versionCmd.SetArgs([]string{"--config-dir", t.TempDir(), "version"})
	require.NoError(t, versionCmd.Execute())

	require.Equal(t, flagOut.String(), versionOut.String())
	require.Equal(t, "v0.1.0\n", versionOut.String())
}

func TestRootHelpIncludesVersion(t *testing.T) {
	previous := Version
	Version = "v0.1.0"
	t.Cleanup(func() { Version = previous })

	cmd := newRootCmd()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{"--help"})

	require.NoError(t, cmd.Execute())
	require.Contains(t, out.String(), "Volcengine PostgreSQL CLI 0.1.0")
}

func TestRootHelpOmitsQuickStart(t *testing.T) {
	cmd := newRootCmd()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{"--help"})

	require.NoError(t, cmd.Execute())
	for _, unwanted := range []string{
		"Commands are scoped to a project, workspace, and branch.",
		"Quick Start:",
		"projects search --name <project-name>",
		"connection-string \\",
	} {
		require.NotContains(t, out.String(), unwanted)
	}
	require.NotContains(t, out.String(), "\t")
}

func TestRootHelpGroupsCommandsByWorkflow(t *testing.T) {
	cmd := newRootCmd()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{"--help"})

	require.NoError(t, cmd.Execute())
	help := out.String()
	for _, want := range []string{
		"Authentication:",
		"  configure         Manage cloud provider AK/SK credentials",
		"  status            Show the resolved CLI configuration and Volcengine credentials",
		"Database Access:",
		"  connection-string Print a Postgres connection string for a branch",
		"Database Resources:",
		"  databases         Manage branch databases",
		"Workspace Settings:",
		"  network           Manage workspace network settings",
		"Tools:",
		"  completion        Generate the autocompletion script for the specified shell",
	} {
		require.Contains(t, help, want)
	}
	require.NotContains(t, help, "Available Commands:")
	require.NotContains(t, help, "Additional Commands:")
}

// TestRootHelpIsEnglishOnly guards against re-introducing a mixed-language,
// jargon-heavy first screen.
func TestRootHelpIsEnglishOnly(t *testing.T) {
	cmd := newRootCmd()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{"--help"})

	require.NoError(t, cmd.Execute())
	help := out.String()

	require.Contains(t, help, "Volcengine PostgreSQL CLI")
	for _, r := range help {
		require.False(t, unicode.Is(unicode.Han, r), "root help must not mix Chinese characters into the English help: found %q", string(r))
	}
}

func TestCommandHelpAvoidsInternalTerms(t *testing.T) {
	for _, args := range [][]string{
		{"workspaces", "list", "--help"},
		{"workspaces", "create", "--help"},
		{"workspaces", "compute-settings", "--help"},
		{"computes", "create", "--help"},
		{"operations", "list", "--help"},
		{"mcp", "serve", "--help"},
	} {
		cmd := newRootCmd()
		var out bytes.Buffer
		cmd.SetOut(&out)
		cmd.SetErr(&out)
		cmd.SetArgs(args)

		require.NoError(t, cmd.Execute(), "args: %v", args)
		help := out.String()
		require.NotContains(t, help, "ByteTree", "args: %v", args)
		require.NotContains(t, help, "CU", "args: %v", args)
	}
}
