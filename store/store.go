// Package store connects to the data store and manages timers and sessions
package store

import (
	"bytes"
	"encoding/json"
	"errors"
	"io/fs"
	"time"

	bolt "go.etcd.io/bbolt"
	"golang.org/x/exp/slices"

	"github.com/ayoisaiah/focus/internal/session"
)

var pathToDB string

var (
	errFocusRunning = errors.New(
		"is Focus already running? Only one instance can be active at a time",
	)
	errNoPausedSession = errors.New(
		"session not found: please start a new session",
	)
)

// Client is a BoltDB database client.
type Client struct {
	*bolt.DB
}

func (c *Client) UpdateSession(sess *session.Session) error {
	key := []byte(sess.StartTime.Format(time.RFC3339))

	value, err := json.Marshal(sess)
	if err != nil {
		return err
	}

	return c.Update(func(tx *bolt.Tx) error {
		return tx.Bucket([]byte("sessions")).Put(key, value)
	})
}

func (c *Client) UpdateTimer(
	dateStarted,
	timerBytes []byte,
) error {
	return c.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte("timers"))

		return b.Put(dateStarted, timerBytes)
	})
}

func (c *Client) GetSession(
	sessionKey []byte,
) (*session.Session, error) {
	var sess session.Session

	err := c.View(func(tx *bolt.Tx) error {
		sessBytes := tx.Bucket([]byte("sessions")).Get(sessionKey)
		if len(sessBytes) == 0 {
			// this will initialise a new session
			return nil
		}

		return json.Unmarshal(sessBytes, &sess)
	})

	return &sess, err
}

func (c *Client) DeleteSessions(sessions []session.Session) error {
	return c.Update(func(tx *bolt.Tx) error {
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

func (c *Client) DeleteTimer(timerKey []byte) error {
	return c.Update(func(tx *bolt.Tx) error {
		return tx.Bucket([]byte("timers")).Delete(timerKey)
	})
}

func (c *Client) Open() error {
	db, err := openDB(pathToDB)
	if err != nil {
		return err
	}

	*c = Client{
		db,
	}

	return nil
}

func (c *Client) RetrievePausedTimers() ([][]byte, error) {
	var timers [][]byte

	err := c.View(func(tx *bolt.Tx) error {
		c := tx.Bucket([]byte("timers")).Cursor()

		for k, v := c.Last(); k != nil; k, v = c.Prev() {
			timers = append(timers, v)
		}

		return nil
	})

	if len(timers) == 0 {
		return nil, errNoPausedSession
	}

	return timers, err
}

func (c *Client) GetSessions(
	startTime, endTime time.Time,
	tags []string,
) ([]session.Session, error) {
	var b [][]byte

	err := c.View(func(tx *bolt.Tx) error {
		c := tx.Bucket([]byte("sessions")).Cursor()
		min := []byte(startTime.Format(time.RFC3339))
		max := []byte(endTime.Format(time.RFC3339))

		//nolint:ineffassign,staticcheck // due to how boltdb works
		sk, sv := c.Seek(min)
		// get the previous session so as to check if
		// it was ended within the specified time bounds
		pk, pv := c.Prev()
		if pk != nil {
			var sess session.Session
			err := json.Unmarshal(pv, &sess)
			if err != nil {
				return err
			}

			// include session in results if it was ended
			// in the bounds of the specified time period
			if sess.EndTime.After(startTime) {
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
				var sess session.Session
				err := json.Unmarshal(v, &sess)
				if err != nil {
					return err
				}

				for _, t := range sess.Tags {
					if slices.Contains(tags, t) {
						b = append(b, v)
					}
				}
			} else {
				b = append(b, v)
			}
		}

		return nil
	})

	//nolint:prealloc // TODO: figure out why pre-allocating causes empty sessions
	var s []session.Session

	for _, v := range b {
		sess := session.Session{}

		err = json.Unmarshal(v, &sess)
		if err != nil {
			return nil, err
		}

		s = append(s, sess)
	}

	return s, err
}

// open creates or opens a database and locks it.
func openDB(pathToDB string) (*bolt.DB, error) {
	var fileMode fs.FileMode = 0o600

	db, err := bolt.Open(
		pathToDB,
		fileMode,
		&bolt.Options{Timeout: 1 * time.Second},
	)
	if err != nil {
		if errors.Is(err, bolt.ErrDatabaseOpen) ||
			errors.Is(err, bolt.ErrTimeout) {
			return nil, errFocusRunning
		}

		return nil, err
	}

	return db, nil
}

// NewClient returns a wrapper to a BoltDB connection.
func NewClient(dbPath string) (*Client, error) {
	pathToDB = dbPath

	db, err := openDB(pathToDB)
	if err != nil {
		return nil, err
	}
	// Create the necessary buckets for storing data if they do not exist already
	err = db.Update(func(tx *bolt.Tx) error {
		_, err = tx.CreateBucketIfNotExists([]byte("sessions"))
		if err != nil {
			return err
		}

		_, err = tx.CreateBucketIfNotExists([]byte("timers"))
		return err
	})
	if err != nil {
		return nil, err
	}

	return &Client{
		db,
	}, nil
}
