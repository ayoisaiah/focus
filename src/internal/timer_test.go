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
		timer.SessionType = work
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

// TestTimer_GetNextSession ensures that the correct session type
// is begun after a session ends.
func TestTimer_GetNextSession(t *testing.T) {
	type testCase struct {
		input             sessionType
		output            sessionType
		longBreakInterval int
		workCycle         int
	}

	cases := []testCase{
		{work, shortBreak, 4, 2},
		{shortBreak, work, 4, 1},
		{longBreak, work, 4, 4},
		{work, longBreak, 4, 4},
	}

	for _, v := range cases {
		timer := &Timer{
			LongBreakInterval: v.longBreakInterval,
			WorkCycle:         v.workCycle,
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

// TestTimer_PrintSession verifies the text that is printed to
// the terminal when a session begins.
func TestTimer_PrintSession(t *testing.T) {
	type testCase struct {
		endTime             string
		sessionType         sessionType
		maxSessions         int
		workCycle           int
		longBreakInterval   int
		twentyFourHourClock bool
		expected            string
	}

	c := &Config{}
	c.defaults()

	cases := []testCase{
		{
			"2021-06-13T13:50:00Z",
			work,
			0,
			2,
			4,
			false,
			fmt.Sprintf(
				"[Work 2/4]: %s (until 01:50:00 PM)",
				c.WorkMessage,
			),
		},
		{
			"2021-06-18T20:00:00Z",
			work,
			8,
			4,
			4,
			true,
			fmt.Sprintf(
				"[Work 4/8]: %s (until 20:00:00)",
				c.WorkMessage,
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
			MaxSessions:         v.maxSessions,
			LongBreakInterval:   v.longBreakInterval,
			WorkCycle:           v.workCycle,
			Counter:             v.workCycle,
			TwentyFourHourClock: v.twentyFourHourClock,
			SessionType:         v.sessionType,
			Msg: message{
				work:       c.WorkMessage,
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
