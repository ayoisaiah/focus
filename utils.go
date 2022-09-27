package focus

import "math"

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

// firstNonEmptyString returns its first non-empty argument, or "" if all
// arguments are empty.
func firstNonEmptyString(ss ...string) string {
	for _, s := range ss {
		if s != "" {
			return s
		}
	}

	return ""
}

func sliceIncludes[T comparable](elems []T, searchElement T) bool {
	for _, v := range elems {
		if v == searchElement {
			return true
		}
	}

	return false
}
