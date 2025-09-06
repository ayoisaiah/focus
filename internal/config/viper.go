package config

import (
	"errors"
	"fmt"
	"os"

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
	keyFlowBell             = "settings.flow_bell"
	keyFlowBellSound        = "settings.flow_bell_sound"
	keyDarkTheme            = "display.dark_theme"
)

// WithViperConfig returns an Option that loads configuration from Viper.
func WithViperConfig(configPath string) Option {
	return func(c *Config) error {
		v := viper.New()

		v.SetConfigFile(configPath)
		v.SetConfigType("yaml")

		setupViper(v, c)

		err := v.ReadInConfig()
		if err == nil {
			return loadViperConfig(v, c)
		}

		if !errors.Is(err, os.ErrNotExist) {
			return errReadConfig.Wrap(err)
		}

		if err := v.WriteConfig(); err != nil {
			return errWriteConfig.Wrap(err)
		}

		return loadViperConfig(v, c)
	}
}

// setupViper configures Viper with defaults and prompt values.
func setupViper(v *viper.Viper, c *Config) {
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
	v.SetDefault(keyFlowBell, true)
	v.SetDefault(keyFlowBellSound, "tibetan_bell")
	v.SetDefault(keyAmbientSound, "")
	v.SetDefault(keySessionCmd, "")

	if c.firstRun {
		v.SetDefault(
			keyWorkDuration,
			fmt.Sprintf("%dm", int(c.Work.Duration.Minutes())),
		)
		v.SetDefault(
			keyShortBreakDuration,
			fmt.Sprintf("%dm", int(c.ShortBreak.Duration.Minutes())),
		)

		v.SetDefault(
			keyLongBreakDuration,
			fmt.Sprintf("%dm", int(c.LongBreak.Duration.Minutes())),
		)

		if c.Settings.LongBreakInterval != 0 {
			v.SetDefault(keyLongBreakInterval, c.Settings.LongBreakInterval)
		}
	}
}

// loadViperConfig loads configuration from Viper into the Config struct.
func loadViperConfig(v *viper.Viper, c *Config) error {
	return v.Unmarshal(c)
}
