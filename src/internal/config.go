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

// Config represents the user's preferences.
type Config struct {
	PomodoroMinutes     int    `yaml:"pomodoro_mins"`
	PomodoroMessage     string `yaml:"pomodoro_msg"`
	ShortBreakMinutes   int    `yaml:"short_break_mins"`
	ShortBreakMessage   string `yaml:"short_break_msg"`
	LongBreakMinutes    int    `yaml:"long_break_mins"`
	LongBreakMessage    string `yaml:"long_break_msg"`
	LongBreakInterval   int    `yaml:"long_break_interval"`
	Notify              bool   `yaml:"notify"`
	AutoStartPomorodo   bool   `yaml:"auto_start_pomodoro"`
	AutoStartBreak      bool   `yaml:"auto_start_break"`
	TwentyFourHourClock bool   `yaml:"24hr_clock"`
}

const (
	pomodoroMinutes   = 25
	shortBreakMinutes = 5
	longBreakMinutes  = 15
	longBreakInterval = 4
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

// configPrompt is the prompt for the app's
// initial configuration.
func (c *Config) prompt(path string) {
	fmt.Println(ascii)

	fmt.Printf("Your preferences will be saved to: %s\n", path)

	fmt.Printf(`
- Follow the prompts below to configure Focus for the first time.
- Type your preferred value, or press ENTER to accept the defaults.
- Edit the configuration file to change any settings, or use command-line arguments (see the --help flag)
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

		break
	}
}

// save stores the current configuration to disk.
func (c *Config) save(path string) error {
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

// Init initialises the app configuration.
// If the config file does not exist,.it prompts the user
// and saves the inputted preferences in a config file.
func (c *Config) Init() error {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return err
	}

	appRoot := filepath.Join(homeDir, configPath)
	pathToConfig := filepath.Join(appRoot, configFileName)

	_, err = os.Stat(pathToConfig)
	if err != nil && !errors.Is(err, os.ErrExist) {
		return c.new(pathToConfig)
	}

	return c.get(pathToConfig)
}

// get retrieves an already existing configuration from
// the filesystem.
func (c *Config) get(pathToConfig string) error {
	c.defaults(false)

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

// defaults sets default values for the config.
// The `willPrompt` flag is used to control
// if default values should be set for the
// values that.are requested in the prompt.
func (c *Config) defaults(willPrompt bool) {
	if !willPrompt {
		c.PomodoroMinutes = pomodoroMinutes
		c.ShortBreakMinutes = shortBreakMinutes
		c.LongBreakMinutes = longBreakMinutes
		c.LongBreakInterval = longBreakInterval
	}

	c.AutoStartBreak = false
	c.AutoStartPomorodo = false
	c.Notify = true
	c.PomodoroMessage = "Focus on your task"
	c.ShortBreakMessage = "Take a breather"
	c.LongBreakMessage = "Take a long break"
	c.TwentyFourHourClock = false
}

// new prompts the user to set a configuration
// for the application. The resulting values are saved
// to the filesystem.
func (c *Config) new(pathToConfig string) error {
	c.defaults(true)

	appRoot := filepath.Dir(pathToConfig)

	// Ensure the config directory exists
	err := os.MkdirAll(appRoot, 0750)
	if err != nil {
		return err
	}

	c.prompt(pathToConfig)

	err = c.save(pathToConfig)
	if err != nil {
		return err
	}

	fmt.Printf("\nYour settings have been saved. Thanks for using Focus!\n\n")

	return nil
}
