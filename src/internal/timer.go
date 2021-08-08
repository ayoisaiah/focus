package focus

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/adrg/xdg"
	"github.com/faiface/beep"
	"github.com/faiface/beep/mp3"
	"github.com/faiface/beep/speaker"
	"github.com/faiface/beep/vorbis"
	"github.com/gen2brain/beeep"
	"github.com/pterm/pterm"
	"github.com/urfave/cli/v2"
)

type Error string

func (e Error) Error() string { return string(e) }

const (
	errUnableToSaveSession = Error("Unable to persist interrupted session")
	errNoPausedSession     = Error(
		"A previously interrupted session was not found. Please start a new session",
	)
)

type countdown struct {
	t int
	m int
	s int
}

type sessionType string

const (
	work       sessionType = "work"
	shortBreak sessionType = "short_break"
	longBreak  sessionType = "long_break"
)

type sessionTimeline struct {
	// StartTime is the start of the session including
	// the start of a paused session
	StartTime time.Time `json:"start_time"`
	// EndTime is the end of a session including
	// when a session is paused or stopped prematurely
	EndTime time.Time `json:"end_time"`
}

// session represents a work or break session.
type session struct {
	// Name is the name of a session
	Name sessionType `json:"name"`
	// Duration is the duration in minutes for a session
	Duration int `json:"duration"`
	// Timeline helps to keep track of how many times
	// a session was paused, for how long, and when it
	// was restarted
	Timeline []sessionTimeline `json:"timeline"`
	// StartTime is the original time the session was started
	StartTime time.Time `json:"start_time"`
	// EndTime the final end time of the session
	EndTime time.Time `json:"end_time"`
	// Completed indicates whether a session was completed
	// or abandoned
	Completed bool `json:"completed"`
}

// getElapsedTimeInSeconds returns the time elapsed
// for the current session in seconds.
func (s *session) getElapsedTimeInSeconds() int {
	var elapsedTimeInSeconds int
	for _, v := range s.Timeline {
		elapsedTimeInSeconds += int(v.EndTime.Sub(v.StartTime).Seconds())
	}

	return elapsedTimeInSeconds
}

// validate ensures that the end time for the current
// session does not exceed what is required to
// complete the session. It mostly helps with normalising
// the end time when the system is hibernated with
// a session in progress, and resumed at a later
// time in the future that surpasses the normal end time.
func (s *session) validateEndTime() {
	elapsed := s.getElapsedTimeInSeconds()

	// If the elapsed time is greater than the duration
	// of the session, the end time must be normalised
	// to a time that will fulfill the exact duration
	// of the session
	if elapsed > s.Duration*60 {
		// secondsBeforeLastPart represents the number of seconds
		// elapsed without including the concluding part of the
		// session timeline
		var secondsBeforeLastPart int

		for i := 0; i < len(s.Timeline)-1; i++ {
			v := s.Timeline[i]
			secondsBeforeLastPart += int(v.EndTime.Sub(v.StartTime).Seconds())
		}

		lastIndex := len(s.Timeline) - 1
		lastPart := s.Timeline[lastIndex]

		secondsLeft := (60 * s.Duration) - secondsBeforeLastPart
		end := lastPart.StartTime.Add(
			time.Duration(secondsLeft * int(time.Second)),
		)
		s.Timeline[lastIndex].EndTime = end
		s.EndTime = end
		s.Completed = true
	}
}

type kind map[sessionType]int

type message map[sessionType]string

// Timer represents a Focus instance.
type Timer struct {
	SessionType         sessionType `json:"session_type"`
	Session             session     `json:"-"`
	Kind                kind        `json:"kind"`
	AutoStartWork       bool        `json:"auto_start_work"`
	AutoStartBreak      bool        `json:"auto_start_break"`
	LongBreakInterval   int         `json:"long_break_interval"`
	MaxSessions         int         `json:"max_sessions"`
	Counter             int         `json:"counter"`
	WorkCycle           int         `json:"iteration"`
	Msg                 message     `json:"msg"`
	ShowNotification    bool        `json:"show_notification"`
	TwentyFourHourClock bool        `json:"24_hour_clock"`
	Store               DB          `json:"-"`
	Sound               string      `json:"sound"`
	SoundOnBreak        bool        `json:"sound_on_break"`
}

// nextSession retrieves the next session.
func (t *Timer) nextSession() sessionType {
	var next sessionType

	switch t.SessionType {
	case work:
		if t.WorkCycle == t.LongBreakInterval {
			next = longBreak
		} else {
			next = shortBreak
		}
	case shortBreak, longBreak:
		next = work
	}

	return next
}

// endSession marks a session as completed
// and updates it in the database.
func (t *Timer) endSession(endTime time.Time) error {
	fmt.Printf("Session completed!\n\n")

	t.Session.Completed = true
	t.Session.EndTime = endTime

	lastIndex := len(t.Session.Timeline) - 1
	t.Session.Timeline[lastIndex].EndTime = endTime

	err := t.saveSession()
	if err != nil {
		return err
	}

	if t.ShowNotification {
		t.notify()
	}

	return nil
}

// getTimeRemaining subtracts the endTime from the currentTime
// and returns the total number of minutes and seconds left.
func (t *Timer) getTimeRemaining(endTime time.Time) countdown {
	difference := time.Until(endTime)
	total := roundTime(difference.Seconds())
	minutes := total / 60
	seconds := total % 60

	return countdown{
		t: total,
		m: minutes,
		s: seconds,
	}
}

// saveSession adds or updates the current session in the database.
// If the current session is not a work session, it will be
// skipped.
func (t *Timer) saveSession() error {
	if t.SessionType != work {
		return nil
	}

	s := t.Session

	s.validateEndTime()

	key := []byte(s.StartTime.Format(time.RFC3339))

	value, err := json.Marshal(s)
	if err != nil {
		return err
	}

	return t.Store.updateSession(key, value)
}

// printSession writes the details of the current
// session to the standard output.
func (t *Timer) printSession(endTime time.Time, w io.Writer) {
	var text string

	switch t.SessionType {
	case work:
		var count int

		var total int

		if t.MaxSessions != 0 {
			count = t.Counter
			total = t.MaxSessions
		} else {
			count = t.WorkCycle
			total = t.LongBreakInterval
		}

		text = fmt.Sprintf(
			pterm.Green("[Work %d/%d]"),
			count,
			total,
		) + ": " + t.Msg[work]
	case shortBreak:
		text = pterm.LightBlue("[Short break]") + ": " + t.Msg[shortBreak]
	case longBreak:
		text = pterm.LightMagenta("[Long break]") + ": " + t.Msg[longBreak]
	}

	var timeFormat string
	if t.TwentyFourHourClock {
		timeFormat = "15:04:05"
	} else {
		timeFormat = "03:04:05 PM"
	}

	fmt.Fprintf(w, "%s (until %s)\n", text, endTime.Format(timeFormat))
}

// notify sends a desktop notification.
func (t *Timer) notify() {
	m := map[sessionType]string{
		work:       "Work session",
		shortBreak: "Short break",
		longBreak:  "Long break",
	}

	msg := m[t.SessionType] + " is finished"

	// pathToIcon will be an empty string if file is not found
	pathToIcon, _ := xdg.SearchDataFile(filepath.Join(configDir, "static", "icon.png"))

	err := beeep.Notify(msg, t.Msg[t.nextSession()], pathToIcon)
	if err != nil {
		pterm.Error.Println(
			fmt.Errorf("Unable to display notification: %w", err),
		)
	}
}

// handleInterruption is used to save the current state
// of the timer if a work session is halted before.
// completion.
func (t *Timer) handleInterruption() {
	c := make(chan os.Signal, 1)

	signal.Notify(c, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-c

		interrruptedTime := time.Now()
		t.Session.EndTime = interrruptedTime

		lastIndex := len(t.Session.Timeline) - 1
		t.Session.Timeline[lastIndex].EndTime = interrruptedTime

		err := t.saveSession()
		if err != nil {
			pterm.Error.Printfln(
				"%s",
				fmt.Errorf("%s: %w", errUnableToSaveSession, err),
			)
			os.Exit(1)
		}

		timerBytes, err := json.Marshal(t)
		if err != nil {
			pterm.Error.Printfln(
				"%s",
				fmt.Errorf("%s: %w", errUnableToSaveSession, err),
			)
			os.Exit(1)
		}

		sessionKey := []byte(t.Session.StartTime.Format(time.RFC3339))

		err = t.Store.saveTimerState(timerBytes, sessionKey)
		if err != nil {
			pterm.Error.Printfln(
				"%s",
				fmt.Errorf("%s: %w", errUnableToSaveSession, err),
			)
			os.Exit(1)
		}

		_ = t.Store.close()

		os.Exit(0)
	}()
}

func (t *Timer) playSound(done chan bool) {
	ext := filepath.Ext(t.Sound)
	// without extension, treat as OGG file
	if ext == "" {
		t.Sound += ".ogg"
	}

	pathToFile, err := xdg.SearchDataFile(filepath.Join(configDir, "static", t.Sound))
	if err != nil {
		pterm.Error.Println(err)
		return
	}

	f, err := os.Open(pathToFile)
	if err != nil {
		pterm.Error.Println(err)
		return
	}

	var streamer beep.StreamSeekCloser

	var format beep.Format

	ext = filepath.Ext(t.Sound)

	switch ext {
	case ".ogg":
		streamer, format, err = vorbis.Decode(f)
	case ".mp3":
		streamer, format, err = mp3.Decode(f)
	}

	if err != nil {
		pterm.Error.Println(err)
		return
	}

	bufferSize := 5

	err = speaker.Init(format.SampleRate, format.SampleRate.N(time.Duration(int(time.Second)/bufferSize)))
	if err != nil {
		pterm.Error.Println(err)
		return
	}

	buffer := beep.NewBuffer(format)

	buffer.Append(streamer)

	streamer.Close()

	s := beep.Loop(-1, buffer.Streamer(0, buffer.Len()))

	speaker.Play(s)

	<-done

	speaker.Clear()
}

func (t *Timer) Run() error {
	t.handleInterruption()

	if t.SessionType == "" {
		t.SessionType = work
	}

	endTime := t.Session.EndTime

	if t.Session.EndTime.IsZero() {
		var err error

		endTime, err = t.initSession()
		if err != nil {
			return err
		}
	}

	return t.start(endTime)
}

// GetInterrupted returns a previously stored work
// session that was interrupted before completion.
func (t *Timer) GetInterrupted() (timerBytes, sessionBytes []byte, err error) {
	timerBytes, sessionBytes, err = t.Store.getTimerState()
	if err != nil {
		return nil, nil, err
	}

	if len(timerBytes) == 0 {
		return nil, nil, errNoPausedSession
	}

	return
}

// Resume attempts to continue an interrupted session
// from where it left off. If the interrupted session is not
// work, it skips right to the next work session
// in the cycle, and continues normally from there.
func (t *Timer) Resume() error {
	timerBytes, sessionBytes, err := t.GetInterrupted()
	if err != nil {
		return err
	}

	err = json.Unmarshal(timerBytes, t)
	if err != nil {
		return err
	}

	if len(sessionBytes) != 0 {
		err = json.Unmarshal(sessionBytes, &t.Session)
		if err != nil {
			return err
		}
	}

	if t.Session.Name != work || t.Session.Completed {
		t.SessionType = work
		// Set to zero value so that a new session is initialised
		t.Session.EndTime = time.Time{}
	} else {
		// Calculate a new end time for the interrupted work
		// session by
		elapsedTimeInSeconds := t.Session.getElapsedTimeInSeconds()
		newEndTime := time.Now().Add(time.Duration(t.Kind[t.SessionType]) * time.Minute).Add(-time.Second * time.Duration(elapsedTimeInSeconds))

		t.Session.EndTime = newEndTime

		t.Session.Timeline = append(t.Session.Timeline, sessionTimeline{
			StartTime: time.Now(),
			EndTime:   newEndTime,
		})
	}

	err = t.Store.deleteTimerState()
	if err != nil {
		return err
	}

	return t.Run()
}

// initSession initialises a new session and saves it
// to the database. It returns the end time for the session
// or an error if saving the session is unsuccessful.
func (t *Timer) initSession() (time.Time, error) {
	t.Counter++

	if t.SessionType == work {
		if t.WorkCycle == t.LongBreakInterval {
			t.WorkCycle = 1
		} else {
			t.WorkCycle++
		}
	}

	startTime := time.Now()
	endTime := startTime.
		Add(time.Duration(t.Kind[t.SessionType] * int(time.Minute)))

	t.Session = session{
		Name:      t.SessionType,
		Duration:  t.Kind[t.SessionType],
		Completed: false,
		StartTime: startTime,
		Timeline: []sessionTimeline{
			{
				StartTime: startTime,
				EndTime:   time.Time{},
			},
		},
	}

	err := t.saveSession()
	if err != nil {
		return time.Time{}, err
	}

	return endTime, nil
}

// start begins a new session.and loops forever,
// alternating between work and break sessions
// unless a maximum number of work sessions
// is set, or the current session is terminated
// manually.
func (t *Timer) start(endTime time.Time) error {
	done := make(chan bool)

	for {
		if t.Sound != "" {
			if t.SessionType == work {
				go t.playSound(done)
			} else if !t.SoundOnBreak {
				done <- true
			}
		}

		t.printSession(endTime, os.Stdout)

		fmt.Print("\033[s")

		timeRemaining := t.getTimeRemaining(endTime)

		t.countdown(timeRemaining)

		ticker := time.NewTicker(time.Second)
		for range ticker.C {
			fmt.Print("\033[u\033[K")

			timeRemaining = t.getTimeRemaining(endTime)

			if timeRemaining.t <= 0 {
				err := t.endSession(endTime)
				if err != nil {
					return err
				}

				break
			}

			t.countdown(timeRemaining)
		}

		if t.Counter == t.MaxSessions {
			return nil
		}

		if t.SessionType != work && !t.AutoStartWork ||
			t.SessionType == work && !t.AutoStartBreak {
			// Block until user input before beginning next session
			reader := bufio.NewReader(os.Stdin)

			fmt.Print("\033[s")
			fmt.Print("Press ENTER to start the next session")

			_, _ = reader.ReadString('\n')

			fmt.Print("\033[u\033[K")
		}

		t.SessionType = t.nextSession()

		var err error

		endTime, err = t.initSession()
		if err != nil {
			return err
		}
	}
}

// countdown prints the time remaining until the end of
// the current session.
func (t *Timer) countdown(timeRemaining countdown) {
	fmt.Printf(
		"🕒%s:%s",
		pterm.Yellow(fmt.Sprintf("%02d", timeRemaining.m)),
		pterm.Yellow(fmt.Sprintf("%02d", timeRemaining.s)),
	)
}

// setOptions configures the Timer instance based
// on command line arguments.
func (t *Timer) setOptions(ctx *cli.Context) {
	if ctx.Uint("work") > 0 {
		t.Kind[work] = int(ctx.Uint("work"))
	}

	if ctx.Uint("short-break") > 0 {
		t.Kind[shortBreak] = int(ctx.Uint("short-break"))
	}

	if ctx.Uint("long-break") > 0 {
		t.Kind[longBreak] = int(ctx.Uint("long-break"))
	}

	if ctx.Uint("long-break-interval") > 0 {
		t.LongBreakInterval = int(ctx.Uint("long-break-interval"))
	}

	if ctx.Uint("max-sessions") > 0 {
		t.MaxSessions = int(ctx.Uint("max-sessions"))
	}

	if ctx.Bool("disable-notifications") {
		t.ShowNotification = false
	}

	if ctx.String("sound") != "" {
		t.Sound = ctx.String("sound")
	}
}

// NewTimer returns a new timer constructed from
// the configuration file and command line arguments.
func NewTimer(ctx *cli.Context, c *Config, store *Store) *Timer {
	t := &Timer{
		Kind: kind{
			work:       c.WorkMinutes,
			shortBreak: c.ShortBreakMinutes,
			longBreak:  c.LongBreakMinutes,
		},
		LongBreakInterval: c.LongBreakInterval,
		Msg: message{
			work:       c.WorkMessage,
			shortBreak: c.ShortBreakMessage,
			longBreak:  c.LongBreakMessage,
		},
		ShowNotification:    c.Notify,
		AutoStartWork:       c.AutoStartWork,
		AutoStartBreak:      c.AutoStartBreak,
		TwentyFourHourClock: c.TwentyFourHourClock,
		Sound:               c.Sound,
		SoundOnBreak:        c.SoundOnBreak,
		Store:               store,
	}

	// Command-line options will override the configuration
	// file
	t.setOptions(ctx)

	return t
}
