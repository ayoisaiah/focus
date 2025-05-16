package app

import (
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"runtime"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/pterm/pterm"
	"github.com/urfave/cli/v2"

	"github.com/ayoisaiah/focus/internal/config"
	"github.com/ayoisaiah/focus/internal/models"
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

	cmd := exec.Command(editor, config.ConfigFilePath())

	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout

	err := cmd.Run()
	if err != nil {
		return err
	}

	return nil
}

// statsAction computes the stats for the specified time period.
func statsAction(ctx *cli.Context) error {
	sessions, db, err := sessionHelper(ctx)
	if err != nil {
		return err
	}

	opts := config.Filter(ctx)

	//nolint:govet // unkeyed fields are fine here
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
// value of --since.
func defaultAction(ctx *cli.Context) error {
	configPath := config.ConfigFilePath()

	cfg, err := config.New(
		config.WithPromptConfig(configPath),
		config.WithViperConfig(configPath),
		config.WithCLIConfig(ctx),
	)
	if err != nil {
		return err
	}

	dbClient, err := store.NewClient(config.DBFilePath())
	if err != nil {
		return err
	}

	t := timer.New(dbClient, cfg)

	p := tea.NewProgram(t)

	_, err = p.Run()

	return err
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
