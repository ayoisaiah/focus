// Package stats reports Focus session statistics
package stats

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/araddon/dateparse"
	"github.com/hako/durafmt"
	"github.com/pterm/pterm"

	"github.com/ayoisaiah/focus/config"
	"github.com/ayoisaiah/focus/internal/session"
	"github.com/ayoisaiah/focus/internal/timeutil"
	"github.com/ayoisaiah/focus/internal/ui"
	"github.com/ayoisaiah/focus/store"
)

var (
	opts *config.StatsConfig
	db   store.DB
)

const (
	barChartChar  = "▇"
	noSessionsMsg = "No sessions found for the specified time range"
)

type aggregatePeriod string

const (
	monthly aggregatePeriod = "Monthly"
	daily   aggregatePeriod = "Daily"
	yearly  aggregatePeriod = "Yearly"
	weekly  aggregatePeriod = "Weekly"
	hourly  aggregatePeriod = "Hourly"
	all     aggregatePeriod = "All"
)

type summary struct {
	tags         map[string]time.Duration
	totalTime    time.Duration
	completed    int
	abandoned    int
	avgCompleted int
	avgAbandoned int
	avgTime      time.Duration
}

type aggregates struct {
	weekly  map[int]time.Duration
	daily   map[int]time.Duration
	yearly  map[int]time.Duration
	monthly map[int]time.Duration
	hourly  map[int]time.Duration
}

// getSessionDuration returns the elapsed time for a session within the
// bounds of the reporting period.
func getSessionDuration(
	sess *session.Session,
) time.Duration {
	var duration time.Duration

outer:
	for _, event := range sess.Timeline {
		if event.StartTime.After(opts.StartTime) && event.EndTime.Before(opts.EndTime) {
			duration += event.EndTime.Sub(event.StartTime)
			continue
		}

		for date := event.StartTime; date.Before(event.EndTime); date = date.Add(1 * time.Minute) {
			// prevent minutes that fall outside the specified bounds
			// from being included
			if date.Before(opts.StartTime) {
				continue
			}

			if date.After(opts.EndTime) {
				break outer
			}

			duration += time.Minute * 1
		}
	}

	return duration
}

func updateAggr(
	event session.Timeline,
	totals *aggregates,
	period aggregatePeriod,
) {
	for date := event.StartTime; date.Before(event.EndTime); date = date.Add(1 * time.Minute) {
		if date.Before(opts.StartTime) {
			continue
		}

		if date.After(opts.EndTime) {
			break
		}

		i := timeutil.DayFormat(date)

		switch period {
		case yearly:
			totals.yearly[date.Year()] += time.Minute * 1
		case monthly:
			totals.monthly[int(date.Month())] += time.Minute * 1
		case weekly:
			totals.weekly[int(date.Weekday())] += time.Minute * 1
		case daily:
			totals.daily[i] += time.Minute * 1
		case hourly:
			totals.hourly[date.Hour()] += time.Minute * 1
		case all:
			totals.monthly[int(date.Month())] += time.Minute * 1
			totals.weekly[int(date.Weekday())] += time.Minute * 1
			totals.daily[i] += time.Minute * 1
			totals.hourly[date.Hour()] += time.Minute * 1
			totals.yearly[date.Year()] += time.Minute * 1
		}
	}
}

func populateMap(max int) map[int]time.Duration {
	m := make(map[int]time.Duration)

	if max == 0 {
		return m
	}

	if max == -1 {
		start := timeutil.RoundToStart(opts.StartTime)

		for date := start; date.Before(opts.EndTime); date = date.AddDate(0, 0, 1) {
			m[timeutil.DayFormat(date)] = time.Duration(0)
		}

		return m
	}

	for i := 0; i <= max; i++ {
		m[i] = time.Duration(0)
	}

	return m
}

func computeAggregates(sessions []session.Session) aggregates {
	var totals aggregates

	totals.yearly = populateMap(0)
	totals.monthly = populateMap(0)
	//nolint:gomnd // 0-6 days
	totals.weekly = populateMap(6)
	totals.daily = populateMap(-1)
	//nolint:gomnd // 0-23 hours
	totals.hourly = populateMap(23)

	for i := range sessions {
		sess := sessions[i]

		for _, event := range sess.Timeline {
			start := event.StartTime
			end := event.EndTime

			if start.After(opts.StartTime) && end.Before(opts.EndTime) {
				if start.Year() == end.Year() {
					totals.yearly[start.Year()] += end.Sub(start)
				} else {
					updateAggr(event, &totals, yearly)
				}

				if start.Month() == end.Month() {
					totals.monthly[int(start.Month())] += end.Sub(start)
				} else {
					updateAggr(event, &totals, monthly)
				}

				if start.Weekday() == end.Weekday() {
					totals.weekly[int(start.Weekday())] += end.Sub(start)
				} else {
					updateAggr(event, &totals, weekly)
				}

				if start.Day() == end.Day() {
					totals.daily[timeutil.DayFormat(start)] += end.Sub(
						start,
					)
				} else {
					updateAggr(event, &totals, daily)
				}

				if start.Hour() == end.Hour() {
					totals.hourly[start.Hour()] += end.Sub(start)
				} else {
					updateAggr(event, &totals, hourly)
				}
			} else {
				updateAggr(event, &totals, all)
			}
		}
	}

	return totals
}

// computeTotals calculates the total minutes, completed sessions,
// and abandoned sessions for the current time period.
func computeTotals(sessions []session.Session) summary {
	var totals summary

	totals.tags = make(map[string]time.Duration)

	for i := range sessions {
		sess := sessions[i]

		duration := getSessionDuration(&sess)

		totals.totalTime += duration

		for _, tag := range sess.Tags {
			totals.tags[tag] += duration
		}

		if len(sess.Tags) == 0 {
			totals.tags["uncategorized"] += duration
		}

		if sess.Completed {
			totals.completed++
		} else {
			totals.abandoned++
		}
	}

	hoursDiff := timeutil.Round(opts.EndTime.Sub(opts.StartTime).Hours())

	numberOfDays := hoursDiff / timeutil.HoursInADay

	totals.avgTime = time.Duration(
		float64(totals.totalTime) / float64(numberOfDays),
	)
	totals.avgCompleted = timeutil.Round(
		float64(totals.completed) / float64(numberOfDays),
	)
	totals.avgAbandoned = timeutil.Round(
		float64(totals.abandoned) / float64(numberOfDays),
	)

	return totals
}

func getBarChart(data map[int]time.Duration, period aggregatePeriod) string {
	if len(data) == 0 {
		return ""
	}

	header := ui.Blue(fmt.Sprintf("\n%s breakdown (minutes)", period))

	type keyValue struct {
		value time.Duration
		key   int
	}

	sl := make([]keyValue, 0, len(data))
	for k, v := range data {
		sl = append(sl, keyValue{v, k})
	}

	sort.SliceStable(sl, func(i, j int) bool {
		return sl[i].key < sl[j].key
	})

	var bars pterm.Bars

	for _, v := range sl {
		var label string

		//nolint:exhaustive // `all` case is not needed
		switch period {
		case yearly:
			label = fmt.Sprintf("%d", v.key)
		case monthly:
			label = time.Month(v.key).String()
		case weekly:
			label = time.Weekday(v.key).String()
		case daily:
			date, _ := dateparse.ParseAny(strconv.Itoa(v.key))
			label = fmt.Sprintf(
				"%s %02d, %d",
				date.Month().String(),
				date.Day(),
				date.Year(),
			)
		case hourly:
			label = fmt.Sprintf("%02d:00", v.key)
		}

		bars = append(bars, pterm.Bar{
			Value: timeutil.Round(v.value.Minutes()),
			Label: label,
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

// getTags retrieves the tag breakdown for the current time period.
func getTags(tags map[string]time.Duration) string {
	var builder strings.Builder

	if len(tags) == 0 {
		return ""
	}

	builder.WriteString(fmt.Sprintf("\n%s\n", ui.Blue("Tags")))

	type keyValue struct {
		key   string
		value time.Duration
	}

	kv := make([]keyValue, 0, len(tags))
	for k, v := range tags {
		kv = append(kv, keyValue{k, v})
	}

	sort.SliceStable(kv, func(i, j int) bool {
		return kv[i].value > kv[j].value
	})

	for _, v := range kv {
		//nolint:gomnd // limit to first 2 units
		duration := durafmt.Parse(v.value).LimitToUnit("hours").LimitFirstN(2)

		tag := fmt.Sprintf(
			"%s: %s\n",
			v.key,
			ui.Green(duration),
		)

		builder.WriteString(tag)
	}

	return builder.String()
}

func getAverages(totals summary) string {
	header := fmt.Sprintf("\n%s\n", ui.Blue("Averages"))

	duration := durafmt.Parse(totals.avgTime)

	timeLogged := fmt.Sprintf(
		"Time logged: %s\n",
		//nolint:gomnd // limit to first 2 units
		ui.Green(duration.LimitToUnit("hours").LimitFirstN(2)),
	)

	completed := fmt.Sprintln(
		"Sessions completed:",
		ui.Green(totals.avgCompleted),
	)

	abandoned := fmt.Sprintln(
		"Sessions abandoned:",
		ui.Green(totals.avgAbandoned),
	)

	return header + timeLogged + completed + abandoned
}

// getSummary retrieves the work session summary for the current time period.
func getSummary(totals summary) string {
	header := fmt.Sprintf("%s\n", ui.Blue("Summary"))

	duration := durafmt.Parse(totals.totalTime)

	timeLogged := fmt.Sprintf(
		"Time logged: %s\n",
		//nolint:gomnd // limit to first 2 units
		ui.Green(duration.LimitToUnit("hours").LimitFirstN(2)),
	)

	completed := fmt.Sprintln(
		"Sessions completed:",
		ui.Green(totals.completed),
	)

	abandoned := fmt.Sprintln(
		"Sessions abandoned:",
		ui.Green(totals.abandoned),
	)

	return header + timeLogged + completed + abandoned
}

// filterSessions ensures that sessions with an invalid end date are ignored.
func filterSessions(sessions []session.Session) []session.Session {
	filtered := sessions[:0]

	for i := range sessions {
		sess := sessions[i]

		if sess.EndTime.IsZero() || sess.EndTime.Before(sess.StartTime) {
			continue
		}

		filtered = append(filtered, sess)
	}

	return filtered
}

// Show displays the relevant statistics for the
// set time period after making the necessary calculations.
func Show() error {
	defer db.Close()

	sessions, err := db.GetSessions(opts.StartTime, opts.EndTime, opts.Tags)
	if err != nil {
		return err
	}

	sessions = filterSessions(sessions)

	// For all-time, set start time to the date of the first session
	if opts.StartTime.IsZero() && len(sessions) > 0 {
		firstSession := sessions[0].StartTime
		opts.StartTime = timeutil.RoundToStart(firstSession)
	}

	totals := computeTotals(sessions)
	aggregates := computeAggregates(sessions)

	reportingStart := opts.StartTime.Format("January 02, 2006")
	reportingEnd := opts.EndTime.Format("January 02, 2006")
	timePeriod := "Reporting period: " + reportingStart + " - " + reportingEnd

	header := pterm.DefaultHeader.WithBackgroundStyle(pterm.NewStyle(pterm.BgYellow)).
		WithTextStyle(pterm.NewStyle(pterm.FgBlack)).
		Sprintfln(timePeriod)

	hoursDiff := timeutil.Round(opts.EndTime.Sub(opts.StartTime).Hours())

	var history string
	//nolint:gocritic // if-else more appropriate
	if hoursDiff > timeutil.HoursInADay &&
		hoursDiff <= timeutil.MaxHoursInAMonth {
		history = getBarChart(aggregates.daily, daily)
	} else if hoursDiff > timeutil.MaxHoursInAYear {
		history = getBarChart(aggregates.yearly, yearly)
	} else {
		history = getBarChart(aggregates.monthly, monthly)
	}

	output := fmt.Sprint(
		header,
		getSummary(totals),
		getAverages(totals),
		getTags(totals.tags),
		history,
		getBarChart(aggregates.weekly, weekly),
		getBarChart(aggregates.hourly, hourly),
	)

	fmt.Fprintln(
		opts.Stdout,
		strings.TrimSpace(output),
	)

	return nil
}

func Init(dbClient store.DB, cfg *config.StatsConfig) {
	db = dbClient
	opts = cfg
}
