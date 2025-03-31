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
	// Config holds all configuration settings
	Config struct {
		Sessions     SessionConfig
		Notification NotificationConfig
		Display      DisplayConfig
		Sound        SoundConfig
		System       SystemConfig
	}

	// SessionConfig holds session-related settings
	SessionConfig struct {
		Durations         map[SessionType]time.Duration
		Messages          map[SessionType]string
		LongBreakInterval int
		AutoStartWork     bool
		AutoStartBreak    bool
		Strict            bool
		Tags              []string
		StartTime         time.Time
	}

	// NotificationConfig holds notification settings
	NotificationConfig struct {
		Enabled bool
		Sounds  map[SessionType]string
	}

	// DisplayConfig holds display-related settings
	DisplayConfig struct {
		Colors         map[SessionType]string
		DarkTheme      bool
		TwentyFourHour bool
	}

	// SoundConfig holds sound-related settings
	SoundConfig struct {
		AmbientSound string
		PlayOnBreak  bool
	}

	// SystemConfig holds system-related settings
	SystemConfig struct {
		ConfigPath string
		DBPath     string
		SessionCmd string
	}

	// Option is a function that modifies Config
	Option func(*Config) error

	// SessionType represents the type of timer session
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

// New creates a new Config with default values and applies options
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
