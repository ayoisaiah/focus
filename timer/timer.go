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

	"github.com/ayoisaiah/focus/config"
	"github.com/ayoisaiah/focus/internal/color"
	"github.com/ayoisaiah/focus/internal/session"
	"github.com/ayoisaiah/focus/internal/static"
	"github.com/ayoisaiah/focus/internal/timeutil"
	"github.com/ayoisaiah/focus/store"
)

var (
	errUnableToSaveSession = errors.New("unable to persist interrupted session")
	errInvalidSoundFormat  = errors.New(
		"unsupported sound file format: Only MP3, OGG, FLAC, and WAV files are allowed",
	)
)

const sessionSettled = "settled"

// Settled fulfills the os.Signal interface.
type Settled struct{}

func (s Settled) String() string {
	return sessionSettled
}

func (s Settled) Signal() {}

type timeRemaining struct {
	t int
	m int
	s int
}

type Timer struct {
	db         store.DB            `json:"-"`
	Opts       *config.TimerConfig `json:"opts"`
	PausedTime []byte              `json:"paused_time"`
	SessionKey []byte              `json:"session_key"`
	WorkCycle  int                 `json:"work_cycle"`
}

// getTimeRemaining subtracts the endTime from the currentTime
// and returns the total number of minutes and seconds left.
func getTimeRemaining(endTime time.Time) timeRemaining {
	difference := time.Until(endTime)
	total := timeutil.Round(difference.Seconds())
	minutes := total / 60
	seconds := total % 60

	return timeRemaining{
		t: total,
		m: minutes,
		s: seconds,
	}
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

// printSession writes the details of the current
// session to the standard output.
func (t *Timer) printSession(
	sess *session.Session,
	endTime time.Time,
) {
	var text string

	switch sess.Name {
	case session.Work:
		total := t.Opts.LongBreakInterval

		text = fmt.Sprintf(
			color.Green("[Work %d/%d]"),
			t.WorkCycle,
			total,
		) + ": " + t.Opts.Message[session.Work]
	case session.ShortBreak:
		text = color.Blue(
			"[Short break]",
		) + ": " + t.Opts.Message[session.ShortBreak]
	case session.LongBreak:
		text = color.Magenta(
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
		color.Highlight(endTime.Format(timeFormat)),
		tags,
	)
}

// notify sends a desktop notification.
func (t *Timer) notify(title, msg string) {
	configDir := filepath.Base(filepath.Dir(t.Opts.PathToConfig))

	// pathToIcon will be an empty string if file is not found
	pathToIcon, _ := xdg.SearchDataFile(
		filepath.Join(configDir, "static", "icon.png"),
	)

	err := beeep.Alert(title, msg, pathToIcon)
	if err != nil {
		pterm.Error.Println(
			fmt.Errorf("Unable to display notification: %w", err),
		)
	}
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
			pterm.Error.Printfln(
				"%s",
				fmt.Errorf("%w: %w", errUnableToSaveSession, err),
			)
			os.Exit(1)
		}

		interrruptedTime := time.Now()
		sess.EndTime = interrruptedTime

		lastIndex := len(sess.Timeline) - 1
		sess.Timeline[lastIndex].EndTime = interrruptedTime

		err := t.saveSession(sess)
		if err != nil {
			exitFunc(err)
		}

		t.SessionKey = []byte(sess.StartTime.Format(time.RFC3339))
		t.PausedTime = []byte(time.Now().Format(time.RFC3339))

		timerBytes, err := json.Marshal(t)
		if err != nil {
			exitFunc(err)
		}

		err = t.db.SaveTimer(t.PausedTime, timerBytes)
		if err != nil {
			exitFunc(err)
		}

		_ = t.db.Close()

		os.Exit(0)
	}()

	return c
}

// prepareAmbientSoundStream returns an audio stream for the ambient sound.
func (t *Timer) prepareAmbientSoundStream() (beep.Streamer, error) {
	ambientSound := t.Opts.AmbientSound

	var (
		f        fs.File
		err      error
		streamer beep.StreamSeekCloser
		format   beep.Format
	)

	ext := filepath.Ext(ambientSound)
	// without extension, treat as OGG file
	if ext == "" {
		ambientSound += ".ogg"

		f, err = static.Files.Open(static.FilePath(ambientSound))
		if err != nil {
			return nil, err
		}
	} else {
		f, err = os.Open(t.Opts.AmbientSound)
		if err != nil {
			return nil, err
		}
	}

	ext = filepath.Ext(ambientSound)

	switch ext {
	case ".ogg":
		streamer, format, err = vorbis.Decode(f)
	case ".mp3":
		streamer, format, err = mp3.Decode(f)
	case ".flac":
		streamer, format, err = flac.Decode(f)
	case ".wav":
		streamer, format, err = wav.Decode(f)
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

	err = streamer.Seek(0)
	if err != nil {
		return nil, err
	}

	s := beep.Loop(-1, streamer)

	return s, nil
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
func (t *Timer) countdown(tr timeRemaining) {
	fmt.Fprintf(
		t.Opts.Stdout,
		"🕒%s:%s",
		pterm.Yellow(fmt.Sprintf("%02d", tr.m)),
		pterm.Yellow(fmt.Sprintf("%02d", tr.s)),
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

// start begins a new session.and blocks until its completion.
func (t *Timer) start(sess *session.Session) {
	endTime := sess.StartTime.
		Add(time.Duration(t.Opts.Duration[sess.Name] * int(time.Minute)))

	if sess.Resuming() {
		// Calculate a new end time for the interrupted work
		// session by
		elapsedTimeInSeconds := sess.GetElapsedTimeInSeconds()
		endTime = time.Now().
			Add(time.Duration(t.Opts.Duration[sess.Name]) * time.Minute).
			Add(-time.Second * time.Duration(elapsedTimeInSeconds))

		sess.EndTime = endTime

		sess.Timeline = append(sess.Timeline, session.Timeline{
			StartTime: time.Now(),
			EndTime:   endTime,
		})
	}

	t.printSession(sess, endTime)

	fmt.Fprint(t.Opts.Stdout, "\033[s")

	remainder := getTimeRemaining(endTime)

	t.countdown(remainder)

	ticker := time.NewTicker(time.Second)
	for range ticker.C {
		fmt.Fprint(t.Opts.Stdout, "\033[u\033[K")

		remainder = getTimeRemaining(endTime)

		if remainder.t <= 0 {
			fmt.Printf("Session completed!\n\n")

			lastIndex := len(sess.Timeline) - 1

			sess.EndTime = endTime
			sess.Completed = true
			sess.Timeline[lastIndex].EndTime = endTime

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

func printTable(data [][]string, writer io.Writer) {
	d := [][]string{
		{"#", "PAUSED DATE", "TAGS"},
	}

	d = append(d, data...)

	table := pterm.DefaultTable
	table.Boxed = true

	str, err := table.WithHasHeader().WithData(d).Srender()
	if err != nil {
		pterm.Error.Printfln("Failed to output session table: %s", err.Error())
		return
	}

	fmt.Fprintln(writer, str)
}

func printPausedTimers(timers []Timer) error {
	tableBody := make([][]string, len(timers))

	for i := range timers {
		t := timers[i]

		keyTime, err := time.Parse(time.RFC3339, string(t.PausedTime))
		if err != nil {
			return err
		}

		row := []string{
			fmt.Sprintf("%d", i+1),
			keyTime.Format("January 02, 2006 03:04:05 PM"),
			strings.Join(t.Opts.Tags, ", "),
		}

		tableBody[i] = row
	}

	printTable(tableBody, os.Stdout)

	return nil
}

func selectPausedTimers(
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
	if len(timers) > index {
		return nil, fmt.Errorf("%d is not associated with a session", num)
	}

	return &timers[index], nil
}

// Recover attempts to recover an interrupted session.
func Recover(
	db store.DB,
	ctx *cli.Context,
) (*Timer, *session.Session, error) {
	var sess *session.Session

	var err error

	b, err := db.RetrievePausedTimers()
	if err != nil {
		return nil, nil, err
	}

	pausedTimers := make([]Timer, len(b))

	for i := range b {
		var t Timer

		err = json.Unmarshal(b[i], &t)
		if err != nil {
			return nil, nil, err
		}

		pausedTimers[i] = t
	}

	var t *Timer

	if ctx.Bool("select") {
		err = printPausedTimers(pausedTimers)
		if err != nil {
			return nil, nil, err
		}

		t, err = selectPausedTimers(pausedTimers)
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

	sess, err = t.db.GetInterrupted(t.SessionKey)
	if err != nil {
		return nil, nil, err
	}

	err = t.db.DeleteTimer(t.PausedTime)
	if err != nil {
		return nil, nil, err
	}

	if ctx.Bool("disable-notification") {
		t.Opts.Notify = false
	}

	if ctx.String("sound") != "" {
		if ctx.String("sound") == "off" {
			t.Opts.AmbientSound = ""
		} else {
			t.Opts.AmbientSound = ctx.String("sound")
		}
	}

	if ctx.String("session-cmd") != "" {
		t.Opts.SessionCmd = ctx.String("session-cmd")
	}

	if sess == nil {
		// Set to zero value so that a new session is initialised
		sess = &session.Session{}
	}

	return t, sess, nil
}

// Run begins the timer and loops forever, alternating between work and
// break sessions until it is terminated with Ctrl-C or a maximum number of work
// sessions is reached.
func (t *Timer) Run(sess *session.Session) (err error) {
	sessName := session.Work

	var streamer beep.Streamer

	if t.Opts.AmbientSound != "" {
		streamer, err = t.prepareAmbientSoundStream()
		if err != nil {
			return err
		}
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
				speaker.Play(streamer)
			} else {
				speaker.Clear()
			}
		}

		t.start(sess)

		c <- Settled{}

		err = t.saveSession(sess)
		if err != nil {
			return err
		}

		next := t.nextSession(sessName)

		if t.Opts.Notify {
			title := sessName + " is finished"
			t.notify(string(title), t.Opts.Message[next])
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
	t := Timer{}
	t.db = dbClient
	t.Opts = cfg

	return &t
}
