package focus

import (
	"bytes"
	"encoding/json"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"time"

	"github.com/adrg/xdg"
	bolt "go.etcd.io/bbolt"
)

var dbFile = "focus.db"

func init() {
	if os.Getenv("FOCUS_ENV") == "development" {
		dbFile = "focus_dev.db"
	}
}

type DB interface {
	init() error
	getSessions(startTime, endTime time.Time, tag []string) ([][]byte, error)
	deleteTimerState() error
	getTimerState() ([]byte, []byte, error)
	saveTimerState(timer, sessionKey []byte) error
	updateSession(key, value []byte) error
	deleteSessions(sessions []session) error
	close() error
	open() error
}

// Store is a wrapper for a BoltDB connection.
type Store struct {
	conn *bolt.DB
}

// init initialises a BoltDB connection
// and creates the necessary buckets for storing data
// if they do not exist already.
func (s *Store) init() error {
	err := s.open()
	if err != nil {
		return err
	}

	// Create the buckets
	return s.conn.Update(func(tx *bolt.Tx) error {
		_, err := tx.CreateBucketIfNotExists([]byte("sessions"))
		if err != nil {
			return err
		}

		_, err = tx.CreateBucketIfNotExists([]byte("timer"))
		return err
	})
}

// updateSession creates or updates a work session
// in the database. The session is created if it doesn't
// exist already, or overwritten if it does.
func (s *Store) updateSession(key, value []byte) error {
	return s.conn.Update(func(tx *bolt.Tx) error {
		return tx.Bucket([]byte("sessions")).Put(key, value)
	})
}

// saveTimerState persists the current timer settings,
// and the key of the interrupted session to the database.
func (s *Store) saveTimerState(timer, sessionKey []byte) error {
	return s.conn.Update(func(tx *bolt.Tx) error {
		err := tx.Bucket([]byte("timer")).Put([]byte("timer"), timer)
		if err != nil {
			return err
		}

		return tx.Bucket([]byte("timer")).
			Put([]byte("interrrupted_session_key"), sessionKey)
	})
}

// getTimerState retrieves the state of the timer as of when it was
// last interrupted, and the corresponding work session (if any).
func (s *Store) getTimerState() (timer, session []byte, err error) {
	err = s.conn.View(func(tx *bolt.Tx) error {
		timer = tx.Bucket([]byte("timer")).Get([]byte("timer"))

		sessionKey := tx.Bucket([]byte("timer")).
			Get([]byte("interrrupted_session_key"))

		session = tx.Bucket([]byte("sessions")).Get(sessionKey)

		return nil
	})

	return timer, session, err
}

// deleteSessions deletes all sessions within the
// specified bounds from the database.
func (s *Store) deleteSessions(sessions []session) error {
	return s.conn.Update(func(tx *bolt.Tx) error {
		for i := range sessions {
			sess := sessions[i]
			id := sess.StartTime.Format(time.RFC3339)

			err := tx.Bucket([]byte("sessions")).Delete([]byte(id))
			if err != nil {
				return err
			}
		}

		return nil
	})
}

// open creates or opens a database and locks it
func (s *Store) open() error {
	pathToDB, err := xdg.DataFile(filepath.Join(configDir, dbFile))
	if err != nil {
		return err
	}

	var fileMode fs.FileMode = 0o600

	db, err := bolt.Open(
		pathToDB,
		fileMode,
		&bolt.Options{Timeout: 1 * time.Second},
	)
	if err != nil {
		if errors.Is(err, bolt.ErrDatabaseOpen) ||
			errors.Is(err, bolt.ErrTimeout) {
			return errFocusRunning
		}

		return err
	}

	s.conn = db

	return nil
}

// close closes the db connection to release file lock.
func (s *Store) close() error {
	return s.conn.Close()
}

// deleteTimerState removes the stored timer state and session key.
// from the database to signify a successful resumption of the session.
func (s *Store) deleteTimerState() error {
	return s.conn.Update(func(tx *bolt.Tx) error {
		err := tx.Bucket([]byte("timer")).Delete([]byte("timer"))
		if err != nil {
			return err
		}

		return tx.Bucket([]byte("timer")).
			Delete([]byte("interrrupted_session_key"))
	})
}

// getSessions retrieves the saved work sessions
// within the specified time period. It checks the previous
// work session just before the `startTime` to see if
// its end time is within the specified bounds. If so, it
// is included in the output. The `tag` parameter is used to filter
// sessions by tag. An empty string signifies no filtering.
func (s *Store) getSessions(
	startTime, endTime time.Time,
	tags []string,
) ([][]byte, error) {
	var b [][]byte

	err := s.conn.View(func(tx *bolt.Tx) error {
		c := tx.Bucket([]byte("sessions")).Cursor()
		min := []byte(startTime.Format(time.RFC3339))
		max := []byte(endTime.Format(time.RFC3339))

		//nolint:ineffassign,staticcheck // due to how boltdb works
		sk, sv := c.Seek(min)
		// get the previous session so as to check if
		// it was ended within the specified time bounds
		pk, pv := c.Prev()
		if pk != nil {
			var sess session
			err := json.Unmarshal(pv, &sess)
			if err != nil {
				return err
			}

			// include session in results if it was ended
			// in the bounds of the specified time period
			if !sess.EndTime.Before(startTime) {
				sk, sv = pk, pv
			} else {
				sk, sv = c.Next()
			}
		} else {
			sk, sv = c.Seek(min)
		}

		for k, v := sk, sv; k != nil && bytes.Compare(k, max) <= 0; k, v = c.Next() {
			// Filter out tags that don't match
			if len(tags) != 0 {
				var sess session
				err := json.Unmarshal(v, &sess)
				if err != nil {
					return err
				}

				for _, t := range sess.Tags {
					if sliceIncludes(tags, t) {
						b = append(b, v)
					}
				}
			} else {
				b = append(b, v)
			}
		}

		return nil
	})

	return b, err
}

// NewStore returns a wrapper to a BoltDB connection or an error.
func NewStore() (*Store, error) {
	store := &Store{}

	err := store.init()
	if err != nil {
		return nil, err
	}

	return store, nil
}
