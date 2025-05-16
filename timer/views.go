package timer

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/key"

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

	if t.CurrentSess.Name == config.Work {
		title = "Your break is over"
		msg = "It's time to refocus and get back to work!"
	}

	s.WriteString(t.Opts.Style.Main.SetString(title).String())
	s.WriteString("\n\n" + t.Opts.Style.Secondary.SetString(msg).String())
	s.WriteString("\n\n" + t.help.ShortHelpView([]key.Binding{
		defaultKeymap.enter,
		defaultKeymap.quit,
	}),
	)

	return s.String()
}

func (t *Timer) timerView() string {
	var s strings.Builder

	switch t.CurrentSess.Name {
	case config.Work:
		s.WriteString(t.Opts.Style.Work.Render())
	case config.ShortBreak:
		s.WriteString(t.Opts.Style.ShortBreak.Render())
	case config.LongBreak:
		s.WriteString(t.Opts.Style.LongBreak.Render())
	}

	var timeFormat string
	if t.Opts.Settings.TwentyFourHour {
		timeFormat = "15:04:05"
	} else {
		timeFormat = "03:04:05 PM"
	}

	if !t.clock.Running() && !t.clock.Timedout() {
		s.WriteString(t.Opts.Style.Secondary.SetString("[Paused]").String())
	} else {
		s.WriteString(
			strings.TrimSpace(
				t.Opts.Style.Hint.SetString("until " + t.CurrentSess.EndTime.Format(timeFormat)).String()),
		)
	}

	if t.CurrentSess.Name == config.Work {
		s.WriteString(
			strings.TrimSpace(
				t.Opts.Style.Hint.SetString(
					fmt.Sprintf(
						" (%d/%d)",
						t.WorkCycle,
						t.Opts.Settings.LongBreakInterval,
					),
				).String()))
	}

	percent := (float64(
		t.clock.Timeout.Seconds(),
	) / float64(
		t.CurrentSess.Duration.Seconds(),
	))

	timeRemaining := t.formatTimeRemaining()

	s.WriteString("\n\n")
	s.WriteString(t.Opts.Style.Main.SetString(timeRemaining).String())
	s.WriteString("\n\n")
	s.WriteString(t.progress.ViewAs(float64(1 - percent)))
	s.WriteString(t.sessionHelpView())

	return s.String()
}

func (t *Timer) settingsView(view string) string {
	if t.settings == soundView {
		return view + "\n\n" + t.soundForm.View()
	}

	return view
}

func (t *Timer) sessionHelpView() string {
	if t.CurrentSess.Name == config.Work {
		return "\n\n" + t.help.ShortHelpView([]key.Binding{
			defaultKeymap.togglePlay,
			defaultKeymap.sound,
			defaultKeymap.quit,
		})
	}

	return "\n\n" + t.help.ShortHelpView([]key.Binding{
		defaultKeymap.esc,
		defaultKeymap.sound,
		defaultKeymap.quit,
	})
}

func (t *Timer) View() string {
	if t.waitForNextSession {
		return t.Opts.Style.Base.Render(t.sessionPromptView())
	}

	if t.clock.Timedout() || t.CurrentSess.Completed {
		return ""
	}

	view := t.settingsView(t.timerView())

	return t.Opts.Style.Base.Render(view)
}
