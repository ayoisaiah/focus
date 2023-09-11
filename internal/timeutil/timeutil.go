// Package timeutil provides utility functions and types for working with
// time-related operations.
package timeutil

import (
	"fmt"
	"math"
	"strconv"
	"time"
)

const minutesInAnHour = 60

const (
	HoursInADay      = 24
	MaxHoursInAMonth = 744  // 31 day months
	MaxHoursInAYear  = 8784 // Leap years
)

type Period string

const (
	PeriodAllTime   Period = "all-time"
	PeriodToday     Period = "today"
	PeriodYesterday Period = "yesterday"
	Period7Days     Period = "7days"
	Period14Days    Period = "14days"
	Period30Days    Period = "30days"
	Period90Days    Period = "90days"
	Period180Days   Period = "180days"
	Period365Days   Period = "365days"
)

var Range = map[Period]int{
	PeriodAllTime:   0,
	PeriodToday:     0,
	PeriodYesterday: -1,
	Period7Days:     -6,
	Period14Days:    -13,
	Period30Days:    -29,
	Period90Days:    -89,
	Period180Days:   -179,
	Period365Days:   -364,
}

var PeriodCollection = []Period{
	PeriodAllTime,
	PeriodToday,
	PeriodYesterday,
	Period7Days,
	Period14Days,
	Period30Days,
	Period90Days,
	Period180Days,
	Period365Days,
}

// Round rounds a time value in seconds, minutes, or hours to the nearest integer.
func Round(t float64) int {
	return int(math.Round(t))
}

// MinsToHoursAndMins expresses a minutes value in hours and mins.
func MinsToHoursAndMins(val int) (hrs, mins int) {
	hrs = int(math.Floor(float64(val) / float64(minutesInAnHour)))
	mins = val % minutesInAnHour

	return
}

// DaysIn returns the number of days in the month for the specified time.
func DaysIn(t time.Time) int {
	m := t.Month()
	year := t.Year()

	return time.Date(year, m+1, 0, 0, 0, 0, 0, time.UTC).Day()
}

// RoundToStart resets the given time to the start of the day.
func RoundToStart(t time.Time) time.Time {
	return time.Date(
		t.Year(),
		t.Month(),
		t.Day(),
		0,
		0,
		0,
		0,
		t.Location(),
	)
}

// RoundToEnd resets the given time to the end of the day.
func RoundToEnd(t time.Time) time.Time {
	return time.Date(
		t.Year(),
		t.Month(),
		t.Day(),
		23,
		59,
		59,
		0,
		t.Location(),
	)
}

func DayFormat(t time.Time) int {
	d := fmt.Sprintf("%d%02d%02d", t.Year(), t.Month(), t.Day())

	i, _ := strconv.Atoi(d)

	return i
}

// ToKey converts a time value to a database key for Bolt.
func ToKey(t time.Time) []byte {
	return []byte(t.Format(time.RFC3339Nano))
}
