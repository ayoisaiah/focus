// Package timer operates the Focus countdown timer and handles the recovery of
// interrupted timers
package timer

import (
	"bufio"
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

	"github.com/ayoisaiah/focus/config"
	"github.com/ayoisaiah/focus/internal/color"
	"github.com/ayoisaiah/focus/internal/session"
	"github.com/ayoisaiah/focus/internal/static"
	internaltime "github.com/ayoisaiah/focus/internal/time"
	"github.com/ayoisaiah/focus/store"
)

var (
	errUnableToSaveSession = errors.New("Unable to persist interrupted session")
	errInvalidSoundFormat  = errors.New(
		"Invalid sound file format. Only MP3, OGG, FLAC, and WAV files are supported",
	)
)

var (
	opts      *config.TimerConfig
	db        store.DB
	workCycle int
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

// getTimeRemaining subtracts the endTime from the currentTime
// and returns the total number of minutes and seconds left.
func getTimeRemaining(endTime time.Time) timeRemaining {
	difference := time.Until(endTime)
	total := internaltime.Round(difference.Seconds())
	minutes := total / 60
	seconds := total % 60

	return timeRemaining{
		t: total,
		m: minutes,
		s: seconds,
	}
}

// runSessionCmd executes the specified command.
func runSessionCmd(sessionCmd string) error {
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
	cmd.Stdin = opts.Stdin
	cmd.Stdout = opts.Stdout
	cmd.Stderr = opts.Stderr

	return cmd.Run()
}

// printSession writes the details of the current
// session to the standard output.
func printSession(
	sess *session.Session,
	endTime time.Time,
) {
	var text string

	switch sess.Name {
	case session.Work:
		total := opts.LongBreakInterval

		text = fmt.Sprintf(
			color.Green("[Work %d/%d]"),
			workCycle,
			total,
		) + ": " + opts.Message[session.Work]
	case session.ShortBreak:
		text = color.Blue(
			"[Short break]",
		) + ": " + opts.Message[session.ShortBreak]
	case session.LongBreak:
		text = color.Magenta(
			"[Long break]",
		) + ": " + opts.Message[session.LongBreak]
	}

	var timeFormat string
	if opts.TwentyFourHourClock {
		timeFormat = "15:04:05"
	} else {
		timeFormat = "03:04:05 PM"
	}

	var tags string
	if len(sess.Tags) > 0 {
		tags = " >>> " + strings.Join(sess.Tags, " | ")
	}

	fmt.Fprintf(
		opts.Stdout,
		"%s (until %s)%s\n",
		text,
		color.Highlight(endTime.Format(timeFormat)),
		tags,
	)
}

// notify sends a desktop notification.
func notify(title, msg string) {
	configDir := filepath.Base(filepath.Dir(opts.PathToConfig))

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
func handleInterruption(sess *session.Session) chan os.Signal {
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
				fmt.Errorf("%s: %w", errUnableToSaveSession, err),
			)
			os.Exit(1)
		}

		interrruptedTime := time.Now()
		sess.EndTime = interrruptedTime

		lastIndex := len(sess.Timeline) - 1
		sess.Timeline[lastIndex].EndTime = interrruptedTime

		err := saveSession(sess)
		if err != nil {
			exitFunc(err)
		}

		sessionKey := []byte(sess.StartTime.Format(time.RFC3339))

		err = db.SaveTimer(sessionKey, opts, workCycle)
		if err != nil {
			exitFunc(err)
		}

		_ = db.Close()

		os.Exit(0)
	}()

	return c
}

// prepareAmbientSoundStream returns an audio stream for the ambient sound.
func prepareAmbientSoundStream() (beep.Streamer, error) {
	ambientSound := opts.AmbientSound

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
		f, err = os.Open(opts.AmbientSound)
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
func wait() error {
	c := make(chan os.Signal, 1)

	signal.Notify(c, os.Interrupt, syscall.SIGTERM)

	go func() {
		s := <-c

		if s.String() == sessionSettled {
			return
		}

		os.Exit(0)
	}()

	err := db.Close()
	if err != nil {
		return err
	}

	reader := bufio.NewReader(opts.Stdin)

	fmt.Fprint(opts.Stdout, "\033[s")
	fmt.Fprint(opts.Stdout, "Press ENTER to start the next session")

	// Block until user input before beginning next session
	_, err = reader.ReadString('\n')

	c <- Settled{}

	if errors.Is(err, io.EOF) {
		return nil
	} else if err != nil {
		return err
	}

	fmt.Print("\033[u\033[K")

	return db.Open()
}

// countdown prints the time remaining until the end of the current session.
func countdown(tr timeRemaining) {
	fmt.Fprintf(
		opts.Stdout,
		"🕒%s:%s",
		pterm.Yellow(fmt.Sprintf("%02d", tr.m)),
		pterm.Yellow(fmt.Sprintf("%02d", tr.s)),
	)
}

// nextSession retrieves the next session.
func nextSession(current session.Name) session.Name {
	var next session.Name

	switch current {
	case session.Work:
		if workCycle == opts.LongBreakInterval {
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
func start(sess *session.Session) {
	endTime := sess.StartTime.
		Add(time.Duration(opts.Duration[sess.Name] * int(time.Minute)))

	if sess.Resuming() {
		// Calculate a new end time for the interrupted work
		// session by
		elapsedTimeInSeconds := sess.GetElapsedTimeInSeconds()
		endTime = time.Now().
			Add(time.Duration(opts.Duration[sess.Name]) * time.Minute).
			Add(-time.Second * time.Duration(elapsedTimeInSeconds))

		sess.EndTime = endTime

		sess.Timeline = append(sess.Timeline, session.Timeline{
			StartTime: time.Now(),
			EndTime:   endTime,
		})
	}

	printSession(sess, endTime)

	fmt.Fprint(opts.Stdout, "\033[s")

	remainder := getTimeRemaining(endTime)

	countdown(remainder)

	ticker := time.NewTicker(time.Second)
	for range ticker.C {
		fmt.Fprint(opts.Stdout, "\033[u\033[K")

		remainder = getTimeRemaining(endTime)

		if remainder.t <= 0 {
			fmt.Printf("Session completed!\n\n")

			lastIndex := len(sess.Timeline) - 1

			sess.EndTime = endTime
			sess.Completed = true
			sess.Timeline[lastIndex].EndTime = endTime

			return
		}

		countdown(remainder)
	}
}

// saveSession creates or updates a work session in the data store.
func saveSession(sess *session.Session) error {
	if sess.Name != session.Work {
		return nil
	}

	sess.Normalise()

	return db.UpdateSession(sess)
}

// newSession initialises a new session and saves it to the data store.
func newSession(name session.Name) (*session.Session, error) {
	now := time.Now()

	sess := &session.Session{
		Name:      name,
		Duration:  opts.Duration[name],
		Tags:      opts.Tags,
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

	err := saveSession(sess)
	if err != nil {
		return sess, err
	}

	return sess, nil
}

// Recover attempts to recover an interrupted session.
func Recover(dbClient store.DB, ctx *cli.Context) (*session.Session, error) {
	var sess *session.Session

	var err error

	db = dbClient

	var pausedKey []byte

	if ctx.Bool("select") {
		pausedKey, err = db.SelectPaused()
		if err != nil {
			return nil, err
		}
	}

	opts, sess, workCycle, err = db.GetInterrupted(pausedKey)
	if err != nil {
		return nil, err
	}

	if ctx.Bool("disable-notification") {
		opts.Notify = false
	}

	if ctx.String("sound") != "" {
		if ctx.String("sound") == "off" {
			opts.AmbientSound = ""
		} else {
			opts.AmbientSound = ctx.String("sound")
		}
	}

	if ctx.String("session-cmd") != "" {
		opts.SessionCmd = ctx.String("session-cmd")
	}

	if sess == nil {
		// Set to zero value so that a new session is initialised
		sess = &session.Session{}
	}

	return sess, nil
}

// Run begins the timer and loops forever, alternating between work and
// break sessions until it is terminated with Ctrl-C or a maximum number of work
// sessions is reached.
func Run(sess *session.Session) (err error) {
	sessName := session.Work

	var streamer beep.Streamer

	if opts.AmbientSound != "" {
		streamer, err = prepareAmbientSoundStream()
		if err != nil {
			return err
		}
	}

	for {
		if !sess.Resuming() {
			sess, err = newSession(sessName)
			if err != nil {
				return err
			}
		}

		if sess.Name == session.Work && !sess.Resuming() {
			if workCycle == opts.LongBreakInterval {
				workCycle = 1
			} else {
				workCycle++
			}
		}

		c := handleInterruption(sess)

		if opts.AmbientSound != "" {
			if sess.Name == session.Work || opts.PlaySoundOnBreak {
				speaker.Play(streamer)
			} else {
				speaker.Clear()
			}
		}

		start(sess)

		c <- Settled{}

		err = saveSession(sess)
		if err != nil {
			return err
		}

		next := nextSession(sessName)

		if opts.Notify {
			title := sessName + " is finished"
			notify(string(title), opts.Message[next])
		}

		sessName = next

		if opts.SessionCmd != "" {
			err = runSessionCmd(opts.SessionCmd)
			if err != nil {
				return err
			}
		}

		if sessName != session.Work && !opts.AutoStartBreak ||
			sessName == session.Work && !opts.AutoStartWork {
			err = wait()
			if err != nil {
				return err
			}
		}
	}
}

func Init(dbClient *store.Client, cfg *config.TimerConfig) {
	db = dbClient
	opts = cfg
}
