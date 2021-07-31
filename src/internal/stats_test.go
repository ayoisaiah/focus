package focus

import (
	"bytes"
	"fmt"
	"testing"
	"time"

	"github.com/pterm/pterm"
)

func init() {
	pterm.DisableColor()
}

type totals struct {
	minutes   int
	completed int
	abandoned int
}

type statsCase struct {
	period   timePeriod
	start    time.Time
	end      time.Time
	count    int
	totals   totals
	averages totals
}

var statsCases = []statsCase{
	{
		period:   "all-time",
		totals:   totals{18611, 391, 109},
		count:    500,
		averages: totals{103, 2, 1},
	},
	{
		start:    time.Date(2021, 1, 1, 0, 0, 0, 0, &time.Location{}),
		end:      time.Date(2021, 1, 31, 23, 59, 59, 0, &time.Location{}),
		count:    96,
		totals:   totals{3584, 76, 20},
		averages: totals{116, 2, 1},
	},
	{
		start:    time.Date(2021, 3, 7, 0, 0, 0, 0, &time.Location{}),
		end:      time.Date(2021, 3, 15, 23, 59, 59, 0, &time.Location{}),
		count:    24,
		totals:   totals{1000, 23, 1},
		averages: totals{111, 3, 0},
	},
	{
		start:    time.Date(2021, 6, 1, 0, 0, 0, 0, &time.Location{}),
		end:      time.Date(2021, 7, 15, 23, 59, 59, 0, &time.Location{}),
		count:    77,
		totals:   totals{2854, 59, 18},
		averages: totals{63, 1, 0},
	},
}

func getStats(t *testing.T, test *statsCase) *Stats {
	t.Helper()

	s := &Stats{}
	s.store = &DBMock{}

	if test.period != "" {
		s.StartTime, s.EndTime = getPeriod(test.period)
	} else {
		s.StartTime = test.start
		s.EndTime = test.end
	}

	err := s.getSessions(s.StartTime, s.EndTime)
	if err != nil {
		t.Fatalf("Unexpected error while retrieving sessions: %s", err.Error())
	}

	if s.StartTime.IsZero() && len(s.Sessions) > 0 {
		earliest := time.Now()
		latest := time.Time{}

		for _, v := range s.Sessions {
			if v.StartTime.Before(earliest) {
				earliest = v.StartTime
			}

			if v.StartTime.After(latest) {
				latest = v.StartTime
			}
		}

		s.StartTime = time.Date(earliest.Year(), earliest.Month(), earliest.Day(), 0, 0, 0, 0, earliest.Location())

		if test.period == periodAllTime {
			s.EndTime = latest
		}
	}

	diff := s.EndTime.Sub(s.StartTime)
	s.HoursDiff = int(diff.Hours())
	s.Data = initData(s.StartTime, s.EndTime, s.HoursDiff)

	s.compute()

	return s
}

func TestStats_Compute(t *testing.T) {
	for _, v := range statsCases {
		s := getStats(t, &v)

		if len(s.Sessions) != v.count {
			t.Errorf("Expected amount of sessions to be: %d, but got %d", v.count, len(s.Sessions))
		}

		for _, v2 := range s.Sessions {
			if v2.StartTime.Before(s.StartTime) || v2.StartTime.After(s.EndTime) {
				t.Errorf("Expected start time: %s to be within bounds: %s to %s", v2.StartTime.Format(time.RFC3339), s.StartTime.Format(time.RFC3339), s.EndTime.Format(time.RFC3339))
			}
		}

		if s.Data.Totals.minutes != v.totals.minutes {
			t.Errorf(
				"Expected total minutes to be: %d, but got: %d",
				v.totals.minutes,
				s.Data.Totals.minutes,
			)
		}

		if s.Data.Totals.completed != v.totals.completed {
			t.Errorf(
				"Expected total completed pomodoros to be: %d, but got: %d",
				v.totals.completed,
				s.Data.Totals.completed,
			)
		}

		if s.Data.Totals.abandoned != v.totals.abandoned {
			t.Errorf(
				"Expected total abandoned pomodoros to be: %d, but got: %d",
				v.totals.abandoned,
				s.Data.Totals.abandoned,
			)
		}

		if s.Data.Averages.minutes != v.averages.minutes {
			t.Errorf(
				"Expected average minutes to be: %d, but got: %d",
				v.averages.minutes,
				s.Data.Averages.minutes,
			)
		}

		if s.Data.Averages.completed != v.averages.completed {
			t.Errorf(
				"Expected average completed pomodoros to be: %d, but got: %d",
				v.averages.completed,
				s.Data.Averages.completed,
			)
		}

		if s.Data.Averages.abandoned != v.averages.abandoned {
			t.Errorf(
				"Expected average abandoned pomodoros to be: %d, but got: %d",
				v.averages.abandoned,
				s.Data.Averages.abandoned,
			)
		}
	}
}

func TestGetPeriod(t *testing.T) {
	type testCase struct {
		period   timePeriod
		minsDiff int
	}

	zero := time.Time{}
	now := time.Now()

	cases := []testCase{
		{
			periodAllTime,
			roundTime(now.Sub(zero).Minutes()),
		},
		{
			periodToday,
			1440,
		},
		{
			periodYesterday,
			1440,
		},
		{
			period7Days,
			10080,
		},
		{
			period14Days,
			20160,
		},
		{
			period30Days,
			43200,
		},
		{
			period90Days,
			129600,
		},
		{
			period180Days,
			259200,
		},
		{
			period365Days,
			525600,
		},
	}

	for _, v := range cases {
		start, end := getPeriod(v.period)

		got := roundTime(end.Sub(start).Minutes())

		if got != v.minsDiff {
			t.Fatalf("Expected period '%s' to yield: %d but got: %d", v.period, v.minsDiff, got)
		}
	}
}

func TestStats_DisplaySummary(t *testing.T) {
	for _, v := range statsCases {
		hours, minutes := minsToHoursAndMins(v.totals.minutes)

		expected := fmt.Sprintf("Summary\nTotal time logged: %d hours %d minutes\nPomodoros completed: %d\nPomodoros abandoned: %d\n", hours, minutes, v.totals.completed, v.totals.abandoned)

		s := getStats(t, &v)

		var buf bytes.Buffer

		s.displaySummary(&buf)

		if expected != buf.String() {
			t.Fatalf("Expected output to be: %s, but got: %s", expected, buf.String())
		}
	}
}

func TestStats_DisplayAverages(t *testing.T) {
	for _, v := range statsCases {
		hours, minutes := minsToHoursAndMins(v.averages.minutes)

		expected := fmt.Sprintf("\nAverages\nAverage time logged per day: %d hours %d minutes\nCompleted pomodoros per day: %d\nAbandoned pomodoros per day: %d\n", hours, minutes, v.averages.completed, v.averages.abandoned)

		s := getStats(t, &v)

		var buf bytes.Buffer

		s.displayAverages(&buf)

		if expected != buf.String() {
			t.Fatalf("Expected output to be %s, but got: %s", expected, buf.String())
		}
	}
}
