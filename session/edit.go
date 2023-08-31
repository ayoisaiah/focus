package session

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/pterm/pterm"
)

// EditTags.edits the tags of the specified sessions.
func EditTags(
	sessions []Session,
	args []string,
	updateFunc func(sessions map[time.Time][]byte) error,
) error {
	if len(sessions) == 0 {
		pterm.Info.Println(noSessionsMsg)
		return nil
	}

	m := make(map[time.Time][]byte)

	for i := range sessions {
		sessions[i].Tags = args

		b, err := json.Marshal(sessions[i])
		if err != nil {
			return err
		}

		m[sessions[i].StartTime] = b
	}

	printSessionsTable(os.Stdout, sessions)

	warning := pterm.Warning.Sprint(
		"The sessions above will be updated. Press ENTER to proceed",
	)

	fmt.Fprint(os.Stdout, warning)

	reader := bufio.NewReader(os.Stdin)

	_, _ = reader.ReadString('\n')

	return updateFunc(m)
}
