package stats

import (
	"fmt"
	"io"
	"strings"

	"github.com/pterm/pterm"

	"github.com/ayoisaiah/focus/internal/color"
	"github.com/ayoisaiah/focus/internal/session"
)

func printTable(data [][]string, writer io.Writer) {
	d := [][]string{
		{"#", "START DATE", "END DATE", "TAGS", "STATUS"},
	}

	d = append(d, data...)

	table := pterm.DefaultTable
	table.Boxed = true

	str, err := table.WithHasHeader().WithData(d).Srender()
	if err != nil {
		pterm.Error.Printfln("Failed to output session table: %s", err.Error())
		return
	}

	fmt.Fprintln(writer, str)
}

func printSessionsTable(w io.Writer, sessions []session.Session) {
	tableBody := make([][]string, 0)

	for i := range sessions {
		sess := sessions[i]

		statusText := color.Green("completed")
		if !sess.Completed {
			statusText = color.Red("abandoned")
		}

		endDate := sess.EndTime.Format("January 02, 2006 03:04 PM")
		if sess.EndTime.IsZero() {
			endDate = ""
		}

		tags := strings.Join(sess.Tags, ", ")

		row := []string{
			fmt.Sprintf("%d", i+1),
			sess.StartTime.Format("January 02, 2006 03:04 PM"),
			endDate,
			tags,
			statusText,
		}

		tableBody = append(tableBody, row)
	}

	printTable(tableBody, w)
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

	printSessionsTable(opts.Stdout, sessions)

	return nil
}
