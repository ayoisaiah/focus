package focus

import (
	"bufio"
	"fmt"
	"os"
	"time"

	"github.com/gen2brain/beeep"
	"github.com/urfave/cli/v2"
)

type countdown struct {
	t int
	m int
	s int
}

type sessionType int

const (
	Pomodoro sessionType = iota
	ShortBreak
	LongBreak
)

type sessionStatus string

const (
	STARTED   sessionStatus = "STARTED"
	STOPPED   sessionStatus = "STOPPED"
	COMPLETED sessionStatus = "COMPLETED"
	SKIPPED   sessionStatus = "SKIPPED"
)

type event struct {
	session         sessionType
	status          sessionStatus
	duration        int
	startTime       time.Time
	expectedEndTime time.Time
	actualEndTime   time.Time
}

type kind map[sessionType]int

type message map[sessionType]string

type Timer struct {
	currentSession      sessionType
	kind                kind
	autoStartPomodoro   bool
	autoStartBreak      bool
	longBreakInterval   int
	maxPomodoros        int
	counter             int
	iteration           int
	msg                 message
	showNotification    bool
	twentyFourHourClock bool
}

// nextSession retrieves the next session.
func (t *Timer) nextSession() sessionType {
	var next sessionType

	switch t.currentSession {
	case Pomodoro:
		if t.iteration == t.longBreakInterval {
			next = LongBreak
		} else {
			next = ShortBreak
		}
	case ShortBreak, LongBreak:
		next = Pomodoro
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

func (t *Timer) printSession(endTime time.Time) {
	var text string

	switch t.currentSession {
	case Pomodoro:
		var count int

		var total int

		if t.maxPomodoros != 0 {
			count = t.counter
			total = t.maxPomodoros
		} else {
			count = t.iteration
			total = t.longBreakInterval
		}

		text = fmt.Sprintf(PrintColor(green, "[Pomodoro %d/%d]"), count, total) + ": " + t.msg[Pomodoro]
	case ShortBreak:
		text = PrintColor(yellow, "[Short break]") + ": " + t.msg[ShortBreak]
	case LongBreak:
		text = PrintColor(blue, "[Long break]") + ": " + t.msg[LongBreak]
	}

	var tf string
	if t.twentyFourHourClock {
		tf = "15:04:05"
	} else {
		tf = "03:04:05 PM"
	}

	fmt.Printf("%s (until %s)\n", text, endTime.Format(tf))
}

func (t *Timer) notify() {
	fmt.Printf("Session completed!\n\n")

	m := map[sessionType]string{
		Pomodoro:   "Pomodoro",
		ShortBreak: "Short break",
		LongBreak:  "Long break",
	}

	if t.showNotification {
		msg := m[t.currentSession] + " is finished"

		// TODO: Handle error
		_ = beeep.Notify(msg, t.msg[t.nextSession()], "")
	}
}

// Start begins a new session.
func (t *Timer) Start(session sessionType) {
	t.currentSession = session

	t.counter++

	if session == Pomodoro {
		if t.iteration == t.longBreakInterval {
			t.iteration = 1
		} else {
			t.iteration++
		}
	}

	endTime := time.Now().Add(time.Duration(t.kind[session]) * time.Minute)

	t.printSession(endTime)

	fmt.Print("\033[s")

	timeRemaining := t.getTimeRemaining(endTime)

	t.countdown(timeRemaining)

	ticker := time.NewTicker(time.Second)
	for range ticker.C {
		fmt.Print("\033[u\033[K")

		timeRemaining = t.getTimeRemaining(endTime)

		if timeRemaining.t <= 0 {
			t.notify()
			break
		}

		t.countdown(timeRemaining)
	}

	if t.counter == t.maxPomodoros {
		return
	}

	if t.currentSession != Pomodoro && !t.autoStartPomodoro || t.currentSession == Pomodoro && !t.autoStartBreak {
		// Block until user input before beginning next session
		reader := bufio.NewReader(os.Stdin)

		fmt.Print("Press ENTER to start the next session\n")

		_, _ = reader.ReadString('\n')
	}

	t.Start(t.nextSession())
}

// countdown prints.
func (t *Timer) countdown(timeRemaining countdown) {
	fmt.Printf("Minutes: %02d Seconds: %02d", timeRemaining.m, timeRemaining.s)
}

// NewTimer returns a new timer constructed from
// command line arguments.
func NewTimer(ctx *cli.Context, c *Config) *Timer {
	t := &Timer{
		kind: kind{
			Pomodoro:   c.PomodoroMinutes,
			ShortBreak: c.ShortBreakMinutes,
			LongBreak:  c.LongBreakMinutes,
		},
		longBreakInterval: c.LongBreakInterval,
		msg: message{
			Pomodoro:   c.PomodoroMessage,
			ShortBreak: c.ShortBreakMessage,
			LongBreak:  c.LongBreakMessage,
		},
		showNotification:    c.Notify,
		autoStartPomodoro:   c.AutoStartPomorodo,
		autoStartBreak:      c.AutoStartBreak,
		twentyFourHourClock: c.TwentyFourHourClock,
	}

	// Command-line flags will override the configuration
	// file
	if ctx.Uint("pomodoro") > 0 {
		t.kind[Pomodoro] = int(ctx.Uint("pomodoro"))
	}

	if ctx.Uint("short-break") > 0 {
		t.kind[ShortBreak] = int(ctx.Uint("short-break"))
	}

	if ctx.Uint("long-break") > 0 {
		t.kind[LongBreak] = int(ctx.Uint("long-break"))
	}

	if ctx.Uint("long-break-interval") > 0 {
		t.longBreakInterval = int(ctx.Uint("long-break-interval"))
	}

	if ctx.Uint("max-pomodoros") > 0 {
		t.maxPomodoros = int(ctx.Uint("max-pomodoros"))
	}

	if ctx.Bool("auto-pomodoro") {
		t.autoStartPomodoro = true
	}

	if ctx.Bool("auto-break") {
		t.autoStartBreak = true
	}

	if ctx.Bool("disable-notifications") {
		t.showNotification = false
	}

	if t.longBreakInterval <= 0 {
		t.longBreakInterval = 4
	}

	if ctx.Bool("24-hour") {
		t.twentyFourHourClock = ctx.Bool("24-hour")
	}

	return t
}
