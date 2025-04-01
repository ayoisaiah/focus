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
		Sessions: config.SessionConfig{
			Durations: map[config.SessionType]time.Duration{
				config.Work:       25 * time.Minute,
				config.ShortBreak: 5 * time.Minute,
				config.LongBreak:  15 * time.Minute,
			},
			Messages: map[config.SessionType]string{
				config.Work:       "Focus on your task",
				config.ShortBreak: "Take a breather",
				config.LongBreak:  "Take a long break",
			},
			LongBreakInterval: 4,
			AutoStartWork:     false,
			AutoStartBreak:    true,
			Strict:            false,
		},
		Notification: config.NotificationConfig{
			Enabled: true,
			Sounds: map[config.SessionType]string{
				config.Work:       "loud_bell",
				config.ShortBreak: "bell",
				config.LongBreak:  "bell",
			},
		},
		Display: config.DisplayConfig{
			Colors: map[config.SessionType]string{
				config.Work:       "#B0DB43",
				config.ShortBreak: "#12EAEA",
				config.LongBreak:  "#C492B1",
			},
			DarkTheme:      true,
			TwentyFourHour: false,
		},
		Sound: config.SoundConfig{
			AmbientSound: "",
			PlayOnBreak:  false,
		},
		System: config.SystemConfig{
			SessionCmd: "",
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

	tc.Want.System.ConfigPath = configPath

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
			Sessions: config.SessionConfig{
				Durations: map[config.SessionType]time.Duration{
					config.Work:       50 * time.Minute,
					config.ShortBreak: 10 * time.Minute,
					config.LongBreak:  30 * time.Minute,
				},
				Messages: map[config.SessionType]string{
					config.Work:       "Focus on your task",
					config.ShortBreak: "Take a short rest",
					config.LongBreak:  "Rest a little longer",
				},
				LongBreakInterval: 6,
				AutoStartWork:     false,
				AutoStartBreak:    false,
				Strict:            false,
			},
			Notification: config.NotificationConfig{
				Enabled: true,
				Sounds: map[config.SessionType]string{
					config.Work:       "loud_bell",
					config.ShortBreak: "loud_bell",
					config.LongBreak:  "loud_bell",
				},
			},
			Display: config.DisplayConfig{
				Colors: map[config.SessionType]string{
					config.Work:       "#B0DB43",
					config.ShortBreak: "#12EAEA",
					config.LongBreak:  "#C492B1",
				},
				DarkTheme:      false,
				TwentyFourHour: false,
			},
			Sound: config.SoundConfig{
				AmbientSound: "",
				PlayOnBreak:  false,
			},
			System: config.SystemConfig{
				SessionCmd: "",
			},
		},
	}

	tc.Want.System.ConfigPath = configPath

	cfg, err := config.New(
		config.WithViperConfig(configPath),
	)
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, tc.Want, cfg)
}
