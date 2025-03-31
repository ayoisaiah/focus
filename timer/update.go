package timer

import (
	"log/slog"
	"time"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/progress"
	btimer "github.com/charmbracelet/bubbles/timer"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/huh"
	"github.com/davecgh/go-spew/spew"
	"github.com/gopxl/beep/v2/speaker"

	"github.com/ayoisaiah/focus/internal/config"
)

// handleTimerTick processes timer tick events.
func (t *Timer) handleTimerTick(msg btimer.TickMsg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	t.clock, cmd = t.clock.Update(msg)

	_ = t.writeStatusFile()

	return t, cmd
}

// handleTimerStartStop manages timer start/stop events.
func (t *Timer) handleTimerStartStop(
	msg btimer.StartStopMsg,
) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	t.clock, cmd = t.clock.Update(msg)

	if t.clock.Running() {
		t.StartTime = time.Now()
		t.Current.SetEndTime()
	} else {
		_ = t.persist()
	}

	if t.SoundStream != nil {
		if !t.clock.Running() {
			_ = speaker.Suspend()
		} else {
			_ = speaker.Resume()
		}
	}

	return t, cmd
}

func (t *Timer) handleKeyPress(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	if t.soundForm != nil {
		form, cmd := t.soundForm.Update(msg)
		if f, ok := form.(*huh.Form); ok {
			t.soundForm = f
			return t, cmd
		}
	}

	return t, cmd
}

func (t *Timer) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case btimer.TickMsg:
		return t.handleTimerTick(msg)

	case btimer.StartStopMsg:
		return t.handleTimerStartStop(msg)

	case btimer.TimeoutMsg:
		_ = t.persist()

		_ = t.postSession()

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
				t.soundForm = huh.NewForm(
					huh.NewGroup(
						huh.NewSelect[string]().
							Key("sound").
							Options(huh.NewOptions(soundOpts...)...).
							Title("Select ambient sound"),
					),
				)
				t.settings = soundView

				return t, t.soundForm.Init()
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

			// TODO: Check strict mode

			cmd = t.clock.Toggle()

			return t, cmd

		case key.Matches(msg, defaultKeymap.quit):
			_ = t.persist()

			return t, tea.Batch(tea.ClearScreen, tea.Quit)
		}

		// return t.handleKeyPress(msg)

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

	if t.soundForm != nil {
		slog.Info(spew.Sdump(msg))

		form, cmd := t.soundForm.Update(msg)
		if f, ok := form.(*huh.Form); ok {
			t.soundForm = f
			return t, cmd
		}
	}

	return t, nil
}
