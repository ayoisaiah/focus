// Package stats reports Focus session statistics
package stats

import (
	"cmp"
	"encoding/json"
	"fmt"
	"slices"
	"sort"
	"strconv"
	"time"

	"github.com/maruel/natural"

	"github.com/ayoisaiah/focus/internal/models"
	"github.com/ayoisaiah/focus/internal/timeutil"
	"github.com/ayoisaiah/focus/store"
)

type (
	// Stats represents the computed focus statistics for a period of time.
	Stats struct {
		Aggregates      Aggregates        `json:"aggregates"`
		StartTime       time.Time         `json:"start_time"`
		EndTime         time.Time         `json:"end_time"`
		DB              store.DB          `json:"-"`
		Sessions        []*models.Session `json:"-"`
		LastDayTimeline []Timeline        `json:"timeline"`
		Summary         Summary           `json:"summary"`
		Tags            []string          `json:"tags"`
	}

	Timeline struct {
		StartTime time.Time     `json:"start_time"`
		Tags      []string      `json:"tags"`
		Duration  time.Duration `json:"duration"`
	}

	Record struct {
		Name     string        `json:"name"`
		Duration time.Duration `json:"duration"`
	}

	statsJSON struct {
		StartTime       time.Time  `json:"start_time"`
		EndTime         time.Time  `json:"end_time"`
		Tags            []Record   `json:"tags"`
		Hourly          []Record   `json:"hourly"`
		LastDayTimeline []Timeline `json:"timeline"`
		Daily           []Record   `json:"daily"`
		Weekday         []Record   `json:"weekday"`
		Weekly          []Record   `json:"weekly"`
		Yearly          []Record   `json:"yearly"`
		Monthly         []Record   `json:"monthly"`
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

	aggregatePeriod string

	Summary struct {
		Tags         map[string]time.Duration `json:"-"`
		TotalTime    time.Duration            `json:"total_time"`
		Completed    int                      `json:"completed"`
		Abandoned    int                      `json:"abandoned"`
		AvgCompleted int                      `json:"avg_completed"`
		AvgAbandoned int                      `json:"avg_abandoned"`
		AvgTime      time.Duration            `json:"avg_time"`
	}

	Aggregates struct {
		startTime time.Time
		endTime   time.Time
		Weekly    map[string]time.Duration `json:"weekly"`
		Weekday   map[string]time.Duration `json:"weekday"`
		Daily     map[string]time.Duration `json:"daily"`
		Yearly    map[string]time.Duration `json:"yearly"`
		Monthly   map[string]time.Duration `json:"monthly"`
		Hourly    map[string]time.Duration `json:"hourly"`
	}
)

const (
	monthly aggregatePeriod = "Monthly"
	weekly  aggregatePeriod = "Weekly"
	daily   aggregatePeriod = "Daily"
	yearly  aggregatePeriod = "Yearly"
	weekday aggregatePeriod = "Weekday"
	hourly  aggregatePeriod = "Hourly"
	all     aggregatePeriod = "All"
)

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
	a.Weekly = a.populateMap(0)
	a.Daily = a.populateMap(-1)

	a.Weekday = map[string]time.Duration{
		"Sunday":    time.Duration(0),
		"Monday":    time.Duration(0),
		"Tuesday":   time.Duration(0),
		"Wednesday": time.Duration(0),
		"Thursday":  time.Duration(0),
		"Friday":    time.Duration(0),
		"Saturday":  time.Duration(0),
	}

	a.Hourly = make(map[string]time.Duration)
	for i := 0; i <= 23; i++ {
		a.Hourly[fmt.Sprintf("%02d:00", i)] = time.Duration(0)
	}
}

// getSessionDuration returns the elapsed time for a session within the
// bounds of the reporting period.
func (s *Stats) getSessionDuration(
	sess *models.Session,
) time.Duration {
	var duration time.Duration

outer:
	for _, event := range sess.Timeline {
		if event.StartTime.After(s.StartTime) && event.EndTime.Before(s.EndTime) {
			duration += event.EndTime.Sub(event.StartTime)
			continue
		}

		for date := event.StartTime; date.Before(event.EndTime); date = date.Add(1 * time.Minute) {
			// prevent minutes that fall outside the specified bounds
			// from being included
			if date.Before(s.StartTime) {
				continue
			}

			if date.After(s.EndTime) {
				break outer
			}

			duration += time.Minute * 1
		}
	}

	return duration
}

func (s *Stats) updateAggr(
	event models.SessionTimeline,
	totals *Aggregates,
	period aggregatePeriod,
) {
	for date := event.StartTime; date.Before(event.EndTime); date = date.Add(1 * time.Minute) {
		if date.Before(s.StartTime) {
			continue
		}

		if date.After(s.EndTime) {
			break
		}

		switch period {
		case yearly:
			totals.Yearly[strconv.Itoa(date.Year())] += time.Minute * 1
		case monthly:
			totals.Monthly[date.Month().String()] += time.Minute * 1
		case weekday:
			totals.Weekday[date.Weekday().String()] += time.Minute * 1
		case weekly:
			y, w := date.ISOWeek()
			totals.Weekly[fmt.Sprintf("%d-W%d", y, w)] += time.Minute * 1
		case daily:
			totals.Daily[date.Format("2006-01-02")] += time.Minute * 1
		case hourly:
			totals.Hourly[date.Format("15:00")] += time.Minute * 1
		case all:
			y, w := date.ISOWeek()
			totals.Monthly[date.Month().String()] += time.Minute * 1
			totals.Weekday[date.Weekday().String()] += time.Minute * 1
			totals.Weekly[fmt.Sprintf("%d-W%d", y, w)] += time.Minute * 1
			totals.Daily[date.Format("2006-01-02")] += time.Minute * 1
			totals.Hourly[date.Format("15:00")] += time.Minute * 1
			totals.Yearly[strconv.Itoa(date.Year())] += time.Minute * 1
		}
	}
}

// filterSessions ensures that sessions with an invalid end date are ignored.
// TODO: Filtering sessions should not be done here.
// func filterSessions(sessions []session.Session) []session.Session {
// 	filtered := sessions[:0]
//
// 	for i := range sessions {
// 		sess := sessions[i]
//
// 		if sess.EndTime.IsZero() || sess.EndTime.Before(sess.StartTime) {
// 			continue
// 		}
//
// 		filtered = append(filtered, sess)
// 	}
//
// 	return filtered
// }

func (s *Stats) computeAggregates() {
	var totals Aggregates

	totals.init(s.StartTime, s.EndTime)

	s.LastDayTimeline = []Timeline{}

	for i := range s.Sessions {
		sess := s.Sessions[i]

		for _, event := range sess.Timeline {
			start := event.StartTime
			end := event.EndTime

			t := Timeline{}

			endTimeBeginning := timeutil.RoundToStart(s.EndTime)

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

			if start.After(s.StartTime) && end.Before(s.EndTime) {
				if start.Year() == end.Year() {
					totals.Yearly[strconv.Itoa(start.Year())] += end.Sub(start)
				} else {
					s.updateAggr(event, &totals, yearly)
				}

				if start.Month() == end.Month() {
					totals.Monthly[start.Month().String()] += end.Sub(
						start,
					)
				} else {
					s.updateAggr(event, &totals, monthly)
				}

				startY, startW := start.ISOWeek()
				_, endW := end.ISOWeek()

				if startW == endW {
					totals.Weekly[fmt.Sprintf("%d-W%d", startY, startW)] += end.Sub(
						start,
					)
				} else {
					s.updateAggr(event, &totals, weekly)
				}

				if start.Weekday() == end.Weekday() {
					totals.Weekday[start.Weekday().String()] += end.Sub(
						start,
					)
				} else {
					s.updateAggr(event, &totals, weekday)
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

// computeSummary calculates the total minutes, completed sessions, and
// abandoned sessions for the current time period.
func (s *Stats) computeSummary() {
	var totals Summary

	totals.Tags = make(map[string]time.Duration)

	for i := range s.Sessions {
		sess := s.Sessions[i]

		duration := s.getSessionDuration(sess)

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

	hoursDiff := timeutil.Round(s.EndTime.Sub(s.StartTime).Hours())

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

func sortByName(recs []Record) {
	slices.SortStableFunc(recs, func(a, b Record) int {
		return cmp.Compare(a.Name, b.Name)
	})
}

func sortNatural(recs []Record) {
	sort.Slice(recs, func(i, j int) bool {
		return natural.Less(recs[i].Name, recs[j].Name)
	})
}

func sortMonths(recs []Record) {
	calendarOrder := map[string]int{
		"January":   1,
		"February":  2,
		"March":     3,
		"April":     4,
		"May":       5,
		"June":      6,
		"July":      7,
		"August":    8,
		"September": 9,
		"October":   10,
		"November":  11,
		"December":  12,
	}

	sort.Slice(recs, func(i, j int) bool {
		return calendarOrder[recs[i].Name] < calendarOrder[recs[j].Name]
	})
}

func sortWeekdays(recs []Record) {
	calendarOrder := map[string]int{
		"Sunday":    1,
		"Monday":    2,
		"Tuesday":   3,
		"Wednesday": 4,
		"Thursday":  5,
		"Friday":    6,
		"Saturday":  7,
	}

	sort.Slice(recs, func(i, j int) bool {
		return calendarOrder[recs[i].Name] < calendarOrder[recs[j].Name]
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
		r.Tags = append(r.Tags, Record{
			Name:     k,
			Duration: v,
		})
	}

	for k, v := range s.Aggregates.Hourly {
		r.Hourly = append(r.Hourly, Record{
			Name:     k,
			Duration: v,
		})
	}

	for k, v := range s.Aggregates.Daily {
		r.Daily = append(r.Daily, Record{
			Name:     k,
			Duration: v,
		})
	}

	for k, v := range s.Aggregates.Weekday {
		r.Weekday = append(r.Weekday, Record{
			Name:     k,
			Duration: v,
		})
	}

	for k, v := range s.Aggregates.Weekly {
		r.Weekly = append(r.Weekly, Record{
			Name:     k,
			Duration: v,
		})
	}

	for k, v := range s.Aggregates.Monthly {
		r.Monthly = append(r.Monthly, Record{
			Name:     k,
			Duration: v,
		})
	}

	for k, v := range s.Aggregates.Yearly {
		r.Yearly = append(r.Yearly, Record{
			Name:     k,
			Duration: v,
		})
	}

	slices.SortStableFunc(r.Tags, func(a, b Record) int {
		return cmp.Compare(b.Duration, a.Duration)
	})

	sortByName(r.Hourly)
	sortByName(r.Daily)
	sortWeekdays(r.Weekday)
	sortNatural(r.Weekly)
	sortMonths(r.Monthly)
	sortByName(r.Yearly)

	return json.Marshal(r)
}
