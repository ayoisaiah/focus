package report

import (
	"fmt"
	"os"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type (
	style struct {
		success lipgloss.Style
		error   lipgloss.Style
	}
	// Status represents the status of a running timer.
	Status struct {
		EndTime           time.Time `json:"end_date"`
		Name              string    `json:"name"`
		Tags              []string  `json:"tags"`
		WorkCycle         int       `json:"work_cycle"`
		LongBreakInterval int       `json:"long_break_interval"`
	}
)

var defaultStyle = style{
	success: lipgloss.NewStyle().
		Foreground(lipgloss.Color("#78BC61")),
	error: lipgloss.NewStyle().
		Foreground(lipgloss.Color("#DA3E52")),
}

func SessionAdded() {
	fmt.Println(defaultStyle.success.Render("session added successfully"))
}

func Error(err error) {
	fmt.Println(defaultStyle.success.Render(err.Error()))
}

func Fatal(err error) tea.Cmd {
	fmt.Println(defaultStyle.success.Render(err.Error()))
	return tea.Quit
}

func Quit(err error) {
	fmt.Println(defaultStyle.success.Render(err.Error()))
	os.Exit(1)
}
