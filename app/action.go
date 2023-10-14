package app

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"runtime"
	"time"

	"github.com/pterm/pterm"
	"github.com/urfave/cli/v2"

	"github.com/ayoisaiah/focus/config"
	"github.com/ayoisaiah/focus/internal/models"
	"github.com/ayoisaiah/focus/internal/timeutil"
	"github.com/ayoisaiah/focus/internal/ui"
	"github.com/ayoisaiah/focus/stats"
	"github.com/ayoisaiah/focus/store"
	"github.com/ayoisaiah/focus/timer"
)

const (
	envUpdateNotifier = "FOCUS_UPDATE_NOTIFIER"
	envNoColor        = "NO_COLOR"
	envFocusNoColor   = "FOCUS_NO_COLOR"
)

var errNoSessionOverlap = errors.New(
	"new sessions cannot overlap with existing ones",
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

func sessionHelper(ctx *cli.Context) ([]*models.Session, store.DB, error) {
	conf := config.Filter(ctx)

	db, err := store.NewClient(config.DBFilePath())
	if err != nil {
		return nil, nil, err
	}

	sessions, err := db.GetSessions(conf.StartTime, conf.EndTime, conf.Tags)
	if err != nil {
		return nil, nil, err
	}

	return sessions, db, nil
}

// deleteAction handles the delete command which deletes one or more
// sessions.
func deleteAction(ctx *cli.Context) error {
	sessions, db, err := sessionHelper(ctx)
	if err != nil {
		return err
	}

	return delSessions(db, sessions)
}

// deleteTimerAction handles the delete-timer command for initiating the
// deletion of one or more paused timers.
func deleteTimerAction(ctx *cli.Context) error {
	db, err := store.NewClient(config.DBFilePath())
	if err != nil {
		return err
	}

	if ctx.Bool("all") {
		return db.DeleteAllTimers()
	}

	return timer.Delete(db)
}

// editConfigAction handles the edit-config command which opens the focus config
// file in the user's default text editor.
func editConfigAction(ctx *cli.Context) error {
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

// editTagsAction handles the edit-tag command which edits tags for the
// specified sessions.
func editTagsAction(ctx *cli.Context) error {
	sessions, db, err := sessionHelper(ctx)
	if err != nil {
		return err
	}

	return editTags(db, sessions, ctx.Args().Slice())
}

// listAction handles the list command and prints a table of all the sessions
// started within a time period.
func listAction(ctx *cli.Context) error {
	sessions, _, err := sessionHelper(ctx)
	if err != nil {
		return err
	}

	if ctx.Bool("json") {
		b, err := json.Marshal(sessions)
		if err != nil {
			return err
		}

		pterm.Println(string(b))

		return nil
	}

	return listSessions(sessions)
}

// resumeAction handles the resume command and resumes a previously paused
// timer.
func resumeAction(ctx *cli.Context) error {
	dbClient, err := store.NewClient(config.DBFilePath())
	if err != nil {
		return err
	}

	t, sess, err := timer.Recover(dbClient, ctx)
	if err != nil {
		return err
	}

	if ctx.Bool("reset") {
		sess = t.NewSession(config.Work, time.Now())
		t.WorkCycle = 1
	}

	if sess == nil || sess.Completed {
		sess = t.NewSession(config.Work, time.Now())
	}

	ui.DarkTheme = t.Opts.DarkTheme

	return t.Run(sess)
}

// statsAction computes the stats for the specified time period.
func statsAction(ctx *cli.Context) error {
	sessions, db, err := sessionHelper(ctx)
	if err != nil {
		return err
	}

	opts := config.Filter(ctx)

	s := &stats.Stats{
		Opts: stats.Opts{
			*opts,
		},
		DB: db,
	}

	s.Compute(sessions)

	if ctx.Bool("json") {
		b, err := s.ToJSON()
		if err != nil {
			return err
		}

		fmt.Println(string(b))

		return nil
	}

	return s.Server(ctx.Uint("port"))
}

// statusAction handles the status command and prints the status of the currently
// running timer.
func statusAction(_ *cli.Context) error {
	t := &timer.Timer{}

	return t.ReportStatus()
}

// defaultAction starts a timer or adds a completed session depending on the
// value of --start.
func defaultAction(ctx *cli.Context) error {
	cfg := config.Timer(ctx)

	dbClient, err := store.NewClient(cfg.PathToDB)
	if err != nil {
		return err
	}

	ui.DarkTheme = cfg.DarkTheme

	t, err := timer.New(dbClient, cfg)
	if err != nil {
		return err
	}

	sinceFlag := ctx.String("since")

	startTime := time.Now()

	if sinceFlag != "" {
		var err error

		startTime, err = timeutil.FromStr(sinceFlag)
		if err != nil {
			return err
		}
	}

	sess := t.NewSession(config.Work, startTime)

	if sinceFlag != "" {
		sessions, err := dbClient.GetSessions(startTime, time.Now(), []string{})
		if err != nil {
			return err
		}

		if len(sessions) > 0 {
			return errNoSessionOverlap
		}
	}

	if time.Now().After(sess.EndTime) {
		sess.Completed = true

		err := t.Persist(sess)
		if err != nil {
			return err
		}

		pterm.Info.Println("session added successfully")

		return nil
	}

	return t.Run(sess)
}

func beforeAction(ctx *cli.Context) error {
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

	pterm.Error.MessageStyle = pterm.NewStyle(pterm.FgRed)
	pterm.Error.Prefix = pterm.Prefix{
		Text:  "ERROR",
		Style: pterm.NewStyle(pterm.BgRed, pterm.FgBlack),
	}

	// Disable colour output if NO_COLOR is set
	if _, exists := os.LookupEnv(envNoColor); exists {
		disableStyling()
	}

	// Disable colour output if FOCUS_NO_COLOR is set
	if _, exists := os.LookupEnv(envFocusNoColor); exists {
		disableStyling()
	}

	if ctx.Bool("no-color") {
		disableStyling()
	}

	return nil
}
