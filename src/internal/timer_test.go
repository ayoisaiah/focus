package focus

import (
	"testing"
	"time"
)

type DBMock struct{}

func (d *DBMock) init() error {
	return nil
}

func (d *DBMock) getSessions(start, end time.Time) ([][]byte, error) {
	return nil, nil
}

func (d *DBMock) deleteTimerState() error {
	return nil
}

func (d *DBMock) getTimerState() (timer, session []byte, err error) {
	return nil, nil, nil
}

func (d *DBMock) saveTimerState(timer, sessionKey []byte) error {
	return nil
}

func (d *DBMock) updateSession(key, value []byte) error {
	return nil
}

// TestTimerInitSession confirms that the endtime
// is perfectly distanced from the start time
// by the specified amount of minutes.
func TestTimerInitSession(t *testing.T) {
	table := []struct {
		duration int
	}{
		{1}, {10}, {25}, {45}, {75}, {90}, {120},
	}

	for _, v := range table {
		timer := &Timer{}
		timer.SessionType = pomodoro
		timer.Kind = make(kind)
		timer.Kind[timer.SessionType] = v.duration
		timer.Store = &DBMock{}

		endTime, err := timer.initSession()
		if err != nil {
			t.Fatal(err)
		}

		startTime := timer.Session.StartTime

		got := endTime.Sub(startTime).Minutes()

		if float64(v.duration) != got {
			t.Errorf("Expected: %d, but got: %f", v.duration, got)
		}
	}
}
