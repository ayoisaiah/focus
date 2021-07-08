package focus

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

const ascii = `
███████╗ ██████╗  ██████╗██╗   ██╗███████╗
██╔════╝██╔═══██╗██╔════╝██║   ██║██╔════╝
█████╗  ██║   ██║██║     ██║   ██║███████╗
██╔══╝  ██║   ██║██║     ██║   ██║╚════██║
██║     ╚██████╔╝╚██████╗╚██████╔╝███████║
╚═╝      ╚═════╝  ╚═════╝ ╚═════╝ ╚══════╝
`

// Config represents the all the environmental variables that should be present
// on start up.
type Config struct {
	PomodoroMinutes   int    `json:"pomodoro_mins"`
	PomodoroMessage   string `json:"pomodoro_msg"`
	ShortBreakMinutes int    `json:"short_break_mins"`
	ShortBreakMessage string `json:"short_break_msg"`
	LongBreakMinutes  int    `json:"long_break_mins"`
	LongBreakMessage  string `json:"long_break_msg"`
	LongBreakInterval int    `json:"long_break_interval"`
}

// Conf represents the application configuration.
var Conf *Config

const (
	pomodoroMinutes   = 25
	shortBreakMinutes = 5
	longBreakMinutes  = 15
	longBreakInterval = 4
	pomodoroMessage   = "Focus on your task"
	shortBreakMessage = "Take a breather"
	longBreakMessage  = "Take a long break"
	configFolder      = ".focus"
	configFileName    = "config.json"
)

// configPrompt is the prompt for the app's
// initial configuration.
func (c *Config) prompt(path string) {
	fmt.Println(ascii)

	fmt.Printf("Your preferences will be saved to: %s\n", path)

	fmt.Printf(`
Follow the prompts below to configure Focus for the first time:
  1. Enter your preferred time lengths in minutes for each session.
  2. Enter your messages.
  3. Run 'focus config' to change your settings at any time.
`)

	reader := bufio.NewReader(os.Stdin)

	fmt.Print("\nPress ENTER to continue")

	_, _ = reader.ReadString('\n')

	for {
		if c.PomodoroMinutes == 0 {
			fmt.Printf("\nPomodoro length in minutes (default: %d): ", pomodoroMinutes)

			num, err := numberPrompt(reader, pomodoroMinutes)
			if err != nil {
				fmt.Println(err)
				continue
			}

			c.PomodoroMinutes = num
		}

		if c.ShortBreakMinutes == 0 {
			fmt.Printf("Short Break in minutes (default: %d): ", shortBreakMinutes)

			num, err := numberPrompt(reader, shortBreakMinutes)
			if err != nil {
				fmt.Println(err)
				continue
			}

			c.ShortBreakMinutes = num
		}

		if c.LongBreakMinutes == 0 {
			fmt.Printf("Long Break in minutes (default: %d): ", longBreakMinutes)

			num, err := numberPrompt(reader, longBreakMinutes)
			if err != nil {
				fmt.Println(err)
				continue
			}

			c.LongBreakMinutes = num
		}

		if c.LongBreakInterval == 0 {
			fmt.Printf("Pomodoro cycles before long break (default: %d): ", longBreakInterval)

			num, err := numberPrompt(reader, longBreakInterval)
			if err != nil {
				fmt.Println(err)
				continue
			}

			c.LongBreakInterval = num
		}

		if c.PomodoroMessage == "" {
			fmt.Printf(
				"Pomodoro message (default: '%s'): ",
				pomodoroMessage,
			)

			input, err := stringPrompt(reader, pomodoroMessage)
			if err != nil {
				fmt.Println(err)
				continue
			}

			c.PomodoroMessage = input
		}

		if c.ShortBreakMessage == "" {
			fmt.Printf(
				"Short Break message (default: '%s'): ",
				shortBreakMessage,
			)

			input, err := stringPrompt(reader, shortBreakMessage)
			if err != nil {
				fmt.Println(err)
				continue
			}

			c.ShortBreakMessage = input
		}

		if c.LongBreakMessage == "" {
			fmt.Printf(
				"Long Break message (default: '%s'): ",
				longBreakMessage,
			)

			input, err := stringPrompt(reader, longBreakMessage)
			if err != nil {
				fmt.Println(err)
				continue
			}

			c.LongBreakMessage = input
		}

		break
	}
}

// save stores the current configuration to disk.
func (c *Config) save(path string) error {
	return saveToDisk(c, path)
}

// newConfig returns a stored config from the filesystem.
// If an existing configuation is not found, it prompts the user
// to set the configuation for the application.
func newConfig() (*Config, error) {
	c := &Config{}

	dir, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}

	path := filepath.Join(dir, configFolder, configFileName)

	b, err := retrieveFromDisk(configFileName)
	if err != nil {
		c.prompt(path)

		err = c.save(path)
		if err != nil {
			return nil, err
		}

		fmt.Printf("\nThanks for using Focus! You can run '%s' to change your settings anytime.\n\n", printColor(yellow, "focus config"))
	} else {
		err = json.Unmarshal(b, c)
		if err != nil {
			return c, err
		}
	}

	return c, nil
}
