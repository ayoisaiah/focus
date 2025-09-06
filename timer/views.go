package timer

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/lipgloss"

	"github.com/ayoisaiah/focus/internal/config"
	"github.com/ayoisaiah/focus/internal/timeutil"
)

// formatTimeRemaining returns the remaining time formatted as "MM:SS".
// Uses the timer's clock to calculate the remaining duration.
func (t *Timer) formatTimeRemaining() string {
	if t.flowMode {
		// In flow mode, show elapsed time
		m, s := timeutil.SecsToMinsAndSecs(t.elapsedTime.Seconds())
		return fmt.Sprintf(
			"%s:%s", fmt.Sprintf("%02d", m), fmt.Sprintf("%02d", s),
		)
	}
	
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

	if t.flowMode {
		// Flow mode display
		if t.taskName != "" {
			s.WriteString(
				lipgloss.NewStyle().
					Foreground(lipgloss.Color(t.Opts.Work.Color)).
					MarginRight(1).
					SetString("📝 " + t.taskName).
					String(),
			)
			s.WriteString("\n")
		}
		
		// Show elapsed vs estimated time
		timeDisplay := t.formatTimeRemaining()
		estimatedM, estimatedS := timeutil.SecsToMinsAndSecs(t.estimatedTime.Seconds())
		estimatedDisplay := fmt.Sprintf("%02d:%02d", estimatedM, estimatedS)
		
		// Check if we're in overtime
		isOvertime := t.elapsedTime > t.estimatedTime
		
		s.WriteString(
			lipgloss.NewStyle().
				Foreground(lipgloss.Color("#888")).
				SetString("Elapsed / Estimated").
				String(),
		)
		s.WriteString("\n\n")
		
		// Color the elapsed time red if in overtime
		if isOvertime {
			s.WriteString(
				lipgloss.NewStyle().
					Foreground(lipgloss.Color("#FF4444")).
					SetString(timeDisplay).
					String(),
			)
		} else {
			s.WriteString(timeDisplay)
		}
		s.WriteString(" / " + estimatedDisplay)
		
		// Progress bar based on estimated time
		percent := t.elapsedTime.Seconds() / t.estimatedTime.Seconds()
		if percent > 1 {
			percent = 1
		}
		
		s.WriteString("\n\n")
		s.WriteString(t.progress.ViewAs(percent))
		s.WriteString("\n")
		s.WriteString(t.helpView())
		
		return s.String()
	}

	// Regular timer mode
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
	if !t.showingSoundMenu {
		return ""
	}
	
	// Styles for the sound menu
	titleStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#FAFAFA")).
		Background(lipgloss.Color("#7D56F4")).
		Padding(0, 1).
		MarginBottom(1)
	
	selectedStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#EE6FF8")).
		Bold(true)
		
	unselectedStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#626262"))
	
	helpStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#626262")).
		MarginTop(1)
	
	var items []string
	
	// Title
	items = append(items, titleStyle.Render("Select ambient sound"))
	items = append(items, "")
	
	// Options
	for i, option := range t.soundOptions {
		if i == t.selectedSoundIndex {
			items = append(items, selectedStyle.Render("› "+option))
		} else {
			items = append(items, unselectedStyle.Render("  "+option))
		}
	}
	
	items = append(items, "")
	items = append(items, helpStyle.Render("↑/↓: navigate • enter: select • esc: cancel"))
	
	// Add border around the whole menu
	menuStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#874BFD")).
		Padding(1, 2).
		MarginTop(1)
	
	return menuStyle.Render(strings.Join(items, "\n"))
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
		return "\n" + t.customHelpView()
	}

	return "\n" + t.help.ShortHelpView([]key.Binding{
		defaultKeymap.esc,
		defaultKeymap.quit,
	})
}

func (t *Timer) customHelpView() string {
	var parts []string
	neonGreen := lipgloss.Color("#39FF14") // Bright neon green
	dimStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#666666")) // Dim gray like original
	
	// Play/pause button - show current state with neon green highlighting
	if t.clock.Running() {
		// Timer is running, show "play" highlighted and "pause" normal
		playText := lipgloss.NewStyle().Foreground(neonGreen).Render("play")
		playPauseText := dimStyle.Render("p ") + playText + dimStyle.Render("/pause")
		parts = append(parts, playPauseText)
	} else {
		// Timer is paused, show "pause" highlighted and "play" normal
		pauseText := lipgloss.NewStyle().Foreground(neonGreen).Render("pause")
		playPauseText := dimStyle.Render("p play/") + pauseText
		parts = append(parts, playPauseText)
	}
	
	// Sound button - highlight in neon green when ambient sound is on
	if t.Opts.Settings.AmbientSound != "" && t.Opts.Settings.AmbientSound != "off" {
		soundText := lipgloss.NewStyle().Foreground(neonGreen).Render("s sound")
		parts = append(parts, soundText)
	} else {
		parts = append(parts, dimStyle.Render("s sound"))
	}
	
	// Quit button - always dim
	parts = append(parts, dimStyle.Render("q quit"))
	
	return strings.Join(parts, dimStyle.Render(" • "))
}

func (t *Timer) View() string {
	// Handle flow mode prompt
	if t.settings == "flowPrompt" && t.soundForm != nil {
		return defaultStyle.base.Render(t.soundForm.View())
	}
	
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

	if t.settings != "" && t.settings != "flowPrompt" {
		view += "\n\n" + t.settingsView()
	}

	return defaultStyle.base.Render(view)
}
