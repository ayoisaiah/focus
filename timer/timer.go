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
	"github.com/ayoisaiah/focus/internal/pathutil"
	"github.com/ayoisaiah/focus/report"
	"github.com/ayoisaiah/focus/store"
)

type (
	settingsView string

	// Timer represents a running timer.
	Timer struct {
		help               help.Model
		StartTime          time.Time           `json:"start_time"`
		SessionKey         time.Time           `json:"session_key"`
		PausedTime         time.Time           `json:"paused_time"`
		SoundStream        beep.Streamer       `json:"-"`
		db                 store.DB            `json:"-"`
		Opts               *config.TimerConfig `json:"opts"`
		Current            *Session
		soundForm          *huh.Form
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
func New(dbClient store.DB, cfg *config.TimerConfig) (*Timer, error) {
	defaultStyle = style{
		work: lipgloss.NewStyle().
			Foreground(lipgloss.Color(cfg.WorkColor)).
			MarginRight(1).
			SetString(cfg.Message[config.Work]),
		shortBreak: lipgloss.NewStyle().
			Foreground(lipgloss.Color(cfg.ShortBreakColor)).
			MarginRight(1).
			SetString(cfg.Message[config.ShortBreak]),
		longBreak: lipgloss.NewStyle().
			Foreground(lipgloss.Color(cfg.LongBreakColor)).
			MarginRight(1).
			SetString(cfg.Message[config.LongBreak]),
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
		soundForm: huh.NewForm(
			huh.NewGroup(
				huh.NewSelect[string]().
					Key("sound").
					Options(huh.NewOptions(soundOpts...)...).
					Title("Select ambient sound"),
			),
		),
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

	return tea.Batch(t.clock.Init(), t.soundForm.Init())
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
	name config.SessType,
) *Session {
	duration := t.Opts.Duration[name]
	startTime := time.Now()
	endTime := startTime.Add(duration)

	return &Session{
		Name:      name,
		Duration:  duration,
		Tags:      t.Opts.Tags,
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
func (t *Timer) nextSession(current config.SessType) config.SessType {
	var next config.SessType

	switch current {
	case config.Work:
		if t.WorkCycle == t.Opts.LongBreakInterval {
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
	t.Current = t.newSession(sessName)

	if t.Current.Name == config.Work && !t.Opts.AutoStartWork ||
		t.Current.Name != config.Work && !t.Opts.AutoStartBreak {
		t.waitForNextSession = true
	}

	// increment or reset the work cycle accordingly
	if sessName == config.Work {
		if t.WorkCycle == t.Opts.LongBreakInterval {
			t.WorkCycle = 1
		} else {
			t.WorkCycle++
		}
	}

	if !t.waitForNextSession {
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

	if t.Opts.Since != "" {
		sess.Adjust(t.Opts.StartTime)

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
	err := t.runSessionCmd(t.Opts.SessionCmd)
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
		LongBreakInterval: t.Opts.LongBreakInterval,
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
	sessName, nextSessName config.SessType,
) {
	if !t.Opts.Notify {
		return
	}

	title := string(sessName + " is finished")

	msg := t.Opts.Message[nextSessName]

	sound := t.Opts.BreakSound

	if sessName != config.Work {
		sound = t.Opts.WorkSound
	}

	configDir := filepath.Base(filepath.Dir(t.Opts.PathToConfig))

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
	dbFilePath := pathutil.DBFilePath()
	statusFilePath := pathutil.StatusFilePath()

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

	switch config.SessType(s.Name) {
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
