package focus

import (
	"encoding/json"
	"os"
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

func TestSessionValidateEndTime(t *testing.T) {
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
						t.Fatalf("Expected end time to be: %s, but got: %s", v.correctEndTime, got)
					}
				}
			}
		})
	}
}
