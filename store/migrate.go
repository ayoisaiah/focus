package store

import (
	"bytes"
	"encoding/json"
	"time"

	"go.etcd.io/bbolt"
)

func migrateSessions(tx *bbolt.Tx) error {
	bucket := tx.Bucket([]byte(sessionBucket))

	cur := bucket.Cursor()

	for k, v := cur.First(); k != nil; k, v = cur.Next() {
		var s session

		err := json.Unmarshal(v, &s)
		if err != nil {
			return err
		}

		newKey := []byte(s.StartTime.Format(time.RFC3339Nano))

		err = bucket.Put(newKey, v)
		if err != nil {
			return err
		}

		err = cur.Delete()
		if err != nil {
			return err
		}
	}

	return nil
}

func migrateTimers(tx *bbolt.Tx) error {
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

		err = bucket.Put(newKey, v)
		if err != nil {
			return err
		}

		err = cur.Delete()
		if err != nil {
			return err
		}
	}

	return nil
}

func (c *Client) migrate(tx *bbolt.Tx) error {
	err := migrateSessions(tx)
	if err != nil {
		return err
	}

	return migrateTimers(tx)
}
