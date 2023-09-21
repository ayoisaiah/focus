package timer

import (
	"bufio"
	"cmp"
	"fmt"
	"os"
	"slices"
	"strconv"
	"strings"

	"github.com/ayoisaiah/focus/internal/models"
	"github.com/ayoisaiah/focus/internal/timeutil"
	"github.com/ayoisaiah/focus/internal/ui"
	"github.com/ayoisaiah/focus/store"
)

// printPausedTimers outputs a list of resumable timers.
func printPausedTimers(
	timers []*models.Timer,
	pausedSess map[string]*models.Session,
) {
	tableBody := make([][]string, len(timers))

	for i := range timers {
		t := timers[i]

		s := pausedSess[string(timeutil.ToKey(t.SessionKey))]

		sess := newSessionFromDB(s)

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
			t.StartTime.Format("Jan 02, 2006 03:04:05 PM"),
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
	timers []*models.Timer,
) (*models.Timer, error) {
	reader := bufio.NewReader(os.Stdin)

	fmt.Fprint(os.Stdout, "\033[s")
	fmt.Fprint(os.Stdout, "Type a number and press ENTER: ")

	input, err := reader.ReadString('\n')
	if err != nil {
		return nil, err
	}

	input = strings.TrimSpace(input)

	if input == "" {
		os.Exit(0)
	}

	num, err := strconv.Atoi(input)
	if err != nil {
		return nil, err
	}

	index := num - 1
	if index >= len(timers) {
		return nil, fmt.Errorf("%d is not associated with a session", num)
	}

	return timers[index], nil
}

// getTimerSessions retrieves all paused timers and their corresponding sessions.
func getTimerSessions(
	db store.DB,
) ([]*models.Timer, map[string]*models.Session, error) {
	pausedTimers, err := db.RetrievePausedTimers()
	if err != nil {
		return nil, nil, err
	}

	pausedSessions := make(map[string]*models.Session)

	for i := range pausedTimers {
		v := pausedTimers[i]

		s, err := db.GetSession(v.SessionKey)
		if err != nil {
			return nil, nil, err
		}

		pausedSessions[string(timeutil.ToKey(v.SessionKey))] = s
	}

	slices.SortStableFunc(pausedTimers, func(a, b *models.Timer) int {
		return cmp.Compare(b.PausedTime.UnixNano(), a.PausedTime.UnixNano())
	})

	return pausedTimers, pausedSessions, nil
}

// selectAndDeleteTimers prompts the user and deletes the selected timers or
// all timers if 0 is specified.
func selectAndDeleteTimers(db store.DB, timers []*models.Timer) error {
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

		err = db.DeleteTimer(timers[num].StartTime)
		if err != nil {
			return err
		}
	}

	return nil
}
