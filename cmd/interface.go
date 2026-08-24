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
	"github.com/spf13/cobra"

	"github.com/volcengine/byted-postgresql-cli/internal/volcengine"
)

type ProviderContext struct {
	Provider        volcengine.Provider
	Region          string
	AIDAPEndpoint   string
	ConsoleEndpoint string
	DisplayName     string
	ConfigDir       string
	EnvPrefix       string
}

type CommandFactory func(ProviderContext) *cobra.Command

type CommandGroup struct {
	Group     *cobra.Group
	Factories []CommandFactory
}

type ProviderCommandTree struct {
	Provider ProviderContext
	Groups   []CommandGroup
}

func authenticationCommandGroup() CommandGroup {
	return CommandGroup{
		Group: &cobra.Group{ID: commandGroupAuthentication, Title: "Authentication:"},
		Factories: []CommandFactory{
			newLoginCmd, newLogoutCmd, newConfigureCmd, newStatusCmd,
		},
	}
}

func discoveryCommandGroup() CommandGroup {
	return CommandGroup{
		Group: &cobra.Group{ID: commandGroupDiscovery, Title: "Discovery:"},
		Factories: []CommandFactory{
			newProjectsCmd, newWorkspacesCmd, newBranchesCmd, newOperationsCmd,
		},
	}
}

func databaseAccessCommandGroup() CommandGroup {
	return CommandGroup{
		Group: &cobra.Group{ID: commandGroupDatabaseAccess, Title: "Database Access:"},
		Factories: []CommandFactory{
			newConnectionStringCmd, newPsqlCmd, newDatabaseToolsCmd, newDatabaseInspectCmd, newSchemaDiffCmd,
		},
	}
}

func resourceCommandGroup(endpointFactory CommandFactory) CommandGroup {
	return CommandGroup{
		Group: &cobra.Group{ID: commandGroupResources, Title: "Database Resources:"},
		Factories: []CommandFactory{
			newDatabasesCmd, newRolesCmd, newComputesCmd, endpointFactory,
		},
	}
}

func settingsCommandGroup() CommandGroup {
	return CommandGroup{
		Group: &cobra.Group{ID: commandGroupSettings, Title: "Workspace Settings:"},
		Factories: []CommandFactory{
			newWorkspaceNetworkCmd, newWorkspaceTagsCmd,
		},
	}
}

func toolsCommandGroup() CommandGroup {
	return CommandGroup{
		Group: &cobra.Group{ID: commandGroupTools, Title: "Tools:"},
		Factories: []CommandFactory{
			newVersionCmd, newConfigCmd, newMCPCmd, newUpdateCmd,
		},
	}
}

func NewProviderCommandTree(ctx ProviderContext) ProviderCommandTree {
	return ProviderCommandTree{
		Provider: ctx,
		Groups: []CommandGroup{
			authenticationCommandGroup(),
			discoveryCommandGroup(),
			databaseAccessCommandGroup(),
			resourceCommandGroup(newVolcengineEndpointsCmd),
			settingsCommandGroup(),
			toolsCommandGroup(),
		},
	}
}

func providerContext(region string) ProviderContext {
	spec, _ := volcengine.ProviderSpecFor(volcengine.ProviderVolcengine)
	if region == "" {
		region = spec.DefaultRegion
	}
	return ProviderContext{
		Provider:        volcengine.ProviderVolcengine,
		Region:          region,
		AIDAPEndpoint:   spec.AIDAPEndpoint(region),
		ConsoleEndpoint: spec.ConsoleEndpoint,
		DisplayName:     spec.DisplayName,
		ConfigDir:       spec.ConfigDir,
		EnvPrefix:       spec.EnvPrefix,
	}
}

func defaultProviderContext() ProviderContext {
	return providerContext(volcengine.DefaultRegion)
}

func providerForCommandTree() volcengine.Provider {
	return volcengine.ProviderVolcengine
}

func rootProviderContext() ProviderContext {
	return providerContext(volcengine.RegionSetting())
}
