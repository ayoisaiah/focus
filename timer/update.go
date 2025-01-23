package timer

import (
	"time"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/progress"
	btimer "github.com/charmbracelet/bubbles/timer"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/huh"

	"github.com/ayoisaiah/focus/config"
)

func (t *Timer) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case btimer.TickMsg:
		t.clock, cmd = t.clock.Update(msg)

		// Persist timer every 60 seconds to aid recovery
		if int(t.clock.Timeout.Seconds())%60 == 0 {
			go func() {
				_ = t.Persist()
			}()
		}

		return t, cmd

	case btimer.StartStopMsg:
		t.clock, cmd = t.clock.Update(msg)

		if t.clock.Running() {
			t.StartTime = time.Now()
			t.Current.SetEndTime()
		} else {
			_ = t.Persist()
		}

		return t, cmd

	case btimer.TimeoutMsg:
		_ = t.Persist()

		cmd = t.initSession()

		return t, cmd

	case tea.KeyMsg:
		switch {
		case key.Matches(msg, defaultKeymap.enter):
			if t.settings != "" {
				break
			}

			t.waitForNextSession = false
			t.clock = btimer.New(t.Current.Duration)
			cmd = t.clock.Init()

			return t, cmd

		case key.Matches(msg, defaultKeymap.sound):
			if !t.clock.Timedout() {
				t.settings = soundView
			}

			return t, nil

		case key.Matches(msg, defaultKeymap.esc):
			// Skip break sessions
			if t.Current.Name != config.Work && t.clock.Running() {
				return t, tea.Batch(t.clock.Stop(), t.initSession())
			}

			t.settings = ""

			return t, nil

		case key.Matches(msg, defaultKeymap.togglePlay):
			if t.Current.Name != config.Work {
				return t, nil
			}

			cmd = t.clock.Toggle()

			return t, cmd

		case key.Matches(msg, defaultKeymap.quit):
			_ = t.Persist()

			return t, tea.Batch(tea.ClearScreen, tea.Quit)
		}

	case tea.WindowSizeMsg:
		t.progress.Width = msg.Width - padding*2 - 4
		if t.progress.Width > maxWidth {
			t.progress.Width = maxWidth
		}

		return t, nil

		// FrameMsg is sent when the progress bar wants to animate itself
	case progress.FrameMsg:
		var progressModel tea.Model

		progressModel, cmd = t.progress.Update(msg)
		t.progress, _ = progressModel.(progress.Model)

		return t, cmd
	}

	form, cmd := t.soundForm.Update(msg)
	if f, ok := form.(*huh.Form); ok {
		t.soundForm = f
		return t, cmd
	}

	return t, nil
}
