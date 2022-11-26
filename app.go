package focus

import (
	"bytes"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"runtime"
	"time"

	"github.com/ayoisaiah/focus/config"
	"github.com/ayoisaiah/focus/internal/color"
	"github.com/ayoisaiah/focus/internal/session"
	"github.com/ayoisaiah/focus/stats"
	"github.com/ayoisaiah/focus/store"
	"github.com/ayoisaiah/focus/timer"
	"github.com/pterm/pterm"
	"github.com/urfave/cli/v2"
)

const (
	envUpdateNotifier = "FOCUS_UPDATE_NOTIFIER"
	envNoColor        = "NO_COLOR"
	envFocusNoColor   = "FOCUS_NO_COLOR"
)

// firstNonEmptyString returns its first non-empty argument, or "" if all
// arguments are empty.
func firstNonEmptyString(ss ...string) string {
	for _, s := range ss {
		if s != "" {
			return s
		}
	}

	return ""
}

func defaultAction(ctx *cli.Context) error {
	if ctx.Bool("no-color") {
		disableStyling()
	}

	cfg := config.GetTimer(ctx)

	dbClient, err := store.NewClient(cfg.PathToDB)
	if err != nil {
		return err
	}

	color.DarkTheme = cfg.DarkTheme

	timer.Init(dbClient, cfg)

	return timer.Run(&session.Session{})
}

func listAction(ctx *cli.Context) error {
	err := statsAction(ctx)
	if err != nil {
		return err
	}

	return stats.List()
}

func editTagsAction(ctx *cli.Context) error {
	err := statsAction(ctx)
	if err != nil {
		return err
	}

	return stats.EditTags(ctx.Args().Slice())
}

func deleteAction(ctx *cli.Context) error {
	err := statsAction(ctx)
	if err != nil {
		return err
	}

	return stats.Delete()
}

func editConfigAction(ctx *cli.Context) error {
	defaultEditor := "vi"

	if runtime.GOOS == "windows" {
		defaultEditor = "C:\\Windows\\system32\\notepad.exe"
	}

	editor := firstNonEmptyString(
		os.Getenv("VISUAL"),
		os.Getenv("EDITOR"),
		defaultEditor,
	)

	cfg := config.GetTimer(ctx)

	cmd := exec.Command(editor, cfg.PathToConfig)

	var stderr bytes.Buffer

	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout

	err := cmd.Run()
	if err != nil {
		fmt.Println(stderr.String())
		return err
	}

	return nil
}

func resumeAction(ctx *cli.Context) error {
	if ctx.Bool("no-color") {
		disableStyling()
	}

	cfg := config.GetTimer(ctx)

	color.DarkTheme = cfg.DarkTheme

	dbClient, err := store.NewClient(cfg.PathToDB)
	if err != nil {
		return err
	}

	sess, err := timer.Recover(dbClient, ctx)
	if err != nil {
		return err
	}

	return timer.Run(sess)
}

func statsAction(ctx *cli.Context) error {
	if ctx.Bool("no-color") {
		pterm.DisableColor()
	}

	cfg := config.GetStats(ctx)

	dbClient, err := store.NewClient(cfg.PathToDB)
	if err != nil {
		return err
	}

	stats.Init(dbClient, cfg)

	return nil
}

func showAction(ctx *cli.Context) error {
	err := statsAction(ctx)
	if err != nil {
		return err
	}

	return stats.Show()
}

func init() {
	// Override the default help template
	cli.AppHelpTemplate = helpText()

	// Override the default version printer
	oldVersionPrinter := cli.VersionPrinter
	cli.VersionPrinter = func(c *cli.Context) {
		oldVersionPrinter(c)
		fmt.Printf(
			"https://github.com/ayoisaiah/focus/releases/%s\n",
			c.App.Version,
		)

		if _, found := os.LookupEnv(envUpdateNotifier); found {
			checkForUpdates(c.App)
		}
	}

	// Disable colour output if NO_COLOR is set
	if _, exists := os.LookupEnv(envNoColor); exists {
		disableStyling()
	}

	// Disable colour output if FOCUS_NO_COLOR is set
	if _, exists := os.LookupEnv(envFocusNoColor); exists {
		disableStyling()
	}

	pterm.Error.MessageStyle = pterm.NewStyle(pterm.FgRed)
	pterm.Error.Prefix = pterm.Prefix{
		Text:  "ERROR",
		Style: pterm.NewStyle(pterm.BgRed, pterm.FgBlack),
	}
}

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

// checkForUpdates alerts the user if there is
// an updated version of Focus from the one currently installed.
func checkForUpdates(app *cli.App) {
	spinner, _ := pterm.DefaultSpinner.Start("Checking for updates...")
	c := http.Client{Timeout: 10 * time.Second}

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
		text := pterm.Sprintf(
			"Congratulations, you are using the latest version of %s",
			app.Name,
		)
		spinner.Success(text)
	} else {
		pterm.Warning.Prefix = pterm.Prefix{
			Text:  "UPDATE AVAILABLE",
			Style: pterm.NewStyle(pterm.BgYellow, pterm.FgBlack),
		}
		pterm.Warning.Printfln("A new release of focus is available: %s at %s", version, resp.Request.URL.String())
	}
}

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
		&cli.StringFlag{
			Name:    "tag",
			Aliases: []string{"t"},
			Usage:   "Match only sessions with a specific tag",
		},
	}

	timerFlags := []cli.Flag{
		&cli.StringFlag{
			Name:    "session-cmd",
			Aliases: []string{"cmd"},
			Usage:   "Execute an arbitrary command after each session",
		},
		&cli.BoolFlag{
			Name:    "disable-notification",
			Aliases: []string{"d"},
			Usage:   "Disable the system notification that appears after a session is completed",
		},
		globalFlags["no-color"],
		&cli.StringFlag{
			Name:  "sound",
			Usage: "Play ambient sounds continuously during a session. Default options: coffee_shop, fireplace, rain,\n\t\t\t\twind, summer_night, playground. Disable sound by setting to 'off'",
		},
		&cli.StringFlag{
			Name:    "tag",
			Aliases: []string{"t"},
			Usage:   "Add comma-delimited tags to a session",
		},
	}

	app := &cli.App{
		Name: "focus",
		Authors: []*cli.Author{
			{
				Name:  "Ayooluwa Isaiah",
				Email: "ayo@freshman.tech",
			},
		},
		Usage:                "Focus is a cross-platform productivity timer for the command-line. It is based on the Pomodoro Technique,\n\t\ta time management method developed by Francesco Cirillo in the late 1980s.",
		UsageText:            "[COMMAND] [OPTIONS]",
		Version:              "v1.2.0",
		EnableBashCompletion: true,
		Commands: []*cli.Command{
			{
				Name:   "resume",
				Usage:  "Resume a previously interrupted session",
				Flags:  timerFlags,
				Action: resumeAction,
			},
			{
				Name:   "edit-config",
				Usage:  "Edit the configuration file",
				Action: editConfigAction,
			},
			{
				Name:   "list",
				Usage:  "List all the sessions within the specified time period",
				Action: listAction,
				Flags:  append(statsFlags, globalFlags["no-color"]),
			},
			{
				Name:   "edit-tags",
				Usage:  "Edit the tags for a set of focus sessions",
				Action: editTagsAction,
				Flags:  append(statsFlags, globalFlags["no-color"]),
			},
			{
				Name:   "delete",
				Usage:  "Permanently delete the all sessions within the specified time period. Will prompt before deleting.",
				Action: deleteAction,
				Flags:  append(statsFlags, globalFlags["no-color"]),
			},
			{
				Name:   "stats",
				Usage:  "Track your progress with detailed statistics reporting. Defaults to a reporting period of 7 days",
				Action: showAction,
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
		Action: defaultAction,
	}

	app.Flags = append(app.Flags, timerFlags...)

	return app
}
