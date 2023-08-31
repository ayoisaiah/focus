package session

import (
	"bufio"
	"fmt"
	"os"

	"github.com/pterm/pterm"
)

// EditTags.edits the tags of the specified sessions.
func EditTags(
	sessions []Session,
	args []string,
	updateFunc func(*Session) error,
) error {
	if len(sessions) == 0 {
		pterm.Info.Println(noSessionsMsg)
		return nil
	}

	for i := range sessions {
		sessions[i].Tags = args
	}

	printSessionsTable(os.Stdout, sessions)

	warning := pterm.Warning.Sprint(
		"The sessions above will be updated. Press ENTER to proceed",
	)

	fmt.Fprint(os.Stdout, warning)

	reader := bufio.NewReader(os.Stdin)

	_, _ = reader.ReadString('\n')

	for i := range sessions {
		sess := sessions[i]

		err := updateFunc(&sess)
		if err != nil {
			return err
		}
	}

	return nil
}
