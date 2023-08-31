package app

import (
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"runtime"
	"time"

	"github.com/pterm/pterm"
	"github.com/urfave/cli/v2"

	"github.com/ayoisaiah/focus/config"
	"github.com/ayoisaiah/focus/internal/ui"
	"github.com/ayoisaiah/focus/session"
	"github.com/ayoisaiah/focus/stats"
	"github.com/ayoisaiah/focus/store"
	"github.com/ayoisaiah/focus/timer"
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

func statsHelper(ctx *cli.Context) error {
	if ctx.Bool("no-color") {
		pterm.DisableColor()
	}

	cfg := config.Stats(ctx)

	dbClient, err := store.NewClient(cfg.PathToDB)
	if err != nil {
		return err
	}

	stats.Init(dbClient, cfg)

	return nil
}

func sessionHelper(ctx *cli.Context) ([]session.Session, store.DB, error) {
	if ctx.Bool("no-color") {
		pterm.DisableColor()
	}

	conf := config.Stats(ctx)

	db, err := store.NewClient(conf.PathToDB)
	if err != nil {
		return nil, nil, err
	}

	b, err := db.GetSessions(conf.StartTime, conf.EndTime, conf.Tags)
	if err != nil {
		return nil, nil, err
	}

	s, err := session.CollectionFromBytes(b)
	if err != nil {
		return nil, nil, err
	}

	return s, db, nil
}

// DeleteAction handles the delete command which is used to delete one or more
// sessions.
func DeleteAction(ctx *cli.Context) error {
	sessions, db, err := sessionHelper(ctx)
	if err != nil {
		return err
	}

	return session.Delete(sessions, db.DeleteSessions)
}

// DeleteTimerAction handles the delete-timer command.
func DeleteTimerAction(ctx *cli.Context) error {
	if ctx.Bool("no-color") {
		disableStyling()
	}

	dbClient, err := store.NewClient(config.DBFilePath())
	if err != nil {
		return err
	}

	return timer.Delete(dbClient)
}

// EditConfigAction handles the edit-config command which opens the focus config
// file in the user's editor.
func EditConfigAction(ctx *cli.Context) error {
	defaultEditor := "nano"

	if runtime.GOOS == "windows" {
		defaultEditor = "C:\\Windows\\system32\\notepad.exe"
	}

	editor := firstNonEmptyString(
		os.Getenv("VISUAL"),
		os.Getenv("EDITOR"),
		defaultEditor,
	)

	cfg := config.Timer(ctx)

	cmd := exec.Command(editor, cfg.PathToConfig)

	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout

	err := cmd.Run()
	if err != nil {
		return err
	}

	return nil
}

// EditTagsAction handles the edit-tag command which is used to edit tags for a
// session.
func EditTagsAction(ctx *cli.Context) error {
	sessions, db, err := sessionHelper(ctx)
	if err != nil {
		return err
	}

	return session.EditTags(sessions, ctx.Args().Slice(), db.UpdateSessions)
}

// ListAction handles the list command and prints a table of all the sessions
// started within a time period.
func ListAction(ctx *cli.Context) error {
	sessions, _, err := sessionHelper(ctx)
	if err != nil {
		return err
	}

	return session.List(sessions)
}

// ResumeAction handles the resume command and resumes a previously paused
// timer.
func ResumeAction(ctx *cli.Context) error {
	if ctx.Bool("no-color") {
		disableStyling()
	}

	dbClient, err := store.NewClient(config.DBFilePath())
	if err != nil {
		return err
	}

	t, sess, err := timer.Recover(dbClient, ctx)
	if err != nil {
		return err
	}

	if sess == nil {
		// Set to zero value so that a new session is initialised
		sess = &session.Session{}
	}

	ui.DarkTheme = t.Opts.DarkTheme

	return t.Run(sess)
}

// StatsAction executes the stats subcommand and outputs the stats for the
// specified time period.
func ShowAction(ctx *cli.Context) error {
	err := statsHelper(ctx)
	if err != nil {
		return err
	}

	return stats.Show()
}

// StatusAction handles the status command and prints the status of the currently
// running timer.
func StatusAction(_ *cli.Context) error {
	t := &timer.Timer{}

	return t.ReportStatus()
}

// DefaultAction handles the default action to start a new timer.
func DefaultAction(ctx *cli.Context) error {
	if ctx.Bool("no-color") {
		disableStyling()
	}

	cfg := config.Timer(ctx)

	dbClient, err := store.NewClient(cfg.PathToDB)
	if err != nil {
		return err
	}

	ui.DarkTheme = cfg.DarkTheme

	t := timer.New(dbClient, cfg)

	return t.Run(&session.Session{})
}
