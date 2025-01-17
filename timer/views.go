package timer

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/lipgloss"

	"github.com/ayoisaiah/focus/config"
)

func (t *Timer) sessionPromptView() string {
	var s strings.Builder

	title := "Your focus session is complete"
	msg := "It's time to take a well-deserved break!"

	if t.Current.Name != config.Work {
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
	s.WriteString(defaultStyle.help.Render("press ENTER to continue.\n"))

	return defaultStyle.base.Render(s.String())
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
	if t.Opts.TwentyFourHourClock {
		timeFormat = "15:04:05"
	} else {
		timeFormat = "03:04:05 PM"
	}

	if !t.clock.Running() {
		s.WriteString(
			lipgloss.NewStyle().
				Foreground(lipgloss.Color("#DB2763")).
				SetString("[Paused]").
				String(),
		)
	} else {
		s.WriteString(
			strings.TrimSpace(
				defaultStyle.help.SetString(fmt.Sprintf("(until %s)", t.Current.EndTime.Format(timeFormat))).String()),
		)
	}

	s.WriteString("\n\n")
	s.WriteString(timeRemaining)
	s.WriteString("\n\n")
	s.WriteString(t.progress.ViewAs(float64(1 - percent)))
	s.WriteString("\n")
	s.WriteString(t.helpView())

	return defaultStyle.base.Render(s.String())
}

func (t *Timer) pickSoundView() string {
	if t.soundForm.State == huh.StateCompleted {
		sound := t.soundForm.GetString("sound")
		t.Opts.AmbientSound = sound
		t.settings = ""

		err := t.setAmbientSound()
		if err != nil {
			return err.Error()
		}

		return ""
	}

	return defaultStyle.base.Render(t.soundForm.View())
}

func (t *Timer) settingsView() string {
	if t.settings == soundView {
		return t.pickSoundView()
	}

	return ""
}

func (t *Timer) View() string {
	if t.waitForNextSession {
		return t.sessionPromptView()
	}

	str := t.timerView()

	if t.settings != "" {
		str += "\n" + t.settingsView()
	}

	return str
}

func (t *Timer) helpView() string {
	return "\n" + t.help.ShortHelpView([]key.Binding{
		defaultKeymap.togglePlay,
		defaultKeymap.sound,
		defaultKeymap.quit,
	})
}
