package focus

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/gen2brain/beeep"
	"github.com/pterm/pterm"
	"github.com/urfave/cli/v2"
)

type Error string

func (e Error) Error() string { return string(e) }

const (
	errUnableToSaveSession = Error("Unable to save incomplete session")
	errNoPausedSession     = Error("Paused session not found. Please start a new session")
)

type countdown struct {
	t int
	m int
	s int
}

type sessionType string

const (
	pomodoro   sessionType = "pomodoro"
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

// session represents a pomodoro or break session.
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

type kind map[sessionType]int

type message map[sessionType]string

// Timer represents a Focus instance.
type Timer struct {
	SessionType         sessionType `json:"session_type"`
	Session             session     `json:"-"`
	Kind                kind        `json:"kind"`
	AutoStartPomodoro   bool        `json:"auto_start_pomodoro"`
	AutoStartBreak      bool        `json:"auto_start_break"`
	LongBreakInterval   int         `json:"long_break_interval"`
	MaxPomodoros        int         `json:"max_pomodoros"`
	Counter             int         `json:"counter"`
	PomodoroCycle       int         `json:"iteration"`
	Msg                 message     `json:"msg"`
	ShowNotification    bool        `json:"show_notification"`
	TwentyFourHourClock bool        `json:"24_hour_clock"`
	Store               DB          `json:"-"`
}

// nextSession retrieves the next session.
func (t *Timer) nextSession() sessionType {
	var next sessionType

	switch t.SessionType {
	case pomodoro:
		if t.PomodoroCycle == t.LongBreakInterval {
			next = longBreak
		} else {
			next = shortBreak
		}
	case shortBreak, longBreak:
		next = pomodoro
	}

	return next
}

// getTimeRemaining subtracts the endTime from the currentTime
// and returns the total number of hours, minutes and seconds
// left.
func (t *Timer) getTimeRemaining(endTime time.Time) countdown {
	currentTime := time.Now()
	difference := endTime.Sub(currentTime)
	total := int(difference.Seconds())
	minutes := total / 60
	seconds := total % 60

	return countdown{
		t: total,
		m: minutes,
		s: seconds,
	}
}

// saveSession adds or updates the current session in the database.
func (t *Timer) saveSession() error {
	if t.SessionType != pomodoro {
		return nil
	}

	ev := t.Session
	key := []byte(ev.StartTime.Format(time.RFC3339))

	value, err := json.Marshal(ev)
	if err != nil {
		return err
	}

	return t.Store.updateSession(key, value)
}

// printSession writes the details of the current
// session to the standard output.
func (t *Timer) printSession(endTime time.Time) {
	var text string

	switch t.SessionType {
	case pomodoro:
		var count int

		var total int

		if t.MaxPomodoros != 0 {
			count = t.Counter
			total = t.MaxPomodoros
		} else {
			count = t.PomodoroCycle
			total = t.LongBreakInterval
		}

		text = fmt.Sprintf(
			PrintColor(green, "[Pomodoro %d/%d]"),
			count,
			total,
		) + ": " + t.Msg[pomodoro]
	case shortBreak:
		text = PrintColor(yellow, "[Short break]") + ": " + t.Msg[shortBreak]
	case longBreak:
		text = PrintColor(blue, "[Long break]") + ": " + t.Msg[longBreak]
	}

	var tf string
	if t.TwentyFourHourClock {
		tf = "15:04:05"
	} else {
		tf = "03:04:05 PM"
	}

	fmt.Printf("%s (until %s)\n", text, endTime.Format(tf))
}

// notify indicates the completion of the session
// and sends a desktop notification if enabled.
func (t *Timer) notify() {
	fmt.Printf("Session completed!\n\n")

	m := map[sessionType]string{
		pomodoro:   "Pomodoro",
		shortBreak: "Short break",
		longBreak:  "Long break",
	}

	if t.ShowNotification {
		msg := m[t.SessionType] + " is finished"

		var pathToIcon string

		homeDir, _ := os.UserHomeDir()
		if homeDir != "" {
			pathToIcon = filepath.Join(homeDir, configPath, "icon.png")
		}

		err := beeep.Notify(msg, t.Msg[t.nextSession()], pathToIcon)
		if err != nil {
			pterm.Error.Println(fmt.Errorf("Unable to display notification: %w", err))
		}
	}
}

// handleInterruption is used to save the current state
// of the timer if a pomodoro session is active.
func (t *Timer) handleInterruption() {
	c := make(chan os.Signal, 1)

	signal.Notify(c, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-c

		fmt.Printf("\n\n")

		if t.SessionType == pomodoro && !t.Session.Completed {
			pausedTime := time.Now()
			t.Session.EndTime = pausedTime

			lastIndex := len(t.Session.Timeline) - 1
			t.Session.Timeline[lastIndex].EndTime = pausedTime

			err := t.saveSession()
			if err != nil {
				pterm.Error.Printfln("%s", fmt.Errorf("%s: %w", errUnableToSaveSession, err))
				os.Exit(1)
			}

			timerBytes, err := json.Marshal(t)
			if err != nil {
				pterm.Error.Printfln("%s", fmt.Errorf("%s: %w", errUnableToSaveSession, err))
				os.Exit(1)
			}

			sessionKey := []byte(t.Session.StartTime.Format(time.RFC3339))

			err = t.Store.saveTimerState(timerBytes, sessionKey)
			if err != nil {
				pterm.Error.Printfln("%s", fmt.Errorf("%s: %w", errUnableToSaveSession, err))
				os.Exit(1)
			}

			pterm.Info.Printfln("Pomodoro session exited prematurely. Use %s to continue later", PrintColor(yellow, "focus resume"))
		}

		os.Exit(0)
	}()
}

func (t *Timer) Run() {
	t.handleInterruption()

	if t.SessionType == "" {
		t.SessionType = pomodoro
	}

	var endTime time.Time

	if t.Session.EndTime.IsZero() {
		endTime = t.initSession()
	} else {
		endTime = t.Session.EndTime
	}

	t.start(endTime)
}

// Resume attempts to retrieve a paused pomodoro session
// and continue from where it left of.
func (t *Timer) Resume() error {
	timerBytes, sessionBytes, err := t.Store.getTimerState()
	if err != nil {
		return err
	}

	if len(timerBytes) == 0 || len(sessionBytes) == 0 {
		return errNoPausedSession
	}

	err = json.Unmarshal(timerBytes, t)
	if err != nil {
		return err
	}

	err = json.Unmarshal(sessionBytes, &t.Session)
	if err != nil {
		return err
	}

	var elapsedTimeInSeconds int
	for _, v := range t.Session.Timeline {
		elapsedTimeInSeconds += int(v.EndTime.Sub(v.StartTime).Seconds())
	}

	newEndTime := time.Now().Add(time.Duration(t.Kind[t.SessionType]) * time.Minute).Add(-time.Second * time.Duration(elapsedTimeInSeconds))

	t.Session.EndTime = newEndTime

	t.Session.Timeline = append(t.Session.Timeline, sessionTimeline{
		StartTime: time.Now(),
		EndTime:   newEndTime,
	})

	err = t.Store.deleteTimerState()
	if err != nil {
		pterm.Error.Println(err)
	}

	t.Run()

	return nil
}

// initSession initialises a new session
// and returns the end time for the session.
func (t *Timer) initSession() time.Time {
	t.Counter++

	if t.SessionType == pomodoro {
		if t.PomodoroCycle == t.LongBreakInterval {
			t.PomodoroCycle = 1
		} else {
			t.PomodoroCycle++
		}
	}

	startTime := time.Now()
	endTime := time.Now().Add(time.Duration(t.Kind[t.SessionType]) * time.Minute)

	t.Session = session{
		Name:      t.SessionType,
		Duration:  t.Kind[t.SessionType],
		Completed: false,
		StartTime: startTime,
		Timeline: []sessionTimeline{
			{
				startTime,
				endTime,
			},
		},
	}

	err := t.saveSession()
	if err != nil {
		pterm.Error.Printfln("%s\n", err)
	}

	return endTime
}

// start begins a new session.
func (t *Timer) start(endTime time.Time) {
	for {
		t.printSession(endTime)

		fmt.Print("\033[s")

		timeRemaining := t.getTimeRemaining(endTime)

		t.countdown(timeRemaining)

		ticker := time.NewTicker(time.Second)
		for range ticker.C {
			fmt.Print("\033[u\033[K")

			timeRemaining = t.getTimeRemaining(endTime)

			if timeRemaining.t <= 0 {
				t.Session.Completed = true
				end := time.Now()
				t.Session.EndTime = end

				lastIndex := len(t.Session.Timeline) - 1
				t.Session.Timeline[lastIndex].EndTime = end

				err := t.saveSession()
				if err != nil {
					pterm.Error.Printfln("%s\n", err)
				}

				t.notify()

				break
			}

			t.countdown(timeRemaining)
		}

		if t.Counter == t.MaxPomodoros {
			return
		}

		if t.SessionType != pomodoro && !t.AutoStartPomodoro ||
			t.SessionType == pomodoro && !t.AutoStartBreak {
			// Block until user input before beginning next session
			reader := bufio.NewReader(os.Stdin)

			fmt.Print("\033[s")
			fmt.Print("Press ENTER to start the next session")

			_, _ = reader.ReadString('\n')

			fmt.Print("\033[u\033[K")
		}

		t.SessionType = t.nextSession()

		endTime = t.initSession()
	}
}

// countdown prints the time remaining until the end of
// the current session.
func (t *Timer) countdown(timeRemaining countdown) {
	fmt.Printf("🕒%s:%s", pterm.Yellow(fmt.Sprintf("%02d", timeRemaining.m)), pterm.Yellow(fmt.Sprintf("%02d", timeRemaining.s)))
}

// setOptions configures the Timer instance based
// on command line arguments.
func (t *Timer) setOptions(ctx *cli.Context) {
	if ctx.Uint("pomodoro") > 0 {
		t.Kind[pomodoro] = int(ctx.Uint("pomodoro"))
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

	if ctx.Uint("max-pomodoros") > 0 {
		t.MaxPomodoros = int(ctx.Uint("max-pomodoros"))
	}

	if ctx.Bool("auto-pomodoro") {
		t.AutoStartPomodoro = true
	}

	if ctx.Bool("auto-break") {
		t.AutoStartBreak = true
	}

	if ctx.Bool("disable-notifications") {
		t.ShowNotification = false
	}

	if t.LongBreakInterval <= 0 {
		t.LongBreakInterval = 4
	}

	if ctx.Bool("24-hour") {
		t.TwentyFourHourClock = ctx.Bool("24-hour")
	}
}

// NewTimer returns a new timer constructed from
// the configuration file and command line arguments.
func NewTimer(ctx *cli.Context, c *Config, store *Store) *Timer {
	t := &Timer{
		Kind: kind{
			pomodoro:   c.PomodoroMinutes,
			shortBreak: c.ShortBreakMinutes,
			longBreak:  c.LongBreakMinutes,
		},
		LongBreakInterval: c.LongBreakInterval,
		Msg: message{
			pomodoro:   c.PomodoroMessage,
			shortBreak: c.ShortBreakMessage,
			longBreak:  c.LongBreakMessage,
		},
		ShowNotification:    c.Notify,
		AutoStartPomodoro:   c.AutoStartPomorodo,
		AutoStartBreak:      c.AutoStartBreak,
		TwentyFourHourClock: c.TwentyFourHourClock,
		Store:               store,
	}

	// Command-line flags will override the configuration
	// file
	t.setOptions(ctx)

	return t
}
