// Package timer operates the Focus countdown timer and handles the recovery of
// interrupted timers
package timer

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/adrg/xdg"
	"github.com/faiface/beep"
	"github.com/faiface/beep/flac"
	"github.com/faiface/beep/mp3"
	"github.com/faiface/beep/speaker"
	"github.com/faiface/beep/vorbis"
	"github.com/faiface/beep/wav"
	"github.com/gen2brain/beeep"
	"github.com/kballard/go-shellquote"
	"github.com/pterm/pterm"
	"github.com/urfave/cli/v2"

	bolt "go.etcd.io/bbolt"

	"github.com/ayoisaiah/focus/config"
	"github.com/ayoisaiah/focus/internal/models"
	"github.com/ayoisaiah/focus/internal/static"
	"github.com/ayoisaiah/focus/internal/ui"
	"github.com/ayoisaiah/focus/store"
)

var (
	errInvalidSoundFormat = errors.New(
		"sound file must be in mp3, ogg, flac, or wav format",
	)

	errInvalidInput = errors.New(
		"invalid input: only comma-separated numbers are accepted",
	)
)

const sessionSettled = "settled"

// Settled fulfills the os.Signal interface.
type Settled struct{}

func (s Settled) String() string {
	return sessionSettled
}

func (s Settled) Signal() {}

// Timer represents a running timer.
type Timer struct {
	db          store.DB            `json:"-"`
	Opts        *config.TimerConfig `json:"opts"`
	SoundStream beep.Streamer       `json:"-"`
	PausedTime  time.Time           `json:"paused_time"`
	StartTime   time.Time           `json:"start_time"`
	SessionKey  time.Time           `json:"session_key"`
	WorkCycle   int                 `json:"work_cycle"`
}

// Status represents the status of a running timer.
type Status struct {
	EndTime           time.Time       `json:"end_date"`
	Name              config.SessType `json:"name"`
	Tags              []string        `json:"tags"`
	WorkCycle         int             `json:"work_cycle"`
	LongBreakInterval int             `json:"long_break_interval"`
}

// Persist saves the current timer and session to the database.
func (t *Timer) Persist(sess *Session) error {
	if sess.Name != config.Work {
		return nil
	}

	sess.Normalise()

	m := map[time.Time]*models.Session{
		sess.StartTime: sess.ToDBModel(),
	}

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

	err = t.db.UpdateTimer(&timer)
	if err != nil {
		return err
	}

	return nil
}

// runSessionCmd executes the specified command.
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

// printSession writes the details of the current session to stdout.
func (t *Timer) printSession(
	sess *Session,
) {
	var text string

	switch sess.Name {
	case config.Work:
		total := t.Opts.LongBreakInterval

		text = fmt.Sprintf(
			ui.Green("[Work %d/%d]"),
			t.WorkCycle,
			total,
		) + ": " + t.Opts.Message[config.Work]
	case config.ShortBreak:
		text = ui.Blue(
			"[Short break]",
		) + ": " + t.Opts.Message[config.ShortBreak]
	case config.LongBreak:
		text = ui.Magenta(
			"[Long break]",
		) + ": " + t.Opts.Message[config.LongBreak]
	}

	var timeFormat string
	if t.Opts.TwentyFourHourClock {
		timeFormat = "15:04:05"
	} else {
		timeFormat = "03:04:05 PM"
	}

	var tags string
	if len(sess.Tags) > 0 {
		tags = " >>> " + strings.Join(sess.Tags, " | ")
	}

	fmt.Fprintf(
		os.Stdout,
		"%s (until %s)%s\n",
		text,
		ui.Highlight(sess.EndTime.Format(timeFormat)),
		tags,
	)
}

// notify sends a desktop notification and plays a notification sound.
func (t *Timer) notify(sessName, nextSessName config.SessType) {
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

	stream, err := t.prepSoundStream(sound)
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

// handleInterruption saves the current state of the timer whenever it is
// interrupted by pressing Ctrl-C.
func (t *Timer) handleInterruption(sess *Session) chan os.Signal {
	c := make(chan os.Signal, 1)

	signal.Notify(c, os.Interrupt, syscall.SIGTERM)

	go func() {
		s := <-c
		// a settled signal indicates that the session was completed normally
		if s.String() == sessionSettled {
			return
		}

		exitFunc := func(err error) {
			pterm.Error.Printfln("unable to save interrupted timer: %v", err)
			os.Exit(1)
		}

		sess.UpdateEndTime()

		_ = os.Remove(config.StatusFilePath())

		err := t.Persist(sess)
		if err != nil {
			exitFunc(err)
		}

		_ = t.db.Close()

		os.Exit(0)
	}()

	return c
}

// prepSoundStream returns an audio stream for the specified sound.
func (t *Timer) prepSoundStream(sound string) (beep.StreamSeekCloser, error) {
	var (
		f      fs.File
		err    error
		stream beep.StreamSeekCloser
		format beep.Format
	)

	ext := filepath.Ext(sound)
	// without extension, treat as OGG file
	if ext == "" {
		sound += ".ogg"

		f, err = static.Files.Open(static.FilePath(sound))
		if err != nil {
			// TODO: Update error
			return nil, err
		}
	} else {
		f, err = os.Open(sound)
		// TODO: Update error
		if err != nil {
			return nil, err
		}
	}

	defer func() {
		_ = f.Close()
	}()

	ext = filepath.Ext(sound)

	switch ext {
	case ".ogg":
		stream, format, err = vorbis.Decode(f)
	case ".mp3":
		stream, format, err = mp3.Decode(f)
	case ".flac":
		stream, format, err = flac.Decode(f)
	case ".wav":
		stream, format, err = wav.Decode(f)
	default:
		return nil, errInvalidSoundFormat
	}

	if err != nil {
		return nil, err
	}

	bufferSize := 10

	err = speaker.Init(
		format.SampleRate,
		format.SampleRate.N(time.Duration(int(time.Second)/bufferSize)),
	)
	if err != nil {
		return nil, err
	}

	err = stream.Seek(0)
	if err != nil {
		return nil, err
	}

	return stream, nil
}

// wait releases the handle to the datastore and waits for user input
// before locking the datastore once more. This allows a new instance of Focus
// to be launched in another terminal.
func (t *Timer) wait(sessName config.SessType) error {
	// only block if auto start options are disabled
	if sessName != config.Work && t.Opts.AutoStartBreak ||
		sessName == config.Work && t.Opts.AutoStartWork {
		return nil
	}

	c := make(chan os.Signal, 1)

	signal.Notify(c, os.Interrupt, syscall.SIGTERM)

	go func() {
		s := <-c

		if s.String() == sessionSettled {
			return
		}

		os.Exit(0)
	}()

	err := t.db.Close()
	if err != nil {
		return err
	}

	reader := bufio.NewReader(os.Stdin)

	fmt.Fprint(os.Stdout, "\033[s")
	fmt.Fprint(os.Stdout, "Press ENTER to start the next session")

	// Block until user input before beginning next session
	_, err = reader.ReadString('\n')

	c <- Settled{}

	if errors.Is(err, io.EOF) {
		return nil
	} else if err != nil {
		return err
	}

	fmt.Print("\033[u\033[K")

	return t.db.Open()
}

// countdown prints the time remaining until the end of the current session.
func (t *Timer) countdown(tr Remainder) {
	fmt.Fprintf(
		os.Stdout,
		"\r🕒%s:%s",
		pterm.Yellow(fmt.Sprintf("%02d", tr.M)),
		pterm.Yellow(fmt.Sprintf("%02d", tr.S)),
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

// start launches or resumes a session and blocks until its completion.
func (t *Timer) start(sess *Session) {
	t.printSession(sess)

	go func() {
		_ = t.writeStatusFile(sess)
	}()

	fmt.Fprint(os.Stdout, "\033[s")

	remainder := sess.Remaining()

	t.countdown(remainder)

	var counter int

	ticker := time.NewTicker(time.Second)
	for range ticker.C {
		fmt.Fprint(os.Stdout, "\033[u\033[K")

		remainder = sess.Remaining()

		// save the timer once every minute to facilitate recovery on sudden
		// shutdowns (e.g. process killed, system crashes etc)
		if counter%60 == 0 {
			s := *sess

			s.UpdateEndTime()

			_ = t.Persist(&s)
		}

		counter++

		if remainder.T <= 0 {
			fmt.Printf("Session completed!\n\n")

			sess.Completed = true

			return
		}

		t.countdown(remainder)
	}

	ticker.Stop()
}

// NewSession initialises a new session.
func (t *Timer) NewSession(name config.SessType, startTime time.Time) *Session {
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

// Run begins the timer and loops forever, alternating between work and
// break sessions until it is terminated with Ctrl-C or a maximum number of work
// sessions is reached.
func (t *Timer) Run(sess *Session) error {
	sessName := config.Work

	for {
		c := t.handleInterruption(sess)

		if t.Opts.AmbientSound != "" {
			if sess.Name == config.Work || t.Opts.PlaySoundOnBreak {
				speaker.Clear()
				speaker.Play(t.SoundStream)
			} else {
				speaker.Clear()
			}
		}

		t.start(sess)

		c <- Settled{}

		err := t.Persist(sess)
		if err != nil {
			return err
		}

		sessName = t.nextSession(sessName)

		t.notify(sess.Name, sessName)

		err = t.runSessionCmd(t.Opts.SessionCmd)
		if err != nil {
			return err
		}

		err = t.wait(sessName)
		if err != nil {
			return err
		}

		sess = t.NewSession(sessName, time.Now())
	}
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

func (t *Timer) setAmbientSound() error {
	var infiniteStream beep.Streamer

	if t.Opts.AmbientSound != "" {
		stream, err := t.prepSoundStream(t.Opts.AmbientSound)
		if err != nil {
			return err
		}

		infiniteStream = beep.Loop(-1, stream)
	}

	t.SoundStream = infiniteStream

	return nil
}

// New creates a new timer.
func New(dbClient store.DB, cfg *config.TimerConfig) (*Timer, error) {
	t := &Timer{
		StartTime: time.Now(),
		db:        dbClient,
		Opts:      cfg,
	}

	err := t.setAmbientSound()

	return t, err
}
