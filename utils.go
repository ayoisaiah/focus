package focus

import (
	"math"

	"github.com/ayoisaiah/focus/config"
	"github.com/pterm/pterm"
)

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

func Green(a any) string {
	cfg := config.Get()
	if cfg.DarkTheme {
		return pterm.LightGreen(a)
	}

	return pterm.Green(a)
}

func Cyan(a any) string {
	cfg := config.Get()
	if cfg.DarkTheme {
		return pterm.LightCyan(a)
	}

	return pterm.Cyan(a)
}

func Magenta(a any) string {
	cfg := config.Get()
	if cfg.DarkTheme {
		return pterm.LightMagenta(a)
	}

	return pterm.Magenta(a)
}

func Blue(a any) string {
	cfg := config.Get()
	if cfg.DarkTheme {
		return pterm.LightBlue(a)
	}

	return pterm.Blue(a)
}

func Red(a any) string {
	cfg := config.Get()
	if cfg.DarkTheme {
		return pterm.LightRed(a)
	}

	return pterm.Red(a)
}

func Highlight(a any) string {
	cfg := config.Get()
	if cfg.DarkTheme {
		return pterm.LightWhite(a)
	}

	return pterm.Black(a)
}
