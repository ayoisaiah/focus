package focus

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"gopkg.in/yaml.v2"
)

const ascii = `
███████╗ ██████╗  ██████╗██╗   ██╗███████╗
██╔════╝██╔═══██╗██╔════╝██║   ██║██╔════╝
█████╗  ██║   ██║██║     ██║   ██║███████╗
██╔══╝  ██║   ██║██║     ██║   ██║╚════██║
██║     ╚██████╔╝╚██████╗╚██████╔╝███████║
╚═╝      ╚═════╝  ╚═════╝ ╚═════╝ ╚══════╝
`

// config represents the application configuration.
type config struct {
	PomodoroMinutes   int    `yaml:"pomodoro_mins"`
	PomodoroMessage   string `yaml:"pomodoro_msg"`
	ShortBreakMinutes int    `yaml:"short_break_mins"`
	ShortBreakMessage string `yaml:"short_break_msg"`
	LongBreakMinutes  int    `yaml:"long_break_mins"`
	LongBreakMessage  string `yaml:"long_break_msg"`
	LongBreakInterval int    `yaml:"long_break_interval"`
}

const (
	pomodoroMinutes   = 25
	shortBreakMinutes = 5
	longBreakMinutes  = 15
	longBreakInterval = 4
	pomodoroMessage   = "Focus on your task"
	shortBreakMessage = "Take a breather"
	longBreakMessage  = "Take a long break"
	configPath        = ".config/focus"
	configFileName    = "config.yml"
)

func numberPrompt(reader *bufio.Reader, defaultVal int) (int, error) {
	input, err := reader.ReadString('\n')
	if err != nil {
		return 0, errors.New(errReadingInput)
	}

	reader.Reset(os.Stdin)

	input = strings.TrimSpace(strings.TrimSuffix(input, "\n"))
	if input == "" {
		return defaultVal, nil
	}

	num, err := strconv.Atoi(input)
	if err != nil {
		return 0, errors.New(errExpectedNumber)
	}

	if num <= 0 {
		return 0, errors.New(errExpectPositiveInteger)
	}

	return num, nil
}

func stringPrompt(reader *bufio.Reader, defaultVal string) (string, error) {
	input, err := reader.ReadString('\n')
	if err != nil {
		return "", errors.New(errReadingInput)
	}

	reader.Reset(os.Stdin)

	input = strings.TrimSpace(strings.TrimSuffix(input, "\n"))
	if input == "" {
		input = defaultVal
	}

	return input, nil
}

// configPrompt is the prompt for the app's
// initial configuration.
func (c *config) prompt(path string) {
	fmt.Println(ascii)

	fmt.Printf("Your preferences will be saved to: %s\n", path)

	fmt.Printf(`
- Follow the prompts below to configure Focus for the first time.
- Type your preferred value, or press ENTER to accept the defaults.
- Run 'focus config' to change your settings at any time.
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
			fmt.Printf("Short break length in minutes (default: %d): ", shortBreakMinutes)

			num, err := numberPrompt(reader, shortBreakMinutes)
			if err != nil {
				fmt.Println(err)
				continue
			}

			c.ShortBreakMinutes = num
		}

		if c.LongBreakMinutes == 0 {
			fmt.Printf("Long break length in minutes (default: %d): ", longBreakMinutes)

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
				"Short break message (default: '%s'): ",
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
				"Long break message (default: '%s'): ",
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
func (c *config) save(path string) error {
	file, err := os.Create(path)
	if err != nil {
		return err
	}

	defer func() {
		ferr := file.Close()
		if ferr != nil {
			err = ferr
		}
	}()

	writer := bufio.NewWriter(file)

	b, err := yaml.Marshal(c)
	if err != nil {
		return err
	}

	_, err = writer.Write(b)
	if err != nil {
		return err
	}

	return writer.Flush()
}

// get retrieves an already existing configuration from
// the filesystem.
func (c *config) get() error {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return err
	}

	configRoot := filepath.Join(homeDir, configPath)
	pathToConfig := filepath.Join(configRoot, configFileName)

	b, err := os.ReadFile(pathToConfig)
	if err != nil {
		return err
	}

	err = yaml.Unmarshal(b, c)
	if err != nil {
		return err
	}

	return nil
}

// new prompts the user to set a configuration
// for the application. The resulting values are saved
// to the filesystem.
func (c *config) new() error {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return err
	}

	configRoot := filepath.Join(homeDir, configPath)
	pathToConfig := filepath.Join(configRoot, configFileName)

	// Ensure the config directory exists
	err = os.MkdirAll(configRoot, 0750)
	if err != nil {
		return err
	}

	c.prompt(pathToConfig)

	err = c.save(pathToConfig)
	if err != nil {
		return err
	}

	fmt.Printf("\nYour settings have been saved. Thanks for using Focus!")

	return nil
}
