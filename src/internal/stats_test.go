package focus

import (
	"encoding/json"
	"os"
	"testing"
	"time"
)

type totals struct {
	totalMins int
	completed int
	abandoned int
}

type test struct {
	period   timePeriod
	start    time.Time
	end      time.Time
	expected totals
}

func getStats(t *testing.T, test *test) *Stats {
	t.Helper()

	s := &Stats{}
	s.store = &DBMock{}

	if test.period != "" {
		s.StartTime, s.EndTime = getPeriod(test.period)
	} else {
		s.StartTime = test.start
		s.EndTime = test.end
	}

	diff := s.EndTime.Sub(s.StartTime)
	s.HoursDiff = int(diff.Hours())
	s.Data = initData(s.StartTime, s.EndTime, s.HoursDiff)

	jsonFile, err := os.ReadFile("../../testdata/pomodoro.json")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	var sessions []session

	err = json.Unmarshal(jsonFile, &sessions)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	s.Sessions = sessions

	s.compute()

	return s
}

func TestStatsTotals(t *testing.T) {
	cases := []test{
		{
			period:   "all-time",
			expected: totals{21400, 257, 243},
		},
		// {
		// 	start:    time.Date(2021, 1, 1, 0, 0, 0, 0, &time.Location{}),
		// 	end:      time.Date(2021, 1, 31, 23, 59, 59, 0, &time.Location{}),
		// 	expected: totals{3756, 50, 35},
		// },
	}

	for _, v := range cases {
		s := getStats(t, &v)

		if s.Data.Totals.minutes != v.expected.totalMins {
			t.Errorf(
				"Expected total minutes to be: %d, but got: %d",
				v.expected.totalMins,
				s.Data.Totals.minutes,
			)
		}

		if s.Data.Totals.completed != v.expected.completed {
			t.Errorf(
				"Expected completed pomodoros to be: %d, but got: %d",
				v.expected.completed,
				s.Data.Totals.completed,
			)
		}

		if s.Data.Totals.abandoned != v.expected.abandoned {
			t.Errorf(
				"Expected abandoned pomodoros to be: %d, but got: %d",
				v.expected.abandoned,
				s.Data.Totals.abandoned,
			)
		}
	}
}
