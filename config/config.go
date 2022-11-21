package config

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"

	"github.com/adrg/xdg"
	"github.com/ayoisaiah/focus/internal/session"
	"github.com/pterm/pterm"
	"github.com/pterm/pterm/putils"
	"github.com/spf13/viper"
	"github.com/urfave/cli/v2"
)

var config = &Config{
	Message:  make(session.Message),
	Duration: make(session.Duration),
}

var once sync.Once

var (
	errReadingInput = errors.New(
		"An error occurred while reading input. Please try again",
	)
	errExpectedInteger = errors.New(
		"Expected an integer that must be greater than zero",
	)
	errInitFailed = errors.New(
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

type Config struct {
	Stderr              io.Writer        `json:"-"`
	Stdout              io.Writer        `json:"-"`
	Stdin               io.Reader        `json:"-"`
	Duration            session.Duration `json:"duration"`
	Message             session.Message  `json:"message"`
	AmbientSound        string           `json:"ambient_sound"`
	PathToConfig        string           `json:"path_to_config"`
	PathToDB            string           `json:"path_to_db"`
	SessionCmd          string           `json:"session_cmd"`
	Tags                []string         `json:"tags"`
	LongBreakInterval   int              `json:"long_break_interval"`
	Notify              bool             `json:"notify"`
	DarkTheme           bool             `json:"dark_theme"`
	TwentyFourHourClock bool             `json:"twenty_four_hour_clock"`
	PlaySoundOnBreak    bool             `json:"ambient_sound_on_break"`
	AutoStartBreak      bool             `json:"auto_start_break"`
	AutoStartWork       bool             `json:"auto_start_work"`
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
	configAmbientSound        = "ambient_sound"
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
	configSessionCmd          = "session_cmd"
	configDarkTheme           = "dark_theme"
)

var (
	configDir      = "focus"
	configFileName = "config.yml"
	dbFile         = "focus.db"
)

func init() {
	if os.Getenv("FOCUS_ENV") == "development" {
		configFileName = "config_dev.yml"
		dbFile = "focus_dev.db"
	}
}

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

// prompt allows the user to state their preferred configuration
// for the most important functions of the program. It is run only
// when a configuration file is not already present (e.g on first run).
func prompt() {
	fmt.Printf("%s\n\n", ascii)

	pterm.Info.Printfln(
		"Your preferences will be saved to: %s\n\n",
		config.PathToConfig,
	)

	_ = putils.BulletListFromString(`Follow the prompts below to configure Focus for the first time.
Type your preferred value, or press ENTER to accept the defaults.
Edit the configuration file (focus edit-config) to change any settings, or use command line arguments (see the --help flag)`, " ").
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
func initConfig() error {
	viper.SetConfigName(configFileName)
	viper.SetConfigType("yaml")

	relPath := filepath.Join(configDir, configFileName)

	pathToConfigFile, err := xdg.ConfigFile(relPath)
	if err != nil {
		pterm.Error.Println(err)
		os.Exit(1)
	}

	config.PathToConfig = pathToConfigFile

	viper.AddConfigPath(filepath.Dir(config.PathToConfig))

	if err := viper.ReadInConfig(); err != nil {
		if errors.As(err, &viper.ConfigFileNotFoundError{}) {
			return create()
		}

		return err
	}

	return nil
}

func set(ctx *cli.Context) {
	config.Stderr = os.Stderr
	config.Stdout = os.Stdout
	config.Stdin = os.Stdin

	pathToDB, err := xdg.DataFile(filepath.Join(configDir, dbFile))
	if err != nil {
		pterm.Error.Printfln("%s: %s", errInitFailed.Error(), err.Error())
		os.Exit(1)
	}

	config.PathToDB = pathToDB

	// set from config file
	config.LongBreakInterval = viper.GetInt(configLongBreakInterval)
	config.AutoStartBreak = viper.GetBool(configAutoStartBreak)
	config.AutoStartWork = viper.GetBool(configAutoStartWork)
	config.Notify = viper.GetBool(configNotify)
	config.TwentyFourHourClock = viper.GetBool(configTwentyFourHourClock)
	config.PlaySoundOnBreak = viper.GetBool(configSoundOnBreak)
	config.AmbientSound = viper.GetString(configAmbientSound)
	config.SessionCmd = viper.GetString(configSessionCmd)
	config.DarkTheme = viper.GetBool(configDarkTheme)
	config.Message[session.Work] = viper.GetString(configWorkMessage)
	config.Message[session.ShortBreak] = viper.GetString(
		configShortBreakMessage,
	)
	config.Message[session.LongBreak] = viper.GetString(configLongBreakMessage)
	config.Duration[session.Work] = viper.GetInt(configWorkMinutes)
	config.Duration[session.ShortBreak] = viper.GetInt(configShortBreakMinutes)
	config.Duration[session.LongBreak] = viper.GetInt(configLongBreakMinutes)

	// set from command-line arguments
	tagArg := ctx.String("tags")

	if tagArg != "" {
		tags := strings.Split(tagArg, ",")
		for i := range tags {
			tags[i] = strings.Trim(tags[i], " ")
		}

		config.Tags = tags
	}

	if ctx.Bool("disable-notification") {
		config.Notify = false
	}

	if ctx.String("sound") != "" {
		if ctx.String("sound") == "off" {
			config.AmbientSound = ""
		} else {
			config.AmbientSound = ctx.String("sound")
		}
	}

	if ctx.String("session-cmd") != "" {
		config.SessionCmd = ctx.String("session-cmd")
	}

	if ctx.Uint("work") > 0 {
		config.Duration[session.Work] = int(ctx.Uint("work"))
	}

	if ctx.Uint("short-break") > 0 {
		config.Duration[session.ShortBreak] = int(ctx.Uint("short-break"))
	}

	if ctx.Uint("long-break") > 0 {
		config.Duration[session.LongBreak] = int(ctx.Uint("long-break"))
	}

	if ctx.Uint("long-break-interval") > 0 {
		config.LongBreakInterval = int(ctx.Uint("long-break-interval"))
	}
}

// create prompts the user to set perferred values
// for key application settings. The results are
// saved to the user's config directory.
func create() error {
	prompt()

	defaults()

	err := viper.WriteConfigAs(config.PathToConfig)
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
func defaults() {
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
	viper.SetDefault(configSessionCmd, "")
	viper.SetDefault(configDarkTheme, true)
}

// Get returns the application configuration.
func Get(ctx *cli.Context) *Config {
	once.Do(func() {
		err := initConfig()
		if err != nil {
			pterm.Error.Printfln("%s: %s", errInitFailed.Error(), err.Error())
			os.Exit(1)
		}

		set(ctx)
	})

	return config
}
