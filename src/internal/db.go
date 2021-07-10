package focus

import (
	"io/fs"
	"os"
	"path/filepath"
	"time"

	bolt "go.etcd.io/bbolt"
)

const (
	dbFile = "focus.db"
)

type Store struct {
	conn *bolt.DB
}

func (s *Store) init() error {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return err
	}

	appRoot := filepath.Join(homeDir, configPath)
	pathToDB := filepath.Join(appRoot, dbFile)

	var fileMode fs.FileMode = 0600

	db, err := bolt.Open(pathToDB, fileMode, &bolt.Options{Timeout: 1 * time.Second})
	if err != nil {
		return err
	}

	s.conn = db

	err = s.conn.Update(func(tx *bolt.Tx) error {
		_, err = tx.CreateBucketIfNotExists([]byte("events"))
		return err
	})

	return err
}

func (s *Store) updateEvent(key, value []byte) error {
	err := s.conn.Update(func(tx *bolt.Tx) error {
		return tx.Bucket([]byte("events")).Put(key, value)
	})

	return err
}
