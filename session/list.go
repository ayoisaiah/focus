package session

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/pterm/pterm"

	"github.com/ayoisaiah/focus/internal/ui"
)

const (
	noSessionsMsg = "No sessions found for the specified time range"
)

// printSessionsTable prints a session table to the command-line.
func printSessionsTable(w io.Writer, sessions []Session) {
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

// List prints out a table of all the sessions that
// were created within the specified time range.
func List(sessions []Session) error {
	if len(sessions) == 0 {
		pterm.Info.Println(noSessionsMsg)
		return nil
	}

	printSessionsTable(os.Stdout, sessions)

	return nil
}
