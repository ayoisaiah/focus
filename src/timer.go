package focus

import (
	"fmt"
	"time"

	"github.com/urfave/cli/v2"
)

type sessionType int

type countdown struct {
	t int
	h int
	m int
	s int
}

const (
	pomodoro sessionType = iota
	shortBreak
	longBreak
)

type sessionStatus string

const (
	STARTED   sessionStatus = "STARTED"
	STOPPED   sessionStatus = "STOPPED"
	COMPLETED sessionStatus = "COMPLETED"
	SKIPPED   sessionStatus = "SKIPPED"
)

type kind map[sessionType]int

type event struct {
	session         sessionType
	status          sessionStatus
	duration        int
	startTime       time.Time
	expectedEndTime time.Time
	actualEndTime   time.Time
}

type timer struct {
	currentSession    sessionType
	kind              kind
	autoStart         bool
	longBreakInterval int
	Events            []event
	maxPomodoros      int
	iteration         int
	pomodoroMessage   string
	longBreakMessage  string
	shortBreakMessage string
}

func (t *timer) nextSession() {
	var next sessionType

	switch t.currentSession {
	case pomodoro:
		if t.iteration == t.longBreakInterval {
			next = longBreak
		} else {
			next = shortBreak
		}
	case shortBreak, longBreak:
		next = pomodoro
	}

	t.start(next)
}

// getTimeRemaining subtracts the endTime from the currentTime
// and returns the total number of hours, minutes and seconds
// left.
func (t *timer) getTimeRemaining(endTime time.Time) countdown {
	currentTime := time.Now()
	difference := endTime.Sub(currentTime)

	total := int(difference.Seconds())
	hours := total / (60 * 60) % 24
	minutes := total / 60 % 60
	seconds := total % 60

	return countdown{
		t: total,
		h: hours,
		m: minutes,
		s: seconds,
	}
}

func (t *timer) printSession(endTime time.Time) {
	var text string

	switch t.currentSession {
	case pomodoro:
		text = fmt.Sprintf(printColor(green, "[Pomodoro %d/%d]"), t.iteration, t.longBreakInterval) + ": " + t.pomodoroMessage
	case shortBreak:
		text = printColor(yellow, "[Short break]") + ": " + t.shortBreakMessage
	case longBreak:
		text = printColor(blue, "[Long break]") + ": " + t.longBreakMessage
	}

	fmt.Printf("%s (until %s)\n", text, endTime.Format("03:04:05 PM"))
}

// start begins a new session.
func (t *timer) start(session sessionType) {
	t.currentSession = session

	if session == pomodoro {
		if t.iteration == t.longBreakInterval {
			t.iteration = 1
		} else {
			t.iteration++
		}
	}

	endTime := time.Now().Add(time.Duration(t.kind[session]) * time.Minute)

	t.printSession(endTime)

	ev := event{
		session:         session,
		status:          STARTED,
		duration:        t.kind[session],
		startTime:       time.Now(),
		expectedEndTime: endTime,
	}

	t.Events = append(t.Events, ev)

	fmt.Print("\033[s")

	ticker := time.NewTicker(time.Second)
	for range ticker.C {
		fmt.Print("\033[u\033[K")

		timeRemaining := t.getTimeRemaining(endTime)

		if timeRemaining.t <= 0 {
			fmt.Printf("Session completed!\n\n")
			break
		}

		fmt.Printf("Hours: %02d Minutes: %02d Seconds: %02d", timeRemaining.h, timeRemaining.m, timeRemaining.s)
	}

	t.nextSession()
}

// newTimer returns a new timer constructed from
// command line arguments.
func newTimer(ctx *cli.Context, c *config) *timer {
	t := &timer{
		kind: kind{
			pomodoro:   c.PomodoroMinutes,
			shortBreak: c.ShortBreakMinutes,
			longBreak:  c.LongBreakMinutes,
		},
		longBreakInterval: c.LongBreakInterval,
		pomodoroMessage:   c.PomodoroMessage,
		shortBreakMessage: c.ShortBreakMessage,
		longBreakMessage:  c.LongBreakMessage,
	}

	if ctx.Uint("pomodoro") > 0 {
		t.kind[pomodoro] = int(ctx.Uint("pomodoro"))
	}

	if ctx.Uint("short-break") > 0 {
		t.kind[shortBreak] = int(ctx.Uint("short-break"))
	}

	if ctx.Uint("long-break") > 0 {
		t.kind[longBreak] = int(ctx.Uint("long-break"))
	}

	if ctx.Uint("long-break-interval") > 0 {
		t.longBreakInterval = int(ctx.Uint("long-break-interval"))
	}

	if t.longBreakInterval <= 0 {
		t.longBreakInterval = 4
	}

	return t
}
