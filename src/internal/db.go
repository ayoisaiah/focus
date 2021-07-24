package focus

import (
	"bytes"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"time"

	bolt "go.etcd.io/bbolt"
)

const (
	dbFile = "focus.db"
)

const (
	errSingleInstanceAllowed = Error(
		"Only one instance of Focus can be active at a time",
	)
)

type DB interface {
	init() error
	getSessions(startTime, endTime time.Time) ([][]byte, error)
	deleteTimerState() error
	getTimerState() ([]byte, []byte, error)
	saveTimerState(timer, sessionKey []byte) error
	updateSession(key, value []byte) error
	deleteSessions(startTime, endTime time.Time) error
	close() error
}

// Store is a wrapper for a BoltDB connection.
type Store struct {
	conn *bolt.DB
}

// init initialises a BoltDB connection
// and creates the buckets for storing data
// if necessary.
func (s *Store) init() error {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return err
	}

	pathToConfigDir := filepath.Join(homeDir, configPath)

	// Ensure the config directory exists
	err = os.MkdirAll(pathToConfigDir, 0750)
	if err != nil {
		return err
	}

	pathToDB := filepath.Join(pathToConfigDir, dbFile)

	var fileMode fs.FileMode = 0600

	db, err := bolt.Open(
		pathToDB,
		fileMode,
		&bolt.Options{Timeout: 1 * time.Second},
	)
	if err != nil {
		return err
	}

	s.conn = db

	err = s.conn.Update(func(tx *bolt.Tx) error {
		_, err = tx.CreateBucketIfNotExists([]byte("sessions"))
		if err != nil {
			return err
		}

		_, err = tx.CreateBucketIfNotExists([]byte("timer"))
		return err
	})

	return err
}

// updateSession creates or updates a pomodoro session
// in the database. The session is created if it doesn't
// exist already, or overwritten if it does.
func (s *Store) updateSession(key, value []byte) error {
	return s.conn.Update(func(tx *bolt.Tx) error {
		return tx.Bucket([]byte("sessions")).Put(key, value)
	})
}

// saveTimerState persists the current timer settings,
// and the key of the paused session to the database.
func (s *Store) saveTimerState(timer, sessionKey []byte) error {
	return s.conn.Update(func(tx *bolt.Tx) error {
		err := tx.Bucket([]byte("timer")).Put([]byte("timer"), timer)
		if err != nil {
			return err
		}

		return tx.Bucket([]byte("timer")).
			Put([]byte("paused_session_key"), sessionKey)
	})
}

// getTimerState retrieves the stored timer and session key.
func (s *Store) getTimerState() (timer, session []byte, err error) {
	err = s.conn.View(func(tx *bolt.Tx) error {
		timer = tx.Bucket([]byte("timer")).Get([]byte("timer"))

		sessionKey := tx.Bucket([]byte("timer")).
			Get([]byte("paused_session_key"))

		session = tx.Bucket([]byte("sessions")).Get(sessionKey)

		return nil
	})

	return timer, session, err
}

// deleteSessions deletes a session from the database.
func (s *Store) deleteSessions(startTime, endTime time.Time) error {
	id := startTime.Format(time.RFC3339)

	return s.conn.Update(func(tx *bolt.Tx) error {
		c := tx.Bucket([]byte("sessions")).Cursor()
		min := []byte(startTime.Format(time.RFC3339))
		max := []byte(endTime.Format(time.RFC3339))

		for k, _ := c.Seek(min); k != nil && bytes.Compare(k, max) <= 0; k, _ = c.Seek(min) {
			err := c.Delete()
			if err != nil {
				return err
			}
		}

		return tx.Bucket([]byte("sessions")).Delete([]byte(id))
	})
}

// close closes the db connection to release file lock.
func (s *Store) close() error {
	return s.conn.Close()
}

// deleteTimerState removes the stored timer and session key.
func (s *Store) deleteTimerState() error {
	return s.conn.Update(func(tx *bolt.Tx) error {
		err := tx.Bucket([]byte("timer")).Delete([]byte("timer"))
		if err != nil {
			return err
		}

		return tx.Bucket([]byte("timer")).Delete([]byte("paused_session_key"))
	})
}

// getSessions retrieves the saved pomodoro sessions
// within the specified time period.
func (s *Store) getSessions(startTime, endTime time.Time) ([][]byte, error) {
	var b [][]byte

	err := s.conn.View(func(tx *bolt.Tx) error {
		c := tx.Bucket([]byte("sessions")).Cursor()
		min := []byte(startTime.Format(time.RFC3339))
		max := []byte(endTime.Format(time.RFC3339))

		for k, v := c.Seek(min); k != nil && bytes.Compare(k, max) <= 0; k, v = c.Next() {
			b = append(b, v)
		}

		return nil
	})

	return b, err
}

func NewStore() (*Store, error) {
	store := &Store{}

	err := store.init()
	if err != nil {
		if errors.Is(err, bolt.ErrDatabaseOpen) ||
			errors.Is(err, bolt.ErrTimeout) {
			return nil, errSingleInstanceAllowed
		}

		return nil, err
	}

	return store, nil
}
