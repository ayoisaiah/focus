package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"time"
)

var (
	// Minimum and maximum duration constraints.
	minSessionDuration = 1 * time.Second
	maxSessionDuration = 720 * time.Minute // 12 hours

	// Valid long break intervals.
	minLongBreakInterval = 4
	maxLongBreakInterval = 10

	// Color format validation.
	hexColorRegex = regexp.MustCompile(`^#[0-9A-Fa-f]{6}$`)
)

// Validate performs validation checks on the Config struct and its fields.
func (c *Config) Validate() error {
	if err := c.validateSessionConfig(c.Work, "work"); err != nil {
		return err
	}

	if err := c.validateSessionConfig(c.ShortBreak, "short break"); err != nil {
		return err
	}

	if err := c.validateSessionConfig(c.LongBreak, "long break"); err != nil {
		return err
	}

	if err := c.validateSessionRelationships(); err != nil {
		return err
	}

	if err := c.validateSettings(); err != nil {
		return err
	}

	return nil
}

// validateSessionConfig validates an individual SessionConfig.
func (c *Config) validateSessionConfig(
	sc SessionConfig,
	sessionType string,
) error {
	if sc.Duration < minSessionDuration || sc.Duration > maxSessionDuration {
		return errInvalidDuration.Fmt(
			sessionType,
			minSessionDuration,
			maxSessionDuration,
		)
	}

	if strings.TrimSpace(sc.Message) == "" {
		return errEmptyMsg.Fmt(sessionType)
	}

	if !hexColorRegex.MatchString(sc.Color) {
		return errInvalidColor.Fmt(sessionType, sc.Color)
	}

	if sc.Sound != "" {
		if err := c.validateSound(sc.Sound, "alert"); err != nil {
			return fmt.Errorf("%s sound invalid: %w", sessionType, err)
		}
	}

	return nil
}

// validateSettings validates the SettingsConfig.
func (c *Config) validateSettings() error {
	if c.Settings.LongBreakInterval < minLongBreakInterval ||
		c.Settings.LongBreakInterval > maxLongBreakInterval {
		return errInvalidDuration
	}

	if c.Settings.AmbientSound != "" {
		if err := c.validateSound(c.Settings.AmbientSound, "ambient"); err != nil {
			return err
		}
	}

	return nil
}

// validateSessionRelationships validates logical relationships between sessions.
func (c *Config) validateSessionRelationships() error {
	if c.ShortBreak.Duration >= c.Work.Duration {
		return errShortBreakTooLong.Fmt(c.ShortBreak.Duration, c.Work.Duration)
	}

	if c.LongBreak.Duration < c.ShortBreak.Duration {
		return errLongBreakTooShort.Fmt(
			c.LongBreak.Duration,
			c.ShortBreak.Duration,
		)
	}

	return nil
}

// It handles both built-in and custom sounds.
func (c *Config) validateSound(sound, group string) error {
	if filepath.Ext(sound) == "" {
		sound = sound + ".ogg"
	}

	ext := strings.ToLower(filepath.Ext(sound))
	validExts := []string{".mp3", ".ogg", ".flac", ".wav"}

	if !slices.Contains(validExts, ext) {
		return errInvalidSoundFormat.Fmt(sound)
	}

	if group == "alert" {
		_, err := os.Stat(filepath.Join(AlertSoundPath(), sound))
		if errors.Is(err, os.ErrNotExist) {
			return errUnknownAlertSound.Fmt(sound)
		}

		return nil
	}

	_, err := os.Stat(filepath.Join(AmbientSoundPath(), sound))
	if errors.Is(err, os.ErrNotExist) {
		return errUnknownAmbientSound.Fmt(sound)
	}

	return nil
}
