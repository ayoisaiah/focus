// Package stats reports Focus session statistics
package stats

import (
	"cmp"
	"encoding/json"
	"fmt"
	"slices"
	"strconv"
	"time"

	"github.com/ayoisaiah/focus/config"
	"github.com/ayoisaiah/focus/internal/timeutil"
	"github.com/ayoisaiah/focus/session"
	"github.com/ayoisaiah/focus/store"
)

type Opts struct {
	config.FilterConfig
}

// Stats represents the computed focus statistics for a period of time.
type Stats struct {
	Aggregates      Aggregates        `json:"aggregates"`
	StartTime       time.Time         `json:"start_time"`
	EndTime         time.Time         `json:"end_time"`
	DB              store.DB          `json:"-"`
	Opts            Opts              `json:"-"`
	Sessions        []session.Session `json:"-"`
	LastDayTimeline []Timeline        `json:"timeline"`
	Summary         Summary           `json:"summary"`
}

type Timeline struct {
	StartTime time.Time     `json:"start_time"`
	Tags      []string      `json:"tags"`
	Duration  time.Duration `json:"duration"`
}

type KV struct {
	Name     string        `json:"name"`
	Duration time.Duration `json:"duration"`
}

type statsJSON struct {
	StartTime       time.Time  `json:"start_time"`
	EndTime         time.Time  `json:"end_time"`
	Tags            []KV       `json:"tags"`
	Hourly          []KV       `json:"hourly"`
	LastDayTimeline []Timeline `json:"timeline"`
	Daily           []KV       `json:"daily"`
	Weekly          []KV       `json:"weekly"`
	Yearly          []KV       `json:"yearly"`
	Monthly         []KV       `json:"monthly"`
	Totals          struct {
		Completed int           `json:"completed"`
		Abandoned int           `json:"abandoned"`
		Duration  time.Duration `json:"duration"`
	} `json:"totals"`
	Averages struct {
		Completed int           `json:"completed"`
		Abandoned int           `json:"abandoned"`
		Duration  time.Duration `json:"duration"`
	} `json:"averages"`
}

type aggregatePeriod string

const (
	monthly aggregatePeriod = "Monthly"
	daily   aggregatePeriod = "Daily"
	yearly  aggregatePeriod = "Yearly"
	weekly  aggregatePeriod = "Weekly"
	hourly  aggregatePeriod = "Hourly"
	all     aggregatePeriod = "All"
)

type Summary struct {
	Tags         map[string]time.Duration `json:"-"`
	TotalTime    time.Duration            `json:"total_time"`
	Completed    int                      `json:"completed"`
	Abandoned    int                      `json:"abandoned"`
	AvgCompleted int                      `json:"avg_completed"`
	AvgAbandoned int                      `json:"avg_abandoned"`
	AvgTime      time.Duration            `json:"avg_time"`
}

type Aggregates struct {
	startTime time.Time
	endTime   time.Time
	Weekly    map[string]time.Duration `json:"weekly"`
	Daily     map[string]time.Duration `json:"daily"`
	Yearly    map[string]time.Duration `json:"yearly"`
	Monthly   map[string]time.Duration `json:"monthly"`
	Hourly    map[string]time.Duration `json:"hourly"`
}

func (a *Aggregates) populateMap(max int) map[string]time.Duration {
	m := make(map[string]time.Duration)

	if max == 0 {
		return m
	}

	if max == -1 {
		start := timeutil.RoundToStart(a.startTime)

		for date := start; date.Before(a.endTime); date = date.AddDate(0, 0, 1) {
			m[date.Format("2006-01-02")] = time.Duration(0)
		}

		return m
	}

	for i := 0; i <= max; i++ {
		m[strconv.Itoa(i)] = time.Duration(0)
	}

	return m
}

func (a *Aggregates) init(start, end time.Time) {
	a.startTime = start
	a.endTime = end

	a.Yearly = a.populateMap(0)
	a.Monthly = a.populateMap(0)
	//nolint:gomnd // 0-6 days
	a.Weekly = a.populateMap(6)
	a.Daily = a.populateMap(-1)

	a.Hourly = make(map[string]time.Duration)
	for i := 0; i <= 23; i++ {
		a.Hourly[fmt.Sprintf("%02d:00", i)] = time.Duration(0)
	}
}

// getSessionDuration returns the elapsed time for a session within the
// bounds of the reporting period.
func (s *Stats) getSessionDuration(
	sess *session.Session,
) time.Duration {
	var duration time.Duration

outer:
	for _, event := range sess.Timeline {
		if event.StartTime.After(s.Opts.StartTime) && event.EndTime.Before(s.Opts.EndTime) {
			duration += event.EndTime.Sub(event.StartTime)
			continue
		}

		for date := event.StartTime; date.Before(event.EndTime); date = date.Add(1 * time.Minute) {
			// prevent minutes that fall outside the specified bounds
			// from being included
			if date.Before(s.Opts.StartTime) {
				continue
			}

			if date.After(s.Opts.EndTime) {
				break outer
			}

			duration += time.Minute * 1
		}
	}

	return duration
}

func (s *Stats) updateAggr(
	event session.Timeline,
	totals *Aggregates,
	period aggregatePeriod,
) {
	for date := event.StartTime; date.Before(event.EndTime); date = date.Add(1 * time.Minute) {
		if date.Before(s.Opts.StartTime) {
			continue
		}

		if date.After(s.Opts.EndTime) {
			break
		}

		switch period {
		case yearly:
			totals.Yearly[strconv.Itoa(date.Year())] += time.Minute * 1
		case monthly:
			totals.Monthly[strconv.Itoa(int(date.Month()))] += time.Minute * 1
		case weekly:
			totals.Weekly[strconv.Itoa(int(date.Weekday()))] += time.Minute * 1
		case daily:
			totals.Daily[date.Format("2006-01-02")] += time.Minute * 1
		case hourly:
			totals.Hourly[date.Format("15:00")] += time.Minute * 1
		case all:
			totals.Monthly[strconv.Itoa(int(date.Month()))] += time.Minute * 1
			totals.Weekly[strconv.Itoa(int(date.Weekday()))] += time.Minute * 1
			totals.Daily[date.Format("2006-01-02")] += time.Minute * 1
			totals.Hourly[date.Format("15:00")] += time.Minute * 1
			totals.Yearly[strconv.Itoa(date.Year())] += time.Minute * 1
		}
	}
}

// filterSessions ensures that sessions with an invalid end date are ignored.
// TODO: Filtering sessions should not be done here.
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

func (s *Stats) computeAggregates() {
	var totals Aggregates

	totals.init(s.Opts.StartTime, s.Opts.EndTime)

	s.LastDayTimeline = []Timeline{}

	for i := range s.Sessions {
		sess := s.Sessions[i]

		for _, event := range sess.Timeline {
			start := event.StartTime
			end := event.EndTime

			t := Timeline{}

			endTimeBeginning := timeutil.RoundToStart(s.Opts.EndTime)

			if end.After(endTimeBeginning) {
				for date := start; date.Before(end); date = date.Add(1 * time.Minute) {
					if date.Before(endTimeBeginning) {
						continue
					}

					t.StartTime = date
					t.Tags = sess.Tags
					t.Duration = end.Sub(date)
					s.LastDayTimeline = append(s.LastDayTimeline, t)

					break
				}
			}

			if start.After(s.Opts.StartTime) && end.Before(s.Opts.EndTime) {
				if start.Year() == end.Year() {
					totals.Yearly[strconv.Itoa(start.Year())] += end.Sub(start)
				} else {
					s.updateAggr(event, &totals, yearly)
				}

				if start.Month() == end.Month() {
					totals.Monthly[strconv.Itoa(int(start.Month()))] += end.Sub(
						start,
					)
				} else {
					s.updateAggr(event, &totals, monthly)
				}

				if start.Weekday() == end.Weekday() {
					totals.Weekly[strconv.Itoa(int(start.Weekday()))] += end.Sub(
						start,
					)
				} else {
					s.updateAggr(event, &totals, weekly)
				}

				if start.Day() == end.Day() {
					totals.Daily[start.Format("2006-01-02")] += end.Sub(
						start,
					)
				} else {
					s.updateAggr(event, &totals, daily)
				}

				if start.Hour() == end.Hour() {
					totals.Hourly[start.Format("15:00")] += end.Sub(start)
				} else {
					s.updateAggr(event, &totals, hourly)
				}
			} else {
				s.updateAggr(event, &totals, all)
			}
		}
	}

	s.Aggregates = totals
}

// computeSummary calculates the total minutes, completed sessions,
// and abandoned sessions for the current time period.
func (s *Stats) computeSummary() {
	var totals Summary

	totals.Tags = make(map[string]time.Duration)

	for i := range s.Sessions {
		sess := s.Sessions[i]

		duration := s.getSessionDuration(&sess)

		totals.TotalTime += duration

		for _, tag := range sess.Tags {
			totals.Tags[tag] += duration
		}

		if len(sess.Tags) == 0 {
			totals.Tags["uncategorized"] += duration
		}

		if sess.Completed {
			totals.Completed++
		} else {
			totals.Abandoned++
		}
	}

	hoursDiff := timeutil.Round(s.Opts.EndTime.Sub(s.Opts.StartTime).Hours())

	numberOfDays := hoursDiff / timeutil.HoursInADay

	totals.AvgTime = time.Duration(
		float64(totals.TotalTime) / float64(numberOfDays),
	)
	totals.AvgCompleted = timeutil.Round(
		float64(totals.Completed) / float64(numberOfDays),
	)
	totals.AvgAbandoned = timeutil.Round(
		float64(totals.Abandoned) / float64(numberOfDays),
	)

	s.Summary = totals
}

func sortByName(s []KV) {
	slices.SortStableFunc(s, func(a, b KV) int {
		return cmp.Compare(a.Name, b.Name)
	})
}

func (s *Stats) ToJSON() ([]byte, error) {
	var r statsJSON

	r.StartTime = s.StartTime
	r.EndTime = s.EndTime

	r.Totals.Completed = s.Summary.Completed
	r.Totals.Abandoned = s.Summary.Abandoned
	r.Totals.Duration = s.Summary.TotalTime
	r.Averages.Completed = s.Summary.AvgCompleted
	r.Averages.Abandoned = s.Summary.AvgAbandoned
	r.Averages.Duration = s.Summary.AvgTime

	r.LastDayTimeline = s.LastDayTimeline

	for k, v := range s.Summary.Tags {
		r.Tags = append(r.Tags, KV{
			Name:     k,
			Duration: v,
		})
	}

	for k, v := range s.Aggregates.Hourly {
		r.Hourly = append(r.Hourly, KV{
			Name:     k,
			Duration: v,
		})
	}

	for k, v := range s.Aggregates.Daily {
		r.Daily = append(r.Daily, KV{
			Name:     k,
			Duration: v,
		})
	}

	for k, v := range s.Aggregates.Weekly {
		r.Weekly = append(r.Weekly, KV{
			Name:     k,
			Duration: v,
		})
	}

	for k, v := range s.Aggregates.Monthly {
		r.Monthly = append(r.Monthly, KV{
			Name:     k,
			Duration: v,
		})
	}

	for k, v := range s.Aggregates.Yearly {
		r.Yearly = append(r.Yearly, KV{
			Name:     k,
			Duration: v,
		})
	}

	slices.SortStableFunc(r.Tags, func(a, b KV) int {
		return cmp.Compare(b.Duration, a.Duration)
	})

	sortByName(r.Hourly)
	sortByName(r.Daily)
	sortByName(r.Weekly)
	sortByName(r.Monthly)
	sortByName(r.Yearly)

	return json.Marshal(r)
}

// Compute calculates Focus statistics for a specific time period.
func (s *Stats) Compute(sessions []session.Session) {
	s.Sessions = sessions

	// TODO: Filter invalid sessions?

	// For all-time, set start time to the date of the first session
	if s.Opts.StartTime.IsZero() && len(sessions) > 0 {
		s.Opts.StartTime = timeutil.RoundToStart(sessions[0].StartTime)
	}

	s.StartTime = s.Opts.StartTime
	s.EndTime = s.Opts.EndTime

	s.computeSummary()
	s.computeAggregates()
}
