package focus

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/adrg/xdg"
	"github.com/pterm/pterm"
	"github.com/spf13/viper"
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
╚═╝      ╚═════╝  ╚═════╝ ╚═════╝ ╚══════╝`

// Config represents the program's configurable properties.
type Config struct {
	Sound               string
	WorkMessage         string
	ShortBreakMessage   string
	LongBreakMessage    string
	ShortBreakMinutes   int
	LongBreakMinutes    int
	LongBreakInterval   int
	WorkMinutes         int
	AutoStartWork       bool
	AutoStartBreak      bool
	TwentyFourHourClock bool
	Notify              bool
	SoundOnBreak        bool
}

const (
	defaultWorkMinutes       = 25
	defaultShortBreakMinutes = 5
	defaultLongBreakMinutes  = 15
	defaultLongBreakInterval = 4
)

const (
	configWorkMinutes         = "work_mins"
	configWorkMessage         = "work_msg"
	configSound               = "sound"
	configShortBreakMinutes   = "short_break_mins"
	configShortBreakMessage   = "short_break_msg"
	configLongBreakMinutes    = "long_break_mins"
	configLongBreakMessage    = "long_break_msg"
	configLongBreakInterval   = "long_break_interval"
	configAutoStartWork       = "auto_start_work"
	configAutoStartBreak      = "auto_start_break"
	configNotify              = "notify"
	configSoundOnBreak        = "sound_on_break"
	configTwentyFourHourClock = "24hr_clock"
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

// configPrompt allows the user to state their preferred configuration
// for the most important functions of the program. It is run only
// when a configuration file is not already present (e.g on first run).
func (c *Config) prompt(path string) {
	fmt.Printf("%s\n\n", ascii)

	pterm.Info.Printfln("Your preferences will be saved to: %s\n\n", path)

	_ = pterm.NewBulletListFromString(`Follow the prompts below to configure Focus for the first time.
Type your preferred value, or press ENTER to accept the defaults.
Edit the configuration file to change any settings, or use command line arguments (see the --help flag)`, " ").
		Render()

	reader := bufio.NewReader(os.Stdin)

	fmt.Print("Press ENTER to continue")

	_, _ = reader.ReadString('\n')

	for {
		if !viper.IsSet(configWorkMinutes) {
			fmt.Printf(
				"\nWork length in minutes (default: %s): ",
				pterm.Green(defaultWorkMinutes),
			)

			num, err := numberPrompt(reader, defaultWorkMinutes)
			if err != nil {
				pterm.Error.Println(err)
				continue
			}

			viper.Set(configWorkMinutes, num)
		}

		if !viper.IsSet(configShortBreakMinutes) {
			fmt.Printf(
				"Short break length in minutes (default: %s): ",
				pterm.Green(defaultShortBreakMinutes),
			)

			num, err := numberPrompt(reader, defaultShortBreakMinutes)
			if err != nil {
				pterm.Error.Println(err)
				continue
			}

			viper.Set(configShortBreakMinutes, num)
		}

		if !viper.IsSet(configLongBreakMinutes) {
			fmt.Printf(
				"Long break length in minutes (default: %s): ",
				pterm.Green(defaultLongBreakMinutes),
			)

			num, err := numberPrompt(reader, defaultLongBreakMinutes)
			if err != nil {
				pterm.Error.Println(err)
				continue
			}

			viper.Set(configLongBreakMinutes, num)
		}

		if !viper.IsSet(configLongBreakInterval) {
			fmt.Printf(
				"Work sessions before long break (default: %s): ",
				pterm.Green(defaultLongBreakInterval),
			)

			num, err := numberPrompt(reader, defaultLongBreakInterval)
			if err != nil {
				pterm.Error.Println(err)
				continue
			}

			viper.Set(configLongBreakInterval, num)
		}

		break
	}
}

// init initialises the application configuration.
// If the config file does not exist,.it prompts the user
// and saves the inputted preferences and defaults in a config file.
func (c *Config) init() error {
	relPath := filepath.Join(configDir, configFileName)

	viper.SetConfigName(configFileName)
	viper.SetConfigType("yaml")

	pathToConfigFile, err := xdg.ConfigFile(relPath)
	if err != nil {
		return err
	}

	viper.AddConfigPath(filepath.Dir(pathToConfigFile))

	if err := viper.ReadInConfig(); err != nil {
		if errors.As(err, &viper.ConfigFileNotFoundError{}) {
			return c.create(pathToConfigFile)
		}

		return err
	}

	return nil
}

func (c *Config) set() {
	c.WorkMinutes = viper.GetInt(configWorkMinutes)
	c.ShortBreakMinutes = viper.GetInt(configShortBreakMinutes)
	c.LongBreakMinutes = viper.GetInt(configLongBreakMinutes)
	c.LongBreakInterval = viper.GetInt(configLongBreakInterval)
	c.AutoStartBreak = viper.GetBool(configAutoStartBreak)
	c.AutoStartWork = viper.GetBool(configAutoStartWork)
	c.Notify = viper.GetBool(configNotify)
	c.WorkMessage = viper.GetString(configWorkMessage)
	c.ShortBreakMessage = viper.GetString(configShortBreakMessage)
	c.LongBreakMessage = viper.GetString(configLongBreakMessage)
	c.TwentyFourHourClock = viper.GetBool(configTwentyFourHourClock)
	c.SoundOnBreak = viper.GetBool(configSoundOnBreak)
	c.Sound = viper.GetString(configSound)
}

// create prompts the user to set perferred values
// for key application settings. The results are
// saved to the user's config directory.
func (c *Config) create(pathToConfig string) error {
	c.prompt(pathToConfig)

	c.defaults()

	err := viper.WriteConfigAs(pathToConfig)
	if err != nil {
		fmt.Println(err)
		return err
	}

	fmt.Println()
	pterm.Success.Printfln(
		"Your settings have been saved. Thanks for using Focus!\n\n",
	)

	return nil
}

// defaults sets program's default configuration values.
func (c *Config) defaults() {
	viper.SetDefault(configWorkMinutes, defaultWorkMinutes)
	viper.SetDefault(configWorkMessage, "Focus on your task")
	viper.SetDefault(configShortBreakMessage, "Take a breather")
	viper.SetDefault(configShortBreakMinutes, defaultShortBreakMinutes)
	viper.SetDefault(configLongBreakMessage, "Take a long break")
	viper.SetDefault(configLongBreakMinutes, defaultLongBreakMinutes)
	viper.SetDefault(configLongBreakInterval, defaultLongBreakInterval)
	viper.SetDefault(configAutoStartBreak, true)
	viper.SetDefault(configAutoStartWork, false)
	viper.SetDefault(configNotify, true)
	viper.SetDefault(configSoundOnBreak, false)
}

// NewConfig returns the application configuration.
func NewConfig() (*Config, error) {
	c := &Config{}

	err := c.init()
	if err != nil {
		return nil, fmt.Errorf("%s: %w", errInitFailed.Error(), err)
	}

	c.set()

	return c, nil
}
