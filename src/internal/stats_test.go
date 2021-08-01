package focus

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/pterm/pterm"
	"github.com/sergi/go-diff/diffmatchpatch"
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
	start    time.Time
	end      time.Time
	count    int
	totals   totals
	averages totals
}

var statsCases = []statsCase{
	{
		start:    time.Date(2021, 1, 1, 0, 0, 0, 0, &time.Location{}),
		end:      time.Date(2021, 7, 1, 23, 59, 59, 0, &time.Location{}),
		totals:   totals{18611, 391, 109},
		count:    500,
		averages: totals{102, 2, 1},
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

type mockStatsCtx map[string]string

func (m mockStatsCtx) Set(name, value string) {
	m[name] = value
}

func (m mockStatsCtx) String(name string) string {
	if _, exists := m[name]; exists {
		return m[name]
	}

	return ""
}

func getStats(t *testing.T, test *statsCase) *Stats {
	t.Helper()

	db := &DBMock{}

	ctx := &mockStatsCtx{}

	start := test.start.Format("2006-01-02")
	end := test.end.Format("2006-01-02")

	ctx.Set("start", start)
	ctx.Set("end", end)

	s, err := NewStats(ctx, db)
	if err != nil {
		t.Fatal(err)
	}

	return s
}

func lineCounter(r io.Reader) (int, error) {
	buf := make([]byte, 32*1024)
	count := 0
	lineSep := []byte{'\n'}

	for {
		c, err := r.Read(buf)
		count += bytes.Count(buf[:c], lineSep)

		switch {
		case errors.Is(err, io.EOF):
			return count, nil

		case err != nil:
			return count, err
		}
	}
}

func TestStats_Compute(t *testing.T) {
	for _, v := range statsCases {
		s := getStats(t, &v)

		var buf bytes.Buffer

		err := s.Show(&buf)
		if err != nil {
			t.Fatal(err)
		}

		if len(s.Sessions) != v.count {
			t.Fatalf(
				"Expected amount of sessions to be: %d, but got %d",
				v.count,
				len(s.Sessions),
			)
		}

		for _, v2 := range s.Sessions {
			if v2.StartTime.Before(s.StartTime) ||
				v2.StartTime.After(s.EndTime) {
				t.Fatalf(
					"Expected start time: %s to be within bounds: %s to %s",
					v2.StartTime.Format(time.RFC3339),
					s.StartTime.Format(time.RFC3339),
					s.EndTime.Format(time.RFC3339),
				)
			}
		}

		if s.Data.Totals.minutes != v.totals.minutes {
			t.Fatalf(
				"Expected total minutes to be: %d, but got: %d",
				v.totals.minutes,
				s.Data.Totals.minutes,
			)
		}

		if s.Data.Totals.completed != v.totals.completed {
			t.Fatalf(
				"Expected total completed pomodoros to be: %d, but got: %d",
				v.totals.completed,
				s.Data.Totals.completed,
			)
		}

		if s.Data.Totals.abandoned != v.totals.abandoned {
			t.Fatalf(
				"Expected total abandoned pomodoros to be: %d, but got: %d",
				v.totals.abandoned,
				s.Data.Totals.abandoned,
			)
		}

		if s.Data.Averages.minutes != v.averages.minutes {
			t.Fatalf(
				"Expected average minutes to be: %d, but got: %d",
				v.averages.minutes,
				s.Data.Averages.minutes,
			)
		}

		if s.Data.Averages.completed != v.averages.completed {
			t.Fatalf(
				"Expected average completed pomodoros to be: %d, but got: %d",
				v.averages.completed,
				s.Data.Averages.completed,
			)
		}

		if s.Data.Averages.abandoned != v.averages.abandoned {
			t.Fatalf(
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
			t.Fatalf(
				"Expected period '%s' to yield: %d but got: %d",
				v.period,
				v.minsDiff,
				got,
			)
		}
	}
}

func TestStats_Show(t *testing.T) {
	for _, v := range statsCases {
		s := getStats(t, &v)

		pterm.DisableStyling()

		var buf bytes.Buffer

		err := s.Show(&buf)
		if err != nil {
			t.Fatal(err)
		}

		header := "Reporting period: " + s.StartTime.Format(
			"January 02, 2006",
		) + " - " + s.EndTime.Format(
			"January 02, 2006",
		) + "\n\n"
		summary := getSummary(&v)
		averages := getAverages(&v)
		history := getHistory(s)
		weekly := getWeekly(s)
		hourly := getHourly(s)

		expected := strings.TrimSpace(
			fmt.Sprint(header, summary, averages, history, weekly, hourly),
		)

		got := strings.TrimSpace(buf.String())

		if got != expected {
			dmp := diffmatchpatch.New()

			diffs := dmp.DiffMain(got, expected, false)

			t.Fatalf(dmp.DiffPrettyText(diffs))
		}
	}
}

func getSummary(v *statsCase) string {
	hours, minutes := minsToHoursAndMins(v.totals.minutes)
	expected := fmt.Sprintf(
		"Summary\nTotal time logged: %d hours %d minutes\nPomodoros completed: %d\nPomodoros abandoned: %d\n",
		hours,
		minutes,
		v.totals.completed,
		v.totals.abandoned,
	)

	return expected
}

func getAverages(v *statsCase) string {
	hours, minutes := minsToHoursAndMins(v.averages.minutes)

	expected := fmt.Sprintf(
		"\nAverages\nAverage time logged per day: %d hours %d minutes\nCompleted pomodoros per day: %d\nAbandoned pomodoros per day: %d\n",
		hours,
		minutes,
		v.averages.completed,
		v.averages.abandoned,
	)

	return expected
}

func getHistory(s *Stats) string {
	type keyValue struct {
		key   string
		value *quantity
	}

	sl := make([]keyValue, 0, len(s.Data.History))
	for k, v := range s.Data.History {
		sl = append(sl, keyValue{k, v})
	}

	sort.Slice(sl, func(i, j int) bool {
		iTime, err := time.Parse(s.Data.HistoryKeyFormat, sl[i].key)
		if err != nil {
			return true
		}

		jTime, err := time.Parse(s.Data.HistoryKeyFormat, sl[j].key)
		if err != nil {
			return true
		}

		return iTime.Before(jTime)
	})

	expected := "\nPomodoro history (minutes)"

	for _, v := range sl {
		expected += fmt.Sprintf("%s: %d\n", v.key, v.value.minutes)
	}

	return expected + "\n"
}

func getWeekly(s *Stats) string {
	sunday := s.Data.Weekday[0].minutes
	monday := s.Data.Weekday[1].minutes
	tuesday := s.Data.Weekday[2].minutes
	wednesday := s.Data.Weekday[3].minutes
	thursday := s.Data.Weekday[4].minutes
	friday := s.Data.Weekday[5].minutes
	saturday := s.Data.Weekday[6].minutes

	expected := fmt.Sprintf(
		"\nWeekly breakdown (minutes)Sunday: %d\nMonday: %d\nTuesday: %d\nWednesday: %d\nThursday: %d\nFriday: %d\nSaturday: %d\n\n",
		sunday,
		monday,
		tuesday,
		wednesday,
		thursday,
		friday,
		saturday,
	)

	return expected
}

func getHourly(s *Stats) string {
	t0 := s.Data.HourofDay[0].minutes
	t1 := s.Data.HourofDay[1].minutes
	t2 := s.Data.HourofDay[2].minutes
	t3 := s.Data.HourofDay[3].minutes
	t4 := s.Data.HourofDay[4].minutes
	t5 := s.Data.HourofDay[5].minutes
	t6 := s.Data.HourofDay[6].minutes
	t7 := s.Data.HourofDay[7].minutes
	t8 := s.Data.HourofDay[8].minutes
	t9 := s.Data.HourofDay[9].minutes
	t10 := s.Data.HourofDay[10].minutes
	t11 := s.Data.HourofDay[11].minutes
	t12 := s.Data.HourofDay[12].minutes
	t13 := s.Data.HourofDay[13].minutes
	t14 := s.Data.HourofDay[14].minutes
	t15 := s.Data.HourofDay[15].minutes
	t16 := s.Data.HourofDay[16].minutes
	t17 := s.Data.HourofDay[17].minutes
	t18 := s.Data.HourofDay[18].minutes
	t19 := s.Data.HourofDay[19].minutes
	t20 := s.Data.HourofDay[20].minutes
	t21 := s.Data.HourofDay[21].minutes
	t22 := s.Data.HourofDay[22].minutes
	t23 := s.Data.HourofDay[23].minutes

	expected := fmt.Sprintf(
		"\nHourly breakdown (minutes)12:00 AM: %d\n01:00 AM: %d\n02:00 AM: %d\n03:00 AM: %d\n04:00 AM: %d\n05:00 AM: %d\n06:00 AM: %d\n07:00 AM: %d\n08:00 AM: %d\n09:00 AM: %d\n10:00 AM: %d\n11:00 AM: %d\n12:00 PM: %d\n01:00 PM: %d\n02:00 PM: %d\n03:00 PM: %d\n04:00 PM: %d\n05:00 PM: %d\n06:00 PM: %d\n07:00 PM: %d\n08:00 PM: %d\n09:00 PM: %d\n10:00 PM: %d\n11:00 PM: %d\n\n",
		t0,
		t1,
		t2,
		t3,
		t4,
		t5,
		t6,
		t7,
		t8,
		t9,
		t10,
		t11,
		t12,
		t13,
		t14,
		t15,
		t16,
		t17,
		t18,
		t19,
		t20,
		t21,
		t22,
		t23,
	)

	return expected
}

func TestStats_List(t *testing.T) {
	for _, v := range statsCases {
		s := getStats(t, &v)

		var buf bytes.Buffer

		err := s.List(&buf)
		if err != nil {
			t.Fatal(err)
		}

		count, err := lineCounter(&buf)
		if err != nil {
			t.Fatal(err)
		}

		expected := v.count + 4 // account for table borders

		if count != expected {
			t.Fatalf(
				"Expected output to be: '%d', but got '%d'",
				expected,
				count,
			)
		}
	}
}

func TestStats_Delete(t *testing.T) {
	for _, v := range statsCases {
		s := getStats(t, &v)

		var buf bytes.Buffer

		var stdin bytes.Buffer

		stdin.Write([]byte("\n"))

		err := s.Delete(&buf, &stdin)
		if err != nil {
			t.Fatal(err)
		}

		count, err := lineCounter(&buf)
		if err != nil {
			t.Fatal(err)
		}

		expected := v.count + 4 // account for table borders

		if count != expected {
			t.Fatalf(
				"Expected output to be: '%d', but got '%d'",
				expected,
				count,
			)
		}
	}
}
