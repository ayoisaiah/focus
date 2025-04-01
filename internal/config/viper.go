package config

import (
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/spf13/viper"
)

// viperKeys defines the mapping between config keys and their Viper counterparts.
const (
	keyWorkDuration       = "work_duration"
	keyShortBreakDuration = "short_break_duration"
	keyLongBreakDuration  = "long_break_duration"
	keyWorkMessage        = "work_msg"
	keyShortBreakMessage  = "short_break_msg"
	keyLongBreakMessage   = "long_break_msg"
	keyLongBreakInterval  = "long_break_interval"
	keyAutoStartWork      = "auto_start_work"
	keyAutoStartBreak     = "auto_start_break"
	keyNotify             = "notify"
	keySoundOnBreak       = "sound_on_break"
	keyAmbientSound       = "sound"
	keyBreakSound         = "break_sound"
	keyWorkSound          = "work_sound"
	keySessionCmd         = "session_cmd"
	keyDarkTheme          = "dark_theme"
	keyTwentyFourHour     = "24hr_clock"
	keyStrict             = "strict"
	keyWorkColor          = "work_color"
	keyShortBreakColor    = "short_break_color"
	keyLongBreakColor     = "long_break_color"
)

// WithViperConfig returns an Option that loads configuration from Viper.
func WithViperConfig(configPath string) Option {
	return func(c *Config) error {
		v := viper.New()

		v.SetConfigFile(configPath)
		v.SetConfigType("yaml")

		if err := setupViper(v, c); err != nil {
			return fmt.Errorf("viper setup failed: %w", err)
		}

		err := v.ReadInConfig()
		if err == nil {
			return loadViperConfig(v, c)
		}

		if !errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("reading config file failed: %w", err)
		}

		if err := v.WriteConfig(); err != nil {
			return fmt.Errorf("writing default config failed: %w", err)
		}

		return loadViperConfig(v, c)
	}
}

// setupViper configures Viper with defaults and prompt values.
func setupViper(v *viper.Viper, c *Config) error {
	// Set defaults
	v.SetDefault(keyWorkDuration, "25m")
	v.SetDefault(keyShortBreakDuration, "5m")
	v.SetDefault(keyLongBreakDuration, "15m")
	v.SetDefault(keyWorkMessage, "Focus on your task")
	v.SetDefault(keyShortBreakMessage, "Take a breather")
	v.SetDefault(keyLongBreakMessage, "Take a long break")
	v.SetDefault(keyLongBreakInterval, 4)
	v.SetDefault(keyAutoStartBreak, true)
	v.SetDefault(keyAutoStartWork, false)
	v.SetDefault(keyNotify, true)
	v.SetDefault(keySoundOnBreak, false)
	v.SetDefault(keyDarkTheme, true)
	v.SetDefault(keyBreakSound, "bell")
	v.SetDefault(keyWorkSound, "loud_bell")
	v.SetDefault(keyStrict, false)
	v.SetDefault(keyWorkColor, "#B0DB43")
	v.SetDefault(keyShortBreakColor, "#12EAEA")
	v.SetDefault(keyLongBreakColor, "#C492B1")
	v.SetDefault(keyAmbientSound, "")
	v.SetDefault(keySessionCmd, "")

	if c.Sessions.Durations != nil {
		v.Set(keyWorkDuration, c.Sessions.Durations[Work].String())
		v.Set(keyShortBreakDuration, c.Sessions.Durations[ShortBreak].String())
		v.Set(keyLongBreakDuration, c.Sessions.Durations[LongBreak].String())
	}

	if c.Sessions.LongBreakInterval != 0 {
		v.Set(keyLongBreakInterval, c.Sessions.LongBreakInterval)
	}

	return nil
}

// loadViperConfig loads configuration from Viper into the Config struct.
func loadViperConfig(v *viper.Viper, c *Config) error {
	if err := loadDurations(v, c); err != nil {
		return fmt.Errorf("loading durations failed: %w", err)
	}

	c.Sessions.Messages = map[SessionType]string{
		Work:       v.GetString(keyWorkMessage),
		ShortBreak: v.GetString(keyShortBreakMessage),
		LongBreak:  v.GetString(keyLongBreakMessage),
	}

	c.Sessions.LongBreakInterval = v.GetInt(keyLongBreakInterval)
	c.Sessions.AutoStartWork = v.GetBool(keyAutoStartWork)
	c.Sessions.AutoStartBreak = v.GetBool(keyAutoStartBreak)
	c.Sessions.Strict = v.GetBool(keyStrict)

	c.Notification.Enabled = v.GetBool(keyNotify)
	c.Notification.Sounds = map[SessionType]string{
		Work:       v.GetString(keyWorkSound),
		ShortBreak: v.GetString(keyBreakSound),
		LongBreak:  v.GetString(keyBreakSound),
	}

	c.Display.Colors = map[SessionType]string{
		Work:       v.GetString(keyWorkColor),
		ShortBreak: v.GetString(keyShortBreakColor),
		LongBreak:  v.GetString(keyLongBreakColor),
	}
	c.Display.DarkTheme = v.GetBool(keyDarkTheme)
	c.Display.TwentyFourHour = v.GetBool(keyTwentyFourHour)

	c.Sound.AmbientSound = v.GetString(keyAmbientSound)
	c.Sound.PlayOnBreak = v.GetBool(keySoundOnBreak)

	c.System.ConfigPath = v.ConfigFileUsed()
	c.System.SessionCmd = v.GetString(keySessionCmd)

	return nil
}

// loadDurations handles parsing duration strings from Viper.
func loadDurations(v *viper.Viper, c *Config) error {
	durations := map[SessionType]string{
		Work:       v.GetString(keyWorkDuration),
		ShortBreak: v.GetString(keyShortBreakDuration),
		LongBreak:  v.GetString(keyLongBreakDuration),
	}

	c.Sessions.Durations = make(map[SessionType]time.Duration)

	for sessType, durStr := range durations {
		dur, err := parseDuration(durStr)
		if err != nil {
			return fmt.Errorf("invalid duration for %s: %w", sessType, err)
		}

		c.Sessions.Durations[sessType] = dur
	}

	return nil
}

// duration strings.
func parseDuration(s string) (time.Duration, error) {
	// Try parsing as duration string first
	dur, err := time.ParseDuration(s)
	if err == nil {
		return dur, nil
	}

	// Try parsing as minutes in case duration unit is absent
	mins, err := time.ParseDuration(s + "m")
	if err != nil {
		return 0, fmt.Errorf("invalid duration format: %s", s)
	}

	return mins, nil
}
