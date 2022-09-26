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
