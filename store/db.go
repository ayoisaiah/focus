package store

import (
	"time"

	"github.com/ayoisaiah/focus/internal/models"
)

// DB is the database storage interface.
type DB interface {
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
	// Close ends the database connection
	Close() error
	// Open initiates a database connection
	Open() error
}
