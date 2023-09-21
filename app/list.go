package app

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/pterm/pterm"

	"github.com/ayoisaiah/focus/internal/models"
	"github.com/ayoisaiah/focus/internal/ui"
)

const (
	noSessionsMsg = "No sessions found for the specified time range"
)

// printSessionsTable prints a session table to the command-line.
func printSessionsTable(w io.Writer, sessions []*models.Session) {
	tableBody := make([][]string, len(sessions))

	for i := range sessions {
		sess := sessions[i]

		statusText := ui.Green("completed")
		if !sess.Completed {
			statusText = ui.Red("abandoned")
		}

		endDate := sess.EndTime.Format("Jan 02, 2006 03:04 PM")
		if sess.EndTime.IsZero() {
			endDate = ""
		}

		tags := strings.Join(sess.Tags, " · ")

		row := []string{
			fmt.Sprintf("%d", i+1),
			sess.StartTime.Format("Jan 02, 2006 03:04 PM"),
			endDate,
			tags,
			statusText,
		}

		tableBody[i] = row
	}

	tableBody = append([][]string{
		{"#", "START DATE", "END DATE", "TAGS", "STATUS"},
	}, tableBody...)

	ui.PrintTable(tableBody, w)
}

// listSessions prints out a table of sessions.
func listSessions(sessions []*models.Session) error {
	if len(sessions) == 0 {
		pterm.Info.Println(noSessionsMsg)
		return nil
	}

	printSessionsTable(os.Stdout, sessions)

	return nil
}
