package werk

import (
	"time"

	"github.com/urfave/cli/v2"
)

type sessionType int

const (
	pomodoro sessionType = iota
	shortBreak
	longBreak
)

type sessionStatus string

const (
	STOPPED   sessionStatus = "STOPPED"
	COMPLETED sessionStatus = "COMPLETED"
	SKIPPED   sessionStatus = "SKIPPED"
)

type Timer struct {
	Pomodoro   int
	ShortBreak int
	LongBreak  int
}

type event struct {
	session         sessionType
	status          sessionStatus
	duration        int
	startTime       time.Time
	expectedEndTime time.Time
	actualEndTime   time.Time
}

type Werk struct {
	Timer
	AutoStart         bool
	LongBreakSessions int
	Events            []event
}

func newWerk(c *cli.Context) *Werk {
	w := &Werk{}

	w.Pomodoro = int(c.Uint("pomodoro"))
	w.LongBreak = int(c.Uint("long"))
	w.ShortBreak = int(c.Uint("short"))
	w.LongBreakSessions = 4

	return w
}
