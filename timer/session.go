package timer

import (
	"time"

	"github.com/ayoisaiah/focus/config"
	"github.com/ayoisaiah/focus/internal/models"
	"github.com/ayoisaiah/focus/internal/timeutil"
)

type Timeline struct {
	// StartTime is the start of the session including
	// the start of a paused session
	StartTime time.Time `json:"start_time"`
	// EndTime is the end of a session including
	// when a session is paused or stopped prematurely
	EndTime time.Time `json:"end_time"`
}

// Session represents an active work or break session.
type Session struct {
	StartTime time.Time       `json:"start_time"`
	EndTime   time.Time       `json:"end_time"`
	Name      config.SessType `json:"name"`
	Tags      []string        `json:"tags"`
	Timeline  []Timeline      `json:"timeline"`
	Duration  time.Duration   `json:"duration"`
	Completed bool            `json:"completed"`
}

// Remainder is the time remaining in an active session.
type Remainder struct {
	T int // total
	M int // minutes
	S int // seconds
}

// Resuming determines if a session is being resumed or not.
func (s *Session) Resuming() bool {
	if s.StartTime.Equal(s.EndTime) || s.Completed {
		return false
	}

	return true
}

// SetEndTime calculates the end time for the session.
func (s *Session) SetEndTime() {
	endTime := s.StartTime.Add(s.Duration)

	if s.Resuming() {
		elapsedTimeInSeconds := s.ElapsedTimeInSeconds()
		endTime = time.Now().
			Add(s.Duration).
			Add(-time.Second * time.Duration(elapsedTimeInSeconds))
	}

	s.EndTime = endTime

	s.Timeline = append(s.Timeline, Timeline{
		StartTime: time.Now(),
		EndTime:   endTime,
	})
}

// Remaining calculates the remaining time  for the session to end.
func (s *Session) Remaining() Remainder {
	difference := time.Until(s.EndTime)
	total := timeutil.Round(difference.Seconds())

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

// getElapsedTimeInSeconds returns the time elapsed
// for the current session in seconds.
func (s *Session) ElapsedTimeInSeconds() float64 {
	var elapsedTimeInSeconds float64
	for _, v := range s.Timeline {
		elapsedTimeInSeconds += v.EndTime.Sub(v.StartTime).Seconds()
	}

	return elapsedTimeInSeconds
}

// Interrupt updates the end time for a session to the time it was interrupted.
func (s *Session) Interrupt() {
	interrruptedTime := time.Now()
	s.EndTime = interrruptedTime

	lastIndex := len(s.Timeline) - 1
	s.Timeline[lastIndex].EndTime = interrruptedTime
}

// Normalise ensures that the end time for the current session does not exceed
// what is required to complete the session. It mostly helps with normalising
// the end time when the system is hibernated with a session in progress, and
// resumed at a later time in the future that surpasses the normal end time.
func (s *Session) Normalise() {
	elapsed := s.ElapsedTimeInSeconds()

	// If the elapsed time is greater than the duration
	// of the session, the end time must be normalised
	// to a time that will fulfill the exact duration
	// of the session
	if elapsed > s.Duration.Seconds() {
		// secondsBeforeLastPart represents the number of seconds
		// elapsed without including the concluding part of the
		// session timeline
		var secondsBeforeLastPart float64

		for i := 0; i < len(s.Timeline)-1; i++ {
			v := s.Timeline[i]
			secondsBeforeLastPart += v.EndTime.Sub(v.StartTime).Seconds()
		}

		lastIndex := len(s.Timeline) - 1
		lastPart := s.Timeline[lastIndex]

		secondsLeft := s.Duration.Seconds() - secondsBeforeLastPart
		end := lastPart.StartTime.Add(
			time.Duration(secondsLeft * float64(time.Second)),
		)
		s.Timeline[lastIndex].EndTime = end
		s.EndTime = end
		s.Completed = true
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
