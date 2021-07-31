package cmd

import (
	"fmt"
	"net/http"
	"time"

	focus "github.com/ayoisaiah/focus/src/internal"
	"github.com/pterm/pterm"
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
{{if .Commands}}
COMMANDS:
{{range .Commands}}{{if not .HideHelp}}   {{join .Names ", "}}{{ "\t"}}{{.Usage}}{{ "\n" }}{{end}}{{end}}{{end}}
{{if .VisibleFlags}}
OPTIONS:{{range .VisibleFlags}}{{ if not (eq .Name "find" "replace" "undo") }}
		 {{if .Aliases}}-{{range $element := .Aliases}}{{$element}},{{end}}{{end}} --{{.Name}} {{ .DefaultText }}
				 {{.Usage}}
		 {{end}}{{end}}{{end}}
DOCUMENTATION:
	https://github.com/ayoisaiah/focus/wiki

WEBSITE:
	https://github.com/ayoisaiah/focus
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

	resp, err := c.Get("https://github.com/ayoisaiah/focus/releases/latest")
	if err != nil {
		pterm.Error.Println("HTTP Error: Failed to check for update")
		return
	}

	defer resp.Body.Close()

	var version string

	_, err = fmt.Sscanf(
		resp.Request.URL.String(),
		"https://github.com/ayoisaiah/focus/releases/tag/%s",
		&version,
	)
	if err != nil {
		pterm.Error.Println("Failed to get latest version")
		return
	}

	if version == app.Version {
		pterm.Info.Printf(
			"Congratulations, you are using the latest version of %s\n",
			app.Name,
		)
	} else {
		pterm.Info.Printf("%s: %s at %s\n", pterm.LightGreen("Update available"), version, resp.Request.URL.String())
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
		Usage:                "Focus is a cross-platform pomodoro timer application for the command line",
		UsageText:            "[COMMAND] [OPTIONS]",
		Version:              "v0.1.0",
		EnableBashCompletion: true,
		Commands: []*cli.Command{
			{
				Name:  "stats",
				Usage: "Track your progress with detailed statistics reporting. Defaults to a reporting period of 7 days",
				Action: func(ctx *cli.Context) error {
					store, err := focus.NewStore()
					if err != nil {
						return err
					}

					stats, err := focus.NewStats(ctx, store)
					if err != nil {
						return err
					}

					if ctx.Bool("delete") {
						return stats.Delete()
					}

					if ctx.Bool("list") {
						return stats.List()
					}

					return stats.Show()
				},
				Flags: []cli.Flag{
					&cli.BoolFlag{
						Name:  "delete",
						Usage: "Delete the all pomodoro sessions within the specified time period",
					},
					&cli.BoolFlag{
						Name:  "list",
						Usage: "List all the pomodoro sessions within the specified time period",
					},
					&cli.StringFlag{
						Name:    "period",
						Aliases: []string{"p"},
						Usage:   "Specify a time period for (defaults to 7days). Possible values are: today, yesterday, 7days, 14days, 30days, 90days, 180days, 365days, all-time",
						Value:   "7days",
					},
					&cli.StringFlag{
						Name:    "start",
						Aliases: []string{"s"},
						Usage:   "Specify a start date in the following format: YYYY-MM-DD [HH:MM:SS PM]",
					},
					&cli.StringFlag{
						Name:    "end",
						Aliases: []string{"e"},
						Usage:   "Specify an end date in the following format: YYYY-MM-DD [HH:MM:SS PM] (defaults to the current time)",
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
				Usage:   "Pomodoro duration in minutes (default: 25)",
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
				Usage:   "The maximum number of pomodoro sessions (unlimited by default)",
			},
			&cli.BoolFlag{
				Name:    "disable-notifications",
				Aliases: []string{"d"},
				Usage:   "Disable the system notification after a session is completed",
			},
			&cli.BoolFlag{
				Name:    "new",
				Aliases: []string{"n"},
				Usage:   "Start a new focus session. Using this option prevents the attempt to resume a previously halted session",
			},
		},
		Action: func(ctx *cli.Context) error {
			store, err := focus.NewStore()
			if err != nil {
				return err
			}

			// Running focus without arguments will attempt
			// to resume an interrupted session
			if ctx.NumFlags() == 0 {
				t := &focus.Timer{
					Store: store,
				}

				_, _, err = t.GetInterrupted()
				if err == nil {
					return t.Resume()
				}
			}

			config, err := focus.NewConfig()
			if err != nil {
				return err
			}

			t := focus.NewTimer(ctx, config, store)

			return t.Run()
		},
	}
}
