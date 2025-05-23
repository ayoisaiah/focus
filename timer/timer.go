// Package timer operates the Focus countdown timer and handles the recovery of
// interrupted timers
package timer

import (
	"bufio"
	"context"
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

	S map[config.SessionType]struct {
		Message  string
		Duration time.Duration
	}

	// Timer represents a running timer.
	Timer struct {
		help               help.Model
		StartTime          time.Time      `json:"start_time"`
		SessionKey         time.Time      `json:"session_key"`
		PausedTime         time.Time      `json:"paused_time"`
		SoundStream        beep.Streamer  `json:"-"`
		db                 store.DB       `json:"-"`
		Opts               *config.Config `json:"opts"`
		Current            *Session
		soundForm          *huh.Form
		S                  S
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

	style struct {
		work       lipgloss.Style
		shortBreak lipgloss.Style
		longBreak  lipgloss.Style
		base       lipgloss.Style
		help       lipgloss.Style
	}
)

const (
	padding  = 2
	maxWidth = 80
)

var (
	defaultStyle  style
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
func New(dbClient store.DB, cfg *config.Config) (*Timer, error) {
	defaultStyle = style{
		work: lipgloss.NewStyle().
			Foreground(lipgloss.Color(cfg.Work.Color)).
			MarginRight(1).
			SetString(cfg.Work.Message),
		shortBreak: lipgloss.NewStyle().
			Foreground(lipgloss.Color(cfg.ShortBreak.Color)).
			MarginRight(1).
			SetString(cfg.ShortBreak.Message),
		longBreak: lipgloss.NewStyle().
			Foreground(lipgloss.Color(cfg.LongBreak.Color)).
			MarginRight(1).
			SetString(cfg.LongBreak.Message),
		base: lipgloss.NewStyle().Padding(1, 1),
		help: lipgloss.NewStyle().
			Foreground(lipgloss.Color("240")).
			MarginTop(2),
	}

	t := &Timer{
		db:       dbClient,
		Opts:     cfg,
		help:     help.New(),
		progress: progress.New(progress.WithDefaultGradient()),
		S: S{
			config.Work: {
				Duration: cfg.Work.Duration,
				Message:  cfg.Work.Message,
			},
			config.ShortBreak: {
				Duration: cfg.ShortBreak.Duration,
				Message:  cfg.Work.Message,
			},
			config.LongBreak: {
				Duration: cfg.LongBreak.Duration,
				Message:  cfg.LongBreak.Message,
			},
		},
	}

	err := t.setAmbientSound()

	return t, err
}

// sessions added with the --since flag.
func (t *Timer) Init() tea.Cmd {
	t.StartTime = time.Now()

	err := t.new()

	// If --since is used to add a completed session
	if t.Current.Completed {
		report.SessionAdded()

		return tea.Quit
	}

	if err != nil {
		return report.Fatal(err)
	}

	return t.clock.Init()
}

// new creates a new timer.
func (t *Timer) new() error {
	sess, err := t.createSession()
	if err != nil {
		return err
	}

	t.Current = sess
	t.WorkCycle = 1
	t.clock = btimer.New(t.Current.Duration)

	return nil
}

// newSession creates a new session.
func (t *Timer) newSession(
	name config.SessionType,
) *Session {
	duration := t.S[name].Duration
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

// initSession prepares the next session based on the current session type.
// It handles work cycle counting, session creation, and auto-start settings.
// Returns a tea.Cmd for initializing the timer if auto-start is enabled.
func (t *Timer) initSession() tea.Cmd {
	sessName := t.nextSession(t.Current.Name)
	newSess := t.newSession(sessName)

	if newSess.Name == config.Work && !t.Opts.Settings.AutoStartWork ||
		newSess.Name != config.Work && !t.Opts.Settings.AutoStartBreak {
		t.waitForNextSession = true
	}

	// increment or reset the work cycle accordingly
	if sessName == config.Work {
		if t.WorkCycle == t.Opts.Settings.LongBreakInterval {
			t.WorkCycle = 1
		} else {
			t.WorkCycle++
		}
	}

	if !t.waitForNextSession {
		t.Current = newSess

		t.clock = btimer.New(t.Current.Duration)
		return t.clock.Init()
	}

	return nil
}

// createSession initializes a new work session with the configured start time.
// If --since is specified, adjusts the session time accordingly.
// Returns the session and marks it completed if the end time is in the past.
func (t *Timer) createSession() (*Session, error) {
	sess := t.newSession(config.Work)

	if !t.Opts.CLI.StartTime.IsZero() {
		sess.Adjust(t.Opts.CLI.StartTime)

		if time.Now().After(sess.EndTime) {
			t.Current = sess

			err := t.persist()
			if err != nil {
				return nil, err
			}

			sess.Completed = true
		}
	}

	return sess, nil
}

func (t *Timer) postSession() error {
	// t.notify(t.Context, t.Current.Name, sessName)
	err := t.runSessionCmd(t.Opts.Settings.Cmd)
	if err != nil {
		return err
	}

	return nil
}

// persist saves the current timer and session to the database.
func (t *Timer) persist() error {
	sess := *t.Current

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
	sess := t.Current

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
	_ context.Context,
	sessName, nextSessName config.SessionType,
) {
	if !t.Opts.Notifications.Enabled {
		return
	}

	title := string(sessName + " is finished")

	msg := t.S[nextSessName].Message

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

	err := beeep.Notify(title, msg, pathToIcon)
	if err != nil {
		pterm.Error.Printfln("unable to display notification: %v", err)
	}

	if sound == "off" || sound == "" {
		return
	}

	stream, err := prepSoundStream(sound)
	if err != nil {
		pterm.Error.Printfln("unable to play sound: %v", err)
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
