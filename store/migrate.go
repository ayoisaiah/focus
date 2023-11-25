package store

import (
	"encoding/json"
	"log/slog"
	"time"

	"go.etcd.io/bbolt"

	"github.com/ayoisaiah/focus/internal/models"
)

// Change session key to RFC3339Nano and update duration to nanoseconds
func migrateSessions_v1_4_0(tx *bbolt.Tx) error {
	bucket := tx.Bucket([]byte(sessionBucket))

	cur := bucket.Cursor()

	for k, v := cur.First(); k != nil; k, v = cur.Next() {
		var s models.Session

		err := json.Unmarshal(v, &s)
		if err != nil {
			return err
		}

		// s.Duration was in minutes, but must now be changed to nanoseconds
		s.Duration = time.Duration(s.Duration) * time.Minute

		newKey := []byte(s.StartTime.Format(time.RFC3339Nano))

		err = cur.Delete()
		if err != nil {
			return err
		}

		b, err := json.Marshal(s)
		if err != nil {
			return err
		}

		err = bucket.Put(newKey, b)
		if err != nil {
			return err
		}
	}

	return nil
}

// Delete all exisiting timers as it won't be possible to resume paused sessions
// after migrating the sessions
func migrateTimers_v1_4_0(tx *bbolt.Tx) error {
	bucket := tx.Bucket([]byte(timerBucket))

	cur := bucket.Cursor()

	for k, _ := cur.First(); k != nil; k, _ = cur.Next() {
		err := cur.Delete()
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
