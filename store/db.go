package store

import (
	"time"

	"github.com/ayoisaiah/focus/internal/session"
)

// DB is the database storage interface.
type DB interface {
	RetrievePausedTimers() ([][]byte, error)
	// GetSessions returns saved sessions according to the time and tag
	// constraints
	GetSessions(
		startTime, endTime time.Time,
		tag []string,
	) ([]session.Session, error)
	// UpdateSession updates a Focus session. The session is created if it doesn't
	// exist already, or overwritten if it does.
	UpdateSession(sess *session.Session) error
	// DeleteSessions deletes one or more saved sessions
	DeleteSessions(sessions []session.Session) error
	// DeleteTimer deletes a previously saved timer state
	DeleteTimer(timerKey []byte) error
	// GetInterrupted returns a previously stored timer and a curresponding work session
	// (if any)
	GetInterrupted(
		sessionKey []byte,
	) (sess *session.Session, err error)
	// SaveTimer stores a timer and the key of an interrupted session
	SaveTimer(pausedTime, timerBytes []byte) error
	// Close ends the database connection
	Close() error
	// Open begins a databse connection
	Open() error
}
