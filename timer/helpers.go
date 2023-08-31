package timer

import (
	"bufio"
	"cmp"
	"encoding/json"
	"fmt"
	"os"
	"slices"
	"strconv"
	"strings"

	"github.com/ayoisaiah/focus/internal/timeutil"
	"github.com/ayoisaiah/focus/internal/ui"
	"github.com/ayoisaiah/focus/session"
	"github.com/ayoisaiah/focus/store"
)

// printPausedTimers outputs a list of resumable timers.
func printPausedTimers(timers []Timer, pausedSess map[string]session.Session) {
	tableBody := make([][]string, len(timers))

	for i := range timers {
		t := timers[i]

		sess := pausedSess[string(timeutil.ToKey(t.SessionKey))]

		sess.SetEndTime()

		r := sess.Remaining()

		cycle := fmt.Sprintf("%d/%d", t.WorkCycle, t.Opts.LongBreakInterval)

		remainder := fmt.Sprintf("%s -> completed", cycle)
		if r.T > 0 {
			remainder = fmt.Sprintf("%s -> %02d:%02d", cycle, r.M, r.S)
		}

		row := []string{
			fmt.Sprintf("%d", i+1),
			t.PausedTime.Format("Jan 02, 2006 03:04:05 PM"),
			t.Started.Format("Jan 02, 2006 03:04:05 PM"),
			remainder,
			strings.Join(t.Opts.Tags, ", "),
		}

		tableBody[i] = row
	}

	tableBody = append([][]string{
		{
			"#",
			"DATE PAUSED",
			"DATE STARTED",
			"CYCLE",
			"TAGS",
		},
	}, tableBody...)

	ui.PrintTable(tableBody, os.Stdout)
}

// selectPausedTimer prompts the user to select from a list of resumable
// timers.
func selectPausedTimer(
	timers []Timer,
) (*Timer, error) {
	reader := bufio.NewReader(os.Stdin)

	fmt.Fprint(os.Stdout, "\033[s")
	fmt.Fprint(os.Stdout, "Type a number and press ENTER: ")

	input, err := reader.ReadString('\n')
	if err != nil {
		return nil, err
	}

	num, err := strconv.Atoi(strings.TrimSpace(input))
	if err != nil {
		return nil, err
	}

	index := num - 1
	if index >= len(timers) {
		return nil, fmt.Errorf("%d is not associated with a session", num)
	}

	return &timers[index], nil
}

// getTimerSessions retrieves all paused timers and their corresponding sessions.
func getTimerSessions(
	db store.DB,
) ([]Timer, map[string]session.Session, error) {
	timers, err := db.RetrievePausedTimers()
	if err != nil {
		return nil, nil, err
	}

	pausedTimers := make([]Timer, len(timers))

	for i := range timers {
		var t Timer

		err := json.Unmarshal(timers[i], &t)
		if err != nil {
			return nil, nil, err
		}

		pausedTimers[i] = t
	}

	pausedSessions := make(map[string]session.Session)

	for _, v := range pausedTimers {
		sessBytes, dbErr := db.GetSession(v.SessionKey)
		if dbErr != nil {
			return nil, nil, dbErr
		}

		sess := &session.Session{}

		err := json.Unmarshal(sessBytes, sess)
		if err != nil {
			return nil, nil, err
		}

		pausedSessions[string(timeutil.ToKey(v.SessionKey))] = *sess
	}

	slices.SortStableFunc(pausedTimers, func(a, b Timer) int {
		return cmp.Compare(b.PausedTime.UnixNano(), a.PausedTime.UnixNano())
	})

	return pausedTimers, pausedSessions, nil
}

// selectAndDeleteTimers prompts the user and deletes the selected timers or
// all timers if 0 is specified.
func selectAndDeleteTimers(db store.DB, timers []Timer) error {
	reader := bufio.NewReader(os.Stdin)

	fmt.Fprint(os.Stdout, "\033[s")
	fmt.Fprint(os.Stdout, "Select the timers to delete and press ENTER: ")

	input, err := reader.ReadString('\n')
	if err != nil {
		return err
	}

	input = strings.TrimSpace(input)

	if input == "" {
		return errInvalidInput
	}

	indices := strings.Split(input, ",")

	for _, v := range indices {
		num, err := strconv.Atoi(v)
		if err != nil {
			continue
		}

		num--

		if len(timers) <= num || num < -1 {
			continue
		}

		if num == -1 {
			return db.DeleteAllTimers()
		}

		err = db.DeleteTimer(timers[num].Started)
		if err != nil {
			return err
		}
	}

	return nil
}
