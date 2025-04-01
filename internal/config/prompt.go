package config

import (
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/charmbracelet/huh"
	"github.com/pterm/pterm"
	"github.com/pterm/pterm/putils"
)

const asciiLogo = `
███████╗ ██████╗  ██████╗██╗   ██╗███████╗
██╔════╝██╔═══██╗██╔════╝██║   ██║██╔════╝
█████╗  ██║   ██║██║     ██║   ██║███████╗
██╔══╝  ██║   ██║██║     ██║   ██║╚════██║
██║     ╚██████╔╝╚██████╗╚██████╔╝███████║
╚═╝      ╚═════╝  ╚═════╝ ╚═════╝ ╚══════╝`

// PromptOptions holds the user's responses to the configuration prompts.
type PromptOptions struct {
	WorkDuration       int
	ShortBreakDuration int
	LongBreakDuration  int
	LongBreakInterval  int
}

// WithPromptConfig returns an Option that configures settings via interactive prompts.
func WithPromptConfig(configPath string) Option {
	return func(c *Config) error {
		_, err := os.Stat(configPath)
		if err == nil || !errors.Is(err, os.ErrNotExist) {
			return err
		}

		opts, err := promptUser()
		if err != nil {
			return fmt.Errorf("user prompt failed: %w", err)
		}

		return applyPromptOptions(c, opts)
	}
}

// promptUser handles the interactive configuration process.
func promptUser() (PromptOptions, error) {
	var opts PromptOptions

	// Display welcome message
	pterm.Println(asciiLogo)

	_ = putils.BulletListFromString(`Follow the prompts below to configure Focus for the first time.
Select your preferred value, or press ENTER to accept the defaults.
Edit the config file with 'focus edit-config' to change any settings.`, " ").
		Render()

	form := huh.NewForm(
		huh.NewGroup(
			huh.NewSelect[int]().
				Title("Work session length").
				Options(
					huh.NewOption("25 minutes", 25).Selected(true),
					huh.NewOption("35 minutes", 35),
					huh.NewOption("50 minutes", 50),
					huh.NewOption("60 minutes", 60),
					huh.NewOption("90 minutes", 90),
				).
				Value(&opts.WorkDuration),
		),
		huh.NewGroup(
			huh.NewSelect[int]().
				Title("Short break length").
				Options(
					huh.NewOption("5 minutes", 5).Selected(true),
					huh.NewOption("10 minutes", 10),
					huh.NewOption("15 minutes", 15),
					huh.NewOption("20 minutes", 20),
				).
				Value(&opts.ShortBreakDuration),
		),
		huh.NewGroup(
			huh.NewSelect[int]().
				Title("Long break length").
				Options(
					huh.NewOption("15 minutes", 15).Selected(true),
					huh.NewOption("20 minutes", 20),
					huh.NewOption("30 minutes", 30),
					huh.NewOption("45 minutes", 45),
				).
				Value(&opts.LongBreakDuration),
		),
		huh.NewGroup(
			huh.NewSelect[int]().
				Title("Work sessions before long break").
				Options(
					huh.NewOption("4 sessions", 4).Selected(true),
					huh.NewOption("6 sessions", 6),
					huh.NewOption("8 sessions", 8),
				).
				Value(&opts.LongBreakInterval),
		),
	)

	err := form.Run()
	if err != nil {
		return opts, fmt.Errorf("form interaction failed: %w", err)
	}

	return opts, nil
}

// applyPromptOptions applies the user's prompt responses to the configuration.
func applyPromptOptions(c *Config, opts PromptOptions) error {
	c.Work.Duration = time.Duration(opts.WorkDuration) * time.Minute
	c.ShortBreak.Duration = time.Duration(opts.ShortBreakDuration) * time.Minute
	c.LongBreak.Duration = time.Duration(opts.LongBreakDuration) * time.Minute
	c.Settings.LongBreakInterval = opts.LongBreakInterval

	return nil
}
