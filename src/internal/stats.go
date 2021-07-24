package focus

import (
	"bufio"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/olekukonko/tablewriter"
	"github.com/pterm/pterm"
	"github.com/urfave/cli/v2"
)

const (
	errParsingDate      = Error("The specified date format must be: YYYY-MM-DD or YYYY-MM-DD HH:MM:SS PM")
	errInvalidDateRange = Error(
		"The end date must not be earlier than the start date",
	)
)

const (
	hoursInADay      = 24
	maxHoursInAMonth = 744 // 31 day months
	minutesInAnHour  = 60
)

const (
	barChartChar = "▇"
)

type timePeriod string

const (
	periodAllTime   timePeriod = "all-time"
	periodToday     timePeriod = "today"
	periodYesterday timePeriod = "yesterday"
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

// getPeriod returns the start and end time according to the
// specified time period.
func getPeriod(period timePeriod) (start, end time.Time) {
	end = time.Now()

	switch period {
	case periodToday:
		start = time.Now()
	case periodYesterday:
		start = time.Now().AddDate(0, 0, -1)
		year, month, day := start.Date()
		end = time.Date(year, month, day, 23, 59, 59, 0, start.Location())
	case period7Days:
		start = time.Now().AddDate(0, 0, -6)
	case period14Days:
		start = time.Now().AddDate(0, 0, -13)
	case period30Days:
		start = time.Now().AddDate(0, 0, -29)
	case period90Days:
		start = time.Now().AddDate(0, 0, -89)
	case period180Days:
		start = time.Now().AddDate(0, 0, -179)
	case period365Days:
		start = time.Now().AddDate(0, 0, -364)
	case periodAllTime:
		return start, end
	default:
		return start, end
	}

	return time.Date(
		start.Year(),
		start.Month(),
		start.Day(),
		0,
		0,
		0,
		0,
		start.Location(),
	), end
}

type Data struct {
	Weekday          map[time.Weekday]*pomo
	HourofDay        map[int]*pomo
	History          map[string]*pomo
	HistoryKeyFormat string
	Totals           pomo
	Averages         pomo
}

func initData(start, end time.Time, hoursDiff int) *Data {
	d := &Data{}

	d.Weekday = make(map[time.Weekday]*pomo)
	d.History = make(map[string]*pomo)
	d.HourofDay = make(map[int]*pomo)

	for i := 0; i <= 6; i++ {
		d.Weekday[time.Weekday(i)] = &pomo{}
	}

	for i := 0; i <= 23; i++ {
		d.HourofDay[i] = &pomo{}
	}

	d.HistoryKeyFormat = "January 2006"
	if hoursDiff > hoursInADay && hoursDiff <= maxHoursInAMonth {
		d.HistoryKeyFormat = "January 02, 2006"
	}

	for date := start; !date.After(end); date = date.Add(time.Duration(hoursInADay) * time.Hour) {
		d.History[date.Format(d.HistoryKeyFormat)] = &pomo{}
	}

	return d
}

// computeAverages calculates the average minutes, completed pomodoros,
// and abandoned pomodoros per day for the specified time period.
func (d *Data) computeAverages(start, end time.Time) {
	end = time.Date(
		end.Year(),
		end.Month(),
		end.Day(),
		23,
		59,
		59,
		0,
		end.Location(),
	)
	hoursDiff := roundTime(end.Sub(start).Hours())
	hoursInADay := 24

	numberOfDays := hoursDiff / hoursInADay

	d.Averages.minutes = roundTime(
		float64(d.Totals.minutes) / float64(numberOfDays),
	)
	d.Averages.completed = roundTime(
		float64(d.Totals.completed) / float64(numberOfDays),
	)
	d.Averages.abandoned = roundTime(
		float64(d.Totals.abandoned) / float64(numberOfDays),
	)
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
			d.History[v.StartTime.Format(d.HistoryKeyFormat)].completed++
			d.Totals.completed++
			d.Totals.minutes += v.Duration
		} else {
			d.Weekday[v.StartTime.Weekday()].abandoned++
			d.HourofDay[v.StartTime.Hour()].abandoned++
			d.History[v.StartTime.Format(d.HistoryKeyFormat)].abandoned++
			d.Totals.abandoned++

			var elapsedTimeInSeconds int
			for _, v2 := range v.Timeline {
				elapsedTimeInSeconds += int(v2.EndTime.Sub(v2.StartTime).Seconds())
			}
			d.Totals.minutes += roundTime(float64(elapsedTimeInSeconds) / float64(minutesInAnHour))
		}

		hourly := map[int]float64{}
		weekday := map[time.Weekday]float64{}
		daily := map[string]float64{}

		for _, v2 := range v.Timeline {
			for date := v2.StartTime; !date.After(v2.EndTime); date = date.Add(1 * time.Minute) {
				var end time.Time
				if date.Add(1 * time.Minute).After(v2.EndTime) {
					end = v2.EndTime
				} else {
					end = date.Add(1 * time.Minute)
				}

				hourly[date.Hour()] += end.Sub(date).Seconds()
				weekday[date.Weekday()] += end.Sub(date).Seconds()
				daily[date.Format(d.HistoryKeyFormat)] += end.Sub(date).
					Seconds()
			}
		}

		for k, val := range weekday {
			d.Weekday[k].minutes += roundTime(val / float64(minutesInAnHour))
		}

		for k, val := range hourly {
			d.HourofDay[k].minutes += roundTime(val / float64(minutesInAnHour))
		}

		for k, val := range daily {
			d.History[k].minutes += roundTime(val / float64(minutesInAnHour))
		}
	}
}

// Stats represents the statistics for a time period.
type Stats struct {
	StartTime time.Time
	EndTime   time.Time
	Sessions  []session
	store     DB
	Data      *Data
	HoursDiff int
}

// getSessions retrieves the pomodoro sessions
// for the specified time period.
func (s *Stats) getSessions(start, end time.Time) error {
	b, err := s.store.getSessions(start, end)
	if err != nil {
		return err
	}

	for _, v := range b {
		sess := session{}

		err = json.Unmarshal(v, &sess)
		if err != nil {
			return err
		}

		s.Sessions = append(s.Sessions, sess)
	}

	return nil
}

// displayHourlyBreakdown prints the hourly breakdown
// for the current time period.
func (s *Stats) displayHourlyBreakdown() {
	fmt.Printf("\n%s", pterm.LightBlue("Hourly breakdown (minutes)"))

	type keyValue struct {
		key   int
		value *pomo
	}

	sl := make([]keyValue, 0, len(s.Data.HourofDay))
	for k, v := range s.Data.HourofDay {
		sl = append(sl, keyValue{k, v})
	}

	sort.SliceStable(sl, func(i, j int) bool {
		return sl[i].key < sl[j].key
	})

	var bars pterm.Bars

	for _, v := range sl {
		val := s.Data.HourofDay[v.key]

		d := time.Date(2000, 1, 1, v.key, 0, 0, 0, time.UTC)

		bars = append(bars, pterm.Bar{
			Label: d.Format("03:04 PM"),
			Value: val.minutes,
		})
	}

	err := pterm.DefaultBarChart.WithHorizontalBarCharacter(barChartChar).
		WithHorizontal().
		WithShowValue().
		WithBars(bars).
		Render()
	if err != nil {
		pterm.Error.Println(err)
	}
}

// displayPomodoroHistory prints the appropriate bar graph
// for the current time period.
func (s *Stats) displayPomodoroHistory() {
	if s.Data.Totals.minutes == 0 {
		return
	}

	fmt.Printf("\n%s", pterm.LightBlue("Pomodoro history (minutes)"))

	type keyValue struct {
		key   string
		value *pomo
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

	var bars pterm.Bars

	for _, v := range sl {
		val := s.Data.History[v.key]

		bars = append(bars, pterm.Bar{
			Label: v.key,
			Value: val.minutes,
		})
	}

	err := pterm.DefaultBarChart.WithHorizontalBarCharacter(barChartChar).
		WithHorizontal().
		WithShowValue().
		WithBars(bars).
		Render()
	if err != nil {
		pterm.Error.Println(err)
	}
}

// displayWeeklyBreakdown prints the weekly breakdown
// for the current time period.
func (s *Stats) displayWeeklyBreakdown() {
	fmt.Printf("\n%s\n", pterm.LightBlue("Weekly breakdown (minutes)"))

	type keyValue struct {
		key   time.Weekday
		value *pomo
	}

	sl := make([]keyValue, 0, len(s.Data.Weekday))
	for k, v := range s.Data.Weekday {
		sl = append(sl, keyValue{k, v})
	}

	sort.SliceStable(sl, func(i, j int) bool {
		return int(sl[i].key) < int(sl[j].key)
	})

	var bars pterm.Bars

	for _, v := range sl {
		val := s.Data.Weekday[v.key]

		bars = append(bars, pterm.Bar{
			Label: v.key.String(),
			Value: val.minutes,
		})
	}

	err := pterm.DefaultBarChart.WithHorizontalBarCharacter(barChartChar).
		WithHorizontal().
		WithShowValue().
		WithBars(bars).
		Render()
	if err != nil {
		pterm.Error.Println(err)
	}
}

func (s *Stats) displayAverages() {
	hoursDiff := roundTime(s.EndTime.Sub(s.StartTime).Hours())

	if hoursDiff > hoursInADay {
		fmt.Printf("\n%s\n", pterm.LightBlue("Averages"))

		hours, minutes := minsToHoursAndMins(s.Data.Averages.minutes)

		fmt.Println(
			"Averaged time logged:",
			pterm.Green(hours),
			pterm.Green("hours"),
			pterm.Green(minutes),
			pterm.Green("minutes"),
		)
		fmt.Println(
			"Completed pomodoros per day:",
			pterm.Green(s.Data.Averages.completed),
		)
		fmt.Println(
			"Abandoned pomodoros per day:",
			pterm.Green(s.Data.Averages.abandoned),
		)
	}
}

func (s *Stats) displayTotals() {
	fmt.Printf("%s\n", pterm.LightBlue("Totals"))

	hours, minutes := minsToHoursAndMins(s.Data.Totals.minutes)

	fmt.Printf(
		"Total time logged: %s %s %s %s\n",
		pterm.Green(hours),
		pterm.Green("hours"),
		pterm.Green(minutes),
		pterm.Green("minutes"),
	)

	fmt.Println("Pomodoros completed:", pterm.Green(s.Data.Totals.completed))
	fmt.Println("Pomodoros abandoned:", pterm.Green(s.Data.Totals.abandoned))
}

func (s *Stats) compute() {
	s.Data.computeTotals(s.Sessions)
	s.Data.computeAverages(s.StartTime, s.EndTime)
}

func printTable(data [][]string) {
	table := tablewriter.NewWriter(os.Stdout)
	table.SetHeader([]string{"#", "Start date", "End date", "Status"})
	table.SetAutoWrapText(false)

	for _, v := range data {
		table.Append(v)
	}

	table.Render()
}

func (s *Stats) Delete() error {
	err := s.List()
	if err != nil {
		return err
	}

	if len(s.Sessions) == 0 {
		return nil
	}

	pterm.Warning.Print("The above sessions will be deleted permanently. Press ENTER to proceed")

	reader := bufio.NewReader(os.Stdin)

	_, _ = reader.ReadString('\n')

	return s.store.deleteSessions(s.StartTime, s.EndTime)
}

func (s *Stats) List() error {
	err := s.getSessions(s.StartTime, s.EndTime)
	if err != nil {
		return err
	}

	if len(s.Sessions) == 0 {
		pterm.Info.Println("No sessions found for the time range")
		return nil
	}

	data := make([][]string, len(s.Sessions))

	for i, v := range s.Sessions {
		statusText := PrintColor("green", "completed")
		if !v.Completed {
			statusText = PrintColor("red", "abandoned")
		}

		endDate := v.EndTime.Format("Jan 02, 2006 03:04 PM")
		if v.EndTime.IsZero() {
			endDate = ""
		}

		sl := []string{
			fmt.Sprintf("%d", i+1),
			v.StartTime.Format("Jan 02, 2006 03:04 PM"),
			endDate,
			statusText,
		}

		data = append(data, sl)
	}

	printTable(data)

	return nil
}

// Show displays the relevant statistics for the
// set time period.
func (s *Stats) Show() error {
	defer s.store.close()

	err := s.getSessions(s.StartTime, s.EndTime)
	if err != nil {
		return err
	}

	s.compute()

	startDate := s.StartTime.Format("January 02, 2006")
	endDate := s.EndTime.Format("January 02, 2006")
	timePeriod := "Reporting period: " + startDate + " - " + endDate

	pterm.DefaultHeader.WithBackgroundStyle(pterm.NewStyle(pterm.BgYellow)).
		WithTextStyle(pterm.NewStyle(pterm.FgBlack)).
		Printfln(timePeriod)

	s.displayTotals()
	s.displayAverages()

	if s.HoursDiff > hoursInADay {
		s.displayPomodoroHistory()
	}

	s.displayWeeklyBreakdown()

	s.displayHourlyBreakdown()

	return nil
}

// NewStats returns an instance of Stats constructed
// from command-line arguments.
func NewStats(ctx *cli.Context, store *Store) (*Stats, error) {
	s := &Stats{}
	s.store = store

	period := ctx.String("period")

	if !contains(statsPeriod, timePeriod(period)) {
		var sl []string
		for _, v := range statsPeriod {
			sl = append(sl, string(v))
		}

		return nil, fmt.Errorf(
			"Period must be one of: %s",
			strings.Join(sl, ", "),
		)
	}

	s.StartTime, s.EndTime = getPeriod(timePeriod(period))

	// start and end arguments override the set period
	start := strings.TrimSpace(ctx.String("start"))
	end := strings.TrimSpace(ctx.String("end"))

	timeFormatLength := 10 // for YYYY-MM-DD

	if start != "" {
		if len(start) == timeFormatLength {
			start += " 12:00:00 AM"
		}

		v, err := time.Parse("2006-1-2 3:4:5 PM", start)
		if err != nil {
			fmt.Println(err)
			return nil, errParsingDate
		}

		s.StartTime = v
	}

	if end != "" {
		if len(end) == timeFormatLength {
			end += " 11:59:59 PM"
		}

		v, err := time.Parse("2006-1-2 3:4:5 PM", end)
		if err != nil {
			return nil, errParsingDate
		}

		s.EndTime = v
	}

	if int(s.EndTime.Sub(s.StartTime).Seconds()) < 0 {
		return nil, errInvalidDateRange
	}

	diff := s.EndTime.Sub(s.StartTime)
	s.HoursDiff = int(diff.Hours())

	s.Data = initData(s.StartTime, s.EndTime, s.HoursDiff)

	return s, nil
}

// roundTime rounds a time value in seconds, minutes, or hours to the nearest integer.
func roundTime(t float64) int {
	return int(math.Round(t))
}

// minsToHoursAndMins expresses a minutes value
// in hours and mins.
func minsToHoursAndMins(val int) (hrs, mins int) {
	hrs = int(math.Floor(float64(val) / float64(minutesInAnHour)))
	mins = val % minutesInAnHour

	return
}
