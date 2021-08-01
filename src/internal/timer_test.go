package focus

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"reflect"
	"strings"
	"testing"
	"time"
)

type DBMock struct{}

func (d *DBMock) init() error {
	return nil
}

func (d *DBMock) getSessions(startTime, endTime time.Time) ([][]byte, error) {
	jsonFile, err := os.ReadFile("../../testdata/pomodoro.json")
	if err != nil {
		return nil, err
	}

	var sessions []session

	err = json.Unmarshal(jsonFile, &sessions)
	if err != nil {
		return nil, err
	}

	result := [][]byte{}

	for _, v := range sessions {
		if v.StartTime.Before(startTime) || v.StartTime.After(endTime) {
			continue
		}

		b, err := json.Marshal(v)
		if err != nil {
			return nil, err
		}

		result = append(result, b)
	}

	return result, nil
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

func (d *DBMock) deleteSessions(startTime, endTime time.Time) error {
	return nil
}

func (d *DBMock) close() error {
	return nil
}

// TestTimer_InitSession confirms that the endtime
// is perfectly distanced from the start time
// by the specified amount of minutes.
func TestTimer_InitSession(t *testing.T) {
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

func TestTimer_GetNextSession(t *testing.T) {
	type testCase struct {
		input             sessionType
		output            sessionType
		longBreakInterval int
		pomodoroCycle     int
	}

	cases := []testCase{
		{pomodoro, shortBreak, 4, 2},
		{shortBreak, pomodoro, 4, 1},
		{longBreak, pomodoro, 4, 4},
		{pomodoro, longBreak, 4, 4},
	}

	for _, v := range cases {
		timer := &Timer{
			LongBreakInterval: v.longBreakInterval,
			PomodoroCycle:     v.pomodoroCycle,
			SessionType:       v.input,
		}

		got := timer.nextSession()

		if got != v.output {
			t.Fatalf(
				"Expected next session to be: %s, but got: %s",
				v.output,
				got,
			)
		}
	}
}

func TestTimer_PrintSession(t *testing.T) {
	type testCase struct {
		endTime             string
		sessionType         sessionType
		maxPomodoros        int
		pomodoroCycle       int
		longBreakInterval   int
		twentyFourHourClock bool
		expected            string
	}

	c := &Config{}
	c.defaults(false)

	cases := []testCase{
		{
			"2021-06-13T13:50:00Z",
			pomodoro,
			0,
			2,
			4,
			false,
			fmt.Sprintf(
				"[Pomodoro 2/4]: %s (until 01:50:00 PM)",
				c.PomodoroMessage,
			),
		},
		{
			"2021-06-18T20:00:00Z",
			pomodoro,
			8,
			4,
			4,
			true,
			fmt.Sprintf(
				"[Pomodoro 4/8]: %s (until 20:00:00)",
				c.PomodoroMessage,
			),
		},
		{
			"2021-07-01T00:10:00Z",
			shortBreak,
			0,
			1,
			4,
			false,
			fmt.Sprintf(
				"[Short break]: %s (until 12:10:00 AM)",
				c.ShortBreakMessage,
			),
		},
		{
			"2021-07-01T15:43:00Z",
			longBreak,
			0,
			4,
			4,
			false,
			fmt.Sprintf(
				"[Long break]: %s (until 03:43:00 PM)",
				c.LongBreakMessage,
			),
		},
	}

	for _, v := range cases {
		timer := &Timer{
			MaxPomodoros:        v.maxPomodoros,
			LongBreakInterval:   v.longBreakInterval,
			PomodoroCycle:       v.pomodoroCycle,
			Counter:             v.pomodoroCycle,
			TwentyFourHourClock: v.twentyFourHourClock,
			SessionType:         v.sessionType,
			Msg: message{
				pomodoro:   c.PomodoroMessage,
				shortBreak: c.ShortBreakMessage,
				longBreak:  c.LongBreakMessage,
			},
		}

		endTime, err := time.Parse(time.RFC3339, v.endTime)
		if err != nil {
			t.Fatal(err)
		}

		var buf bytes.Buffer

		timer.printSession(endTime, &buf)

		got := strings.TrimSpace(buf.String())

		if got != v.expected {
			t.Fatalf(
				"Expected print session output to be: '%s', but got: '%s'",
				v.expected,
				got,
			)
		}
	}
}

func TestTimer_GetTimeRemaining(t *testing.T) {
	cases := []countdown{
		{1827, 30, 27},
		{3232, 53, 52},
		{100, 1, 40},
		{360, 6, 0},
		{0, 0, 0},
		{-20, 0, -20},
		{-765, -12, -45},
	}

	for _, v := range cases {
		timer := &Timer{}
		endTime := time.Now().Add(time.Duration(v.t * int(time.Second)))

		got := timer.getTimeRemaining(endTime)

		if !reflect.DeepEqual(v, got) {
			t.Fatalf("Expected: %v, but got: %v", v, got)
		}
	}
}

func TestSession_ValidateEndTime(t *testing.T) {
	type testCase struct {
		startTime      string
		correctEndTime string
	}

	cases := []testCase{
		{
			startTime:      "2021-03-24T06:43:00Z",
			correctEndTime: "2021-03-24T09:25:00Z",
		},
		{
			startTime:      "2021-04-27T07:27:00Z",
			correctEndTime: "2021-04-27T08:10:00Z",
		},
		{
			startTime:      "2021-02-05T08:48:00Z",
			correctEndTime: "2021-02-05T12:41:00Z",
		},
		{
			startTime:      "2021-04-20T09:16:00Z",
			correctEndTime: "2021-04-20T10:04:00Z",
		},
	}

	for _, v := range cases {
		t.Run(v.startTime, func(t *testing.T) {
			b, err := os.ReadFile("../../testdata/bad_end_time.json")
			if err != nil {
				t.Fatal(err)
			}

			ss := []session{}

			err = json.Unmarshal(b, &ss)
			if err != nil {
				t.Fatal(err)
			}

			for _, v2 := range ss {
				if v.startTime == v2.StartTime.Format(time.RFC3339) {
					v2.validateEndTime()

					got := v2.EndTime.Format(time.RFC3339)
					if got != v.correctEndTime {
						t.Fatalf(
							"Expected end time to be: %s, but got: %s",
							v.correctEndTime,
							got,
						)
					}
				}
			}
		})
	}
}
