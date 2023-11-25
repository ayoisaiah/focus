package store

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"time"

	"go.etcd.io/bbolt"

	"github.com/ayoisaiah/focus/internal/models"
)

func migrateSessions_v1_4_0(tx *bbolt.Tx) error {
	bucket := tx.Bucket([]byte(sessionBucket))

	cur := bucket.Cursor()

	for k, v := cur.First(); k != nil; k, v = cur.Next() {
		var s models.Session

		err := json.Unmarshal(v, &s)
		if err != nil {
			return err
		}

		newKey := []byte(s.StartTime.Format(time.RFC3339Nano))

		err = cur.Delete()
		if err != nil {
			return err
		}

		err = bucket.Put(newKey, v)
		if err != nil {
			return err
		}
	}

	return nil
}

func migrateTimers_v1_4_0(tx *bbolt.Tx) error {
	bucket := tx.Bucket([]byte(timerBucket))

	cur := bucket.Cursor()

	type timer struct {
		DateStarted time.Time `json:"date_started"`
	}

	for k, v := cur.First(); k != nil; k, v = cur.Next() {
		var t timer

		err := json.Unmarshal(v, &t)
		if err != nil {
			return err
		}

		newKey := []byte(t.DateStarted.Format(time.RFC3339Nano))

		v = bytes.Replace(v, []byte("date_started"), []byte("start_time"), 1)

		err = cur.Delete()
		if err != nil {
			return err
		}

		err = bucket.Put(newKey, v)
		if err != nil {
			return err
		}
	}

	return nil
}

func (c *Client) migrate_v1_4_0(tx *bbolt.Tx) error {
	slog.Info(
		"running db migrations to v1.4.0 format",
	)

	err := migrateSessions_v1_4_0(tx)
	if err != nil {
		return err
	}

	return migrateTimers_v1_4_0(tx)
}
