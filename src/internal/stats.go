package focus

import (
	"encoding/json"
	"errors"
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

var (
	errParsingDate      = errors.New("The date format must be: YYYY-MM-DD")
	errInvalidDateRange = errors.New("The end date must not be less than the start date")
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

var statsPeriod = []timePeriod{periodAllTime, periodToday, periodYesterday, period7Days, period14Days, period30Days, period90Days, period180Days, period365Days}

type pomo struct {
	minutes   int
	completed int
	abandoned int
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

type Data struct {
	Weekday   map[time.Weekday]*pomo
	HourofDay map[int]*pomo
	Totals    pomo
	Averages  pomo
}

func initData() *Data {
	s := &Data{}

	s.Weekday = make(map[time.Weekday]*pomo)
	s.HourofDay = make(map[int]*pomo)

	for i := 0; i <= 6; i++ {
		s.Weekday[time.Weekday(i)] = &pomo{}
	}

	for i := 0; i <= 23; i++ {
		s.HourofDay[i] = &pomo{}
	}

	return s
}

// computeAverages calculates the average minutes, completed pomodoros,
// and abandoned pomodoros per day for the specified time period.
func (d *Data) computeAverages(start, end time.Time) {
	hoursDiff := int(math.Round(end.Sub(start).Hours()))
	hoursInADay := 24

	numberOfDays := hoursDiff / hoursInADay

	d.Averages.minutes = int(math.Round(float64(d.Totals.minutes) / float64(numberOfDays)))
	d.Averages.completed = int(math.Round(float64(d.Totals.completed) / float64(numberOfDays)))
	d.Averages.abandoned = int(math.Round(float64(d.Totals.abandoned) / float64(numberOfDays)))
}

// computeTotals calculates the the computeTotals minutes, completed pomodoros,
// and abandoned pomodoros per day for the current time period.
func (d *Data) computeTotals(sessions []session) {
	for _, v := range sessions {
		if v.EndTime.IsZero() {
			continue
		}

		if v.Completed {
			d.Weekday[v.StartTime.Weekday()].completed++
			d.HourofDay[v.StartTime.Hour()].completed++
			d.Totals.completed++
			d.Totals.minutes += v.Duration
		} else {
			d.Weekday[v.StartTime.Weekday()].abandoned++
			d.HourofDay[v.StartTime.Hour()].abandoned++
			d.Totals.abandoned++

			var elapsedTimeInSeconds int
			for _, v2 := range v.Timeline {
				elapsedTimeInSeconds += int(v2.EndTime.Sub(v2.StartTime).Seconds())
			}
			d.Totals.minutes += int(math.Round(float64(elapsedTimeInSeconds) / float64(hourMins)))
		}

		hourly := map[int]float64{}
		weekday := map[time.Weekday]float64{}

		for _, v2 := range v.Timeline {
			for d := v2.StartTime; !d.After(v2.EndTime); d = d.Add(1 * time.Minute) {
				var end time.Time
				if d.Add(1 * time.Minute).After(v2.EndTime) {
					end = v2.EndTime
				} else {
					end = d.Add(1 * time.Minute)
				}

				hourly[d.Hour()] += end.Sub(d).Seconds()
				weekday[d.Weekday()] += end.Sub(d).Seconds()
			}
		}

		for k, val := range weekday {
			d.Weekday[k].minutes += int(math.Round(val / float64(hourMins)))
		}

		for k, val := range hourly {
			d.HourofDay[k].minutes += int(math.Round(val / float64(hourMins)))
		}
	}
}

// Stats represents the statistics for a time period.
type Stats struct {
	StartTime time.Time
	EndTime   time.Time
	Sessions  []session
	store     *Store
	sortValue statsSort
	Data      *Data
}

// getSessions retrieves the pomodoro sessions
// for the specified time period.
func (s *Stats) getSessions(start, end time.Time) {
	b, err := s.store.getSessions(start, end)
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

// displayHourlyBreakdown prints the hourly breakdown
// for the current time period.
func (s *Stats) displayHourlyBreakdown() {
	fmt.Printf("\n%s\n", pterm.Blue("Hourly breakdown"))

	type keyValue struct {
		key   int
		value *pomo
	}

	sl := make([]keyValue, 0, len(s.Data.HourofDay))
	for k, v := range s.Data.HourofDay {
		sl = append(sl, keyValue{k, v})
	}

	switch s.sortValue {
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
		val := s.Data.HourofDay[v.key]
		completed := strconv.Itoa(val.completed)
		abandoned := strconv.Itoa(val.abandoned)
		total := strconv.Itoa(val.minutes)

		d := time.Date(2000, 1, 1, v.key, 0, 0, 0, time.UTC)
		data = append(data, []string{d.Format("03:04 PM"), total, completed, abandoned})
	}

	printTable("hours", data)
}

// displayWeeklyBreakdown prints the weekly breakdown
// for the current time period.
func (s *Stats) displayWeeklyBreakdown() {
	fmt.Printf("\n%s\n", pterm.Blue("Weekly breakdown"))

	type keyValue struct {
		key   time.Weekday
		value *pomo
	}

	sl := make([]keyValue, 0, len(s.Data.Weekday))
	for k, v := range s.Data.Weekday {
		sl = append(sl, keyValue{k, v})
	}

	switch s.sortValue {
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
		val := s.Data.Weekday[v.key]
		completed := strconv.Itoa(val.completed)
		abandoned := strconv.Itoa(val.abandoned)
		total := strconv.Itoa(val.minutes)

		data = append(data, []string{v.key.String(), total, completed, abandoned})
	}

	printTable("weekday", data)
}

func (s *Stats) displayAverages() {
	hoursDiff := int(math.Round(s.EndTime.Sub(s.StartTime).Hours()))
	hoursInADay := 24

	if hoursDiff > hoursInADay {
		fmt.Printf("\n%s\n", pterm.Blue("Averages"))

		hours := int(math.Floor(float64(s.Data.Totals.minutes) / float64(hourMins)))
		minutes := s.Data.Totals.minutes % hourMins

		fmt.Println("Averaged time logged:", pterm.Green(hours), pterm.Green("hours"), pterm.Green(minutes), pterm.Green("minutes"))
		fmt.Println("Completed pomodoros per day:", pterm.Green(s.Data.Averages.completed))
		fmt.Println("Abandoned pomodoros per day:", pterm.Green(s.Data.Averages.abandoned))
	}
}

func (s *Stats) displayTotals() {
	fmt.Printf("%s\n", pterm.Blue("Totals"))

	hours := int(math.Floor(float64(s.Data.Totals.minutes) / float64(hourMins)))
	minutes := s.Data.Totals.minutes % hourMins

	fmt.Printf("Total time logged: %s %s %s %s\n", pterm.Green(hours), pterm.Green("hours"), pterm.Green(minutes), pterm.Green("minutes"))

	fmt.Println("Pomodoros completed:", pterm.Green(s.Data.Totals.completed))
	fmt.Println("Pomodoros abandoned:", pterm.Green(s.Data.Totals.abandoned))
}

func (s *Stats) compute() {
	s.Data.computeTotals(s.Sessions)
	s.Data.computeAverages(s.StartTime, s.EndTime)
}

// Show displays the relevant statistics for the
// set time period.
func (s *Stats) Show() {
	s.getSessions(s.StartTime, s.EndTime)
	s.compute()

	startDate := s.StartTime.Format("January 02, 2006")
	endDate := s.EndTime.Format("January 02, 2006")
	timePeriod := startDate + " - " + endDate

	pterm.DefaultHeader.WithBackgroundStyle(pterm.NewStyle(pterm.BgYellow)).WithTextStyle(pterm.NewStyle(pterm.FgBlack)).Printfln(timePeriod)

	s.displayTotals()
	s.displayAverages()
	s.displayWeeklyBreakdown()
	s.displayHourlyBreakdown()
}

// NewStats returns an instance of Stats constructed
// from command-line arguments.
func NewStats(ctx *cli.Context, store *Store) (*Stats, error) {
	s := &Stats{}
	s.store = store
	s.Data = initData()

	s.sortValue = statsSort(ctx.String("sort"))
	period := ctx.String("period")

	start := ctx.String("start")
	end := ctx.String("end")

	if start != "" {
		v, err := time.Parse("2006-01-02", start)
		if err != nil {
			return nil, errParsingDate
		}

		s.StartTime = time.Date(v.Year(), v.Month(), v.Day(), 0, 0, 0, 0, v.Location())
	}

	if end != "" {
		v, err := time.Parse("2006-01-02", end)
		if err != nil {
			return nil, errParsingDate
		}

		s.EndTime = time.Date(v.Year(), v.Month(), v.Day(), 23, 59, 59, 0, v.Location())
	}

	if int(s.EndTime.Sub(s.StartTime).Seconds()) < 0 {
		return nil, errInvalidDateRange
	}

	if !contains(statsPeriod, timePeriod(period)) {
		var sl []string
		for _, v := range statsPeriod {
			sl = append(sl, string(v))
		}

		return nil, fmt.Errorf("Period must be one of: %s", strings.Join(sl, ", "))
	}

	if period != "" {
		// The set time period overrides start and end times
		s.StartTime, s.EndTime = getPeriod(timePeriod(period))
	}

	return s, nil
}
