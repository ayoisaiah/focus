package timer

import (
	"cmp"
	"slices"

	"github.com/ayoisaiah/focus/internal/models"
	"github.com/ayoisaiah/focus/internal/timeutil"
	"github.com/ayoisaiah/focus/store"
)

// getTimerSessions retrieves all paused timers and their corresponding sessions.
func getTimerSessions(
	db store.DB,
) ([]*models.Timer, map[string]*models.Session, error) {
	pausedTimers, err := db.RetrievePausedTimers()
	if err != nil {
		return nil, nil, err
	}

	pausedSessions := make(map[string]*models.Session)

	for i := range pausedTimers {
		v := pausedTimers[i]

		s, err := db.GetSession(v.SessionKey)
		if err != nil {
			return nil, nil, err
		}

		pausedSessions[string(timeutil.ToKey(v.SessionKey))] = s
	}

	slices.SortStableFunc(pausedTimers, func(a, b *models.Timer) int {
		return cmp.Compare(b.PausedTime.UnixNano(), a.PausedTime.UnixNano())
	})

	return pausedTimers, pausedSessions, nil
}
