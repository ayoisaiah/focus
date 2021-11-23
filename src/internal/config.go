package focus

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/adrg/xdg"
	"github.com/pterm/pterm"
	"gopkg.in/yaml.v3"
)

const (
	errReadingInput = Error(
		"An error occurred while reading input. Please try again",
	)
	errExpectedInteger = Error(
		"Expected an integer that must be greater than zero",
	)
	errInitFailed = Error(
		"Unable to initialise Focus settings from the configuration file",
	)
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
	WorkMinutes         int    `yaml:"work_mins"`
	WorkMessage         string `yaml:"work_msg"`
	ShortBreakMinutes   int    `yaml:"short_break_mins"`
	ShortBreakMessage   string `yaml:"short_break_msg"`
	LongBreakMinutes    int    `yaml:"long_break_mins"`
	LongBreakMessage    string `yaml:"long_break_msg"`
	LongBreakInterval   int    `yaml:"long_break_interval"`
	Notify              bool   `yaml:"notify"`
	AutoStartWork       bool   `yaml:"auto_start_work"`
	AutoStartBreak      bool   `yaml:"auto_start_break"`
	TwentyFourHourClock bool   `yaml:"24hr_clock"`
	Sound               string `yaml:"sound"`
	SoundOnBreak        bool   `yaml:"sound_on_break"`
}

const (
	defaultWorkMinutes       = 25
	defaultShortBreakMinutes = 5
	defaultLongBreakMinutes  = 15
	defaultLongBreakInterval = 4
)

var (
	configDir      = "focus"
	configFileName = "config.yml"
)

func numberPrompt(reader *bufio.Reader, defaultVal int) (int, error) {
	input, err := reader.ReadString('\n')
	if err != nil {
		return 0, errReadingInput
	}

	reader.Reset(os.Stdin)

	input = strings.TrimSpace(strings.TrimSuffix(input, "\n"))
	if input == "" {
		return defaultVal, nil
	}

	num, err := strconv.Atoi(input)
	if err != nil {
		return 0, errExpectedInteger
	}

	if num <= 0 {
		return 0, errExpectedInteger
	}

	return num, nil
}

// configPrompt is the prompt for the app's initial configuration.
func (c *Config) prompt(path string) {
	fmt.Println(ascii)

	pterm.Info.Printfln("Your preferences will be saved to: %s\n\n", path)

	_ = pterm.NewBulletListFromString(`Follow the prompts below to configure Focus for the first time.
Type your preferred value, or press ENTER to accept the defaults.
Edit the configuration file to change any settings, or use command line arguments (see the --help flag)`, " ").
		Render()

	reader := bufio.NewReader(os.Stdin)

	fmt.Print("Press ENTER to continue")

	_, _ = reader.ReadString('\n')

	for {
		if c.WorkMinutes == 0 {
			fmt.Printf(
				"\nWork length in minutes (default: %s): ",
				pterm.Green(defaultWorkMinutes),
			)

			num, err := numberPrompt(reader, defaultWorkMinutes)
			if err != nil {
				pterm.Error.Println(err)
				continue
			}

			c.WorkMinutes = num
		}

		if c.ShortBreakMinutes == 0 {
			fmt.Printf(
				"Short break length in minutes (default: %s): ",
				pterm.Green(defaultShortBreakMinutes),
			)

			num, err := numberPrompt(reader, defaultShortBreakMinutes)
			if err != nil {
				pterm.Error.Println(err)
				continue
			}

			c.ShortBreakMinutes = num
		}

		if c.LongBreakMinutes == 0 {
			fmt.Printf(
				"Long break length in minutes (default: %s): ",
				pterm.Green(defaultLongBreakMinutes),
			)

			num, err := numberPrompt(reader, defaultLongBreakMinutes)
			if err != nil {
				pterm.Error.Println(err)
				continue
			}

			c.LongBreakMinutes = num
		}

		if c.LongBreakInterval == 0 {
			fmt.Printf(
				"Work sessions before long break (default: %s): ",
				pterm.Green(defaultLongBreakInterval),
			)

			num, err := numberPrompt(reader, defaultLongBreakInterval)
			if err != nil {
				pterm.Error.Println(err)
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

// init initialises the application configuration.
// If the config file does not exist,.it prompts the user
// and saves the inputted preferences in a config file.
func (c *Config) init() error {
	relPath := filepath.Join(configDir, configFileName)

	pathToConfigFile, err := xdg.ConfigFile(relPath)
	if err != nil {
		return err
	}

	_, err = xdg.SearchConfigFile(relPath)
	if err != nil {
		return c.create(pathToConfigFile)
	}

	return c.get(pathToConfigFile)
}

// get retrieves an already existing configuration from
// the filesystem.
func (c *Config) get(pathToConfig string) error {
	b, err := os.ReadFile(pathToConfig)
	if err != nil {
		return err
	}

	var nc Config

	err = yaml.Unmarshal(b, &nc)
	if err != nil {
		pterm.Warning.Printfln(
			"Unable to initialise Focus from config file due to errors: %v. Using default settings.",
			err,
		)

		return nil
	}

	// Account for empty config files
	if nc == (Config{}) {
		pterm.Warning.Printfln(
			"Unable to initialise Focus from empty config file. Using default settings.",
		)

		return nil
	}

	*c = nc

	return c.save(pathToConfig)
}

// defaults sets default values for the config object.
func (c *Config) defaults() {
	c.WorkMinutes = defaultWorkMinutes
	c.ShortBreakMinutes = defaultShortBreakMinutes
	c.LongBreakMinutes = defaultLongBreakMinutes
	c.LongBreakInterval = defaultLongBreakInterval
	c.AutoStartBreak = true
	c.AutoStartWork = false
	c.Notify = true
	c.WorkMessage = "Focus on your task"
	c.ShortBreakMessage = "Take a breather"
	c.LongBreakMessage = "Take a long break"
	c.TwentyFourHourClock = false
	c.SoundOnBreak = false
}

// create prompts the user to set perferred values
// for key application settings. The results are
// saved to the filesystem to facilitate reuse.
func (c *Config) create(pathToConfig string) error {
	c.prompt(pathToConfig)

	err := c.save(pathToConfig)
	if err != nil {
		return err
	}

	fmt.Println()
	pterm.Success.Printfln(
		"Your settings have been saved. Thanks for using Focus!\n\n",
	)

	return nil
}

// NewConfig returns the application configuration.
func NewConfig() (*Config, error) {
	c := &Config{}

	c.defaults()

	err := c.init()
	if err != nil {
		return nil, fmt.Errorf("%s: %w", errInitFailed.Error(), err)
	}

	return c, nil
}
