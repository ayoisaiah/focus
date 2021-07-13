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

func (d *DBMock) getTimerState() ([]byte, []byte, error) {
	return nil, nil, nil
}

func (d *DBMock) saveTimerState(timer, sessionKey []byte) error {
	return nil
}

func (d *DBMock) updateSession(key, value []byte) error {
	return nil
}

func setupDB(t *testing.T) {
	t.Helper()
}

func TestTimerInitSession(t *testing.T) {
	setupDB(t)

	table := []struct {
		duration int
	}{
		{10}, {25}, {45}, {75},
	}

	for _, v := range table {
		timer := &Timer{}
		timer.SessionType = pomodoro
		timer.Kind = make(kind)
		timer.Kind[timer.SessionType] = v.duration
		timer.Store = &DBMock{}

		now := time.Now()

		endTime := timer.initSession()
		got := int(endTime.Sub(now).Minutes())

		if v.duration != got {
			t.Errorf("Expected: %d, but got: %d", v.duration, got)
		}
	}
}
