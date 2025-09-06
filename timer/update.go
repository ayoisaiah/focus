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

	// In flow mode, track elapsed time and play bells
	if t.flowMode && t.clock.Running() {
		// Only calculate elapsed time if StartTime is properly set (not zero)
		if !t.StartTime.IsZero() {
			totalTime := time.Since(t.StartTime)
			// Subtract accumulated paused time (but not current time since we're running)
			t.elapsedTime = totalTime - t.pausedDuration
		} else {
			t.elapsedTime = 0
		}
		
		// Play bells if enabled
		if t.Opts.Settings.FlowBell && !t.StartTime.IsZero() {
			halfwayTime := t.estimatedTime / 2
			
			// Play bell and notify at 50% of estimated time
			if !t.halfwayBellPlayed && t.elapsedTime >= halfwayTime {
				t.halfwayBellPlayed = true
				t.playFlowBell() // Call directly in main thread like ambient sounds
				go t.notifyFlowMilestone("halfway")
			}
			
			// Play bell and notify at 100% of estimated time
			if !t.completeBellPlayed && t.elapsedTime >= t.estimatedTime {
				t.completeBellPlayed = true
				t.playFlowBell() // Call directly in main thread like ambient sounds
				go t.notifyFlowMilestone("complete")
			}
		}
	}

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
		// In flow mode, only set StartTime if it's not already set
		// This preserves the original start time across pause/resume cycles
		if t.flowMode {
			if t.StartTime.IsZero() {
				t.StartTime = time.Now()
			}
			// If we were paused, add the paused duration to accumulated paused time
			if !t.PausedTime.IsZero() {
				t.pausedDuration += time.Since(t.PausedTime)
				t.PausedTime = time.Time{} // Reset paused time
			}
		} else {
			t.StartTime = time.Now()
		}
		t.Current.SetEndTime()
	} else {
		// When pausing in flow mode, record the pause time
		if t.flowMode {
			t.PausedTime = time.Now()
		}
		_ = t.persist()
	}

	// Don't suspend/resume speaker when ambient sound is playing
	// as it can interfere with continuous background audio
	if t.SoundStream != nil && t.Opts.Settings.AmbientSound != "" {
		// Skip speaker suspend/resume when ambient sound is active
		return t, cmd
	}
	
	// Only manage speaker suspend/resume for non-ambient sounds
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

	// Handle flow mode form input (still using huh)
	if t.soundForm != nil && t.settings == "flowPrompt" {
		// Check for quit key before passing to form
		if keyMsg, ok := msg.(tea.KeyMsg); ok {
			switch keyMsg.String() {
			case "ctrl+c":
				// Allow quit from flow mode form
				_ = t.persist()
				return t, tea.Batch(tea.ClearScreen, tea.Quit)
			}
		}
		
		slog.Info(spew.Sdump(msg))

		form, cmd := t.soundForm.Update(msg)
		if f, ok := form.(*huh.Form); ok {
			t.soundForm = f
			
			// Check if flow mode form is completed
			if t.soundForm.State == huh.StateCompleted {
				t.settings = ""
				t.soundForm = nil
				err := t.initFlowTimer()
				if err != nil {
					return t, nil
				}
				// Set StartTime immediately for flow mode since we want to start counting now
				t.StartTime = time.Now()
				return t, tea.Batch(t.clock.Init(), t.clock.Start())
			}
			
			return t, cmd
		}
	}

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
		// Handle custom sound menu navigation
		if t.showingSoundMenu {
			switch msg.String() {
			case "up", "k":
				if t.selectedSoundIndex > 0 {
					t.selectedSoundIndex--
				}
				return t, nil
			case "down", "j":
				if t.selectedSoundIndex < len(t.soundOptions)-1 {
					t.selectedSoundIndex++
				}
				return t, nil
			case "enter":
				// Apply selected sound
				selectedSound := t.soundOptions[t.selectedSoundIndex]
				if selectedSound == "off" {
					t.Opts.Settings.AmbientSound = ""
				} else {
					t.Opts.Settings.AmbientSound = selectedSound
				}
				
				// Close menu
				t.showingSoundMenu = false
				t.settings = ""
				
				// Apply sound change
				err := t.setAmbientSound()
				if err != nil {
					// Continue even if sound fails
				}
				return t, nil
			case "esc":
				// Cancel sound selection
				t.showingSoundMenu = false
				t.settings = ""
				return t, nil
			case "ctrl+c":
				// Allow quit even when in sound menu
				_ = t.persist()
				return t, tea.Batch(tea.ClearScreen, tea.Quit)
			}
		}
		
		switch {
		case key.Matches(msg, defaultKeymap.enter):
			if t.settings != "" {
				break
			}

			if t.waitForNextSession {
				t.waitForNextSession = false

				sessName := t.nextSession(t.Current.Name)
				t.Current = t.newSession(sessName)

				t.clock = btimer.New(t.Current.Duration)
				cmd = t.clock.Init()
			}

			return t, cmd

		case key.Matches(msg, defaultKeymap.sound):
			// Don't allow sound selection during flow mode prompt
			if t.settings == "flowPrompt" {
				return t, nil
			}
			
			// Only allow sound selection when timer is running or paused, not when timed out
			if !t.clock.Timedout() {
				// Use custom sound selection instead of huh form
				t.soundOptions = config.SoundOpts()
				t.selectedSoundIndex = 0
				t.showingSoundMenu = true
				t.settings = soundView
				
				// Find current sound in options if it exists
				currentSound := t.Opts.Settings.AmbientSound
				if currentSound == "" {
					currentSound = "off"
				}
				for i, opt := range t.soundOptions {
					if opt == currentSound {
						t.selectedSoundIndex = i
						break
					}
				}
			}

			return t, nil

		case key.Matches(msg, defaultKeymap.esc):
			// If in flow mode prompt, cancel and quit
			if t.settings == "flowPrompt" {
				return t, tea.Quit
			}
			
			// Skip break sessions
			if t.Current.Name != config.Work && t.clock.Running() {
				return t, tea.Batch(t.clock.Stop(), t.initSession())
			}

			t.settings = ""

			return t, nil

		case key.Matches(msg, defaultKeymap.togglePlay):
			// Don't allow toggle play during flow mode prompt
			if t.settings == "flowPrompt" {
				return t, nil
			}
			
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

	return t, nil
}
