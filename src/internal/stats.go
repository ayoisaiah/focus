package focus

import (
	"encoding/json"
	"fmt"
	"math"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/olekukonko/tablewriter"
	"github.com/pterm/pterm"
	"github.com/urfave/cli/v2"
)

type timePeriod string

const (
	PeriodAllTime   timePeriod = "all-time"
	PeriodToday     timePeriod = "today"
	PeriodYesterday timePeriod = "yesterday"
	Period24Hours   timePeriod = "24hours"
	Period7Days     timePeriod = "7days"
	Period14Days    timePeriod = "14days"
	Period30Days    timePeriod = "30days"
	Period90Days    timePeriod = "90days"
	Period180Days   timePeriod = "180days"
	Period365Days   timePeriod = "365days"
)

var StatsPeriod = []timePeriod{PeriodAllTime, PeriodToday, PeriodYesterday, Period7Days, Period14Days, Period30Days, Period90Days, Period180Days, Period365Days}

type pomo struct {
	totalMins          int
	completedPomodoros int
	abandonedPomodoros int
}

type Stats struct {
	StartDate time.Time
	EndDate   time.Time
	pomo
	Sessions  []session
	Weekday   map[time.Weekday]*pomo
	HourofDay map[int]*pomo
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

func (s *Stats) hourly() {
	type keyValue struct {
		key   int
		value *pomo
	}

	sl := make([]keyValue, 0, len(s.HourofDay))
	for k, v := range s.HourofDay {
		sl = append(sl, keyValue{k, v})
	}

	sort.SliceStable(sl, func(i, j int) bool {
		return sl[i].key < sl[j].key
	})

	var data = make([][]string, len(sl))

	for _, v := range sl {
		val := s.HourofDay[v.key]
		completed := strconv.Itoa(val.completedPomodoros)
		abandoned := strconv.Itoa(val.abandonedPomodoros)
		total := strconv.Itoa(val.totalMins)

		data = append(data, []string{fmt.Sprintf("%02d:00", v.key), total, completed, abandoned})
	}

	printTable("hours", data)
}

func (s *Stats) weekdays() {
	type keyValue struct {
		key   time.Weekday
		value *pomo
	}

	sl := make([]keyValue, 0, len(s.Weekday))
	for k, v := range s.Weekday {
		sl = append(sl, keyValue{k, v})
	}

	sort.SliceStable(sl, func(i, j int) bool {
		return int(sl[i].key) < int(sl[j].key)
	})

	var data = make([][]string, len(sl))

	for _, v := range sl {
		val := s.Weekday[v.key]
		completed := strconv.Itoa(val.completedPomodoros)
		abandoned := strconv.Itoa(val.abandonedPomodoros)
		total := strconv.Itoa(val.totalMins)

		data = append(data, []string{v.key.String(), total, completed, abandoned})
	}

	printTable("weekday", data)
}

func (s *Stats) total() {
	for _, v := range s.Sessions {
		if v.EndTime.IsZero() {
			continue
		}

		if v.Completed {
			s.Weekday[v.StartTime.Weekday()].completedPomodoros++
			s.HourofDay[v.StartTime.Hour()].completedPomodoros++
			s.completedPomodoros++
		} else {
			s.Weekday[v.StartTime.Weekday()].abandonedPomodoros++
			s.HourofDay[v.StartTime.Hour()].abandonedPomodoros++
			s.abandonedPomodoros++
		}

		s.Weekday[v.StartTime.Weekday()].totalMins += int(math.Round(v.EndTime.Sub(v.StartTime).Minutes()))
		s.HourofDay[v.StartTime.Hour()].totalMins += int(math.Round(v.EndTime.Sub(v.StartTime).Minutes()))
		s.totalMins += int(math.Round(v.EndTime.Sub(v.StartTime).Minutes()))
	}

	hours := int(math.Floor(float64(s.totalMins) / float64(60)))
	minutes := s.totalMins % 60

	fmt.Printf("Time logged: %s %s %s %s\n", pterm.Green(hours), pterm.Green("hours"), pterm.Green(minutes), pterm.Green("minutes"))

	fmt.Println("Pomodoros completed:", pterm.Green(s.completedPomodoros))
	fmt.Println("Pomodoros abandoned:", pterm.Green(s.abandonedPomodoros))
}

func (s *Stats) Run() {
	s.getSessions()

	startDate := s.StartDate.Format("January 02, 2006")
	endDate := s.EndDate.Format("January 02, 2006")
	timePeriod := startDate + " — " + endDate

	pterm.DefaultHeader.Printfln(timePeriod)

	pterm.FgBlue.Printfln("Totals")
	s.total()

	fmt.Println()
	pterm.FgBlue.Printfln("Averages")
	s.average()

	fmt.Println()
	pterm.FgBlue.Printfln("Weekly breakdown")
	s.weekdays()

	fmt.Println()
	pterm.FgBlue.Printfln("Hourly breakdown")
	s.hourly()
}

// getPeriod returns the start and end time according to the
// specified time period.
func getPeriod(period timePeriod) (startTime, endTime time.Time) {
	switch period {
	case PeriodToday:
		now := time.Now()
		return time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location()), now
	case PeriodYesterday:
		yesterday := time.Now().AddDate(0, 0, -1)
		year, month, day := yesterday.Date()

		return time.Date(year, month, day, 0, 0, 0, 0, yesterday.Location()), time.Date(year, month, day, 23, 59, 59, 0, yesterday.Location())
	case Period24Hours:
		return time.Now().AddDate(0, 0, -1), time.Now()
	case Period7Days:
		return time.Now().AddDate(0, 0, -7), time.Now()
	case Period14Days:
		return time.Now().AddDate(0, 0, -14), time.Now()
	case Period30Days:
		return time.Now().AddDate(0, 0, -30), time.Now()
	case Period90Days:
		return time.Now().AddDate(0, 0, -90), time.Now()
	case Period180Days:
		return time.Now().AddDate(0, 0, -180), time.Now()
	case Period365Days:
		return time.Now().AddDate(0, 0, -365), time.Now()
	case PeriodAllTime:
		return time.Time{}, time.Now()
	}

	return time.Time{}, time.Now()
}

func (s *Stats) average() {
	hoursDiff := int(math.Round(s.EndDate.Sub(s.StartDate).Hours()))
	hoursInADay := 24

	if hoursDiff > hoursInADay {
		numberOfDays := hoursDiff / hoursInADay
		avgMins := math.Round(float64(s.totalMins) / float64(numberOfDays))
		avgCompleted := math.Round(float64(s.completedPomodoros) / float64(numberOfDays))
		avgAbandoned := math.Round(float64(s.abandonedPomodoros) / float64(numberOfDays))

		fmt.Println("Daily minutes:", pterm.Green(int(avgMins)))
		fmt.Println("Completed pomodoros per day:", pterm.Green(int(avgCompleted)))
		fmt.Println("Abandoned pomodoros per day:", pterm.Green(int(avgAbandoned)))
	}
}

// NewStats returns an instance of Stats.
func NewStats(ctx *cli.Context) (*Stats, error) {
	s := &Stats{}

	s.Weekday = make(map[time.Weekday]*pomo)
	s.HourofDay = make(map[int]*pomo)

	for i := 0; i <= 6; i++ {
		s.Weekday[time.Weekday(i)] = &pomo{}
	}

	for i := 0; i <= 23; i++ {
		s.HourofDay[i] = &pomo{}
	}

	p := ctx.String("period")

	if !contains(StatsPeriod, timePeriod(p)) {
		var sl []string
		for _, v := range StatsPeriod {
			sl = append(sl, string(v))
		}

		return nil, fmt.Errorf("Period must be one of: %s", strings.Join(sl, ", "))
	}

	s.StartDate, s.EndDate = getPeriod(timePeriod(p))

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
func contains(s []timePeriod, e timePeriod) bool {
	for _, a := range s {
		if a == e {
			return true
		}
	}

	return false
}

func printTable(title string, data [][]string) {
	table := tablewriter.NewWriter(os.Stdout)
	table.SetHeader([]string{title, "minutes", "completed", "abandoned"})
	table.SetAutoWrapText(false)

	for _, v := range data {
		table.Append(v)
	}

	table.Render()
}
