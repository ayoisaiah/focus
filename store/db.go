package store

import (
	"time"

	"github.com/ayoisaiah/focus/internal/models"
)

// DB is the database storage interface.
type DB interface {
	RetrievePausedTimers() ([]*models.Timer, error)
	// GetSessions returns saved sessions according to the specified time and tag
	// constraints
	GetSessions(
		since, until time.Time,
		tags []string,
	) ([]*models.Session, error)
	// UpdateSessions updates one or more Focus sessions.
	// Each session is created if it doesn't
	// exist already, or overwritten if it does.
	UpdateSessions(map[time.Time]*models.Session) error
	// DeleteSessions deletes one or more saved sessions
	DeleteSessions(startTimes []time.Time) error
	// GetSession returns a previously created session. If the session does not
	// exist, no error is returned.
	GetSession(
		startTime time.Time,
	) (sess *models.Session, err error)
	// UpdateTimer stores a timer and the key of an interrupted session
	UpdateTimer(timer *models.Timer) error
	// DeleteTimer deletes a previously saved timer if it exists
	DeleteTimer(startTime time.Time) error
	// DeleteAllTimers deletes all the saved timers in the database
	DeleteAllTimers() error
	// Close ends the database connection
	Close() error
	// Open initiates a database connection
	Open() error
}
