package config

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/adrg/xdg"
	"github.com/pterm/pterm"
)

type (
	// Config holds all configuration settings.
	Config struct {
		Work          SessionConfig `mapstructure:"work"`
		ShortBreak    SessionConfig `mapstructure:"short_break"`
		LongBreak     SessionConfig `mapstructure:"long_break"`
		CLI           CLIConfig
		Settings      SettingsConfig `mapstructure:"settings"`
		Display       DisplayConfig  `mapstructue:"display"`
		Notifications NotificationConfig
	}

	SessionConfig struct {
		Message  string        `mapstructure:"message"`
		Color    string        `mapstructue:"color"`
		Sound    string        `mapstructure:"sound"`
		Duration time.Duration `mapstructure:"duration"`
	}

	// SettingsConfig contains general application settings.
	SettingsConfig struct {
		AmbientSound      string `mapstructure:"ambient_sound"`
		Cmd               string `mapstructure:"cmd"`
		LongBreakInterval int    `mapstructure:"long_break_interval"`
		AutoStartBreak    bool   `mapstructure:"auto_start_break"`
		AutoStartWork     bool   `mapstructure:"auto_start_work"`
		SoundOnBreak      bool   `mapstructure:"sound_on_break"`
		Strict            bool   `mapstructure:"strict"`
		TwentyFourHour    bool   `mapstructure:"24hr_clock"`
	}

	// arguments.
	CLIConfig struct {
		StartTime time.Time
		Tags      []string
	}

	// NotificationConfig holds notification settings.
	NotificationConfig struct {
		Enabled bool `mapstructure:"enabled"`
	}

	// DisplayConfig holds display-related settings.
	DisplayConfig struct {
		DarkTheme bool `mapstructure:"dark_theme"`
	}

	// Option is a function that modifies Config.
	Option func(*Config) error

	// SessionType represents the type of timer session.
	SessionType string
)

const Version = "v1.4.2"

const (
	Work       SessionType = "Work session"
	ShortBreak SessionType = "Short break"
	LongBreak  SessionType = "Long break"
)

var (
	configDir      = "focus"
	configFileName = "config.yml"
	dbFileName     = "focus.db"
	statusFileName = "status.json"
	logFileName    = "focus.log"
	dbFilePath     string
	configFilePath string
	statusFilePath string
	logFilePath    string
)

var (
	Stdin  io.Reader = os.Stdin
	Stdout io.Writer = os.Stdout
	Stderr io.Writer = os.Stderr
)

func Dir() string {
	return configDir
}

func DBFilePath() string {
	return dbFilePath
}

func StatusFilePath() string {
	return statusFilePath
}

func LogFilePath() string {
	return logFilePath
}

func ConfigFilePath() string {
	return configFilePath
}

func InitializePaths() {
	focusEnv := strings.TrimSpace(os.Getenv("FOCUS_ENV"))
	if focusEnv != "" {
		configFileName = fmt.Sprintf("config_%s.yml", focusEnv)
		dbFileName = fmt.Sprintf("focus_%s.db", focusEnv)
		statusFileName = fmt.Sprintf("status_%s.json", focusEnv)
		logFileName = fmt.Sprintf("focus_%s.log", focusEnv)
	}

	var err error

	relPath := filepath.Join(configDir, configFileName)

	configFilePath, err = xdg.ConfigFile(relPath)
	if err != nil {
		pterm.Error.Println(err)
		os.Exit(1)
	}

	dataDir, err := xdg.DataFile(configDir)
	if err != nil {
		pterm.Error.Println(err)
		os.Exit(1)
	}

	dbFilePath = filepath.Join(dataDir, dbFileName)

	statusFilePath = filepath.Join(dataDir, statusFileName)

	logFilePath = filepath.Join(dataDir, "log", logFileName)
}

// New creates a new Config with default values and applies options.
func New(opts ...Option) (*Config, error) {
	cfg := &Config{}

	for _, opt := range opts {
		if err := opt(cfg); err != nil {
			return nil, fmt.Errorf("config option error: %w", err)
		}
	}

	// if err := cfg.Validate(); err != nil {
	//     return nil, fmt.Errorf("config validation error: %w", err)
	// }

	return cfg, nil
}
