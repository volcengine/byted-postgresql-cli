package cli

import (
	"strings"

	"github.com/spf13/cobra"
)

var usageFlagPlaceholders = map[string]string{
	"branch-id":        "<branch-id>",
	"compute-id":       "<compute-id>",
	"job-id":           "<job-id>",
	"parent-branch-id": "<branch-id>",
	"subnet-id":        "<subnet-id>",
	"vpc-id":           "<vpc-id>",
	"workspace-id":     "<workspace-id>",
}

func formatUseLine(cmd *cobra.Command) string {
	usage := cmd.UseLine()
	resourceArgs := make([]string, 0, len(usageFlagPlaceholders))
	for _, name := range []string{
		"workspace-id",
		"parent-branch-id",
		"branch-id",
		"compute-id",
		"job-id",
		"vpc-id",
		"subnet-id",
	} {
		if cmd.LocalNonPersistentFlags().Lookup(name) != nil &&
			!strings.Contains(usage, "--"+name) {
			usage = strings.Replace(usage, " [flags]", " --"+name+" "+usageFlagPlaceholders[name]+" [flags]", 1)
		}
		if strings.Contains(usage, "--"+name+" ") {
			resourceArgs = append(resourceArgs, "--"+name+" "+usageFlagPlaceholders[name])
		}
	}
	if len(resourceArgs) >= 2 {
		for _, arg := range resourceArgs {
			name := strings.SplitN(arg, " ", 2)[0]
			usage = strings.ReplaceAll(usage, " "+name+" ", " \\\n    "+name+" ")
		}
	}
	return usage
}

func isNavigationCommand(cmd *cobra.Command) bool {
	return cmd.HasParent() && cmd.HasAvailableSubCommands()
}

const usageTemplate = `Usage:{{if .Runnable}}
  {{formatUseLine .}}{{end}}{{if .HasAvailableSubCommands}}
  {{.CommandPath}} [command]{{end}}{{if gt (len .Aliases) 0}}

Aliases:
  {{.NameAndAliases}}{{end}}{{if and .HasExample (not (isNavigationCommand .))}}

Examples:
{{.Example}}{{end}}{{if .HasAvailableSubCommands}}{{$cmds := .Commands}}{{if eq (len .Groups) 0}}

Available Commands:{{range $cmds}}{{if (or .IsAvailableCommand (eq .Name "help"))}}
  {{rpad .Name .NamePadding }} {{.Short}}{{end}}{{end}}{{else}}{{range $group := .Groups}}

{{.Title}}{{range $cmds}}{{if (and (eq .GroupID $group.ID) (or .IsAvailableCommand (eq .Name "help")))}}
  {{rpad .Name .NamePadding }} {{.Short}}{{end}}{{end}}{{end}}{{if not .AllChildCommandsHaveGroup}}

Additional Commands:{{range $cmds}}{{if (and (eq .GroupID "") (or .IsAvailableCommand (eq .Name "help")))}}
  {{rpad .Name .NamePadding }} {{.Short}}{{end}}{{end}}{{end}}{{end}}{{end}}{{if .HasAvailableLocalFlags}}

Flags:
{{.LocalFlags.FlagUsages | trimTrailingWhitespaces}}{{end}}{{if .HasAvailableInheritedFlags}}

Global Flags:
{{.InheritedFlags.FlagUsages | trimTrailingWhitespaces}}{{end}}{{if .HasHelpSubCommands}}

Additional help topics:{{range .Commands}}{{if .IsAdditionalHelpTopicCommand}}
  {{rpad .CommandPath .CommandPathPadding}} {{.Short}}{{end}}{{end}}{{end}}{{if .HasAvailableSubCommands}}

Use "{{.CommandPath}} [command] --help" for more information about a command.{{end}}
`

const helpTemplate = `{{if not (isNavigationCommand .)}}{{with .Short}}{{. | trimTrailingWhitespaces}}

{{end}}{{with .Long}}{{. | trimTrailingWhitespaces}}

{{end}}{{end}}{{if or .Runnable .HasSubCommands}}{{.UsageString}}{{end}}`

func init() {
	cobra.AddTemplateFunc("formatUseLine", formatUseLine)
	cobra.AddTemplateFunc("isNavigationCommand", isNavigationCommand)
}
