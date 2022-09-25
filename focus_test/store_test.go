package focus_test

import (
	"encoding/json"
	"os"
	"time"
)

type DBMock struct{}

func (d *DBMock) init() error {
	return nil
}

func (d *DBMock) getSessions(
	startTime, endTime time.Time,
	tags []string,
) ([][]byte, error) {
	jsonFile, err := os.ReadFile("../../testdata/work.json")
	if err != nil {
		return nil, err
	}

	var sessions []session

	err = json.Unmarshal(jsonFile, &sessions)
	if err != nil {
		return nil, err
	}

	result := [][]byte{}

	for _, v := range sessions {
		if v.StartTime.Before(startTime) || v.StartTime.After(endTime) {
			continue
		}

		b, err := json.Marshal(v)
		if err != nil {
			return nil, err
		}

		result = append(result, b)
	}

	return result, nil
}

func (d *DBMock) deleteTimerState() error {
	return nil
}

func (d *DBMock) getTimerState() (timer, session []byte, err error) {
	return nil, nil, nil
}

func (d *DBMock) saveTimerState(timer, sessionKey []byte) error {
	return nil
}

func (d *DBMock) updateSession(key, value []byte) error {
	return nil
}

func (d *DBMock) deleteSessions(sessions []session) error {
	return nil
}

func (d *DBMock) close() error {
	return nil
}
