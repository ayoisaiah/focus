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

// delSessions deletes all the specified sessions. It requests for confirmation
// before proceeding with the operation.
func delSessions(
	db store.DB,
	sessions []*models.Session,
) error {
	if len(sessions) == 0 {
		return nil
	}

	t := make([]time.Time, len(sessions))

	for i := range sessions {
		t[i] = sessions[i].StartTime
	}

	printSessionsTable(os.Stdout, sessions)

	warning := pterm.Warning.Sprint(
		"The above sessions will be deleted permanently. Press ENTER to proceed",
	)

	fmt.Fprint(os.Stdout, warning)

	reader := bufio.NewReader(os.Stdin)

	_, _ = reader.ReadString('\n')

	return db.DeleteSessions(t)
}
