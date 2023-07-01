package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/spf13/viper"
	"github.com/urfave/cli/v2"

	"github.com/ayoisaiah/focus/internal/session"
)

type ConfigTest struct {
	Name       string
	ConfigFile string
	Expected   TimerConfig
}

var configs = []ConfigTest{
	{
		Name:       "Normal config",
		ConfigFile: "config1.yml",
		Expected: TimerConfig{
			Duration: map[session.Name]int{
				session.Work:       50,
				session.ShortBreak: 10,
				session.LongBreak:  30,
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
		},
	},
	{
		Name:       "No config (defaults)",
		ConfigFile: "",
		Expected: TimerConfig{
			Duration: map[session.Name]int{
				session.Work:       25,
				session.ShortBreak: 5,
				session.LongBreak:  15,
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
		},
	},
	{
		Name:       "invalid config",
		ConfigFile: "config2.yml",
		Expected: TimerConfig{
			Duration: map[session.Name]int{
				session.Work:       25,
				session.ShortBreak: 5,
				session.LongBreak:  15,
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

func TestTimerConfig(t *testing.T) {
	for _, tc := range configs {
		t.Run(tc.Name, func(t *testing.T) {
			viper.Reset()

			tc.Expected.PathToConfig = pathToConfig
			tc.Expected.PathToDB = pathToDB

			if tc.ConfigFile == "" {
				err := os.Remove(pathToConfig)
				if err != nil {
					t.Fatal(err)
				}
			} else {
				err := copyFile(
					filepath.Join("testdata", tc.ConfigFile),
					pathToConfig,
				)
				if err != nil {
					t.Fatal(err)
				}
			}

			ctx := cli.NewContext(&cli.App{}, nil, nil)

			// reset the config
			timerCfg = nil

			result := GetTimer(ctx)

			if diff := cmp.Diff(
				result,
				&tc.Expected,
				cmpopts.IgnoreFields(
					TimerConfig{},
					"Stderr",
					"Stdout",
					"Stdin",
				),
			); diff != "" {
				t.Errorf(
					"TestTimerConfig(): [%s] mismatch (-want +got):\n%s",
					tc.Name,
					diff,
				)
			}
		})
	}
}
