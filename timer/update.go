package timer

import (
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/progress"
	btimer "github.com/charmbracelet/bubbles/timer"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/huh"
	"github.com/gopxl/beep/v2/speaker"

	"github.com/ayoisaiah/focus/internal/config"
)

// handleTimerTick processes timer tick events.
func (t *Timer) handleTimerTick(msg btimer.TickMsg) (cmd tea.Cmd) {
	t.clock, cmd = t.clock.Update(msg)

	go func() {
		_ = t.writeStatusFile()
	}()

	return cmd
}

// handleTimerStartStop manages timer start/stop events.
func (t *Timer) handleTimerStartStop(
	msg btimer.StartStopMsg,
) (cmd tea.Cmd) {
	t.clock, cmd = t.clock.Update(msg)

	// Perform side effects
	if t.clock.Running() {
		t.CurrentSess.SetEndTime()

		if t.SoundStream != nil {
			_ = speaker.Resume()
		}
	} else {
		_ = t.persist()

		if t.SoundStream != nil {
			_ = speaker.Suspend()
		}
	}

	return cmd
}

// handleKeyPress handles key presses.
func (t *Timer) handleKeyPress(msg tea.KeyMsg) (cmd tea.Cmd) {
	switch {
	case key.Matches(msg, defaultKeymap.enter):
		if t.waitForNextSession {
			t.waitForNextSession = false

			return t.initSession()
		}

	case key.Matches(msg, defaultKeymap.sound):
		if !t.clock.Timedout() {
			t.soundForm = huh.NewForm(
				huh.NewGroup(
					huh.NewSelect[string]().
						Key("sound").
						Options(huh.NewOptions(config.SoundOpts()...)...).
						Title("Select ambient sound"),
				),
			)
			t.settings = soundView

			return t.soundForm.Init()
		}

	case key.Matches(msg, defaultKeymap.esc):
		if t.settings != "" {
			t.settings = ""
			return nil
		}

		// Skip break sessions
		if t.CurrentSess.Name != config.Work && t.clock.Running() {
			return tea.Batch(t.clock.Stop(), t.initSession())
		}

	case key.Matches(msg, defaultKeymap.togglePlay):
		if t.CurrentSess.Name != config.Work || t.Opts.Settings.Strict {
			return nil
		}

		if t.clock.Timedout() {
			return nil
		}

		return t.clock.Toggle()

	case key.Matches(msg, defaultKeymap.quit):
		_ = t.persist()

		return tea.Batch(tea.ClearScreen, tea.Quit)
	}

	return cmd
}

func (t *Timer) handleSettings(msg tea.Msg) tea.Cmd {
	if t.settings == soundView {
		return t.handleSoundUpdate(msg)
	}

	return nil
}

func (t *Timer) handleSoundUpdate(msg tea.Msg) tea.Cmd {
	form, cmd := t.soundForm.Update(msg)

	if f, ok := form.(*huh.Form); ok {
		t.soundForm = f

		if f.State == huh.StateCompleted {
			sound := t.soundForm.GetString("sound")
			t.Opts.Settings.AmbientSound = sound
			t.settings = ""

			_ = t.setAmbientSound()
		}
	}

	return cmd
}

func (t *Timer) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case btimer.TickMsg:
		return t, t.handleTimerTick(msg)

	case btimer.StartStopMsg:
		return t, t.handleTimerStartStop(msg)

	case btimer.TimeoutMsg:
		_ = t.persist()

		_ = t.postSession()

		nextSess := t.nextSession(t.CurrentSess.Name)

		if nextSess == config.Work && !t.Opts.Settings.AutoStartWork ||
			nextSess != config.Work && !t.Opts.Settings.AutoStartBreak {
			t.waitForNextSession = true
			t.settings = ""
		} else {
			cmd = t.initSession()
		}

		return t, cmd

	case tea.KeyMsg:
		return t, tea.Batch(t.handleSettings(msg), t.handleKeyPress(msg))

	case tea.WindowSizeMsg:
		t.progress.Width = msg.Width - padding*2 - 4
		t.progress.Width = min(t.progress.Width, maxWidth)

		return t, nil

	// FrameMsg is sent when the progress bar wants to animate itself
	case progress.FrameMsg:
		var progressModel tea.Model

		progressModel, cmd = t.progress.Update(msg)
		t.progress, _ = progressModel.(progress.Model)

		return t, cmd
	default:
		return t, tea.Batch(t.handleSettings(msg))
	}

}
