// Package config is responsible for setting the program config from
// the config file and command-line arguments
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

	"github.com/pterm/pterm"
	"github.com/spf13/viper"
	"github.com/urfave/cli/v2"

	"github.com/ayoisaiah/focus/internal/session"
)

var once sync.Once

var timerCfg = &TimerConfig{
	Message:  make(session.Message),
	Duration: make(session.Duration),
	Stderr:   os.Stderr,
	Stdout:   os.Stdout,
	Stdin:    os.Stdin,
}

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

const (
	defaultWorkMinutes       = 25
	defaultShortBreakMinutes = 5
	defaultLongBreakMinutes  = 15
	defaultLongBreakInterval = 4
)

const (
	configWorkMinutes         = "work_mins"
	configWorkMessage         = "work_msg"
	configAmbientSound        = "sound"
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

// TimerConfig represents the program configuration derived from the config file
// and command-line arguments.
type TimerConfig struct {
	Stderr              io.Writer        `json:"-"`
	Stdout              io.Writer        `json:"-"`
	Stdin               io.Reader        `json:"-"`
	Duration            session.Duration `json:"duration"`
	Message             session.Message  `json:"message"`
	AmbientSound        string           `json:"sound"`
	PathToConfig        string           `json:"path_to_config"`
	PathToDB            string           `json:"path_to_db"`
	SessionCmd          string           `json:"session_cmd"`
	Tags                []string         `json:"tags"`
	LongBreakInterval   int              `json:"long_break_interval"`
	Notify              bool             `json:"notify"`
	DarkTheme           bool             `json:"dark_theme"`
	TwentyFourHourClock bool             `json:"twenty_four_hour_clock"`
	PlaySoundOnBreak    bool             `json:"sound_on_break"`
	AutoStartBreak      bool             `json:"auto_start_break"`
	AutoStartWork       bool             `json:"auto_start_work"`
}

func numberPrompt(reader *bufio.Reader, defaultVal int) (int, error) {
	input, err := reader.ReadString('\n')
	if err != nil {
		return 0, errReadingInput
	}

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

// prompt allows the user to state their preferred values for the most
// important timer settings. It is run only when a configuration file
// is not already present (e.g on first run).
func prompt() {
	pterm.Printf("%s\n\n", ascii)

	pterm.Info.Printfln(
		"Your preferences will be saved to: %s\n\n",
		timerCfg.PathToConfig,
	)

	_ = pterm.NewBulletListFromString(`Follow the prompts below to configure Focus for the first time.
Type your preferred value, or press ENTER to accept the defaults.
Edit the configuration file (focus edit-config) to change any settings, or use command line arguments (see the --help flag)`, " ").
		Render()

	reader := bufio.NewReader(os.Stdin)

	pterm.Print("Press ENTER to continue")

	_, _ = reader.ReadString('\n')

	for {
		if !viper.IsSet(configWorkMinutes) {
			pterm.Printf(
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
			pterm.Printf(
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
			pterm.Printf(
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
			pterm.Printf(
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

// overrideConfigFromArgs retrieves user-defined configuration set through
// command-line arguments and updates the timer configuration.
func overrideConfigFromArgs(ctx *cli.Context) {
	tagArg := ctx.String("tag")

	if tagArg != "" {
		tags := strings.Split(tagArg, ",")
		for i := range tags {
			tags[i] = strings.Trim(tags[i], " ")
		}

		timerCfg.Tags = tags
	}

	if ctx.Bool("disable-notification") {
		timerCfg.Notify = false
	}

	if ctx.Bool("sound-on-break") {
		timerCfg.PlaySoundOnBreak = true
	}

	if ctx.String("sound") != "" {
		if ctx.String("sound") == "off" {
			timerCfg.AmbientSound = ""
		} else {
			timerCfg.AmbientSound = ctx.String("sound")
		}
	}

	if ctx.String("session-cmd") != "" {
		timerCfg.SessionCmd = ctx.String("session-cmd")
	}

	if ctx.Uint("work") > 0 {
		timerCfg.Duration[session.Work] = int(ctx.Uint("work"))
	}

	if ctx.Uint("short-break") > 0 {
		timerCfg.Duration[session.ShortBreak] = int(ctx.Uint("short-break"))
	}

	if ctx.Uint("long-break") > 0 {
		timerCfg.Duration[session.LongBreak] = int(ctx.Uint("long-break"))
	}

	if ctx.Uint("long-break-interval") > 0 {
		timerCfg.LongBreakInterval = int(ctx.Uint("long-break-interval"))
	}
}

func warnOnInvalidConfig(configKey string, defaultVal int) {
	pterm.Warning.Printfln(
		"config error: invalid %s value, using default (%d)",
		configKey,
		defaultVal,
	)
}

// updateConfigFromFile retrieves configuration values from the config
// file and uses to update the timer configuration.
func updateConfigFromFile() {
	longBreakInterval := viper.GetInt(configLongBreakInterval)
	if longBreakInterval < 1 {
		warnOnInvalidConfig(configLongBreakInterval, defaultLongBreakInterval)
		longBreakInterval = defaultLongBreakInterval
	}

	workMins := viper.GetInt(configWorkMinutes)
	if workMins < 1 {
		warnOnInvalidConfig(configWorkMinutes, defaultWorkMinutes)
		workMins = defaultWorkMinutes
	}

	shortBreakMins := viper.GetInt(configShortBreakMinutes)
	if shortBreakMins < 1 {
		warnOnInvalidConfig(configShortBreakMinutes, defaultShortBreakMinutes)
		shortBreakMins = defaultShortBreakMinutes
	}

	longBreakMins := viper.GetInt(configLongBreakMinutes)
	if longBreakMins < 1 {
		warnOnInvalidConfig(configLongBreakMinutes, defaultLongBreakMinutes)
		longBreakMins = defaultLongBreakMinutes
	}

	timerCfg.LongBreakInterval = longBreakInterval
	timerCfg.Duration[session.Work] = workMins
	timerCfg.Duration[session.ShortBreak] = shortBreakMins
	timerCfg.Duration[session.LongBreak] = longBreakMins

	timerCfg.AutoStartBreak = viper.GetBool(configAutoStartBreak)
	timerCfg.AutoStartWork = viper.GetBool(configAutoStartWork)
	timerCfg.Notify = viper.GetBool(configNotify)
	timerCfg.TwentyFourHourClock = viper.GetBool(configTwentyFourHourClock)
	timerCfg.PlaySoundOnBreak = viper.GetBool(configSoundOnBreak)
	timerCfg.AmbientSound = viper.GetString(configAmbientSound)
	timerCfg.SessionCmd = viper.GetString(configSessionCmd)

	if viper.IsSet(configDarkTheme) {
		timerCfg.DarkTheme = viper.GetBool(configDarkTheme)
	} else {
		timerCfg.DarkTheme = true
	}

	timerCfg.Message[session.Work] = viper.GetString(configWorkMessage)
	timerCfg.Message[session.ShortBreak] = viper.GetString(
		configShortBreakMessage,
	)
	timerCfg.Message[session.LongBreak] = viper.GetString(
		configLongBreakMessage,
	)
}

// setTimerConfig overrides the default configuaration with user-defined
// settings retrieved from the config file and command-line arguments. The
// latter overrides the former.
func setTimerConfig(ctx *cli.Context) {
	timerCfg.PathToDB = pathToDB

	// set from config file
	updateConfigFromFile()

	// set from command-line arguments
	overrideConfigFromArgs(ctx)
}

// createTimerConfig saves the user's configuration to disk
// after prompting for key settings.
func createTimerConfig() error {
	prompt()

	timerDefaults()

	err := viper.WriteConfigAs(timerCfg.PathToConfig)
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

// timerDefaults sets the timer's default configuration values.
func timerDefaults() {
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
	viper.SetDefault(configAmbientSound, "")
	viper.SetDefault(configSessionCmd, "")
	viper.SetDefault(configDarkTheme, true)
}

// initTimerConfig initialises the application configuration.
// If the config file does not exist,.it prompts the user
// and saves the inputted preferences and defaults in a config file.
func initTimerConfig() error {
	viper.SetConfigName(configFileName)
	viper.SetConfigType("yaml")

	timerCfg.PathToConfig = pathToConfig

	viper.AddConfigPath(filepath.Dir(timerCfg.PathToConfig))

	if err := viper.ReadInConfig(); err != nil {
		if errors.As(err, &viper.ConfigFileNotFoundError{}) {
			return createTimerConfig()
		}

		return err
	}

	return nil
}

// GetTimer initializes and returns the timer configuration.
func GetTimer(ctx *cli.Context) *TimerConfig {
	once.Do(func() {
		err := initTimerConfig()
		if err != nil {
			pterm.Error.Printfln("%s: %s", errInitFailed.Error(), err.Error())
			os.Exit(1)
		}

		setTimerConfig(ctx)
	})

	return timerCfg
}
