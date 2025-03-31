package config

import (
	"fmt"
	"strings"
	"time"

	"github.com/ayoisaiah/focus/internal/timeutil"
	"github.com/urfave/cli/v2"
)

// CLIOptions represents command-line configuration options
type CLIOptions struct {
	Work              string
	ShortBreak        string
	LongBreak         string
	LongBreakInterval uint
	Tags              string
	AmbientSound      string
	BreakSound        string
	WorkSound         string
	SessionCmd        string
	Since             string
	DisableNotify     bool
	SoundOnBreak      bool
	Strict            bool
}

// WithCLIConfig returns an Option that loads configuration from CLI flags
func WithCLIConfig(ctx *cli.Context) Option {
	return func(c *Config) error {
		opts := CLIOptions{
			Work:              ctx.String("work"),
			ShortBreak:        ctx.String("short-break"),
			LongBreak:         ctx.String("long-break"),
			LongBreakInterval: ctx.Uint("long-break-interval"),
			Tags:              ctx.String("tag"),
			AmbientSound:      ctx.String("sound"),
			BreakSound:        ctx.String("break-sound"),
			WorkSound:         ctx.String("work-sound"),
			SessionCmd:        ctx.String("session-cmd"),
			Since:             ctx.String("since"),
			DisableNotify:     ctx.Bool("disable-notification"),
			SoundOnBreak:      ctx.Bool("sound-on-break"),
			Strict:            ctx.Bool("strict"),
		}

		return applyCLIOptions(c, opts)
	}
}

// applyCLIOptions applies CLI options to the config
func applyCLIOptions(c *Config, opts CLIOptions) error {
	// Handle session durations
	if err := applyCLIDurations(c, opts); err != nil {
		return fmt.Errorf("applying CLI durations: %w", err)
	}

	// Handle tags
	if opts.Tags != "" {
		c.Sessions.Tags = splitAndTrimTags(opts.Tags)
	}

	// Handle notifications
	if opts.DisableNotify {
		c.Notification.Enabled = false
	}

	// Handle strict mode
	if opts.Strict {
		c.Sessions.Strict = true
	}

	// Handle sounds
	if err := applyCLISounds(c, opts); err != nil {
		return fmt.Errorf("applying CLI sounds: %w", err)
	}

	// Handle session command
	if opts.SessionCmd != "" {
		c.System.SessionCmd = opts.SessionCmd
	}

	// Handle since time
	if opts.Since != "" {
		startTime, err := timeutil.FromStr(opts.Since)
		if err != nil {
			return fmt.Errorf("invalid since time: %w", err)
		}
		c.Sessions.StartTime = startTime
	} else {
		c.Sessions.StartTime = time.Now()
	}

	return nil
}

// applyCLIDurations handles parsing and applying duration settings from CLI
func applyCLIDurations(c *Config, opts CLIOptions) error {
	durationsMap := map[SessionType]string{
		Work:       opts.Work,
		ShortBreak: opts.ShortBreak,
		LongBreak:  opts.LongBreak,
	}

	for sessType, durStr := range durationsMap {
		if durStr != "" {
			dur, err := parseDuration(durStr)
			if err != nil {
				return fmt.Errorf("invalid duration for %s: %w", sessType, err)
			}
			c.Sessions.Durations[sessType] = dur
		}
	}

	if opts.LongBreakInterval > 0 {
		c.Sessions.LongBreakInterval = int(opts.LongBreakInterval)
	}

	return nil
}

// applyCLISounds handles sound-related CLI options
func applyCLISounds(c *Config, opts CLIOptions) error {
	// Handle ambient sound
	if opts.AmbientSound != "" {
		if opts.AmbientSound == "off" {
			c.Sound.AmbientSound = ""
		} else {
			c.Sound.AmbientSound = opts.AmbientSound
		}
	}

	// Handle break sound
	if opts.BreakSound != "" {
		if opts.BreakSound == "off" {
			c.Notification.Sounds[ShortBreak] = ""
			c.Notification.Sounds[LongBreak] = ""
		} else {
			c.Notification.Sounds[ShortBreak] = opts.BreakSound
			c.Notification.Sounds[LongBreak] = opts.BreakSound
		}
	}

	// Handle work sound
	if opts.WorkSound != "" {
		if opts.WorkSound == "off" {
			c.Notification.Sounds[Work] = ""
		} else {
			c.Notification.Sounds[Work] = opts.WorkSound
		}
	}

	// Handle sound on break setting
	c.Sound.PlayOnBreak = opts.SoundOnBreak

	return nil
}

// splitAndTrimTags splits a comma-separated tag string and trims whitespace
func splitAndTrimTags(tags string) []string {
	split := strings.Split(tags, ",")

	trimmed := make([]string, len(split))

	for i, tag := range split {
		trimmed[i] = strings.TrimSpace(tag)
	}

	return trimmed
}
