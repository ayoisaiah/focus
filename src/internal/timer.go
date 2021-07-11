package focus

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gen2brain/beeep"
	"github.com/urfave/cli/v2"
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

type session struct {
	Name      sessionType `json:"name"`
	Duration  int         `json:"duration"`
	StartTime time.Time   `json:"start_time"`
	EndTime   time.Time   `json:"end_time"`
	Completed bool        `json:"completed"`
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
	Iteration           int         `json:"iteration"`
	Msg                 message     `json:"msg"`
	ShowNotification    bool        `json:"show_notification"`
	TwentyFourHourClock bool        `json:"24_hour_clock"`
	AllowPausing        bool        `json:"allow_pausing"`
}

// nextSession retrieves the next session.
func (t *Timer) nextSession() sessionType {
	var next sessionType

	switch t.SessionType {
	case pomodoro:
		if t.Iteration == t.LongBreakInterval {
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

	return store.updateSession(key, value)
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
			count = t.Iteration
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

		// TODO: Handle error
		_ = beeep.Notify(msg, t.Msg[t.nextSession()], "")
	}
}

// handleInterruption is used to save the current state
// of the timer if a pomodoro session is active.
func (t *Timer) handleInterruption() {
	c := make(chan os.Signal, 1)

	signal.Notify(c, os.Interrupt, syscall.SIGTERM)

	errUnableToSave := errors.New("unable to save incomplete session")

	go func() {
		<-c

		if t.SessionType == pomodoro {
			t.Session.EndTime = time.Now()

			err := t.saveSession()
			if err != nil {
				fmt.Printf("\n\n%s", fmt.Errorf("%s: %w", errUnableToSave, err))
				goto exit
			}

			if !t.AllowPausing {
				goto exit
			}

			timerBytes, err := json.Marshal(t)
			if err != nil {
				fmt.Printf("\n\n%s", fmt.Errorf("%s: %w", errUnableToSave, err))
				goto exit
			}

			sessionKey := []byte(t.Session.StartTime.Format(time.RFC3339))

			err = store.saveTimerState(timerBytes, sessionKey)
			if err != nil {
				fmt.Printf("\n\n%s", fmt.Errorf("%s: %w", errUnableToSave, err))
				goto exit
			}

			fmt.Printf("\n\nPomodoro session is paused. Use %s to continue later", PrintColor(yellow, "focus resume"))
		}

	exit:
		os.Exit(1)
	}()
}

// Run.
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
	timerBytes, sessionBytes, err := store.getTimerState()
	if err != nil {
		return err
	}

	if len(timerBytes) == 0 || len(sessionBytes) == 0 {
		return fmt.Errorf("No existing paused session was found. Please start a new session")
	}

	err = json.Unmarshal(timerBytes, t)
	if err != nil {
		return err
	}

	err = json.Unmarshal(sessionBytes, &t.Session)
	if err != nil {
		return err
	}

	elapsedTimeInSeconds := int(t.Session.EndTime.Sub(t.Session.StartTime).Seconds())
	newEndTime := time.Now().Add(time.Duration(t.Kind[t.SessionType]) * time.Minute).Add(-time.Second * time.Duration(elapsedTimeInSeconds))

	t.Session.EndTime = newEndTime

	err = store.deleteTimerState()
	if err != nil {
		fmt.Println(err)
	}

	t.Run()

	return nil
}

// initSession initialises a new session
// and returns the end time for the session.
func (t *Timer) initSession() time.Time {
	t.Counter++

	if t.SessionType == pomodoro {
		if t.Iteration == t.LongBreakInterval {
			t.Iteration = 1
		} else {
			t.Iteration++
		}
	}

	t.Session = session{
		Name:      t.SessionType,
		Duration:  t.Kind[t.SessionType],
		Completed: false,
		StartTime: time.Now(),
	}

	// TODO: Handle error
	_ = t.saveSession()

	return time.Now().Add(time.Duration(t.Kind[t.SessionType]) * time.Minute)
}

// start begins a new session.
func (t *Timer) start(endTime time.Time) {
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
			t.Session.EndTime = time.Now()

			_ = t.saveSession()

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

	t.start(t.initSession())
}

// countdown prints the time remaining until the end of
// the current session.
func (t *Timer) countdown(timeRemaining countdown) {
	fmt.Printf("Minutes: %02d Seconds: %02d", timeRemaining.m, timeRemaining.s)
}

// setOptions configures the Timer instance based
// on command line options.
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

	if ctx.Bool("allow-pausing") {
		t.AllowPausing = true
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
func NewTimer(ctx *cli.Context, c *Config) *Timer {
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
		AllowPausing:        c.AllowPausing,
	}

	// Command-line flags will override the configuration
	// file
	t.setOptions(ctx)

	return t
}
