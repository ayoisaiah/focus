package stats

import (
	"bufio"
	"fmt"

	"github.com/pterm/pterm"
)

// Delete attempts to delete all sessions that fall in the specified time range.
// It requests for confirmation before proceeding with the permanent removal of
// the sessions from the database.
func Delete() error {
	sessions, err := db.GetSessions(opts.StartTime, opts.EndTime, opts.Tags)
	if err != nil {
		return err
	}

	if len(sessions) == 0 {
		return nil
	}

	printSessionsTable(opts.Stdout, sessions)

	warning := pterm.Warning.Sprint(
		"The above sessions will be deleted permanently. Press ENTER to proceed",
	)
	fmt.Fprint(opts.Stdout, warning)

	reader := bufio.NewReader(opts.Stdin)

	_, _ = reader.ReadString('\n')

	return db.DeleteSessions(sessions)
}
