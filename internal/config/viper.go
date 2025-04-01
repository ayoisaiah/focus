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
	keyWorkDuration         = "work.duration"
	keyWorkMessage          = "work.message"
	keyWorkSound            = "work.sound"
	keyWorkColor            = "work.color"
	keyShortBreakDuration   = "short_break.duration"
	keyShortBreakMessage    = "short_break.message"
	keyShortBreakSound      = "short_break.sound"
	keyShortBreakColor      = "short_break.color"
	keyLongBreakDuration    = "long_break.duration"
	keyLongBreakMessage     = "long_break.message"
	keyLongBreakSound       = "long_break.sound"
	keyLongBreakColor       = "long_break.color"
	keyLongBreakInterval    = "settings.long_break_interval"
	keyAutoStartWork        = "settings.auto_start_work"
	keyAutoStartBreak       = "settings.auto_start_break"
	keySoundOnBreak         = "settings.sound_on_break"
	keyStrict               = "settings.strict"
	keyNotificationsEnabled = "notifications.enabled"
	keyAmbientSound         = "settings.ambient_sound"
	keySessionCmd           = "settings.cmd"
	keyTwentyFourHour       = "settings.24hr_clock"
	keyDarkTheme            = "display.dark_theme"
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
	v.SetDefault(keyWorkMessage, "Focus on your task")
	v.SetDefault(keyWorkColor, "#B0DB43")
	v.SetDefault(keyWorkSound, "loud_bell")
	v.SetDefault(keyShortBreakDuration, "5m")
	v.SetDefault(keyShortBreakMessage, "Take a breather")
	v.SetDefault(keyShortBreakColor, "#12EAEA")
	v.SetDefault(keyShortBreakSound, "bell")
	v.SetDefault(keyLongBreakColor, "#C492B1")
	v.SetDefault(keyLongBreakMessage, "Take a long break")
	v.SetDefault(keyLongBreakDuration, "15m")
	v.SetDefault(keyLongBreakSound, "bell")
	v.SetDefault(keyLongBreakInterval, 4)
	v.SetDefault(keyAutoStartBreak, true)
	v.SetDefault(keyAutoStartWork, false)
	v.SetDefault(keyNotificationsEnabled, true)
	v.SetDefault(keySoundOnBreak, false)
	v.SetDefault(keyDarkTheme, true)
	v.SetDefault(keyStrict, false)
	v.SetDefault(keyAmbientSound, "")
	v.SetDefault(keySessionCmd, "")

	// if c.Sessions.Durations != nil {
	// 	v.Set(keyWorkDuration, c.Sessions.Durations[Work].String())
	// 	v.Set(keyShortBreakDuration, c.Sessions.Durations[ShortBreak].String())
	// 	v.Set(keyLongBreakDuration, c.Sessions.Durations[LongBreak].String())
	// }
	//
	// if c.Sessions.LongBreakInterval != 0 {
	// 	v.Set(keyLongBreakInterval, c.Sessions.LongBreakInterval)
	// }

	return nil
}

// loadViperConfig loads configuration from Viper into the Config struct.
func loadViperConfig(v *viper.Viper, c *Config) error {
	// if err := loadDurations(v, c); err != nil {
	// 	return fmt.Errorf("loading durations failed: %w", err)
	// }
	v.Unmarshal(c)

	return nil
}

// loadDurations handles parsing duration strings from Viper.
// func loadDurations(v *viper.Viper, c *Config) error {
// 	durations := map[SessionType]string{
// 		Work:       v.GetString(keyWorkDuration),
// 		ShortBreak: v.GetString(keyShortBreakDuration),
// 		LongBreak:  v.GetString(keyLongBreakDuration),
// 	}
//
// 	c.Sessions.Durations = make(map[SessionType]time.Duration)
//
// 	for sessType, durStr := range durations {
// 		dur, err := parseDuration(durStr)
// 		if err != nil {
// 			return fmt.Errorf("invalid duration for %s: %w", sessType, err)
// 		}
//
// 		c.Sessions.Durations[sessType] = dur
// 	}
//
// 	return nil
// }

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
