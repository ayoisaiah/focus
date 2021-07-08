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

type Timer struct {
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

func (t *Timer) nextSession() {
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
func (t *Timer) getTimeRemaining(endTime time.Time) countdown {
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

func (t *Timer) printSession(endTime time.Time) {
	var text string

	switch t.currentSession {
	case pomodoro:
		text = printColor(green, "Focus on your task! "+fmt.Sprintf("(%d/%d)", t.iteration, t.longBreakInterval))
	case shortBreak:
		text = printColor(yellow, "Take a breather!")
	case longBreak:
		text = printColor(yellow, "Take a long break!")
	}

	fmt.Printf("%s (until %s)\n", text, endTime.Format("03:04:05 PM"))
}

// start begins a new session.
func (t *Timer) start(session sessionType) {
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
			fmt.Printf("Countdown reached!\n\n")
			break
		}

		fmt.Printf("Hours: %02d Minutes: %02d Seconds: %02d", timeRemaining.h, timeRemaining.m, timeRemaining.s)
	}

	t.nextSession()
}

// newTimer returns a new timer constructed from
// command line arguments.
func newTimer(c *cli.Context) (*Timer, error) {
	config, err := newConfig()
	if err != nil {
		return nil, err
	}

	t := &Timer{
		kind: kind{
			pomodoro:   config.PomodoroMinutes,
			shortBreak: config.ShortBreakMinutes,
			longBreak:  config.LongBreakMinutes,
		},
		longBreakInterval: config.LongBreakInterval,
		pomodoroMessage:   config.PomodoroMessage,
		shortBreakMessage: config.ShortBreakMessage,
		longBreakMessage:  config.LongBreakMessage,
	}

	if c.Uint("pomodoro") > 0 {
		t.kind[pomodoro] = int(c.Uint("pomodoro"))
	}

	if c.Uint("shortBreak") > 0 {
		t.kind[shortBreak] = int(c.Uint("shortBreak"))
	}

	if c.Uint("longBreak") > 0 {
		t.kind[longBreak] = int(c.Uint("longBreak"))
	}

	if c.Uint("long-break-interval") > 0 {
		t.longBreakInterval = int(c.Uint("long-break-interval"))
	}

	if t.longBreakInterval <= 0 {
		t.longBreakInterval = 4
	}

	return t, nil
}
