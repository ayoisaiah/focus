package focus

import (
	"encoding/json"
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/urfave/cli/v2"
)

type statsPeriod string

const (
	periodAllTime   statsPeriod = "all-time"
	periodToday     statsPeriod = "today"
	periodYesterday statsPeriod = "yesterday"
	period24Hours   statsPeriod = "24hours"
	period7Days     statsPeriod = "7days"
	period14Days    statsPeriod = "14days"
	period30Days    statsPeriod = "30days"
	period90Days    statsPeriod = "90days"
	period180Days   statsPeriod = "180days"
	period365Days   statsPeriod = "365days"
)

var period = []statsPeriod{periodAllTime, periodToday, periodYesterday, period7Days, period14Days, period30Days, period90Days, period180Days, period365Days}

type Stats struct {
	StartDate          time.Time
	EndDate            time.Time
	Sessions           []session
	totalMins          int
	completedPomodoros int
	abandonedPomodoros int
	Weekday            map[time.Weekday]int
	HourofDay          map[int]int
}

func (s *Stats) getSessions() {
	b, err := store.getSessions(s.StartDate, s.EndDate)
	if err != nil {
		fmt.Println(err)
		return
	}

	for _, v := range b {
		sess := session{}

		err = json.Unmarshal(v, &sess)
		if err != nil {
			fmt.Println(err)
			return
		}

		s.Sessions = append(s.Sessions, sess)
	}
}

func (s *Stats) total() {
	for _, v := range s.Sessions {
		if v.EndTime.IsZero() {
			continue
		}

		s.Weekday[v.StartTime.Weekday()]++
		s.HourofDay[v.StartTime.Hour()]++

		if v.Completed {
			s.completedPomodoros++
		} else {
			s.abandonedPomodoros++
		}

		s.totalMins += int(math.Round(v.EndTime.Sub(v.StartTime).Minutes()))
	}

	fmt.Println("Total minutes worked: ", s.totalMins)
	fmt.Println("Total pomodoros completed: ", s.completedPomodoros)
	fmt.Println("Total pomodoros abandoned: ", s.abandonedPomodoros)
	fmt.Println("Weekdays")

	for k, v := range s.Weekday {
		fmt.Println(k.String(), " -> ", v)
	}
}

func (s *Stats) Run() {
	s.getSessions()
	s.total()
}

func getPeriod(period statsPeriod) (startTime, endTime time.Time) {
	switch period {
	case periodToday:
		now := time.Now()
		return time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location()), now
	case periodYesterday:
		yesterday := time.Now().AddDate(0, 0, -1)
		year, month, day := yesterday.Date()

		return time.Date(year, month, day, 0, 0, 0, 0, yesterday.Location()), time.Date(year, month, day, 23, 59, 59, 0, yesterday.Location())
	case period24Hours:
		return time.Now().AddDate(0, 0, -1), time.Now()
	case period7Days:
		return time.Now().AddDate(0, 0, -7), time.Now()
	case period14Days:
		return time.Now().AddDate(0, 0, -14), time.Now()
	case period30Days:
		return time.Now().AddDate(0, 0, -30), time.Now()
	case period90Days:
		return time.Now().AddDate(0, 0, -90), time.Now()
	case period180Days:
		return time.Now().AddDate(0, 0, -180), time.Now()
	case period365Days:
		return time.Now().AddDate(0, 0, -365), time.Now()
	case periodAllTime:
		return time.Time{}, time.Now()
	}

	return time.Time{}, time.Now()
}

func NewStats(ctx *cli.Context) (*Stats, error) {
	s := &Stats{}

	s.Weekday = make(map[time.Weekday]int)
	s.HourofDay = make(map[int]int)

	for i := 0; i <= 6; i++ {
		s.Weekday[time.Weekday(i)] = 0
	}

	for i := 0; i <= 23; i++ {
		s.HourofDay[i] = 0
	}

	p := ctx.String("period")

	if !contains(period, statsPeriod(p)) {
		var sl []string
		for _, v := range period {
			sl = append(sl, string(v))
		}

		return nil, fmt.Errorf("Period must be one of: %s", strings.Join(sl, ", "))
	}

	s.StartDate, s.EndDate = getPeriod(statsPeriod(p))

	start := ctx.String("start")
	end := ctx.String("end")

	if start != "" {
		v, err := time.Parse("2006-01-02", start)
		if err != nil {
			return nil, err
		}

		s.StartDate = time.Date(v.Year(), v.Month(), v.Day(), 0, 0, 0, 0, v.Location())
	}

	if end != "" {
		v, err := time.Parse("2006-01-02", end)
		if err != nil {
			return nil, err
		}

		s.EndDate = time.Date(v.Year(), v.Month(), v.Day(), 23, 59, 59, 0, v.Location())
	}

	if int(s.EndDate.Sub(s.StartDate).Seconds()) < 0 {
		return nil, fmt.Errorf("The end date must not be less than the start date")
	}

	return s, nil
}

// contains checks if a string is present in
// a string slice.
func contains(s []statsPeriod, e statsPeriod) bool {
	for _, a := range s {
		if a == e {
			return true
		}
	}

	return false
}
