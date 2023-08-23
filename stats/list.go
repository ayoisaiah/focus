package stats

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/pterm/pterm"

	"github.com/ayoisaiah/focus/internal/session"
	"github.com/ayoisaiah/focus/internal/ui"
)

// printSessionsTable prints a session table to the command-line.
func printSessionsTable(w io.Writer, sessions []session.Session) {
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
		{"#", "START DATE", "END DATE", "TAGGED", "STATUS"},
	}, tableBody...)

	ui.PrintTable(tableBody, w)
}

// List prints out a table of all the sessions that
// were created within the specified time range.
func List() error {
	sessions, err := db.GetSessions(opts.StartTime, opts.EndTime, opts.Tags)
	if err != nil {
		return err
	}

	if len(sessions) == 0 {
		pterm.Info.Println(noSessionsMsg)
		return nil
	}

	printSessionsTable(os.Stdout, sessions)

	return nil
}
