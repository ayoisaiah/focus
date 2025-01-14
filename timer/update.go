package timer

import (
	"time"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/progress"
	btimer "github.com/charmbracelet/bubbles/timer"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/huh"
)

func (t *Timer) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case btimer.TickMsg:
		t.clock, cmd = t.clock.Update(msg)
		t.update()

		return t, cmd

	case btimer.StartStopMsg:
		t.clock, cmd = t.clock.Update(msg)

		if t.clock.Running() {
			t.StartTime = time.Now()
			t.Current.SetEndTime()
		} else {
			t.Current.UpdateEndTime()
			_ = t.Persist(t.Context, t.Current)
		}

		return t, cmd

	case btimer.TimeoutMsg:
		_ = t.endSession()

		if !t.waitForNextSession {
			cmd = t.Init()
		}

		return t, cmd

	case tea.KeyMsg:
		switch {
		case key.Matches(msg, defaultKeymap.beginSess):
			if t.settings != "" {
				break
			}

			t.waitForNextSession = false
			cmd = t.Init()

			return t, cmd

		case key.Matches(msg, defaultKeymap.sound):
			t.settings = soundView
			return t, nil

		case key.Matches(msg, defaultKeymap.esc):
			t.settings = ""
			return t, nil

		case key.Matches(msg, defaultKeymap.togglePlay):
			cmd = t.clock.Toggle()
			return t, cmd

		case key.Matches(msg, defaultKeymap.quit):
			t.Current.UpdateEndTime()
			_ = t.Persist(t.Context, t.Current)
			return t, tea.Quit
		}

	case tea.WindowSizeMsg:
		t.progress.Width = msg.Width - padding*2 - 4
		if t.progress.Width > maxWidth {
			t.progress.Width = maxWidth
		}

		return t, nil

		// FrameMsg is sent when the progress bar wants to animate itself
	case progress.FrameMsg:
		progressModel, cmd := t.progress.Update(msg)
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
