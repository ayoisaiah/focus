// Package config is responsible for setting the program config from
// the config file and command-line arguments
package config

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/pterm/pterm"
	"github.com/spf13/viper"
	"github.com/urfave/cli/v2"

	"github.com/ayoisaiah/focus/session"
)

var once sync.Once

var timerCfg = &TimerConfig{
	Message:  make(session.Message),
	Duration: make(session.Duration),
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
	defaultWorkMins          = 25
	defaultShortBreakMins    = 5
	defaultLongBreakMins     = 15
	defaultLongBreakInterval = 4
)

const SoundOff = "off"

// Legacy config.
const (
	legacyWorkMins       = "work_mins"
	legacyShortBreakMins = "short_break_mins"
	legacyLongBreakMins  = "long_break_mins"
)

const (
	configWorkDur             = "work_duration"
	configWorkMessage         = "work_msg"
	configAmbientSound        = "sound"
	configShortBreakDur       = "short_break_duration"
	configShortBreakMessage   = "short_break_msg"
	configLongBreakDur        = "long_break_duration"
	configLongBreakMessage    = "long_break_msg"
	configLongBreakInterval   = "long_break_interval"
	configAutoStartWork       = "auto_start_work"
	configAutoStartBreak      = "auto_start_break"
	configNotify              = "notify"
	configSoundOnBreak        = "sound_on_break"
	configTwentyFourHourClock = "24hr_clock"
	configSessionCmd          = "session_cmd"
	configDarkTheme           = "dark_theme"
	configBreakSound          = "break_sound"
	configWorkSound           = "work_sound"
)

// TimerConfig represents the program configuration derived from the config file
// and command-line arguments.
type TimerConfig struct {
	Duration            session.Duration `json:"duration"`
	Message             session.Message  `json:"message"`
	AmbientSound        string           `json:"sound"`
	BreakSound          string           `json:"break_sound"`
	WorkSound           string           `json:"work_sound"`
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

func parseTime(s, key string, d int) time.Duration {
	_, err := strconv.Atoi(s)
	if err == nil {
		s += "m"
	}

	dur, err := time.ParseDuration(s)
	if err != nil {
		warnOnInvalidConfig(key, d)

		dur = time.Duration(d) * time.Minute
	}

	return dur
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
		if !viper.IsSet(configWorkDur) {
			pterm.Printf(
				"\nWork length in minutes (default: %s): ",
				pterm.Green(defaultWorkMins),
			)

			num, err := numberPrompt(reader, defaultWorkMins)
			if err != nil {
				pterm.Error.Println(err)
				continue
			}

			viper.Set(configWorkDur, strconv.Itoa(num)+"m")
		}

		if !viper.IsSet(configShortBreakDur) {
			pterm.Printf(
				"Short break length in minutes (default: %s): ",
				pterm.Green(defaultShortBreakMins),
			)

			num, err := numberPrompt(reader, defaultShortBreakMins)
			if err != nil {
				pterm.Error.Println(err)
				continue
			}

			viper.Set(configShortBreakDur, strconv.Itoa(num)+"m")
		}

		if !viper.IsSet(configLongBreakDur) {
			pterm.Printf(
				"Long break length in minutes (default: %s): ",
				pterm.Green(defaultLongBreakMins),
			)

			num, err := numberPrompt(reader, defaultLongBreakMins)
			if err != nil {
				pterm.Error.Println(err)
				continue
			}

			viper.Set(configLongBreakDur, strconv.Itoa(num)+"m")
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

	timerCfg.PlaySoundOnBreak = ctx.Bool("sound-on-break")

	ambientSound := ctx.String("sound")
	if ambientSound != "" {
		if ambientSound == SoundOff {
			timerCfg.AmbientSound = ""
		} else {
			timerCfg.AmbientSound = ambientSound
		}
	}

	breakSound := ctx.String("break-sound")
	if breakSound != "" {
		if breakSound == SoundOff {
			timerCfg.BreakSound = ""
		} else {
			timerCfg.BreakSound = breakSound
		}
	}

	workSound := ctx.String("work-sound")
	if workSound != "" {
		if workSound == SoundOff {
			timerCfg.WorkSound = ""
		} else {
			timerCfg.WorkSound = workSound
		}
	}

	if ctx.String("session-cmd") != "" {
		timerCfg.SessionCmd = ctx.String("session-cmd")
	}

	if ctx.String("work") != "" {
		timerCfg.Duration[session.Work] = parseTime(
			ctx.String("work"),
			configWorkDur,
			defaultWorkMins,
		)
	}

	if ctx.String("short-break") != "" {
		timerCfg.Duration[session.ShortBreak] = parseTime(
			ctx.String("short-break"),
			configShortBreakDur,
			defaultShortBreakMins,
		)
	}

	if ctx.String("long-break") != "" {
		timerCfg.Duration[session.LongBreak] = parseTime(
			ctx.String("short-break"),
			configShortBreakDur,
			defaultShortBreakMins,
		)
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

	workDurConfig := viper.GetString(legacyWorkMins)

	if viper.GetString(configWorkDur) != "" {
		workDurConfig = viper.GetString(configWorkDur)
	}

	workDur := parseTime(workDurConfig, configWorkDur, defaultWorkMins)

	shortBreakDurConfig := viper.GetString(legacyShortBreakMins)

	if viper.GetString(configShortBreakDur) != "" {
		shortBreakDurConfig = viper.GetString(configShortBreakDur)
	}

	shortBreakDur := parseTime(
		shortBreakDurConfig,
		configShortBreakDur,
		defaultShortBreakMins,
	)

	longBreakDurConfig := viper.GetString(legacyLongBreakMins)

	if viper.GetString(configLongBreakDur) != "" {
		longBreakDurConfig = viper.GetString(configLongBreakDur)
	}

	longBreakDur := parseTime(
		longBreakDurConfig,
		configLongBreakDur,
		defaultLongBreakMins,
	)

	timerCfg.LongBreakInterval = longBreakInterval
	timerCfg.Duration[session.Work] = workDur
	timerCfg.Duration[session.ShortBreak] = shortBreakDur
	timerCfg.Duration[session.LongBreak] = longBreakDur

	timerCfg.AutoStartBreak = viper.GetBool(configAutoStartBreak)
	timerCfg.AutoStartWork = viper.GetBool(configAutoStartWork)
	timerCfg.Notify = viper.GetBool(configNotify)
	timerCfg.TwentyFourHourClock = viper.GetBool(configTwentyFourHourClock)
	timerCfg.PlaySoundOnBreak = viper.GetBool(configSoundOnBreak)
	timerCfg.AmbientSound = viper.GetString(configAmbientSound)
	timerCfg.SessionCmd = viper.GetString(configSessionCmd)
	timerCfg.BreakSound = viper.GetString(configBreakSound)
	timerCfg.WorkSound = viper.GetString(configWorkSound)

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
	timerCfg.PathToDB = dbFilePath

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
	viper.SetDefault(configWorkDur, defaultWorkMins*time.Minute)
	viper.SetDefault(configWorkMessage, "Focus on your task")
	viper.SetDefault(configShortBreakMessage, "Take a breather")
	viper.SetDefault(configShortBreakDur, defaultShortBreakMins*time.Minute)
	viper.SetDefault(configLongBreakMessage, "Take a long break")
	viper.SetDefault(configLongBreakDur, defaultLongBreakMins*time.Minute)
	viper.SetDefault(configLongBreakInterval, defaultLongBreakInterval)
	viper.SetDefault(configAutoStartBreak, true)
	viper.SetDefault(configAutoStartWork, false)
	viper.SetDefault(configNotify, true)
	viper.SetDefault(configSoundOnBreak, false)
	viper.SetDefault(configAmbientSound, "")
	viper.SetDefault(configSessionCmd, "")
	viper.SetDefault(configDarkTheme, true)
	viper.SetDefault(configBreakSound, "bell")
	viper.SetDefault(configWorkSound, "loud_bell")
}

// initTimerConfig initialises the application configuration.
// If the config file does not exist,.it prompts the user
// and saves the inputted preferences and defaults in a config file.
func initTimerConfig() error {
	viper.SetConfigName(configFileName)
	viper.SetConfigType("yaml")

	timerCfg.PathToConfig = configFilePath

	viper.AddConfigPath(filepath.Dir(timerCfg.PathToConfig))

	if err := viper.ReadInConfig(); err != nil {
		if errors.As(err, &viper.ConfigFileNotFoundError{}) {
			return createTimerConfig()
		}

		return err
	}

	return nil
}

// Timer initializes and returns the timer configuration.
func Timer(ctx *cli.Context) *TimerConfig {
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
