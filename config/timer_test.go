package config

import (
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/spf13/viper"
	"github.com/urfave/cli/v2"

	"github.com/ayoisaiah/focus/session"
)

type TimerTest struct {
	Name       string
	ConfigFile string
	PromptFile string
	Expected   TimerConfig
}

var timerTestCases = []TimerTest{
	{
		Name:       "Normal config",
		ConfigFile: "config1.yml",
		Expected: TimerConfig{
			Duration: map[session.Name]time.Duration{
				session.Work:       50 * time.Minute,
				session.ShortBreak: 10 * time.Minute,
				session.LongBreak:  30 * time.Minute,
			},
			Message: map[session.Name]string{
				session.Work:       "Focus on your task",
				session.ShortBreak: "Take a breather",
				session.LongBreak:  "Take a long break",
			},
			LongBreakInterval:   4,
			Notify:              true,
			DarkTheme:           true,
			TwentyFourHourClock: false,
			PlaySoundOnBreak:    false,
			AutoStartBreak:      true,
			AutoStartWork:       false,
			WorkSound:           "loud_bell",
			BreakSound:          "bell",
		},
	},
	{
		Name:       "No config (accept all defaults)",
		ConfigFile: "",
		PromptFile: "defaults.txt",
		Expected: TimerConfig{
			Duration: map[session.Name]time.Duration{
				session.Work:       25 * time.Minute,
				session.ShortBreak: 5 * time.Minute,
				session.LongBreak:  15 * time.Minute,
			},
			Message: map[session.Name]string{
				session.Work:       "Focus on your task",
				session.ShortBreak: "Take a breather",
				session.LongBreak:  "Take a long break",
			},
			LongBreakInterval:   4,
			Notify:              true,
			DarkTheme:           true,
			TwentyFourHourClock: false,
			PlaySoundOnBreak:    false,
			AutoStartBreak:      true,
			AutoStartWork:       false,
			WorkSound:           "loud_bell",
			BreakSound:          "bell",
		},
	},
	{
		Name:       "No config (prompt with custom values)",
		ConfigFile: "",
		PromptFile: "prompt.txt",
		Expected: TimerConfig{
			Duration: map[session.Name]time.Duration{
				session.Work:       40 * time.Minute,
				session.ShortBreak: 12 * time.Minute,
				session.LongBreak:  22 * time.Minute,
			},
			Message: map[session.Name]string{
				session.Work:       "Focus on your task",
				session.ShortBreak: "Take a breather",
				session.LongBreak:  "Take a long break",
			},
			LongBreakInterval:   5,
			Notify:              true,
			DarkTheme:           true,
			TwentyFourHourClock: false,
			PlaySoundOnBreak:    false,
			AutoStartBreak:      true,
			AutoStartWork:       false,
			WorkSound:           "loud_bell",
			BreakSound:          "bell",
		},
	},
	{
		Name:       "invalid config",
		ConfigFile: "config2.yml",
		PromptFile: "",
		Expected: TimerConfig{
			Duration: map[session.Name]time.Duration{
				session.Work:       25 * time.Minute,
				session.ShortBreak: 5 * time.Minute,
				session.LongBreak:  15 * time.Minute,
			},
			Message: map[session.Name]string{
				session.Work:       "Focus on your task",
				session.ShortBreak: "Take a breather",
				session.LongBreak:  "Take a long break",
			},
			LongBreakInterval:   4,
			Notify:              true,
			DarkTheme:           true,
			TwentyFourHourClock: false,
			PlaySoundOnBreak:    false,
			AutoStartBreak:      false,
			AutoStartWork:       false,
			WorkSound:           "",
			BreakSound:          "",
		},
	},
}

func copyFile(src, dest string) error {
	input, err := os.ReadFile(src)
	if err != nil {
		return err
	}

	err = os.WriteFile(dest, input, 0o644)
	if err != nil {
		return err
	}

	return nil
}

func resetTimerConfig() {
	timerCfg = &TimerConfig{
		Message:  make(session.Message),
		Duration: make(session.Duration),
	}

	once = sync.Once{}

	viper.Reset()
}

func TestTimer(t *testing.T) {
	for _, tc := range timerTestCases {
		t.Run(tc.Name, func(t *testing.T) {
			resetTimerConfig()

			tc.Expected.PathToConfig = configFilePath
			tc.Expected.PathToDB = dbFilePath

			if tc.ConfigFile == "" {
				_ = os.Remove(configFilePath)
			} else {
				err := copyFile(
					filepath.Join("testdata", tc.ConfigFile),
					configFilePath,
				)
				if err != nil {
					t.Fatal(err)
				}
			}

			ctx := cli.NewContext(&cli.App{}, nil, nil)

			oldStdin := os.Stdin

			if tc.PromptFile != "" {
				f, err := os.Open(filepath.Join("testdata", tc.PromptFile))
				if err != nil {
					t.Fatal(err)
				}

				os.Stdin = f
			}

			result := Timer(ctx)

			// restore stdin
			os.Stdin = oldStdin

			if diff := cmp.Diff(
				result,
				&tc.Expected,
			); diff != "" {
				t.Errorf(
					"TestTimerConfig(): [%s] mismatch (-got +want):\n%s",
					tc.Name,
					diff,
				)
			}
		})
	}
}
