// Package store connects to the data store and manages timers and sessions
package store

import (
	"bytes"
	"encoding/json"
	"errors"
	"io/fs"
	"slices"
	"time"

	bolt "go.etcd.io/bbolt"

	"github.com/ayoisaiah/focus/config"
	"github.com/ayoisaiah/focus/internal/timeutil"
)

var (
	errFocusRunning = errors.New(
		"is Focus already running? Only one instance can be active at a time",
	)
	errNoPausedTimer = errors.New(
		"no paused timers were found",
	)
)

const (
	sessionBucket = "sessions"
	timerBucket   = "timers"
)

// Client is a BoltDB database client.
type Client struct {
	*bolt.DB
}

func (c *Client) UpdateSessions(sessions map[time.Time][]byte) error {
	for k, v := range sessions {
		key := timeutil.ToKey(k)

		return c.Update(func(tx *bolt.Tx) error {
			return tx.Bucket([]byte(sessionBucket)).Put(key, v)
		})
	}

	return nil
}

func (c *Client) UpdateTimer(
	startTime time.Time,
	timerBytes []byte,
) error {
	return c.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(timerBucket))

		return b.Put(timeutil.ToKey(startTime), timerBytes)
	})
}

func (c *Client) GetSession(
	startTime time.Time,
) ([]byte, error) {
	var sessBytes []byte

	err := c.View(func(tx *bolt.Tx) error {
		key := timeutil.ToKey(startTime)

		sessBytes = tx.Bucket([]byte(sessionBucket)).Get(key)

		return nil
	})

	return sessBytes, err
}

func (c *Client) DeleteSessions(startTimes []time.Time) error {
	return c.Update(func(tx *bolt.Tx) error {
		for i := range startTimes {
			key := timeutil.ToKey(startTimes[i])

			err := tx.Bucket([]byte(sessionBucket)).Delete(key)
			if err != nil {
				return err
			}
		}

		return nil
	})
}

func (c *Client) DeleteTimer(startTime time.Time) error {
	return c.Update(func(tx *bolt.Tx) error {
		return tx.Bucket([]byte(timerBucket)).Delete(timeutil.ToKey(startTime))
	})
}

func (c *Client) DeleteAllTimers() error {
	err := c.Update(func(tx *bolt.Tx) error {
		c := tx.Bucket([]byte(timerBucket)).Cursor()

		for k, _ := c.First(); k != nil; k, _ = c.Next() {
			err := c.Delete()
			if err != nil {
				return err
			}
		}

		return nil
	})

	return err
}

func (c *Client) Open() error {
	db, err := openDB(config.DBFilePath())
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
		c := tx.Bucket([]byte(timerBucket)).Cursor()

		for k, v := c.Last(); k != nil; k, v = c.Prev() {
			timers = append(timers, v)
		}

		return nil
	})

	if len(timers) == 0 {
		return nil, errNoPausedTimer
	}

	return timers, err
}

type session struct {
	StartTime time.Time `json:"start_time"`
	EndTime   time.Time `json:"end_time"`
	Tags      []string  `json:"tags"`
}

func (c *Client) GetSessions(
	startTime, endTime time.Time,
	tags []string,
) ([][]byte, error) {
	var b [][]byte

	err := c.View(func(tx *bolt.Tx) error {
		c := tx.Bucket([]byte(sessionBucket)).Cursor()
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
				var sess session
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

	return b, err
}

// openDB creates or opens a database.
func openDB(dbFilePath string) (*bolt.DB, error) {
	var fileMode fs.FileMode = 0o600

	db, err := bolt.Open(
		dbFilePath,
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
func NewClient(dbFilePath string) (*Client, error) {
	db, err := openDB(dbFilePath)
	if err != nil {
		return nil, err
	}
	// Create the necessary buckets for storing data if they do not exist already
	err = db.Update(func(tx *bolt.Tx) error {
		_, err = tx.CreateBucketIfNotExists([]byte(sessionBucket))
		if err != nil {
			return err
		}

		_, err = tx.CreateBucketIfNotExists([]byte(timerBucket))
		return err
	})
	if err != nil {
		return nil, err
	}

	return &Client{
		db,
	}, nil
}
