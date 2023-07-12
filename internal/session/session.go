// Package session defines focus sessions
package session

import "time"

// Name represents the session name.
type Name string

const (
	Work       Name = "Work session"
	ShortBreak Name = "Short break"
	LongBreak  Name = "Long break"
)

// Message maps a session to a message.
type Message map[Name]string

// Duration maps a session to time duration value.
type Duration map[Name]time.Duration

type Timeline struct {
	// StartTime is the start of the session including
	// the start of a paused session
	StartTime time.Time `json:"start_time"`
	// EndTime is the end of a session including
	// when a session is paused or stopped prematurely
	EndTime time.Time `json:"end_time"`
}

// Session represents a work or break session.
type Session struct {
	StartTime time.Time     `json:"start_time"`
	EndTime   time.Time     `json:"end_time"`
	Name      Name          `json:"name"`
	Tags      []string      `json:"tags"`
	Timeline  []Timeline    `json:"timeline"`
	Duration  time.Duration `json:"duration"` // minutes
	Completed bool          `json:"completed"`
}

// Resuming determines if a session is being resumed or not.
func (s *Session) Resuming() bool {
	if s.StartTime.Equal(s.EndTime) || s.Completed {
		return false
	}

	return true
}

// getElapsedTimeInSeconds returns the time elapsed
// for the current session in seconds.
func (s *Session) GetElapsedTimeInSeconds() float64 {
	var elapsedTimeInSeconds float64
	for _, v := range s.Timeline {
		elapsedTimeInSeconds += v.EndTime.Sub(v.StartTime).Seconds()
	}

	return elapsedTimeInSeconds
}

// Normalise ensures that the end time for the current session does not exceed
// what is required to complete the session. It mostly helps with normalising
// the end time when the system is hibernated with a session in progress, and
// resumed at a later time in the future that surpasses the normal end time.
func (s *Session) Normalise() {
	elapsed := s.GetElapsedTimeInSeconds()

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
