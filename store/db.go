package store

import (
	"time"

	"github.com/ayoisaiah/focus/config"
	"github.com/ayoisaiah/focus/internal/session"
)

// DB is the database storage interface.
type DB interface {
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
	// DeleteTimerState deletes a previously saved timer and interrupted session
	// key
	DeleteTimer() error
	// GetInterrupted returns a previously stored timer and a curresponding work session
	// (if any)
	GetInterrupted() (opts *config.TimerConfig, sess *session.Session, workCycle int, err error)
	// SaveTimer stores a timer and the key of an interrupted session
	SaveTimer(sessionKey []byte, opts *config.TimerConfig, workCycle int) error
	// Close ends the database connection
	Close() error
	// Open begins a databse connection
	Open() error
}
