package focus

import (
	"github.com/urfave/cli/v2"

	"github.com/ayoisaiah/focus/app"
)

// GetApp retrieves the focus app instance.
func GetApp() *cli.App {
	globalFlags := map[string]cli.Flag{
		"no-color": &cli.BoolFlag{
			Name:  "no-color",
			Usage: "Disable coloured output",
		},
	}

	statsFlags := []cli.Flag{
		&cli.StringFlag{
			Name:    "end",
			Aliases: []string{"e"},
			Usage:   "Specify an end date in the following format: YYYY-MM-DD [HH:MM:SS PM] (defaults to the current time)",
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
			Name:    "tag",
			Aliases: []string{"t"},
			Usage:   "Filter sessions by tag",
		},
	}

	resumeFlags := []cli.Flag{
		&cli.BoolFlag{
			Name:    "select",
			Aliases: []string{"s"},
			Usage:   "Select a paused session from a list",
		},
	}

	timerFlags := []cli.Flag{
		&cli.BoolFlag{
			Name:    "disable-notification",
			Aliases: []string{"d"},
			Usage:   "Disable the system notification that appears after a session is completed",
		},
		globalFlags["no-color"],
		&cli.StringFlag{
			Name:    "session-cmd",
			Aliases: []string{"cmd"},
			Usage:   "Execute an arbitrary command after each session",
		},
		&cli.StringFlag{
			Name:  "sound",
			Usage: "Play ambient sounds continuously during a session. Default options: coffee_shop, fireplace, rain,\n\t\t\t\twind, birds, playground, tick_tock. Disable sound by setting to 'off'",
		},
		&cli.BoolFlag{
			Name:    "sound-on-break",
			Aliases: []string{"sob"},
			Usage:   "Enable ambient sound in break sessions",
		},
		&cli.StringFlag{
			Name:    "work-sound",
			Aliases: []string{"ws"},
			Usage:   "Sound to play when a break session has ended. Defaults to loud_bell",
		},
		&cli.StringFlag{
			Name:    "break-sound",
			Aliases: []string{"bs"},
			Usage:   "Sound to play when a work session has ended. Defaults to bell",
		},
		&cli.StringFlag{
			Name:    "tag",
			Aliases: []string{"t"},
			Usage:   "Add comma-delimited tags to a session",
		},
	}

	focusApp := &cli.App{
		Name: "focus",
		Authors: []*cli.Author{
			{
				Name:  "Ayooluwa Isaiah",
				Email: "ayo@freshman.tech",
			},
		},
		Usage:                "Focus is a cross-platform productivity timer for the command-line. It is based on the Pomodoro Technique,\n\t\ta time management method developed by Francesco Cirillo in the late 1980s.",
		UsageText:            "[COMMAND] [OPTIONS]",
		Version:              "v1.3.0",
		EnableBashCompletion: true,
		Commands: []*cli.Command{
			{
				Name:   "resume",
				Usage:  "Resume a previously interrupted session",
				Flags:  append(timerFlags, resumeFlags...),
				Action: app.ResumeAction,
			},
			{
				Name:   "edit-config",
				Usage:  "Edit the configuration file",
				Action: app.EditConfigAction,
			},
			{
				Name:   "list",
				Usage:  "List all the sessions within the specified time period",
				Action: app.ListAction,
				Flags:  append(statsFlags, globalFlags["no-color"]),
			},
			{
				Name:   "edit-tag",
				Usage:  "Edit the tags for a set of focus sessions",
				Action: app.EditTagsAction,
				Flags:  append(statsFlags, globalFlags["no-color"]),
			},
			{
				Name:   "delete",
				Usage:  "Permanently delete the all sessions within the specified time period. Will prompt before deleting.",
				Action: app.DeleteAction,
				Flags:  append(statsFlags, globalFlags["no-color"]),
			},
			{
				Name:   "stats",
				Usage:  "Track your progress with detailed statistics reporting. Defaults to a reporting period of 7 days",
				Action: app.ShowAction,
				Flags:  append(statsFlags, globalFlags["no-color"]),
			},
		},
		Flags: []cli.Flag{
			&cli.UintFlag{
				Name:    "short-break",
				Aliases: []string{"s"},
				Usage:   "Short break duration in minutes (default: 5)",
			},
			&cli.UintFlag{
				Name:    "long-break",
				Aliases: []string{"l"},
				Usage:   "Long break duration in minutes (default: 15)",
			},
			&cli.UintFlag{
				Name:    "long-break-interval",
				Aliases: []string{"int"},
				Usage:   "The number of work sessions before a long break (default: 4)",
			},
			&cli.UintFlag{
				Name:    "work",
				Aliases: []string{"w"},
				Usage:   "Work duration in minutes (default: 25)",
			},
		},
		Action: app.DefaultAction,
	}

	focusApp.Flags = append(focusApp.Flags, timerFlags...)

	return focusApp
}
