package focus

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"sort"
	"strings"
	"time"

	"github.com/olekukonko/tablewriter"
	"github.com/pterm/pterm"
)

const (
	errParsingDate = Error(
		"The specified date format must be: YYYY-MM-DD or YYYY-MM-DD HH:MM:SS PM",
	)
	errInvalidDateRange = Error(
		"The end date must not be earlier than the start date",
	)
)

const (
	hoursInADay      = 24
	maxHoursInAMonth = 744  // 31 day months
	maxHoursInAYear  = 8784 // Leap years
	minutesInAnHour  = 60
)

const (
	barChartChar  = "▇"
	noSessionsMsg = "No sessions found for the specified time range"
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

var statsPeriod = []timePeriod{
	periodAllTime,
	periodToday,
	periodYesterday,
	period7Days,
	period14Days,
	period30Days,
	period90Days,
	period180Days,
	period365Days,
}

type quantity struct {
	minutes   int
	completed int
	abandoned int
}

// getPeriod returns the start and end time according to the
// specified time period.
func getPeriod(period timePeriod) (start, end time.Time) {
	now := time.Now()

	end = time.Date(
		now.Year(),
		now.Month(),
		now.Day(),
		23,
		59,
		59,
		0,
		now.Location(),
	)

	switch period {
	case periodToday:
		start = now
	case periodYesterday:
		start = now.AddDate(0, 0, -1)
		year, month, day := start.Date()
		end = time.Date(year, month, day, 23, 59, 59, 0, start.Location())
	case period7Days:
		start = now.AddDate(0, 0, -6)
	case period14Days:
		start = now.AddDate(0, 0, -13)
	case period30Days:
		start = now.AddDate(0, 0, -29)
	case period90Days:
		start = now.AddDate(0, 0, -89)
	case period180Days:
		start = now.AddDate(0, 0, -179)
	case period365Days:
		start = now.AddDate(0, 0, -364)
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

// Data represents the computed statistics data
// for the current time period.
type Data struct {
	Weekday          map[time.Weekday]*quantity
	HourofDay        map[int]*quantity
	History          map[string]*quantity
	Tags             map[string]*quantity
	HistoryKeyFormat string
	Totals           quantity
	Averages         quantity
}

// initData creates an instance of Data with
// all its values initialised properly.
func initData(start, end time.Time, hoursDiff int) *Data {
	d := &Data{}

	d.Weekday = make(map[time.Weekday]*quantity)
	d.History = make(map[string]*quantity)
	d.Tags = make(map[string]*quantity)
	d.HourofDay = make(map[int]*quantity)

	for i := 0; i <= 6; i++ {
		d.Weekday[time.Weekday(i)] = &quantity{}
	}

	for i := 0; i <= 23; i++ {
		d.HourofDay[i] = &quantity{}
	}

	// Decide whether to compute the work history
	// in terms of days, or months
	d.HistoryKeyFormat = "January 2006"
	if hoursDiff > hoursInADay && hoursDiff <= maxHoursInAMonth {
		d.HistoryKeyFormat = "January 02, 2006"
	} else if hoursDiff > maxHoursInAYear {
		d.HistoryKeyFormat = "2006"
	}

	for date := start; !date.After(end); date = date.Add(time.Duration(hoursInADay) * time.Hour) {
		d.History[date.Format(d.HistoryKeyFormat)] = &quantity{}
	}

	return d
}

// computeAverages calculates the average minutes, completed sessions,
// and abandoned sessions per day for the specified time period.
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

// calculateSessionDuration returns the session duration in seconds.
// It ensures that minutes that are not within the bounds of the
// reporting period, are not included.
func (d *Data) calculateSessionDuration(
	s *session,
	statsStart, statsEnd time.Time,
) float64 {
	var seconds float64

	hourly := map[int]float64{}
	weekday := map[time.Weekday]float64{}
	daily := map[string]float64{}

	for _, v := range s.Timeline {
		var durationAdded bool

		for date := v.StartTime; !date.After(v.EndTime); date = date.Add(1 * time.Minute) {
			// prevent minutes that fall outside the specified bounds
			// from being included
			if date.Before(statsStart) || date.After(statsEnd) {
				continue
			}

			var end time.Time
			if date.Add(1 * time.Minute).After(v.EndTime) {
				end = v.EndTime
			} else {
				end = date.Add(1 * time.Minute)
			}

			secs := end.Sub(date).Seconds()

			hourly[date.Hour()] += secs
			weekday[date.Weekday()] += secs
			daily[date.Format(d.HistoryKeyFormat)] += secs

			if !durationAdded {
				durationAdded = true
				seconds += v.EndTime.Sub(date).Seconds()
			}
		}
	}

	for k, val := range weekday {
		d.Weekday[k].minutes += roundTime(val / float64(minutesInAnHour))
	}

	for k, val := range hourly {
		d.HourofDay[k].minutes += roundTime(val / float64(minutesInAnHour))
	}

	for k, val := range daily {
		if _, exists := d.History[k]; exists {
			d.History[k].minutes += roundTime(val / float64(minutesInAnHour))
		}
	}

	return seconds
}

// computeTotals calculates the total minutes, completed sessions,
// and abandoned sessions for the current time period.
func (d *Data) computeTotals(sessions []session, startTime, endTime time.Time) {
	for i := range sessions {
		s := sessions[i]
		if len(s.Tags) == 0 {
			s.Tags = []string{"uncategorised"}
		}

		if s.EndTime.IsZero() {
			continue
		}

		duration := roundTime(
			d.calculateSessionDuration(
				&s,
				startTime,
				endTime,
			) / float64(
				minutesInAnHour,
			),
		)

		for _, t := range s.Tags {
			if _, exists := d.Tags[t]; !exists {
				d.Tags[t] = &quantity{}
			}

			d.Tags[t].minutes += duration

			if s.Completed {
				d.Tags[t].completed++
			} else {
				d.Tags[t].abandoned++
			}
		}

		d.Totals.minutes += duration

		if s.Completed {
			d.Weekday[s.StartTime.Weekday()].completed++
			d.HourofDay[s.StartTime.Hour()].completed++

			if _, exists := d.History[s.StartTime.Format(d.HistoryKeyFormat)]; exists {
				d.History[s.StartTime.Format(d.HistoryKeyFormat)].completed++
			}

			d.Totals.completed++
		} else {
			d.Weekday[s.StartTime.Weekday()].abandoned++
			d.HourofDay[s.StartTime.Hour()].abandoned++

			if _, exists := d.History[s.StartTime.Format(d.HistoryKeyFormat)]; exists {
				d.History[s.StartTime.Format(d.HistoryKeyFormat)].abandoned++
			}

			d.Totals.abandoned++
		}
	}
}

// Stats represents the statistics for a time period.
type Stats struct {
	StartTime time.Time
	EndTime   time.Time
	store     DB
	Data      *Data
	Sessions  []session
	Tags      []string
	HoursDiff int
}

// getSessions retrieves the work sessions
// for the specified time period.
func (s *Stats) getSessions() error {
	b, err := s.store.getSessions(s.StartTime, s.EndTime, s.Tags)
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

// getHourlyBreakdown retrieves the hourly breakdown
// for the current time period.
func (s *Stats) getHourlyBreakdown() string {
	header := fmt.Sprintf("\n%s", pterm.LightBlue("Hourly breakdown (minutes)"))

	type keyValue struct {
		value *quantity
		key   int
	}

	sl := make([]keyValue, 0, len(s.Data.HourofDay))
	for k, v := range s.Data.HourofDay {
		sl = append(sl, keyValue{v, k})
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

	chart, err := pterm.DefaultBarChart.WithHorizontalBarCharacter(barChartChar).
		WithHorizontal().
		WithShowValue().
		WithBars(bars).
		Srender()
	if err != nil {
		pterm.Error.Println(err)
		return ""
	}

	return header + chart
}

// getWorkHistory retrieves the work history bar graph
// for the current time period.
func (s *Stats) getWorkHistory() string {
	if s.Data.Totals.minutes == 0 {
		return ""
	}

	header := fmt.Sprintf("\n%s", pterm.LightBlue("Work history (minutes)"))

	type keyValue struct {
		value *quantity
		key   string
	}

	sl := make([]keyValue, 0, len(s.Data.History))
	for k, v := range s.Data.History {
		sl = append(sl, keyValue{v, k})
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

	chart, err := pterm.DefaultBarChart.WithHorizontalBarCharacter(barChartChar).
		WithHorizontal().
		WithShowValue().
		WithBars(bars).
		Srender()
	if err != nil {
		pterm.Error.Println(err)
		return ""
	}

	return header + chart
}

// getWeeklyBreakdown retrieves weekly breakdown
// for the current time period.
func (s *Stats) getWeeklyBreakdown() string {
	header := fmt.Sprintf("\n%s", pterm.LightBlue("Weekly breakdown (minutes)"))

	type keyValue struct {
		value *quantity
		key   time.Weekday
	}

	sl := make([]keyValue, 0, len(s.Data.Weekday))
	for k, v := range s.Data.Weekday {
		sl = append(sl, keyValue{v, k})
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

	chart, err := pterm.DefaultBarChart.WithHorizontalBarCharacter(barChartChar).
		WithHorizontal().
		WithShowValue().
		WithBars(bars).
		Srender()
	if err != nil {
		pterm.Error.Println(err)
		return ""
	}

	return header + chart
}

// getAverages retrieves the average time logged for the
// current time period.
func (s *Stats) getAverages() string {
	hoursDiff := roundTime(s.EndTime.Sub(s.StartTime).Hours())

	if hoursDiff > hoursInADay {
		header := fmt.Sprintf("\n%s\n", pterm.LightBlue("Averages"))

		hours, minutes := minsToHoursAndMins(s.Data.Averages.minutes)

		timeLogged := fmt.Sprintln(
			"Average time logged per day:",
			pterm.Green(hours),
			pterm.Green("hours"),
			pterm.Green(minutes),
			pterm.Green("minutes"),
		)

		completed := fmt.Sprintln(
			"Completed sessions per day:",
			pterm.Green(s.Data.Averages.completed),
		)

		abandoned := fmt.Sprintln(
			"Abandoned sessions per day:",
			pterm.Green(s.Data.Averages.abandoned),
		)

		return header + timeLogged + completed + abandoned
	}

	return ""
}

// getTags retrieves the tag breakdown for the current time period.
func (s *Stats) getTags() string {
	var builder strings.Builder

	builder.WriteString(fmt.Sprintf("\n%s\n", pterm.LightBlue("Tags")))

	type KeyValue struct {
		Key   string
		Value int
	}

	kv := make([]KeyValue, 0, len(s.Data.Tags))
	for k, v := range s.Data.Tags {
		kv = append(kv, KeyValue{k, v.minutes})
	}

	sort.SliceStable(kv, func(i, j int) bool {
		return kv[i].Value > kv[j].Value
	})

	for _, v := range kv {
		hrs, mins := minsToHoursAndMins(s.Data.Tags[v.Key].minutes)

		tag := fmt.Sprintf(
			"%s: %s %s %s %s\n",
			v.Key,
			pterm.Green(hrs),
			pterm.Green("hours"),
			pterm.Green(mins),
			pterm.Green("minutes"),
		)

		builder.WriteString(tag)
	}

	return builder.String()
}

// getSummary retrieves the work session summary for the current
// time period.
func (s *Stats) getSummary() string {
	header := fmt.Sprintf("%s\n", pterm.LightBlue("Summary"))

	totalHrs, totalMins := minsToHoursAndMins(s.Data.Totals.minutes)

	timeLogged := fmt.Sprintf(
		"Total time logged: %s %s %s %s\n",
		pterm.Green(totalHrs),
		pterm.Green("hours"),
		pterm.Green(totalMins),
		pterm.Green("minutes"),
	)

	completed := fmt.Sprintln(
		"Work sessions completed:",
		pterm.Green(s.Data.Totals.completed),
	)

	abandoned := fmt.Sprintln(
		"Work sessions abandoned:",
		pterm.Green(s.Data.Totals.abandoned),
	)

	return header + timeLogged + completed + abandoned
}

func (s *Stats) compute() {
	s.Data.computeTotals(s.Sessions, s.StartTime, s.EndTime)
	s.Data.computeAverages(s.StartTime, s.EndTime)
}

func printTable(data [][]string, w io.Writer) {
	table := tablewriter.NewWriter(w)
	table.SetHeader([]string{"#", "Start date", "End date", "Tag", "Status"})
	table.SetAutoWrapText(false)

	for _, v := range data {
		table.Append(v)
	}

	table.Render()
}

// EditTag.is used to edit the tags of the specified sessions.
func (s *Stats) EditTag(w io.Writer, r io.Reader) error {
	tag := s.Tags
	// So that getSessions() does not filter by tag
	s.Tags = []string{}

	err := s.getSessions()
	if err != nil {
		return err
	}

	if len(s.Sessions) == 0 {
		pterm.Info.Println(noSessionsMsg)
		return nil
	}

	for i := range s.Sessions {
		s.Sessions[i].Tags = tag
	}

	printSessionsTable(w, s.Sessions)

	warning := pterm.Warning.Sprint(
		"The sessions above will be updated. Press ENTER to proceed",
	)
	fmt.Fprint(w, warning)

	reader := bufio.NewReader(r)

	_, _ = reader.ReadString('\n')

	for i := range s.Sessions {
		sess := s.Sessions[i]

		key := []byte(sess.StartTime.Format(time.RFC3339))

		value, err := json.Marshal(sess)
		if err != nil {
			return err
		}

		err = s.store.updateSession(key, value)
		if err != nil {
			return err
		}
	}

	return nil
}

// Delete attempts to delete all sessions that fall
// in the specified time range. It requests for
// confirmation before proceeding with the permanent
// removal of the sessions from the database.
func (s *Stats) Delete(w io.Writer, r io.Reader) error {
	err := s.List(w)
	if err != nil {
		return err
	}

	if len(s.Sessions) == 0 {
		return nil
	}

	warning := pterm.Warning.Sprint(
		"The above sessions will be deleted permanently. Press ENTER to proceed",
	)
	fmt.Fprint(w, warning)

	reader := bufio.NewReader(r)

	_, _ = reader.ReadString('\n')

	return s.store.deleteSessions(s.Sessions)
}

func printSessionsTable(w io.Writer, sessions []session) {
	data := make([][]string, len(sessions))

	for i := range sessions {
		sess := sessions[i]

		statusText := pterm.Green("completed")
		if !sess.Completed {
			statusText = pterm.Red("abandoned")
		}

		endDate := sess.EndTime.Format("Jan 02, 2006 03:04 PM")
		if sess.EndTime.IsZero() {
			endDate = ""
		}

		tags := strings.Join(sess.Tags, ", ")

		sl := []string{
			fmt.Sprintf("%d", i+1),
			sess.StartTime.Format("Jan 02, 2006 03:04 PM"),
			endDate,
			tags,
			statusText,
		}

		data = append(data, sl)
	}

	printTable(data, w)
}

// List prints out a table of all the sessions that
// were created within the specified time range.
func (s *Stats) List(w io.Writer) error {
	err := s.getSessions()
	if err != nil {
		return err
	}

	if len(s.Sessions) == 0 {
		pterm.Info.Println(noSessionsMsg)
		return nil
	}

	printSessionsTable(w, s.Sessions)

	return nil
}

// Show displays the relevant statistics for the
// set time period after making the necessary calculations.
func (s *Stats) Show(w io.Writer) error {
	defer s.store.close()

	err := s.getSessions()
	if err != nil {
		return err
	}

	if s.StartTime.IsZero() && len(s.Sessions) > 0 {
		fs := s.Sessions[0].StartTime
		s.StartTime = time.Date(
			fs.Year(),
			fs.Month(),
			fs.Day(),
			0,
			0,
			0,
			0,
			fs.Location(),
		)
	}

	diff := s.EndTime.Sub(s.StartTime)
	s.HoursDiff = int(diff.Hours())

	s.Data = initData(s.StartTime, s.EndTime, s.HoursDiff)

	s.compute()

	reportingStart := s.StartTime.Format("January 02, 2006")
	reportingEnd := s.EndTime.Format("January 02, 2006")
	timePeriod := "Reporting period: " + reportingStart + " - " + reportingEnd

	header := pterm.DefaultHeader.WithBackgroundStyle(pterm.NewStyle(pterm.BgYellow)).
		WithTextStyle(pterm.NewStyle(pterm.FgBlack)).
		Sprintfln(timePeriod)

	summary := s.getSummary()
	averages := s.getAverages()

	var workHistory string
	if s.HoursDiff > hoursInADay {
		workHistory = s.getWorkHistory()
	}

	var tags string
	if len(s.Tags) == 0 {
		tags = s.getTags()
	}

	weekly := s.getWeeklyBreakdown()

	hourly := s.getHourlyBreakdown()

	fmt.Fprintln(
		w,
		strings.TrimSpace(
			header+summary+averages+tags+workHistory+weekly+hourly,
		),
	)

	return nil
}

type statsCtx interface {
	String(name string) string
}

// NewStats returns an instance of Stats constructed
// from command-line arguments.
func NewStats(ctx statsCtx, store DB) (*Stats, error) {
	s := &Stats{}
	s.store = store

	if (ctx.String("tag")) != "" {
		s.Tags = strings.Split(ctx.String("tag"), ",")
	}

	period := ctx.String("period")

	if period != "" && !contains(statsPeriod, timePeriod(period)) {
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

	// start and end options will override the set period
	start := strings.TrimSpace(ctx.String("start"))
	end := strings.TrimSpace(ctx.String("end"))

	timeFormatLength := 10 // for YYYY-MM-DD

	if start != "" {
		if len(start) == timeFormatLength {
			start += " 12:00:00 AM"
		}

		v, err := time.Parse("2006-1-2 3:4:5 PM", start)
		if err != nil {
			return nil, errParsingDate
		}

		// Using time.Date allows setting the correct time zone
		// instead of UTC time
		s.StartTime = time.Date(
			v.Year(),
			v.Month(),
			v.Day(),
			v.Hour(),
			v.Minute(),
			v.Second(),
			0,
			time.Now().Location(),
		)
	}

	if end != "" {
		if len(end) == timeFormatLength {
			end += " 11:59:59 PM"
		}

		v, err := time.Parse("2006-1-2 3:4:5 PM", end)
		if err != nil {
			return nil, errParsingDate
		}

		s.EndTime = time.Date(
			v.Year(),
			v.Month(),
			v.Day(),
			v.Hour(),
			v.Minute(),
			v.Second(),
			0,
			time.Now().Location(),
		)
	}

	if int(s.EndTime.Sub(s.StartTime).Seconds()) < 0 {
		return nil, errInvalidDateRange
	}

	return s, nil
}
