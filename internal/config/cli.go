package config

import (
	"fmt"
	"strings"
	"time"

	"github.com/urfave/cli/v2"

	"github.com/ayoisaiah/focus/internal/timeutil"
)

// CLIOptions represents command-line configuration options.
type CLIOptions struct {
	Since             string
	ShortBreak        string
	LongBreak         string
	Tags              string
	AmbientSound      string
	BreakSound        string
	WorkSound         string
	SessionCmd        string
	Work              string
	LongBreakInterval uint
	DisableNotify     bool
	SoundOnBreak      bool
	Strict            bool
}

// WithCLIConfig returns an Option that loads configuration from CLI flags.
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

// applyCLIOptions applies CLI options to the config.
func applyCLIOptions(c *Config, opts CLIOptions) error {
	if err := applyCLIDurations(c, opts); err != nil {
		return fmt.Errorf("applying CLI durations: %w", err)
	}

	if opts.Tags != "" {
		c.CLI.Tags = splitAndTrimTags(opts.Tags)
	}

	if opts.DisableNotify {
		c.Notifications.Enabled = false
	}

	if opts.Strict {
		c.Settings.Strict = true
	}

	if err := applyCLISounds(c, opts); err != nil {
		return fmt.Errorf("applying CLI sounds: %w", err)
	}

	if opts.SessionCmd != "" {
		c.Settings.Cmd = opts.SessionCmd
	}

	if opts.Since != "" {
		startTime, err := timeutil.FromStr(opts.Since)
		if err != nil {
			return fmt.Errorf("invalid since time: %w", err)
		}

		c.CLI.StartTime = startTime
	} else {
		c.CLI.StartTime = time.Now()
	}

	return nil
}

// applyCLIDurations handles parsing and applying duration settings from CLI.
func applyCLIDurations(c *Config, opts CLIOptions) error {
	durationsMap := map[SessionType]string{
		Work:       opts.Work,
		ShortBreak: opts.ShortBreak,
		LongBreak:  opts.LongBreak,
	}

	for sessType, durStr := range durationsMap {
		if durStr != "" {
			dur, err := time.ParseDuration(durStr)
			if err != nil {
				return errInvalidCLIDuration.Fmt(sessType, err)
			}

			if sessType == Work {
				c.Work.Duration = dur
			}

			if sessType == ShortBreak {
				c.ShortBreak.Duration = dur
			}

			if sessType == LongBreak {
				c.LongBreak.Duration = dur
			}
		}
	}

	if opts.LongBreakInterval > 0 {
		c.Settings.LongBreakInterval = int(opts.LongBreakInterval)
	}

	return nil
}

// applyCLISounds handles sound-related CLI options.
func applyCLISounds(c *Config, opts CLIOptions) error {
	if opts.AmbientSound != "" {
		if opts.AmbientSound == "off" {
			c.Settings.AmbientSound = ""
		} else {
			c.Settings.AmbientSound = opts.AmbientSound
		}
	}

	if opts.BreakSound != "" {
		if opts.BreakSound == "off" {
			c.ShortBreak.Sound = ""
			c.LongBreak.Sound = ""
		} else {
			c.ShortBreak.Sound = opts.BreakSound
			c.LongBreak.Sound = opts.BreakSound
		}
	}

	if opts.WorkSound != "" {
		if opts.WorkSound == "off" {
			c.Work.Sound = ""
		} else {
			c.Work.Sound = opts.WorkSound
		}
	}

	c.Settings.SoundOnBreak = opts.SoundOnBreak

	return nil
}

// splitAndTrimTags splits a comma-separated tag string and trims whitespace.
func splitAndTrimTags(tags string) []string {
	split := strings.Split(tags, ",")

	trimmed := make([]string, len(split))

	for i, tag := range split {
		trimmed[i] = strings.TrimSpace(tag)
	}

	return trimmed
}
