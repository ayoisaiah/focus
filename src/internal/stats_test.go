package focus

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"testing"
	"time"

	"github.com/andreyvit/diff"
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
				"Expected total completed sessions to be: %d, but got: %d",
				v.totals.completed,
				s.Data.Totals.completed,
			)
		}

		if s.Data.Totals.abandoned != v.totals.abandoned {
			t.Fatalf(
				"Expected total abandoned sessions to be: %d, but got: %d",
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
				"Expected average completed sessions to be: %d, but got: %d",
				v.averages.completed,
				s.Data.Averages.completed,
			)
		}

		if s.Data.Averages.abandoned != v.averages.abandoned {
			t.Fatalf(
				"Expected average abandoned sessions to be: %d, but got: %d",
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
	for i, v := range statsCases {
		s := getStats(t, &v)

		pterm.DisableStyling()

		var buf bytes.Buffer

		err := s.Show(&buf)
		if err != nil {
			t.Fatal(err)
		}

		got := buf.String()

		expected := goldenFile(
			t,
			fmt.Sprintf("stats_show_%d.golden", i+1),
			got,
			*update,
		)

		if got != expected {
			t.Fatalf(diff.LineDiff(got, expected))
		}
	}
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
