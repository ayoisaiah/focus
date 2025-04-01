package timer

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/lipgloss"

	"github.com/ayoisaiah/focus/internal/config"
	"github.com/ayoisaiah/focus/internal/timeutil"
)

// formatTimeRemaining returns the remaining time formatted as "MM:SS".
// Uses the timer's clock to calculate the remaining duration.
func (t *Timer) formatTimeRemaining() string {
	m, s := timeutil.SecsToMinsAndSecs(t.clock.Timeout.Seconds())

	return fmt.Sprintf(
		"%s:%s", fmt.Sprintf("%02d", m), fmt.Sprintf("%02d", s),
	)
}

func (t *Timer) sessionPromptView() string {
	var s strings.Builder

	title := "Your focus session is complete"
	msg := "It's time to take a well-deserved break!"

	if t.Current.Name == config.Work {
		title = "Your break is over"
		msg = "Time to refocus and get back to work!"
	}

	s.WriteString(
		lipgloss.NewStyle().
			Foreground(lipgloss.Color("#DB2763")).
			SetString(title).
			String(),
	)
	s.WriteString("\n\n" + msg)

	return s.String()
}

func (t *Timer) timerView() string {
	var s strings.Builder

	percent := (float64(
		t.clock.Timeout.Seconds(),
	) / float64(
		t.Current.Duration.Seconds(),
	))

	timeRemaining := t.formatTimeRemaining()

	switch t.Current.Name {
	case config.Work:
		s.WriteString(defaultStyle.work.Render())
	case config.ShortBreak:
		s.WriteString(defaultStyle.shortBreak.Render())
	case config.LongBreak:
		s.WriteString(defaultStyle.longBreak.Render())
	}

	var timeFormat string
	if t.Opts.Settings.TwentyFourHour {
		timeFormat = "15:04:05"
	} else {
		timeFormat = "03:04:05 PM"
	}

	if !t.clock.Running() && !t.clock.Timedout() {
		s.WriteString(
			lipgloss.NewStyle().
				Foreground(lipgloss.Color("#DB2763")).
				SetString("[Paused]").
				String(),
		)
	} else {
		s.WriteString(
			strings.TrimSpace(
				defaultStyle.help.SetString("until " + t.Current.EndTime.Format(timeFormat)).String()),
		)
	}

	if t.Current.Name == config.Work {
		s.WriteString(
			strings.TrimSpace(
				defaultStyle.help.SetString(
					fmt.Sprintf(
						" (%d/%d)",
						t.WorkCycle,
						t.Opts.Settings.LongBreakInterval,
					),
				).String()))
	}

	s.WriteString("\n\n")
	s.WriteString(timeRemaining)
	s.WriteString("\n\n")
	s.WriteString(t.progress.ViewAs(float64(1 - percent)))
	s.WriteString("\n")
	s.WriteString(t.helpView())

	return s.String()
}

func (t *Timer) pickSoundView() string {
	if t.soundForm.State == huh.StateCompleted {
		sound := t.soundForm.GetString("sound")
		t.Opts.Settings.AmbientSound = sound
		t.settings = ""

		err := t.setAmbientSound()
		if err != nil {
			return err.Error()
		}

		return ""
	}

	return t.soundForm.View()
}

func (t *Timer) settingsView() string {
	if t.settings == soundView {
		return t.pickSoundView()
	}

	return ""
}

func (t *Timer) helpView() string {
	if t.waitForNextSession {
		return "\n" + t.help.ShortHelpView([]key.Binding{
			defaultKeymap.enter,
			defaultKeymap.quit,
		})
	}

	if t.Current.Name == config.Work {
		return "\n" + t.help.ShortHelpView([]key.Binding{
			defaultKeymap.togglePlay,
			defaultKeymap.sound,
			defaultKeymap.quit,
		})
	}

	return "\n" + t.help.ShortHelpView([]key.Binding{
		defaultKeymap.esc,
		defaultKeymap.quit,
	})
}

func (t *Timer) View() string {
	if t.waitForNextSession {
		return defaultStyle.base.Render(
			t.sessionPromptView(),
			"\n",
			t.helpView(),
		)
	}

	if t.clock.Timedout() || t.Current.Completed {
		return ""
	}

	view := t.timerView()

	if t.settings != "" {
		view += "\n\n" + t.settingsView()
	}

	return defaultStyle.base.Render(view)
}
