package stats

import (
	"bufio"
	"fmt"

	"github.com/pterm/pterm"
)

// EditTags.edits the tags of the specified sessions.
func EditTags(args []string) error {
	sessions, err := db.GetSessions(opts.StartTime, opts.EndTime, opts.Tags)
	if err != nil {
		return err
	}

	if len(sessions) == 0 {
		pterm.Info.Println(noSessionsMsg)
		return nil
	}

	for i := range sessions {
		sessions[i].Tags = args
	}

	printSessionsTable(opts.Stdout, sessions)

	warning := pterm.Warning.Sprint(
		"The sessions above will be updated. Press ENTER to proceed",
	)
	fmt.Fprint(opts.Stdout, warning)

	reader := bufio.NewReader(opts.Stdin)

	_, _ = reader.ReadString('\n')

	for i := range sessions {
		sess := sessions[i]

		err = db.UpdateSession(&sess)
		if err != nil {
			return err
		}
	}

	return nil
}
