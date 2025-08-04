// Package timer operates the Focus countdown timer and handles the recovery of
// interrupted timers
package timer

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/adrg/xdg"
	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/progress"
	btimer "github.com/charmbracelet/bubbles/timer"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/lipgloss"
	"github.com/faiface/beep"
	"github.com/faiface/beep/speaker"
	"github.com/gen2brain/beeep"
	"github.com/kballard/go-shellquote"
	"github.com/pterm/pterm"
	bolt "go.etcd.io/bbolt"
	bolterr "go.etcd.io/bbolt/errors"

	"github.com/ayoisaiah/focus/internal/config"
	"github.com/ayoisaiah/focus/internal/models"
	"github.com/ayoisaiah/focus/report"
	"github.com/ayoisaiah/focus/store"
)

type (
	settingsView string

	SessParams map[config.SessionType]struct {
		Message  string
		Duration time.Duration
	}

	// Timer represents a running timer.
	Timer struct {
		help               help.Model
		SoundStream        beep.Streamer  `json:"-"`
		db                 store.DB       `json:"-"`
		Opts               *config.Config `json:"opts"`
		CurrentSess        *Session
		soundForm          *huh.Form
		SessParams         SessParams
		settings           settingsView
		progress           progress.Model
		clock              btimer.Model
		WorkCycle          int `json:"work_cycle"`
		waitForNextSession bool
	}

	keymap struct {
		togglePlay key.Binding
		sound      key.Binding
		enter      key.Binding
		quit       key.Binding
		esc        key.Binding
	}
)

const (
	padding  = 2
	maxWidth = 80
)

var (
	baseStyle     = lipgloss.NewStyle().Padding(1, 1)
	defaultKeymap = keymap{
		togglePlay: key.NewBinding(
			key.WithKeys("p"),
			key.WithHelp("p", "play/pause"),
		),
		sound: key.NewBinding(
			key.WithKeys("s"),
			key.WithHelp("s", "sound"),
		),
		enter: key.NewBinding(
			key.WithKeys("enter"),
			key.WithHelp(
				"enter",
				"continue",
			),
		),
		quit: key.NewBinding(
			key.WithKeys("ctrl+c", "q"),
			key.WithHelp("q", "quit"),
		),
		esc: key.NewBinding(
			key.WithKeys("esc"),
			key.WithHelp("esc", "skip"),
		),
	}
)

var soundView settingsView = "sound"

// New creates a new timer.
func New(dbClient store.DB, cfg *config.Config) *Timer {
	t := &Timer{
		db:       dbClient,
		Opts:     cfg,
		help:     help.New(),
		progress: progress.New(),
		SessParams: SessParams{
			config.Work: {
				Duration: cfg.Work.Duration,
				Message:  cfg.Work.Message,
			},
			config.ShortBreak: {
				Duration: cfg.ShortBreak.Duration,
				Message:  cfg.ShortBreak.Message,
			},
			config.LongBreak: {
				Duration: cfg.LongBreak.Duration,
				Message:  cfg.LongBreak.Message,
			},
		},
	}

	t.progress.PercentageStyle = t.Opts.Style.Main

	return t
}

// Init initializes the timer for an interactive session, setting up the
// current session and its clock. It handles the special case where a completed
// session was added via --since reporting the successful addition and then
// exiting the application. Otherwise, for a new interactive session,
// it returns a command to initialize the timer's internal clock.
func (t *Timer) Init() tea.Cmd {
	err := t.setupSession()

	// If --since is used to add a completed session
	if t.CurrentSess.Completed {
		report.SessionAdded()

		return tea.Quit
	}

	if err != nil {
		return report.Fatal(err)
	}

	return t.clock.Init()
}

// setupSession creates a session and initializes the timer with it.
func (t *Timer) setupSession() error {
	sess := t.newSession(config.Work)

	if !t.Opts.CLI.StartTime.IsZero() {
		sess.Adjust(t.Opts.CLI.StartTime)

		if time.Now().After(sess.EndTime) {
			sess.Completed = true
			t.CurrentSess = sess

			return t.persist()
		}
	}

	t.CurrentSess = sess
	t.WorkCycle = 1
	t.clock = btimer.New(t.CurrentSess.Duration)
	t.setProgressColor(sess)

	return nil
}

// newSession creates a new session.
func (t *Timer) newSession(
	name config.SessionType,
) *Session {
	duration := t.SessParams[name].Duration
	startTime := time.Now()
	endTime := startTime.Add(duration)

	return &Session{
		Name:      name,
		Duration:  duration,
		Tags:      t.Opts.CLI.Tags,
		Completed: false,
		StartTime: startTime,
		EndTime:   endTime,
		Timeline: []Timeline{
			{
				StartTime: startTime,
				EndTime:   endTime,
			},
		},
	}
}

// nextSession determines the type of the next session based on the current
// session and work cycle count.
func (t *Timer) nextSession(current config.SessionType) config.SessionType {
	var next config.SessionType

	switch current {
	case config.Work:
		if t.WorkCycle == t.Opts.Settings.LongBreakInterval {
			next = config.LongBreak
		} else {
			next = config.ShortBreak
		}
	case config.ShortBreak, config.LongBreak:
		next = config.Work
	}

	return next
}

func (t *Timer) setProgressColor(newSess *Session) {
	if newSess.Name == config.Work {
		t.progress.FullColor = config.ColorWork.Light

		if t.Opts.Display.DarkTheme {
			t.progress.FullColor = config.ColorWork.Dark
		}
	}

	if newSess.Name == config.ShortBreak {
		t.progress.FullColor = config.ColorShortBreak.Light

		if t.Opts.Display.DarkTheme {
			t.progress.FullColor = config.ColorShortBreak.Dark
		}
	}

	if newSess.Name == config.LongBreak {
		t.progress.FullColor = config.ColorLongBreak.Light

		if t.Opts.Display.DarkTheme {
			t.progress.FullColor = config.ColorLongBreak.Dark
		}
	}
}

// immediately.
func (t *Timer) initSession() tea.Cmd {
	sessName := t.nextSession(t.CurrentSess.Name)
	newSess := t.newSession(sessName)

	t.setProgressColor(newSess)

	// increment or reset the work cycle accordingly
	if sessName == config.Work {
		if t.WorkCycle == t.Opts.Settings.LongBreakInterval {
			t.WorkCycle = 1
		} else {
			t.WorkCycle++
		}
	}

	t.CurrentSess = newSess

	t.clock = btimer.New(t.CurrentSess.Duration)

	return t.clock.Init()
}

func (t *Timer) postSession() error {
	t.notify(t.CurrentSess.Name, t.nextSession(t.CurrentSess.Name))

	err := t.runSessionCmd(t.Opts.Settings.Cmd)
	if err != nil {
		return err
	}

	return nil
}

// persist saves the current timer and session to the database.
func (t *Timer) persist() error {
	sess := *t.CurrentSess

	if sess.Name != config.Work {
		return nil
	}

	sess.UpdateEndTime(t.clock.Timedout())

	sess.Normalise()

	sessModel := sess.ToDBModel()

	m := map[time.Time]*models.Session{
		sess.StartTime: sessModel,
	}

	err := t.db.UpdateSessions(m)
	if err != nil {
		return err
	}

	return nil
}

// writeStatusFile writes the current timer status to a JSON file.
// The status includes session details, work cycle count, and timing information.
// This file is used by other processes to query the timer's current state.
func (t *Timer) writeStatusFile() error {
	sess := t.CurrentSess

	s := report.Status{
		Name:              string(sess.Name),
		WorkCycle:         t.WorkCycle,
		Tags:              sess.Tags,
		LongBreakInterval: t.Opts.Settings.LongBreakInterval,
		EndTime:           sess.EndTime,
	}

	statusFilePath := config.StatusFilePath()

	statusFile, err := os.Create(statusFilePath)
	if err != nil {
		return err
	}

	defer func() {
		ferr := statusFile.Close()
		if ferr != nil {
			err = ferr
		}
	}()

	b, err := json.Marshal(s)
	if err != nil {
		return err
	}

	writer := bufio.NewWriter(statusFile)

	_, err = writer.Write(b)
	if err != nil {
		return err
	}

	return writer.Flush()
}

// runSessionCmd executes the specified command (if any) after a session
// completes.
func (t *Timer) runSessionCmd(sessionCmd string) error {
	if sessionCmd == "" {
		return nil
	}

	cmdSlice, err := shellquote.Split(sessionCmd)
	if err != nil {
		return fmt.Errorf("unable to parse session_cmd option: %w", err)
	}

	if len(cmdSlice) == 0 {
		return nil
	}

	name := cmdSlice[0]
	args := cmdSlice[1:]

	cmd := exec.Command(name, args...)

	return cmd.Run()
}

// notify sends a desktop notification and plays a notification sound when a
// session ends if enabled.
func (t *Timer) notify(
	sessName, nextSessName config.SessionType,
) {
	if !t.Opts.Notifications.Enabled {
		return
	}

	title := string(sessName + " is finished")

	msg := t.SessParams[nextSessName].Message

	// TODO: Need to update this
	sound := t.Opts.ShortBreak.Sound

	if sessName != config.Work {
		sound = t.Opts.Work.Sound
	}

	configDir := filepath.Base(filepath.Dir(config.ConfigFilePath()))

	// pathToIcon will be an empty string if file is not found
	pathToIcon, _ := xdg.SearchDataFile(
		filepath.Join(configDir, "static", "icon.png"),
	)

	_ = beeep.Notify(title, msg, pathToIcon)

	if sound == "off" || sound == "" {
		return
	}

	stream, err := prepSoundStream(sound)
	if err != nil {
		// TODO: Log error
		return
	}

	done := make(chan bool)

	speaker.Play(beep.Seq(stream, beep.Callback(func() {
		done <- true
	})))

	<-done

	stream.Close()

	speaker.Clear()
	speaker.Close()
}

// ReportStatus reports the status of the currently running timer.
func (t *Timer) ReportStatus() error {
	dbFilePath := config.DBFilePath()
	statusFilePath := config.StatusFilePath()

	var fileMode fs.FileMode = 0o600

	_, err := bolt.Open(dbFilePath, fileMode, &bolt.Options{
		Timeout: 100 * time.Millisecond,
	})
	// This means focus is not running, so no status to report
	if err == nil {
		return nil
	}

	if !errors.Is(err, bolterr.ErrTimeout) {
		return err
	}

	fileBytes, err := os.ReadFile(statusFilePath)
	if err != nil {
		// missing file should not return an error
		return nil
	}

	var s report.Status

	err = json.Unmarshal(fileBytes, &s)
	if err != nil {
		return err
	}

	sess := &Session{
		EndTime: s.EndTime,
	}
	tr := sess.Remaining()

	if tr.T < 0 {
		return nil
	}

	var text string

	switch config.SessionType(s.Name) {
	case config.Work:
		text = fmt.Sprintf("[Work %d/%d]",
			s.WorkCycle,
			s.LongBreakInterval,
		)
	case config.ShortBreak:
		text = "[Short break]"
	case config.LongBreak:
		text = "[Long break]"
	}

	pterm.Printfln("%s: %02d:%02d", text, tr.M, tr.S)

	return nil
}
