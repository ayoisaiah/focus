package time

import "math"

const minutesInAnHour = 60

// roundTime rounds a time value in seconds, minutes, or hours to the nearest integer.
func Round(t float64) int {
	return int(math.Round(t))
}

// MinsToHoursAndMins expresses a minutes value in hours and mins.
func MinsToHoursAndMins(val int) (hrs, mins int) {
	hrs = int(math.Floor(float64(val) / float64(minutesInAnHour)))
	mins = val % minutesInAnHour

	return
}
