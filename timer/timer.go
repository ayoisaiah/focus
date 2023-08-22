// Package timer operates the Focus countdown timer and handles the recovery of
// interrupted timers
package timer

import (
	"bufio"
	"cmp"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"slices"
	"strconv"
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
	"github.com/ayoisaiah/focus/internal/session"
	"github.com/ayoisaiah/focus/internal/static"
	"github.com/ayoisaiah/focus/internal/timeutil"
	"github.com/ayoisaiah/focus/internal/ui"
	"github.com/ayoisaiah/focus/store"
)

var errInvalidSoundFormat = errors.New(
	"file must be in mp3, ogg, flac, or wav format",
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
	db         store.DB            `json:"-"`
	Opts       *config.TimerConfig `json:"opts"`
	PausedTime time.Time           `json:"paused_time"`
	Started    time.Time           `json:"date_started"`
	SessionKey []byte              `json:"session_key"`
	WorkCycle  int                 `json:"work_cycle"`
}

// Status represents the status of a running timer.
type Status struct {
	EndTime           time.Time    `json:"end_date"`
	Name              session.Name `json:"name"`
	Tags              []string     `json:"tags"`
	WorkCycle         int          `json:"work_cycle"`
	LongBreakInterval int          `json:"long_break_interval"`
}

// persist updates the timer in the database so that it may be
// recovered later.
func (t *Timer) persist(sess *session.Session) error {
	if sess.Name != session.Work {
		return nil
	}

	if !sess.Completed {
		t.SessionKey = []byte(sess.StartTime.Format(time.RFC3339))
	}

	t.PausedTime = time.Now()

	timerBytes, err := json.Marshal(t)
	if err != nil {
		return err
	}

	err = t.db.UpdateTimer(timeutil.ToKey(t.Started), timerBytes)
	if err != nil {
		return err
	}

	return nil
}

// runSessionCmd executes the specified command.
func (t *Timer) runSessionCmd(sessionCmd string) error {
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
	cmd.Stdin = t.Opts.Stdin
	cmd.Stdout = t.Opts.Stdout
	cmd.Stderr = t.Opts.Stderr

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
		pterm.Error.Printfln("unable to read status file: %v", err)
		return nil
	}

	var s Status

	err = json.Unmarshal(fileBytes, &s)
	if err != nil {
		return err
	}

	sess := &session.Session{
		EndTime: s.EndTime,
	}
	tr := sess.Remaining()

	if tr.T < 0 {
		return nil
	}

	var text string

	switch s.Name {
	case session.Work:
		text = fmt.Sprintf("[Work %d/%d]",
			s.WorkCycle,
			s.LongBreakInterval,
		)
	case session.ShortBreak:
		text = "[Short break]"
	case session.LongBreak:
		text = "[Long break]"
	}

	pterm.Printfln("%s: %02d:%02d", text, tr.M, tr.S)

	return nil
}

func (t *Timer) writeStatusFile(
	sess *session.Session,
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

// printSession writes the details of the current
// session to the standard output.
func (t *Timer) printSession(
	sess *session.Session,
) {
	var text string

	switch sess.Name {
	case session.Work:
		total := t.Opts.LongBreakInterval

		text = fmt.Sprintf(
			ui.Green("[Work %d/%d]"),
			t.WorkCycle,
			total,
		) + ": " + t.Opts.Message[session.Work]
	case session.ShortBreak:
		text = ui.Blue(
			"[Short break]",
		) + ": " + t.Opts.Message[session.ShortBreak]
	case session.LongBreak:
		text = ui.Magenta(
			"[Long break]",
		) + ": " + t.Opts.Message[session.LongBreak]
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
func (t *Timer) notify(title, msg, sound string) {
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

	speaker.Clear()
}

// handleInterruption saves the current state of the timer whenever it is
// interrupted by pressing Ctrl-C.
func (t *Timer) handleInterruption(sess *session.Session) chan os.Signal {
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

		interrruptedTime := time.Now()
		sess.EndTime = interrruptedTime

		lastIndex := len(sess.Timeline) - 1
		sess.Timeline[lastIndex].EndTime = interrruptedTime

		_ = os.Remove(config.StatusFilePath())

		err := t.saveSession(sess)
		if err != nil {
			exitFunc(err)
		}

		err = t.persist(sess)
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
func (t *Timer) wait() error {
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

	reader := bufio.NewReader(t.Opts.Stdin)

	fmt.Fprint(t.Opts.Stdout, "\033[s")
	fmt.Fprint(t.Opts.Stdout, "Press ENTER to start the next session")

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
func (t *Timer) countdown(tr session.Remainder) {
	fmt.Fprintf(
		t.Opts.Stdout,
		"🕒%s:%s",
		pterm.Yellow(fmt.Sprintf("%02d", tr.M)),
		pterm.Yellow(fmt.Sprintf("%02d", tr.S)),
	)
}

// nextSession retrieves the next session.
func (t *Timer) nextSession(current session.Name) session.Name {
	var next session.Name

	switch current {
	case session.Work:
		if t.WorkCycle == t.Opts.LongBreakInterval {
			next = session.LongBreak
		} else {
			next = session.ShortBreak
		}
	case session.ShortBreak, session.LongBreak:
		next = session.Work
	}

	return next
}

// start starts or resumes a session.and blocks until its completion.
func (t *Timer) start(sess *session.Session) {
	sess.SetEndTime()

	t.printSession(sess)

	_ = t.writeStatusFile(sess)

	fmt.Fprint(t.Opts.Stdout, "\033[s")

	remainder := sess.Remaining()

	t.countdown(remainder)

	ticker := time.NewTicker(time.Second)
	for range ticker.C {
		fmt.Fprint(t.Opts.Stdout, "\033[u\033[K")

		remainder = sess.Remaining()

		if remainder.T <= 0 {
			fmt.Printf("Session completed!\n\n")

			sess.Completed = true

			return
		}

		t.countdown(remainder)
	}
}

// saveSession creates or updates a work session in the data store.
func (t *Timer) saveSession(sess *session.Session) error {
	if sess.Name != session.Work {
		return nil
	}

	sess.Normalise()

	return t.db.UpdateSession(sess)
}

// newSession initialises a new session and saves it to the data store.
func (t *Timer) newSession(name session.Name) (*session.Session, error) {
	now := time.Now()

	sess := &session.Session{
		Name:      name,
		Duration:  t.Opts.Duration[name],
		Tags:      t.Opts.Tags,
		Completed: false,
		StartTime: now,
		EndTime:   now,
		Timeline: []session.Timeline{
			{
				StartTime: now,
				EndTime:   now,
			},
		},
	}

	err := t.saveSession(sess)
	if err != nil {
		return sess, err
	}

	return sess, nil
}

// printPausedTimers outputs a list of resumable timers.
func printPausedTimers(timers []Timer, pausedSess map[string]session.Session) {
	tableBody := make([][]string, len(timers))

	for i := range timers {
		t := timers[i]

		sess := pausedSess[string(t.SessionKey)]

		sess.SetEndTime()

		r := sess.Remaining()

		cycle := fmt.Sprintf("%d/%d", t.WorkCycle, t.Opts.LongBreakInterval)

		remainder := fmt.Sprintf("%s -> completed", cycle)
		if r.T > 0 {
			remainder = fmt.Sprintf("%s -> %02d:%02d", cycle, r.M, r.S)
		}

		row := []string{
			fmt.Sprintf("%d", i+1),
			t.PausedTime.Format("Jan 02, 2006 03:04:05 PM"),
			t.Started.Format("Jan 02, 2006 03:04:05 PM"),
			remainder,
			strings.Join(t.Opts.Tags, ", "),
		}

		tableBody[i] = row
	}

	tableBody = append([][]string{
		{
			"#",
			"DATE PAUSED",
			"DATE STARTED",
			"CYCLE",
			"TAGS",
		},
	}, tableBody...)

	ui.PrintTable(tableBody, os.Stdout)
}

// selectPausedTimer prompts the user to select from a list of resumable
// timers.
func selectPausedTimer(
	timers []Timer,
) (*Timer, error) {
	reader := bufio.NewReader(os.Stdin)

	fmt.Fprint(os.Stdout, "\033[s")
	fmt.Fprint(os.Stdout, "Type a number and press ENTER: ")

	input, err := reader.ReadString('\n')
	if err != nil {
		return nil, err
	}

	num, err := strconv.Atoi(strings.TrimSpace(input))
	if err != nil {
		return nil, err
	}

	index := num - 1
	if index >= len(timers) {
		return nil, fmt.Errorf("%d is not associated with a session", num)
	}

	return &timers[index], nil
}

// overrideOptsOnResume overrides timer options if specified through
// command-line arguments.
func (t *Timer) overrideOptsOnResume(ctx *cli.Context) {
	if ctx.Bool("disable-notification") {
		t.Opts.Notify = false
	}

	ambientSound := ctx.String("sound")
	if ambientSound != "" {
		if ambientSound == config.SoundOff {
			t.Opts.AmbientSound = ""
		} else {
			t.Opts.AmbientSound = ambientSound
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
}

func getTimerSessions(
	timers [][]byte,
	db store.DB,
) ([]Timer, map[string]session.Session, error) {
	pausedTimers := make([]Timer, len(timers))

	for i := range timers {
		var t Timer

		err := json.Unmarshal(timers[i], &t)
		if err != nil {
			return nil, nil, err
		}

		pausedTimers[i] = t
	}

	pausedSessions := make(map[string]session.Session)

	for _, v := range pausedTimers {
		s, dbErr := db.GetSession(v.SessionKey)
		if dbErr != nil {
			return nil, nil, dbErr
		}

		pausedSessions[string(v.SessionKey)] = *s
	}

	slices.SortStableFunc(pausedTimers, func(a, b Timer) int {
		return cmp.Compare(b.PausedTime.UnixNano(), a.PausedTime.UnixNano())
	})

	return pausedTimers, pausedSessions, nil
}

// Recover attempts to recover an interrupted session.
func Recover(
	db store.DB,
	ctx *cli.Context,
) (*Timer, *session.Session, error) {
	b, err := db.RetrievePausedTimers()
	if err != nil {
		return nil, nil, err
	}

	pausedTimers, pausedSessions, err := getTimerSessions(b, db)
	if err != nil {
		return nil, nil, err
	}

	var t *Timer

	if ctx.Bool("select") {
		printPausedTimers(pausedTimers, pausedSessions)

		t, err = selectPausedTimer(pausedTimers)
		if err != nil {
			return nil, nil, err
		}
	} else {
		t = &pausedTimers[0]
	}

	t.db = db

	t.Opts.Stdin = os.Stdin
	t.Opts.Stdout = os.Stdout
	t.Opts.Stderr = os.Stderr

	sess, err := t.db.GetSession(t.SessionKey)
	if err != nil {
		return nil, nil, err
	}

	err = t.db.DeleteTimer(timeutil.ToKey(t.Started))
	if err != nil {
		return nil, nil, err
	}

	t.overrideOptsOnResume(ctx)

	return t, sess, nil
}

// Run begins the timer and loops forever, alternating between work and
// break sessions until it is terminated with Ctrl-C or a maximum number of work
// sessions is reached.
func (t *Timer) Run(sess *session.Session) (err error) {
	sessName := session.Work

	var infiniteStream beep.Streamer

	if t.Opts.AmbientSound != "" {
		stream, streamErr := t.prepSoundStream(t.Opts.AmbientSound)
		if streamErr != nil {
			return streamErr
		}

		infiniteStream = beep.Loop(-1, stream)
	}

	for {
		if !sess.Resuming() {
			sess, err = t.newSession(sessName)
			if err != nil {
				return err
			}
		}

		if sess.Name == session.Work && !sess.Resuming() {
			if t.WorkCycle == t.Opts.LongBreakInterval {
				t.WorkCycle = 1
			} else {
				t.WorkCycle++
			}
		}

		c := t.handleInterruption(sess)

		if t.Opts.AmbientSound != "" {
			if sess.Name == session.Work || t.Opts.PlaySoundOnBreak {
				speaker.Clear()
				speaker.Play(infiniteStream)
			} else {
				speaker.Clear()
			}
		}

		t.start(sess)

		c <- Settled{}

		err = t.persist(sess)
		if err != nil {
			return err
		}

		err = t.saveSession(sess)
		if err != nil {
			return err
		}

		next := t.nextSession(sessName)

		if t.Opts.Notify {
			title := sessName + " is finished"

			notifySound := t.Opts.BreakSound

			if sessName != session.Work {
				notifySound = t.Opts.WorkSound
			}

			t.notify(string(title), t.Opts.Message[next], notifySound)
		}

		sessName = next

		if t.Opts.SessionCmd != "" {
			err = t.runSessionCmd(t.Opts.SessionCmd)
			if err != nil {
				return err
			}
		}

		if sessName != session.Work && !t.Opts.AutoStartBreak ||
			sessName == session.Work && !t.Opts.AutoStartWork {
			err = t.wait()
			if err != nil {
				return err
			}
		}
	}
}

// New creates a new timer.
func New(dbClient *store.Client, cfg *config.TimerConfig) *Timer {
	return &Timer{
		Started: time.Now(),
		db:      dbClient,
		Opts:    cfg,
	}
}
