package focus

import (
	"testing"
	"time"
)

type totals struct {
	totalMins int
	completed int
	abandoned int
}

type test struct {
	period timePeriod
	start  time.Time
	end    time.Time
	count  int
	totals totals
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

	err := s.getSessions(s.StartTime, s.EndTime)
	if err != nil {
		t.Fatalf("Unexpected error while retrieving sessions: %s", err.Error())
	}

	s.compute()

	return s
}

func TestStats(t *testing.T) {
	cases := []test{
		{
			period: "all-time",
			totals: totals{18611, 391, 109},
			count:  500,
		},
		{
			start:  time.Date(2021, 1, 1, 0, 0, 0, 0, &time.Location{}),
			end:    time.Date(2021, 1, 31, 23, 59, 59, 0, &time.Location{}),
			count:  96,
			totals: totals{3584, 76, 20},
		},
		{
			start:  time.Date(2021, 3, 7, 0, 0, 0, 0, &time.Location{}),
			end:    time.Date(2021, 3, 15, 23, 59, 59, 0, &time.Location{}),
			count:  24,
			totals: totals{1000, 23, 1},
		},
		{
			start:  time.Date(2021, 6, 1, 0, 0, 0, 0, &time.Location{}),
			end:    time.Date(2021, 7, 15, 23, 59, 59, 0, &time.Location{}),
			count:  77,
			totals: totals{2854, 59, 18},
		},
	}

	for _, v := range cases {
		s := getStats(t, &v)

		if len(s.Sessions) != v.count {
			t.Errorf("Expected amount of sessions to be: %d, but got %d", v.count, len(s.Sessions))
		}

		for _, v2 := range s.Sessions {
			if v2.StartTime.Before(s.StartTime) || v2.StartTime.After(s.EndTime) {
				t.Errorf("Expected start time: %s to be within bounds: %s to %s", v2.StartTime.Format(time.RFC3339), s.StartTime.Format(time.RFC3339), s.EndTime.Format(time.RFC3339))
			}
		}

		if s.Data.Totals.minutes != v.totals.totalMins {
			t.Errorf(
				"Expected total minutes to be: %d, but got: %d",
				v.totals.totalMins,
				s.Data.Totals.minutes,
			)
		}

		if s.Data.Totals.completed != v.totals.completed {
			t.Errorf(
				"Expected completed pomodoros to be: %d, but got: %d",
				v.totals.completed,
				s.Data.Totals.completed,
			)
		}

		if s.Data.Totals.abandoned != v.totals.abandoned {
			t.Errorf(
				"Expected abandoned pomodoros to be: %d, but got: %d",
				v.totals.abandoned,
				s.Data.Totals.abandoned,
			)
		}
	}
}
