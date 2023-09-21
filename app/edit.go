package app

import (
	"bufio"
	"fmt"
	"os"
	"time"

	"github.com/pterm/pterm"

	"github.com/ayoisaiah/focus/internal/models"
	"github.com/ayoisaiah/focus/store"
)

// editTags.edits the tags of the specified sessions.
func editTags(
	db store.DB,
	sessions []*models.Session,
	args []string,
) error {
	if len(sessions) == 0 {
		pterm.Info.Println(noSessionsMsg)
		return nil
	}

	m := make(map[time.Time]*models.Session)

	for i := range sessions {
		sessions[i].Tags = args

		m[sessions[i].StartTime] = sessions[i]
	}

	printSessionsTable(os.Stdout, sessions)

	warning := pterm.Warning.Sprint(
		"The sessions above will be updated. Press ENTER to proceed",
	)

	fmt.Fprint(os.Stdout, warning)

	reader := bufio.NewReader(os.Stdin)

	_, _ = reader.ReadString('\n')

	return db.UpdateSessions(m)
}
