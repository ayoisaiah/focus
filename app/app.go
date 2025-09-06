package app

import (
	"github.com/pterm/pterm"
	"github.com/urfave/cli/v2"

	"github.com/ayoisaiah/focus/internal/config"
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
	focusApp := &cli.App{
		Name: "focus",
		Authors: []*cli.Author{
			{
				Name:  "Ayooluwa Isaiah",
				Email: "ayo@freshman.tech",
			},
		},
		Usage: `
		Focus is a cross-platform productivity timer for the command-line. It is 
		based on the Pomodoro Technique, a time management method developed by 
		Francesco Cirillo in the late 1980s.`,
		UsageText:            "[COMMAND] [OPTIONS]",
		Version:              config.Version,
		EnableBashCompletion: true,
		Commands: []*cli.Command{
			{
				Name:   "edit-config",
				Usage:  "Edit the configuration file",
				Action: editConfigAction,
			},
			{
				Name: "stats",
				Usage: `
				Track your progress with detailed statistics reporting. Defaults to a 
				reporting period of 7 days`,
				Action: statsAction,
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
			disableNotificationFlag,
			soundFlag,
			soundOnBreakFlag,
			workSoundFlag,
			breakSoundFlag,
			sessionCmdFlag,
			addTagFlag,
			strictFlag,
			flowTimerFlag,
		},
		Action: defaultAction,
		Before: beforeAction,
	}

	return focusApp
}
