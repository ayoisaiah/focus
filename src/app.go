package werk

import (
	"fmt"
	"net/http"
	"time"

	"github.com/urfave/cli/v2"
)

func init() {
	// Override the default help template
	cli.AppHelpTemplate = `DESCRIPTION:
	{{.Usage}}

USAGE:
   {{.HelpName}} {{if .UsageText}}{{ .UsageText }}{{end}}
{{if len .Authors}}
AUTHOR:
   {{range .Authors}}{{ . }}{{end}}{{end}}
{{if .Version}}
VERSION:
	 {{.Version}}{{end}}
{{if .VisibleFlags}}
FLAGS:{{range .VisibleFlags}}{{ if (eq .Name "find" "undo" "replace") }}
		 {{if .Aliases}}-{{range $element := .Aliases}}{{$element}},{{end}}{{end}} --{{.Name}} {{.DefaultText}}
				 {{.Usage}}
		 {{end}}{{end}}
OPTIONS:{{range .VisibleFlags}}{{ if not (eq .Name "find" "replace" "undo") }}
		 {{if .Aliases}}-{{range $element := .Aliases}}{{$element}},{{end}}{{end}} --{{.Name}} {{ .DefaultText }}
				 {{.Usage}}
		 {{end}}{{end}}{{end}}
DOCUMENTATION:
	https://github.com/ayoisaiah/f2/wiki

WEBSITE:
	https://github.com/ayoisaiah/f2
`

	// Override the default version printer
	oldVersionPrinter := cli.VersionPrinter
	cli.VersionPrinter = func(c *cli.Context) {
		oldVersionPrinter(c)
		checkForUpdates(GetApp())
	}
}

func checkForUpdates(app *cli.App) {
	fmt.Println("Checking for updates...")

	c := http.Client{Timeout: 20 * time.Second}
	resp, err := c.Get("https://github.com/ayoisaiah/f2/releases/latest")
	if err != nil {
		fmt.Println("HTTP Error: Failed to check for update")
		return
	}

	defer resp.Body.Close()

	var version string
	_, err = fmt.Sscanf(
		resp.Request.URL.String(),
		"https://github.com/ayoisaiah/f2/releases/tag/%s",
		&version,
	)
	if err != nil {
		fmt.Println("Failed to get latest version")
		return
	}

	if version == app.Version {
		fmt.Printf(
			"Congratulations, you are using the latest version of %s\n",
			app.Name,
		)
	} else {
		fmt.Printf("%s: %s at %s\n", printColor("green", "Update available"), version, resp.Request.URL.String())
	}
}

// GetApp retrieves the werk app instance
func GetApp() *cli.App {
	return &cli.App{
		Name: "Werk",
		Authors: []*cli.Author{
			{
				Name:  "Ayooluwa Isaiah",
				Email: "ayo@freshman.tech",
			},
		},
		Usage:                "Werk is a cross-platform pomodoro app for the command line",
		UsageText:            "FLAGS [OPTIONS] [PATHS...]",
		Version:              "v0.1.0",
		EnableBashCompletion: true,
		Flags: []cli.Flag{
			&cli.UintFlag{
				Name:    "long",
				Usage:   "Long break duration in minutes",
				Aliases: []string{"l"},
				Value:   15,
			},
			&cli.UintFlag{
				Name:    "short",
				Usage:   "Short break duration in minutes",
				Aliases: []string{"s"},
				Value:   5,
			},
			&cli.UintFlag{
				Name:    "pomodoro",
				Usage:   "Pomodoro interval duration in minutes",
				Aliases: []string{"p"},
				Value:   25,
			},
		},
		Action: func(c *cli.Context) error {
			newWerk(c)
			return nil
		},
	}
}
