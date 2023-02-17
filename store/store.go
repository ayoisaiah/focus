// Package store handles connections to the data store and managing sessions
package store

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/ayoisaiah/focus/config"
	"github.com/ayoisaiah/focus/internal/session"
	"github.com/pterm/pterm"
	bolt "go.etcd.io/bbolt"
	"golang.org/x/exp/slices"
)

var pathToDB string

var (
	errFocusRunning = errors.New(
		"Is Focus already running? Only one instance can be active at a time",
	)

	errNoPausedSession = errors.New(
		"Paused session not found, please start a new session",
	)
)

type timerState struct {
	Opts       *config.TimerConfig `json:"opts"`
	WorkCycle  int                 `json:"work_cycle"`
	SessionKey []byte              `json:"session_key"`
	Timestamp  string              `json:"timestamp"`
}

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

func (c *Client) SaveTimer(
	sessionKey []byte,
	opts *config.TimerConfig,
	workCycle int,
) error {
	timestamp := time.Now().Format(time.RFC3339)

	value, err := json.Marshal(timerState{
		opts,
		workCycle,
		sessionKey,
		timestamp,
	})
	if err != nil {
		return err
	}

	return c.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte("timers"))

		return b.Put([]byte(timestamp), value)
	})
}

func (c *Client) GetInterrupted(
	pausedKey []byte,
) (opts *config.TimerConfig, sess *session.Session, workCycle int, err error) {
	var t timerState

	var timerKey, timerBytes []byte

	t.Opts = &config.TimerConfig{
		Stdout: os.Stdout,
		Stderr: os.Stderr,
		Stdin:  os.Stdin,
	}

	err = c.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte("timers"))

		c := b.Cursor()

		if len(pausedKey) > 0 {
			timerKey, timerBytes = c.Seek(pausedKey)
		} else {
			timerKey, timerBytes = c.Last()
		}

		if len(timerBytes) == 0 {
			return errNoPausedSession
		}

		err = json.Unmarshal(timerBytes, &t)
		if err != nil {
			return err
		}

		opts = t.Opts
		workCycle = t.WorkCycle

		sessBytes := tx.Bucket([]byte("sessions")).Get(t.SessionKey)
		if len(sessBytes) == 0 {
			return nil
		}

		sess = &session.Session{}

		err = json.Unmarshal(sessBytes, sess)
		if err != nil {
			return err
		}

		return nil
	})
	if err != nil {
		return
	}

	err = c.DeleteTimer(timerKey)

	//nolint:nakedret // ok to use naked return
	return
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

func printTable(data [][]string, writer io.Writer) {
	d := [][]string{
		{"#", "PAUSED DATE", "TAGS"},
	}

	d = append(d, data...)

	table := pterm.DefaultTable
	table.Boxed = true

	str, err := table.WithHasHeader().WithData(d).Srender()
	if err != nil {
		pterm.Error.Printfln("Failed to output session table: %s", err.Error())
		return
	}

	fmt.Fprintln(writer, str)
}

func (c *Client) SelectPaused() ([]byte, error) {
	var selected []byte

	m := make(map[int][]byte, 0)

	err := c.View(func(tx *bolt.Tx) error {
		c := tx.Bucket([]byte("timers")).Cursor()

		tableBody := make([][]string, 0)

		var counter int

		for k, v := c.Last(); k != nil; k, v = c.Prev() {
			counter++

			m[counter] = k

			var t timerState

			err := json.Unmarshal(v, &t)
			if err != nil {
				return err
			}

			keyTime, err := time.Parse(time.RFC3339, t.Timestamp)
			if err != nil {
				return err
			}

			row := []string{
				fmt.Sprintf("%d", counter),
				keyTime.Format("January 02, 2006 03:04:05 PM"),
				strings.Join(t.Opts.Tags, ", "),
			}

			tableBody = append(tableBody, row)
		}

		if len(tableBody) == 0 {
			return errNoPausedSession
		}

		printTable(tableBody, os.Stdout)

		reader := bufio.NewReader(os.Stdin)

		fmt.Fprint(os.Stdout, "\033[s")
		fmt.Fprint(os.Stdout, "Type a number and press ENTER: ")

		// Block until user input before beginning next session
		input, err := reader.ReadString('\n')
		if err != nil {
			return err
		}

		num, err := strconv.Atoi(strings.TrimSpace(input))
		if err != nil {
			return err
		}

		if key, ok := m[num]; !ok {
			return fmt.Errorf("%d is not associated with a session", num)
		} else {
			selected = key
		}

		return nil
	})

	return selected, err
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
