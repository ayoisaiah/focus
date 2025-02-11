package app

import (
	"github.com/pterm/pterm"
	"github.com/urfave/cli/v2"

	"github.com/ayoisaiah/focus/internal/config"
)

var (
	endTimeFlag = &cli.StringFlag{
		Name:    "end",
		Aliases: []string{"e"},
		Usage:   "Specify an end date in the following format: YYYY-MM-DD [HH:MM:SS PM] (defaults to the current time)",
	}

	periodFlag = &cli.StringFlag{
		Name:    "period",
		Aliases: []string{"p"},
		Usage:   "Specify a time period. Possible values are: today, yesterday, 7days, 14days, 30days, 90days, 180days, 365days, all-time",
	}

	startTimeFlag = &cli.StringFlag{
		Name:    "start",
		Aliases: []string{"s"},
		Usage:   "Specify a start date in the following format: YYYY-MM-DD [HH:MM:SS PM]",
	}

	sinceFlag = &cli.StringFlag{
		Name:  "since",
		Usage: "Start or add a new session in the past (e.g. '20 mins ago'). Must not overlap with any existing sessions",
	}

	filterTagFlag = &cli.StringFlag{
		Name:    "tag",
		Aliases: []string{"t"},
		Usage:   "Filter sessions by tag",
	}

	noColorFlag = &cli.BoolFlag{
		Name:  "no-color",
		Usage: "Disable coloured output",
	}

	strictFlag = &cli.BoolFlag{
		Name:  "strict",
		Usage: "When strict mode is enabled, you can't resume a paused session",
	}

	disableNotificationFlag = &cli.BoolFlag{
		Name:    "disable-notification",
		Aliases: []string{"d"},
		Usage:   "Disable the system notification that appears after a session is completed",
	}

	sessionCmdFlag = &cli.StringFlag{
		Name:    "session-cmd",
		Aliases: []string{"cmd"},
		Usage:   "Execute an arbitrary command after each session",
	}

	soundFlag = &cli.StringFlag{
		Name:  "sound",
		Usage: "Play ambient sounds continuously during a session. Default options: coffee_shop, fireplace, rain,\n\t\t\t\twind, birds, playground, tick_tock. Disable sound by setting to 'off'",
	}

	soundOnBreakFlag = &cli.BoolFlag{
		Name:    "sound-on-break",
		Aliases: []string{"sob"},
		Usage:   "Enable ambient sound in break sessions",
	}

	workSoundFlag = &cli.StringFlag{
		Name:    "work-sound",
		Aliases: []string{"ws"},
		Usage:   "Sound to play when a break session has ended. Defaults to loud_bell",
	}

	breakSoundFlag = &cli.StringFlag{
		Name:    "break-sound",
		Aliases: []string{"bs"},
		Usage:   "Sound to play when a work session has ended. Defaults to bell",
	}

	addTagFlag = &cli.StringFlag{
		Name:    "tag",
		Aliases: []string{"t"},
		Usage:   "Add comma-delimited tags to a session",
	}

	listJSONFlag = &cli.BoolFlag{
		Name:  "json",
		Usage: "List Focus sessions in JSON format",
	}

	statsJSONFlag = &cli.BoolFlag{
		Name:  "json",
		Usage: "Output Focus statistics as JSON",
	}

	statsPortFlag = &cli.UintFlag{
		Name:  "port",
		Usage: "Specify the port for the statistics server",
		Value: 1111,
	}

	resetTimerFlag = &cli.BoolFlag{
		Name:    "reset",
		Aliases: []string{"r"},
		Usage:   "Reset the timer to the beginning of the set",
	}

	shortBreakFlag = &cli.StringFlag{
		Name:    "short-break",
		Aliases: []string{"s"},
		Usage:   "Short break duration in minutes (default: 5)",
	}

	longBreakFlag = &cli.StringFlag{
		Name:    "long-break",
		Aliases: []string{"l"},
		Usage:   "Long break duration in minutes (default: 15)",
	}

	longBreakIntervalFlag = &cli.UintFlag{
		// TODO: Rename to set?
		Name:    "long-break-interval",
		Aliases: []string{"int"},
		Usage:   "The number of work sessions before a long break (default: 4)",
	}

	workFlag = &cli.StringFlag{
		Name:    "work",
		Aliases: []string{"w"},
		Usage:   "Work duration in minutes (default: 25)",
	}
)

// disableStyling disables all styling provided by pterm.
func disableStyling() {
	pterm.DisableColor()
	pterm.DisableStyling()
	pterm.Debug.Prefix.Text = ""
	pterm.Info.Prefix.Text = ""
	pterm.Success.Prefix.Text = ""
	pterm.Warning.Prefix.Text = ""
	pterm.Error.Prefix.Text = ""
	pterm.Fatal.Prefix.Text = ""
}

// Get retrieves the focus app instance.
func Get() *cli.App {
	defaultPeriod := *periodFlag
	defaultPeriod.Value = "7days"

	statsFlags := []cli.Flag{
		startTimeFlag,
		endTimeFlag,
		&defaultPeriod,
		filterTagFlag,
		noColorFlag,
		statsJSONFlag,
		statsPortFlag,
	}

	filterFlags := []cli.Flag{
		startTimeFlag,
		endTimeFlag,
		periodFlag,
		filterTagFlag,
		noColorFlag,
	}

	timerFlags := []cli.Flag{
		disableNotificationFlag,
		soundFlag,
		soundOnBreakFlag,
		workSoundFlag,
		breakSoundFlag,
		sessionCmdFlag,
		addTagFlag,
		strictFlag,
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
		Version:              config.Version,
		EnableBashCompletion: true,
		Commands: []*cli.Command{
			{
				Name:   "delete",
				Usage:  "Permanently delete the specified sessions",
				Action: deleteAction,
				Flags:  filterFlags,
			},
			{
				Name:   "edit-config",
				Usage:  "Edit the configuration file",
				Action: editConfigAction,
			},
			{
				Name:   "edit-tag",
				Usage:  "Edit the tags for a set of focus sessions",
				Action: editTagsAction,
				Flags:  filterFlags,
			},
			{
				Name:   "list",
				Usage:  "List all the sessions within the specified time period",
				Action: listAction,
				Flags:  append(filterFlags, listJSONFlag),
			},
			{
				Name:  "resume",
				Usage: "Resume a previously interrupted timer",
				Flags: append(
					timerFlags,
					resetTimerFlag,
				),
				Action: resumeAction,
			},
			{
				Name:   "stats",
				Usage:  "Track your progress with detailed statistics reporting. Defaults to a reporting period of 7 days",
				Action: statsAction,
				Flags:  statsFlags,
			},
			{
				Name:   "status",
				Usage:  "Print the status of the timer",
				Action: statusAction,
			},
		},
		Flags: []cli.Flag{
			shortBreakFlag,
			longBreakFlag,
			longBreakIntervalFlag,
			workFlag,
			sinceFlag,
		},
		Action: defaultAction,
		Before: beforeAction,
		After:  afterAction,
	}

	focusApp.Flags = append(focusApp.Flags, timerFlags...)

	return focusApp
}
