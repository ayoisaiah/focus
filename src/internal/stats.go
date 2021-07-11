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

const hourMins = 60

type statsSort string

const (
	sortMinutes   statsSort = "minutes"
	sortCompleted statsSort = "completed"
	sortAbandoned statsSort = "abandoned"
)

type timePeriod string

const (
	periodAllTime   timePeriod = "all-time"
	periodToday     timePeriod = "today"
	periodYesterday timePeriod = "yesterday"
	period24Hours   timePeriod = "24hours"
	period7Days     timePeriod = "7days"
	period14Days    timePeriod = "14days"
	period30Days    timePeriod = "30days"
	period90Days    timePeriod = "90days"
	period180Days   timePeriod = "180days"
	period365Days   timePeriod = "365days"
)

var StatsPeriod = []timePeriod{periodAllTime, periodToday, periodYesterday, period7Days, period14Days, period30Days, period90Days, period180Days, period365Days}

type pomo struct {
	minutes   int
	completed int
	abandoned int
}

type Stats struct {
	StartDate time.Time
	EndDate   time.Time
	pomo
	Sessions  []session
	Weekday   map[time.Weekday]*pomo
	HourofDay map[int]*pomo
	Sort      statsSort
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
	fmt.Printf("\n%s\n", pterm.Blue("Hourly breakdown"))

	type keyValue struct {
		key   int
		value *pomo
	}

	sl := make([]keyValue, 0, len(s.HourofDay))
	for k, v := range s.HourofDay {
		sl = append(sl, keyValue{k, v})
	}

	switch s.Sort {
	case sortMinutes:
		sort.SliceStable(sl, func(i, j int) bool {
			return sl[i].value.minutes > sl[j].value.minutes
		})
	case sortCompleted:
		sort.SliceStable(sl, func(i, j int) bool {
			return sl[i].value.completed > sl[j].value.completed
		})
	case sortAbandoned:
		sort.SliceStable(sl, func(i, j int) bool {
			return sl[i].value.abandoned > sl[j].value.abandoned
		})
	default:
		sort.SliceStable(sl, func(i, j int) bool {
			return sl[i].key < sl[j].key
		})
	}

	var data = make([][]string, len(sl))

	for _, v := range sl {
		val := s.HourofDay[v.key]
		completed := strconv.Itoa(val.completed)
		abandoned := strconv.Itoa(val.abandoned)
		total := strconv.Itoa(val.minutes)

		d := time.Date(2000, 1, 1, v.key, 0, 0, 0, time.UTC)
		data = append(data, []string{d.Format("03:04 PM"), total, completed, abandoned})
	}

	printTable("hours", data)
}

func (s *Stats) weekdays() {
	fmt.Printf("\n%s\n", pterm.Blue("Weekly breakdown"))

	type keyValue struct {
		key   time.Weekday
		value *pomo
	}

	sl := make([]keyValue, 0, len(s.Weekday))
	for k, v := range s.Weekday {
		sl = append(sl, keyValue{k, v})
	}

	switch s.Sort {
	case sortMinutes:
		sort.SliceStable(sl, func(i, j int) bool {
			return sl[i].value.minutes > sl[j].value.minutes
		})
	case sortCompleted:
		sort.SliceStable(sl, func(i, j int) bool {
			return sl[i].value.completed > sl[j].value.completed
		})
	case sortAbandoned:
		sort.SliceStable(sl, func(i, j int) bool {
			return sl[i].value.abandoned > sl[j].value.abandoned
		})
	default:
		sort.SliceStable(sl, func(i, j int) bool {
			return int(sl[i].key) < int(sl[j].key)
		})
	}

	var data = make([][]string, len(sl))

	for _, v := range sl {
		val := s.Weekday[v.key]
		completed := strconv.Itoa(val.completed)
		abandoned := strconv.Itoa(val.abandoned)
		total := strconv.Itoa(val.minutes)

		data = append(data, []string{v.key.String(), total, completed, abandoned})
	}

	printTable("weekday", data)
}

func (s *Stats) total() {
	fmt.Printf("%s\n", pterm.Blue("Totals"))

	for _, v := range s.Sessions {
		if v.EndTime.IsZero() {
			continue
		}

		if v.Completed {
			s.Weekday[v.StartTime.Weekday()].completed++
			s.HourofDay[v.StartTime.Hour()].completed++
			s.completed++
		} else {
			s.Weekday[v.StartTime.Weekday()].abandoned++
			s.HourofDay[v.StartTime.Hour()].abandoned++
			s.abandoned++
		}

		hourly := map[int]float64{}
		weekday := map[time.Weekday]float64{}

		for d := v.StartTime; !d.After(v.EndTime); d = d.Add(1 * time.Minute) {
			var end time.Time
			if d.Add(1 * time.Minute).After(v.EndTime) {
				end = v.EndTime
			} else {
				end = d.Add(1 * time.Minute)
			}

			hourly[d.Hour()] += end.Sub(d).Seconds()
			weekday[d.Weekday()] += end.Sub(d).Seconds()
		}

		for k, val := range weekday {
			s.Weekday[k].minutes += int(math.Round(val / float64(hourMins)))
		}

		for k, val := range hourly {
			s.HourofDay[k].minutes += int(math.Round(val / float64(hourMins)))
		}

		s.minutes += int(math.Round(v.EndTime.Sub(v.StartTime).Minutes()))
	}

	hours := int(math.Floor(float64(s.minutes) / float64(hourMins)))
	minutes := s.minutes % hourMins

	fmt.Printf("Total time logged: %s %s %s %s\n", pterm.Green(hours), pterm.Green("hours"), pterm.Green(minutes), pterm.Green("minutes"))

	fmt.Println("Pomodoros completed:", pterm.Green(s.completed))
	fmt.Println("Pomodoros abandoned:", pterm.Green(s.abandoned))
}

// Show displays the relevant statistics for the
// set time period.
func (s *Stats) Show() {
	s.getSessions()

	startDate := s.StartDate.Format("January 02, 2006")
	endDate := s.EndDate.Format("January 02, 2006")
	timePeriod := startDate + " - " + endDate

	pterm.DefaultHeader.WithBackgroundStyle(pterm.NewStyle(pterm.BgYellow)).WithTextStyle(pterm.NewStyle(pterm.FgBlack)).Printfln(timePeriod)

	s.total()
	s.average()
	s.weekdays()
	s.hourly()
}

// getPeriod returns the start and end time according to the
// specified time period.
func getPeriod(period timePeriod) (startTime, endTime time.Time) {
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

func (s *Stats) average() {
	hoursDiff := int(math.Round(s.EndDate.Sub(s.StartDate).Hours()))
	hoursInADay := 24

	if hoursDiff > hoursInADay {
		fmt.Printf("\n%s\n", pterm.Blue("Averages"))

		numberOfDays := hoursDiff / hoursInADay
		avgMins := math.Round(float64(s.minutes) / float64(numberOfDays))
		avgCompleted := math.Round(float64(s.completed) / float64(numberOfDays))
		avgAbandoned := math.Round(float64(s.abandoned) / float64(numberOfDays))

		hourMins := 60
		hours := int(math.Floor(avgMins / float64(hourMins)))
		minutes := int(avgMins) % hourMins

		fmt.Println("Averaged time logged:", pterm.Green(hours), pterm.Green("hours"), pterm.Green(minutes), pterm.Green("minutes"))
		fmt.Println("Completed pomodoros per day:", pterm.Green(int(avgCompleted)))
		fmt.Println("Abandoned pomodoros per day:", pterm.Green(int(avgAbandoned)))
	}
}

// NewStats returns an instance of Stats.
func NewStats(ctx *cli.Context) (*Stats, error) {
	s := &Stats{}

	s.Sort = statsSort(ctx.String("sort"))

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
	table.SetHeader([]string{title, "total minutes", "total completed", "total abandoned"})
	table.SetAutoWrapText(false)

	for _, v := range data {
		table.Append(v)
	}

	table.Render()
}
