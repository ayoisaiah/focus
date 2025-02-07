package timer

import (
	"time"

	"github.com/ayoisaiah/focus/internal/config"
	"github.com/ayoisaiah/focus/internal/models"
	"github.com/ayoisaiah/focus/internal/timeutil"
)

type (
	Timeline struct {
		// StartTime is the start of the session including
		// the start of a paused session
		StartTime time.Time `json:"start_time"`
		// EndTime is the end of a session including
		// when a session is paused or stopped prematurely
		EndTime time.Time `json:"end_time"`
	}

	// Session represents an active work or break session.
	Session struct {
		StartTime time.Time       `json:"start_time"`
		EndTime   time.Time       `json:"end_time"`
		Name      config.SessType `json:"name"`
		Tags      []string        `json:"tags"`
		Timeline  []Timeline      `json:"timeline"`
		Duration  time.Duration   `json:"duration"`
		Completed bool            `json:"completed"`
	}

	// Remainder is the time remaining in an active session.
	Remainder struct {
		T int // total
		M int // minutes
		S int // seconds
	}
)

// IsResuming determines if a session is being resumed or not.
func (s *Session) IsResuming() bool {
	if s.EndTime.IsZero() || s.Completed {
		return false
	}

	return true
}

func (s *Session) Adjust(since time.Time) {
	s.StartTime = since
	s.EndTime = s.StartTime.Add(s.Duration)

	s.Timeline[0].EndTime = s.EndTime
}

// SetEndTime calculates the end time for the current session.
func (s *Session) SetEndTime() {
	endTime := s.StartTime.Add(s.Duration)

	if !s.IsResuming() {
		s.EndTime = endTime
		lastIndex := len(s.Timeline) - 1
		s.Timeline[lastIndex].EndTime = endTime

		return
	}

	now := time.Now()

	elapsedTimeInSeconds := s.ElapsedTimeInSeconds()
	endTime = now.
		Add(s.Duration).
		Add(-time.Second * time.Duration(elapsedTimeInSeconds))

	s.EndTime = endTime

	s.Timeline = append(s.Timeline, Timeline{
		StartTime: now,
		EndTime:   endTime,
	})
}

// Remaining calculates the time remaining for the session to end.
func (s *Session) Remaining() Remainder {
	monotonicDiff := time.Until(s.EndTime)

	total := timeutil.Round(monotonicDiff.Seconds())

	if total < 0 {
		total = 0
	}

	minutes := total / 60
	seconds := total % 60

	return Remainder{
		T: total,
		M: minutes,
		S: seconds,
	}
}

// ElapsedTimeInSeconds returns the time elapsed for the current session
// in seconds using monotonic timings.
func (s *Session) ElapsedTimeInSeconds() float64 {
	var elapsedTimeInSeconds float64
	for _, v := range s.Timeline {
		elapsedTimeInSeconds += v.EndTime.Sub(v.StartTime).Seconds()
	}

	return elapsedTimeInSeconds
}

// RealElapsedTimeInSeconds returns the time elapsed for the current session
// in seconds using real timings.
func (s *Session) RealElapsedTimeInSeconds() float64 {
	var elapsedTimeInSeconds float64

	for _, v := range s.Timeline {
		start := v.StartTime.Round(0)
		end := v.EndTime.Round(0)

		elapsedTimeInSeconds += end.Sub(start).Seconds()
	}

	return elapsedTimeInSeconds
}

// UpdateEndTime sets the session end time to the current time.
func (s *Session) UpdateEndTime(isCompleted bool) {
	endTime := time.Now()
	s.EndTime = endTime
	s.Completed = isCompleted

	lastIndex := len(s.Timeline) - 1
	s.Timeline[lastIndex].EndTime = endTime
}

// Normalise ensures that the end time for the current session perfectly
// correlates with what is required to complete the session.
// It mostly helps with normalising the end time when the system is suspended
// with a session in progress, and resumed at a later time in the future
// that surpasses the normal end time.
func (s *Session) Normalise() {
	elapsed := s.ElapsedTimeInSeconds()
	realElapsed := s.RealElapsedTimeInSeconds()
	durationSecs := s.Duration.Seconds()
	diffSecs := realElapsed - elapsed

	// Normalize end time to precisely fulfill the session duration
	if s.Completed {
		// secondsBeforeLastPart is the number of seconds elapsed without including
		// the concluding part of the session timeline
		var secondsBeforeLastPart float64

		for i := 0; i < len(s.Timeline)-1; i++ {
			v := s.Timeline[i]
			secondsBeforeLastPart += v.EndTime.Sub(v.StartTime).Seconds()
		}

		lastIndex := len(s.Timeline) - 1
		lastPart := s.Timeline[lastIndex]

		secondsLeft := durationSecs - secondsBeforeLastPart
		end := lastPart.StartTime.Add(
			time.Duration(secondsLeft * float64(time.Second)),
		)

		s.Timeline[lastIndex].EndTime = end
		s.EndTime = end
		s.Completed = true
	} else if diffSecs > 1 {
		// For interrupted timers, normalize the end time when the
		// monotonic and real timings are significantly different, likely due to
		// suspending the computer while the timer was running.
		lastIndex := len(s.Timeline) - 1
		end := s.EndTime.Add(-time.Duration(diffSecs * float64(time.Second)))

		s.EndTime = end
		s.Timeline[lastIndex].EndTime = end
	}
}

// ToDBModel converts an active session to a database model.
func (s *Session) ToDBModel() *models.Session {
	sess := &models.Session{}

	sess.StartTime = s.StartTime
	sess.EndTime = s.EndTime
	sess.Name = s.Name
	sess.Tags = s.Tags
	sess.Duration = s.Duration
	sess.Completed = s.Completed

	for _, v := range s.Timeline {
		timeline := models.SessionTimeline{
			StartTime: v.StartTime,
			EndTime:   v.EndTime,
		}

		sess.Timeline = append(sess.Timeline, timeline)
	}

	return sess
}
