package cmd

import (
	"fmt"
	"net/http"
	"time"

	focus "github.com/ayoisaiah/focus/src/internal"
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
		fmt.Printf("%s: %s at %s\n", focus.PrintColor("green", "Update available"), version, resp.Request.URL.String())
	}
}

// GetApp retrieves the focus app instance.
func GetApp() *cli.App {
	return &cli.App{
		Name: "Focus",
		Authors: []*cli.Author{
			{
				Name:  "Ayooluwa Isaiah",
				Email: "ayo@freshman.tech",
			},
		},
		Usage:                "Focus is a cross-platform pomodoro app for the command line",
		UsageText:            "FLAGS [OPTIONS] [PATHS...]",
		Version:              "v0.1.0",
		EnableBashCompletion: true,
		Commands: []*cli.Command{
			{
				Name: "resume",
				Action: func(ctx *cli.Context) error {
					t := &focus.Timer{}

					return t.Resume()
				},
			},
			{
				Name: "stats",
				Action: func(ctx *cli.Context) error {
					stats, err := focus.NewStats(ctx)
					if err != nil {
						return err
					}

					stats.Run()

					return nil
				},
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:    "period",
						Aliases: []string{"p"},
						Usage:   "The time period for the statistics",
						Value:   string(focus.Period7Days),
					},
					&cli.StringFlag{
						Name:    "start",
						Aliases: []string{"s"},
						Usage:   "The start date",
					},
					&cli.StringFlag{
						Name:    "end",
						Aliases: []string{"e"},
						Usage:   "The end date",
					},
				},
			},
		},
		Flags: []cli.Flag{
			&cli.UintFlag{
				Name:    "long-break",
				Usage:   "Long break duration in minutes (default: 15)",
				Aliases: []string{"l"},
			},
			&cli.UintFlag{
				Name:    "short-break",
				Usage:   "Short break duration in minutes (default: 5)",
				Aliases: []string{"s"},
			},
			&cli.UintFlag{
				Name:    "pomodoro",
				Usage:   "Pomodoro interval duration in minutes (default: 25)",
				Aliases: []string{"p"},
			},
			&cli.UintFlag{
				Name:    "long-break-interval",
				Aliases: []string{"int"},
				Usage:   "The number of pomodoro sessions before a long break (default: 4)",
			},
			&cli.UintFlag{
				Name:    "max-pomodoros",
				Aliases: []string{"max"},
				Usage:   "The maximum number of pomodoro sessions (default: unlimited)",
			},
			&cli.BoolFlag{
				Name:  "24-hour",
				Usage: "Switch from 12-hour clock to 24-hour clock",
			},
			&cli.BoolFlag{
				Name:    "auto-pomodoro",
				Aliases: []string{"ap"},
				Usage:   "Start pomodoro sessions automatically without user interaction",
			},
			&cli.BoolFlag{
				Name:    "auto-break",
				Aliases: []string{"ab"},
				Usage:   "Start break sessions automatically without user interaction",
			},
			&cli.BoolFlag{
				Name:    "disable-notifications",
				Aliases: []string{"d"},
				Usage:   "Disable notification alerts after a session is completed",
			},
			&cli.BoolFlag{
				Name:  "allow-pausing",
				Usage: "Enable interrupted pomodoro sessions to be resumed",
			},
		},
		Action: func(ctx *cli.Context) error {
			c := &focus.Config{}

			err := c.Init()
			if err != nil {
				fmt.Println(
					fmt.Errorf("Unable to initialise Focus from configuration file: %w\n", err),
				)
			}

			t := focus.NewTimer(ctx, c)
			t.Run()

			return nil
		},
	}
}
