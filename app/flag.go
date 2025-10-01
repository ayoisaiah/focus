package app

import "github.com/urfave/cli/v2"

var (
	sinceFlag = &cli.StringFlag{
		Name:  "since",
		Usage: "Start or add a new session in the past (e.g. '20 mins ago'). Must not overlap with any existing sessions",
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

	statsPortFlag = &cli.UintFlag{
		Name:  "port",
		Usage: "Specify the port for the statistics server",
		Value: 1111,
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
