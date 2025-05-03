package config

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/adrg/xdg"

	"github.com/ayoisaiah/focus/internal/pathutil"
	"github.com/ayoisaiah/focus/report"
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
		firstRun      bool
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
	appName        = "focus"
	configFile     = "config.yml"
	dbFile         = "focus.db"
	statusFile     = "status.json"
	logFile        = "focus.log"
	dbFilePath     string
	configFilePath string
	statusFilePath string
)

var (
	Stdin  io.Reader = os.Stdin
	Stdout io.Writer = os.Stdout
	Stderr io.Writer = os.Stderr
)

func init() {
	focusEnv := strings.TrimSpace(os.Getenv("FOCUS_ENV"))

	if focusEnv != "" {
		configFile = fmt.Sprintf("config_%s.yml", focusEnv)
		dbFile = fmt.Sprintf("focus_%s.db", focusEnv)
		statusFile = fmt.Sprintf("status_%s.json", focusEnv)
	}

	var err error

	configFilePath, err = xdg.ConfigFile(filepath.Join(appName, configFile))
	if err != nil {
		report.Quit(err)
	}

	dbFilePath, err = xdg.DataFile(filepath.Join(appName, dbFile))
	if err != nil {
		report.Quit(err)
	}
	// statusFilePath = filepath.Join(dataDir, statusFile)
}

func Dir() string {
	return appName
}

func DBFilePath() string {
	return dbFilePath
}

func alertSoundPath() string {
	return filepath.Join(xdg.DataHome, appName, "alert_sound")
}

func AmbientSoundPath() string {
	return filepath.Join(xdg.DataHome, appName, "ambient_sound")
}

func StatusFilePath() string {
	return statusFilePath
}

func ConfigFilePath() string {
	return configFilePath
}

func SoundOpts() []string {
	var sounds []string

	dirs, err := os.ReadDir(AmbientSoundPath())
	if err == nil {
		for _, v := range dirs {
			sounds = append(sounds, pathutil.StripExtension(v.Name()))
		}
	}

	return sounds
}

// New creates a new Config with default values and applies options.
func New(opts ...Option) (*Config, error) {
	cfg := &Config{}

	for _, opt := range opts {
		if err := opt(cfg); err != nil {
			return nil, errConfigOption.Wrap(err)
		}
	}

	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("config validation error: %w", err)
	}

	return cfg, nil
}
