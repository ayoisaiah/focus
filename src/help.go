package cmd

import (
	"fmt"

	"github.com/pterm/pterm"
)

func helpText() string {
	description := fmt.Sprintf(
		"%s\n\t\t{{.Usage}}\n\n",
		pterm.Yellow("DESCRIPTION"),
	)
	usage := fmt.Sprintf(
		"%s\n\t\t{{.HelpName}} {{if .UsageText}}{{ .UsageText }}{{end}}\n\n",
		pterm.Yellow("USAGE"),
	)
	author := fmt.Sprintf(
		"{{if len .Authors}}%s\n\t\t{{range .Authors}}{{ . }}{{end}}{{end}}\n\n",
		pterm.Yellow("AUTHOR"),
	)

	version := fmt.Sprintf(
		"{{if .Version}}%s\n\t\t{{.Version}}{{end}}\n\n",
		pterm.Yellow("VERSION"),
	)
	commands := fmt.Sprintf(
		"%s\n{{range .Commands}}{{if not .HideHelp}}   %s{{ `\t`}}{{.Usage}}{{ `\n` }}{{end}}{{end}}\n\n",
		pterm.Yellow("COMMANDS"),
		pterm.Green("{{join .Names `, `}}"),
	)
	options := fmt.Sprintf(
		"%s\n{{range .VisibleFlags}}\t\t{{if .Aliases}}{{range $element := .Aliases}}%s,{{end}}{{end}} %s\n\t\t\t\t{{.Usage}}\n\n{{end}}",
		pterm.Yellow("OPTIONS"),
		pterm.Green("-{{$element}}"),
		pterm.Green("--{{.Name}} {{.DefaultText}}"),
	)

	docs := fmt.Sprintf(
		"%s\n\t\t%s\n\n",
		pterm.Yellow("DOCUMENTATION"),
		"https://github.com/ayoisaiah/focus/wiki",
	)
	website := fmt.Sprintf(
		"%s\n\t\thttps://github.com/ayoisaiah/focus\n",
		pterm.Yellow("WEBSITE"),
	)

	return description + usage + author + version + commands + options + docs + website
}
