package report

import (
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/pterm/pterm"
)

func SessionAdded() {
	pterm.Info.Print("session added successfully")
}

func Error(err error) {
	pterm.Error.Println(err)
}

func Fatal(err error) tea.Cmd {
	pterm.Error.Println(err)
	return tea.Quit
}

func Quit(err error) {
	pterm.Error.Println(err)
	os.Exit(1)
}
