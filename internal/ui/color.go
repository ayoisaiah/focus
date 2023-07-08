package ui

import (
	"github.com/pterm/pterm"
)

var DarkTheme bool

func Green(a any) string {
	if DarkTheme {
		return pterm.LightGreen(a)
	}

	return pterm.Green(a)
}

func Cyan(a any) string {
	if DarkTheme {
		return pterm.LightCyan(a)
	}

	return pterm.Cyan(a)
}

func Magenta(a any) string {
	if DarkTheme {
		return pterm.LightMagenta(a)
	}

	return pterm.Magenta(a)
}

func Blue(a any) string {
	if DarkTheme {
		return pterm.LightBlue(a)
	}

	return pterm.Blue(a)
}

func Red(a any) string {
	if DarkTheme {
		return pterm.LightRed(a)
	}

	return pterm.Red(a)
}

func Highlight(a any) string {
	if DarkTheme {
		return pterm.LightWhite(a)
	}

	return pterm.Black(a)
}
