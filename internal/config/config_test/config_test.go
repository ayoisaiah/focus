package config_test

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/ayoisaiah/focus/internal/config"
	"github.com/ayoisaiah/focus/internal/testutil"
)

type TestCase struct {
	Want       *config.Config
	Name       string
	GoldenFile string
	Snapshot   []byte `json:"-"`
}

func (t TestCase) Output() (out []byte, name string) {
	return t.Snapshot, t.GoldenFile
}

// defaultConfig returns a new Config instance with default values.
func defaultConfig() *config.Config {
	return &config.Config{
		Work: config.SessionConfig{
			Message:  "Focus on your task",
			Color:    "#B0DB43",
			Sound:    "loud_bell",
			Duration: 25 * time.Minute,
		},
		ShortBreak: config.SessionConfig{
			Message:  "Take a breather",
			Color:    "#12EAEA",
			Sound:    "bell",
			Duration: 5 * time.Minute,
		},
		LongBreak: config.SessionConfig{
			Message:  "Take a long break",
			Color:    "#C492B1",
			Sound:    "bell",
			Duration: 15 * time.Minute,
		},
		Settings: config.SettingsConfig{
			AmbientSound:      "",
			AutoStartBreak:    true,
			AutoStartWork:     false,
			Cmd:               "",
			LongBreakInterval: 4,
			SoundOnBreak:      false,
			Strict:            false,
			TwentyFourHour:    false,
		},
		Notifications: config.NotificationConfig{
			Enabled: true,
		},
		Display: config.DisplayConfig{
			DarkTheme: true,
		},
	}
}

func TestViperWriteConfig(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yml")

	tc := TestCase{
		Name:       "write default config to file",
		GoldenFile: "defaults",
		Want:       defaultConfig(),
	}

	cfg, err := config.New(
		config.WithViperConfig(configPath),
	)
	if err != nil {
		t.Fatal(err)
	}

	tc.Snapshot, err = os.ReadFile(configPath)
	if err != nil {
		t.Fatal("failed to read config", err)
	}

	testutil.CompareGoldenFile(t, tc)

	assert.Equal(t, tc.Want, cfg)
}

func TestViperReadConfig(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yml")

	err := testutil.CopyFile("testdata/modified_config.golden", configPath)
	if err != nil {
		t.Fatal(err)
	}

	tc := TestCase{
		Name: "read a modified config file",
		Want: &config.Config{
			Work: config.SessionConfig{
				Message:  "Focus on your task",
				Color:    "#B0DB43",
				Sound:    "loud_bell",
				Duration: 50 * time.Minute,
			},
			ShortBreak: config.SessionConfig{
				Message:  "Take a short rest",
				Color:    "#12EAEA",
				Sound:    "loud_bell",
				Duration: 10 * time.Minute,
			},
			LongBreak: config.SessionConfig{
				Message:  "Rest a little longer",
				Color:    "#C492B1",
				Sound:    "loud_bell",
				Duration: 30 * time.Minute,
			},
			Settings: config.SettingsConfig{
				AmbientSound:      "",
				AutoStartBreak:    true,
				AutoStartWork:     false,
				Cmd:               "",
				LongBreakInterval: 6,
				SoundOnBreak:      false,
				Strict:            false,
				TwentyFourHour:    false,
			},
			Notifications: config.NotificationConfig{
				Enabled: true,
			},
			Display: config.DisplayConfig{
				DarkTheme: true,
			},
		},
	}

	cfg, err := config.New(
		config.WithViperConfig(configPath),
	)
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, tc.Want, cfg)
}
