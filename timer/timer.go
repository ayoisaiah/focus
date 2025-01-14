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
	"log/slog"
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
	"github.com/urfave/cli/v2"
	bolt "go.etcd.io/bbolt"

	"github.com/ayoisaiah/focus/config"
	"github.com/ayoisaiah/focus/internal/models"
	"github.com/ayoisaiah/focus/internal/timeutil"
	"github.com/ayoisaiah/focus/store"
)

const (
	padding  = 2
	maxWidth = 80
)

var (
	defaultStyle  style
	defaultKeymap keymap
)

type settingsView string

var soundView settingsView = "sound"

type (
	// Timer represents a running timer.
	Timer struct {
		clock              btimer.Model
		soundForm          *huh.Form
		db                 store.DB            `json:"-"`
		Opts               *config.TimerConfig `json:"opts"`
		SoundStream        beep.Streamer       `json:"-"`
		PausedTime         time.Time           `json:"paused_time"`
		StartTime          time.Time           `json:"start_time"`
		SessionKey         time.Time           `json:"session_key"`
		WorkCycle          int                 `json:"work_cycle"`
		Current            *Session
		Context            context.Context
		waitForNextSession bool
		help               help.Model
		progress           progress.Model
		settings           settingsView
	}

	keymap struct {
		togglePlay key.Binding
		sound      key.Binding
		beginSess  key.Binding
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

	// Status represents the status of a running timer.
	Status struct {
		EndTime           time.Time       `json:"end_date"`
		Name              config.SessType `json:"name"`
		Tags              []string        `json:"tags"`
		WorkCycle         int             `json:"work_cycle"`
		LongBreakInterval int             `json:"long_break_interval"`
	}
)

// Persist saves the current timer and session to the database.
func (t *Timer) Persist(c context.Context, sess *Session) error {
	if sess.Name != config.Work {
		return nil
	}

	sess.Normalise(c)

	sessModel := sess.ToDBModel()

	m := map[time.Time]*models.Session{
		sess.StartTime: sessModel,
	}

	logMsg := "syncing in-progress session to db"
	if sess.Completed {
		logMsg = "syncing completed session to db"
	}

	slog.InfoContext(c, logMsg, slog.Any("session", sessModel))

	err := t.db.UpdateSessions(m)
	if err != nil {
		return err
	}

	t.SessionKey = sess.StartTime

	timer := models.Timer{
		Opts:       t.Opts,
		PausedTime: time.Now(),
		SessionKey: t.SessionKey,
		WorkCycle:  t.WorkCycle,
		StartTime:  t.StartTime,
	}

	slog.InfoContext(c, "syncing timer to db", slog.Any("timer", timer))

	err = t.db.UpdateTimer(&timer)
	if err != nil {
		return err
	}

	return nil
}

// runSessionCmd executes the specified command.
func (t *Timer) runSessionCmd(c context.Context, sessionCmd string) error {
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

	slog.InfoContext(
		c,
		"executing session command",
		slog.Any("name", name),
		slog.Any("args", args),
	)

	cmd := exec.Command(name, args...)

	return cmd.Run()
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

	if !errors.Is(err, bolt.ErrDatabaseOpen) &&
		!errors.Is(err, bolt.ErrTimeout) {
		return err
	}

	fileBytes, err := os.ReadFile(statusFilePath)
	if err != nil {
		// missing file should not return an error
		return nil
	}

	var s Status

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

	switch s.Name {
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

func (t *Timer) writeStatusFile(
	sess *Session,
) error {
	s := Status{
		Name:              sess.Name,
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

// notify sends a desktop notification and plays a notification sound.
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

// formatTimeRemaining returns the remaining time formatted as "MM:SS".
func (t *Timer) formatTimeRemaining() string {
	m, s := timeutil.SecsToMinsAndSecs(t.clock.Timeout.Seconds())

	return fmt.Sprintf(
		"%s:%s", fmt.Sprintf("%02d", m), fmt.Sprintf("%02d", s),
	)
}

// nextSession retrieves the name of the next session.
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

func (t *Timer) update() {
	if int(t.clock.Timeout.Seconds())%60 == 0 {
		go func(sess *Session) {
			s := *sess

			s.UpdateEndTime()

			_ = t.Persist(t.Context, &s)
		}(t.Current)
	}
}

func (t *Timer) endSession() error {
	t.Current.UpdateEndTime()
	t.Current.Completed = true

	err := t.Persist(t.Context, t.Current)
	if err != nil {
		return err
	}

	sessName := t.nextSession(t.Current.Name)

	t.notify(t.Context, t.Current.Name, sessName)

	err = t.runSessionCmd(t.Context, t.Opts.SessionCmd)
	if err != nil {
		return err
	}

	if sessName == config.Work && !t.Opts.AutoStartWork ||
		sessName != config.Work && !t.Opts.AutoStartBreak {
		t.waitForNextSession = true
	}

	t.Current = t.NewSession(sessName, time.Now())

	return nil
}

// NewSession initialises a new session.
func (t *Timer) NewSession(
	name config.SessType,
	startTime time.Time,
) *Session {
	sess := &Session{
		Name:      name,
		Duration:  t.Opts.Duration[name],
		Tags:      t.Opts.Tags,
		Completed: false,
		StartTime: startTime,
		Timeline: []Timeline{
			{
				StartTime: startTime,
			},
		},
	}

	sess.SetEndTime()

	// increment or reset the work cycle accordingly
	if name == config.Work {
		if t.WorkCycle == t.Opts.LongBreakInterval {
			t.WorkCycle = 1
		} else {
			t.WorkCycle++
		}
	}

	return sess
}

// overrideOptsOnResume overrides timer options if specified through
// command-line arguments.
func (t *Timer) overrideOptsOnResume(ctx *cli.Context) error {
	if ctx.Bool("disable-notification") {
		t.Opts.Notify = false
	}

	ambientSound := ctx.String("sound")
	if ambientSound != "" {
		if ambientSound == config.SoundOff {
			t.Opts.AmbientSound = ""
		} else {
			t.Opts.AmbientSound = ambientSound

			err := t.setAmbientSound()
			if err != nil {
				return err
			}
		}
	}

	breakSound := ctx.String("break-sound")
	if breakSound != "" {
		if breakSound == config.SoundOff {
			t.Opts.BreakSound = ""
		} else {
			t.Opts.BreakSound = breakSound
		}
	}

	workSound := ctx.String("work-sound")
	if workSound != "" {
		if workSound == config.SoundOff {
			t.Opts.WorkSound = ""
		} else {
			t.Opts.WorkSound = workSound
		}
	}

	if ctx.String("session-cmd") != "" {
		t.Opts.SessionCmd = ctx.String("session-cmd")
	}

	return nil
}

// Delete permanently removes one or more paused timers.
func Delete(db store.DB) error {
	pausedTimers, pausedSessions, err := getTimerSessions(db)
	if err != nil {
		return err
	}

	printPausedTimers(pausedTimers, pausedSessions)

	return selectAndDeleteTimers(db, pausedTimers)
}

func newSessionFromDB(s *models.Session) *Session {
	sess := &Session{}

	sess.StartTime = s.StartTime
	sess.EndTime = s.EndTime
	sess.Name = s.Name
	sess.Tags = s.Tags
	sess.Duration = s.Duration
	sess.Completed = s.Completed

	for _, v := range s.Timeline {
		timeline := Timeline{
			StartTime: v.StartTime,
			EndTime:   v.EndTime,
		}

		sess.Timeline = append(sess.Timeline, timeline)
	}

	return sess
}

// Recover attempts to recover an interrupted timer.
func Recover(
	db store.DB,
	ctx *cli.Context,
) (*Timer, *Session, error) {
	pausedTimers, pausedSessions, err := getTimerSessions(db)
	if err != nil {
		return nil, nil, err
	}

	var selectedTimer *models.Timer

	if ctx.Bool("select") {
		printPausedTimers(pausedTimers, pausedSessions)

		selectedTimer, err = selectPausedTimer(pausedTimers)
		if err != nil {
			return nil, nil, err
		}
	} else {
		selectedTimer = pausedTimers[0]
	}

	s, err := db.GetSession(selectedTimer.SessionKey)
	if err != nil {
		return nil, nil, err
	}

	t, err := New(db, selectedTimer.Opts)
	if err != nil {
		return nil, nil, err
	}

	t.PausedTime = selectedTimer.PausedTime
	t.StartTime = selectedTimer.StartTime
	t.SessionKey = selectedTimer.SessionKey
	t.WorkCycle = selectedTimer.WorkCycle

	sess := newSessionFromDB(s)

	sess.SetEndTime()

	err = t.overrideOptsOnResume(ctx)

	return t, sess, err
}

func (t *Timer) Init() tea.Cmd {
	t.StartTime = time.Now()
	t.clock = btimer.New(t.Current.Duration)

	if t.Opts.AmbientSound != "" {
		if t.Current.Name == config.Work || t.Opts.PlaySoundOnBreak {
			speaker.Clear()
			speaker.Play(t.SoundStream)
		} else {
			speaker.Clear()
		}
	}

	return tea.Batch(t.clock.Init(), t.soundForm.Init())

	// return t.clock.Init()
}

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

	defaultKeymap = keymap{
		togglePlay: key.NewBinding(
			key.WithKeys("p"),
			key.WithHelp("p", "play/pause"),
		),
		sound: key.NewBinding(
			key.WithKeys("s"),
			key.WithHelp("s", "sound"),
		),
		beginSess: key.NewBinding(
			key.WithKeys("enter"),
			key.WithHelp(
				"enter",
				"Press ENTER to start the next session",
			),
		),
		quit: key.NewBinding(
			key.WithKeys("ctrl+c", "q"),
			key.WithHelp("q", "quit"),
		),
		esc: key.NewBinding(
			key.WithKeys("esc"),
			key.WithHelp("esc", "cancel"),
		),
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
