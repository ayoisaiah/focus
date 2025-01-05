// Package config is responsible for setting the program config from
// the config file and command-line arguments
package config

import (
	"errors"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/charmbracelet/huh"
	"github.com/pterm/pterm"
	"github.com/pterm/pterm/putils"
	"github.com/spf13/viper"
	"github.com/urfave/cli/v2"
)

type (
	SessType string

	// Message maps a session to a message.
	Message map[SessType]string

	// Duration maps a session to time duration value.
	Duration map[SessType]time.Duration

	// TimerConfig represents the program configuration derived from the config file
	// and command-line arguments.
	TimerConfig struct {
		Duration            Duration `json:"duration"`
		Message             Message  `json:"message"`
		AmbientSound        string   `json:"sound"`
		BreakSound          string   `json:"break_sound"`
		WorkSound           string   `json:"work_sound"`
		PathToConfig        string   `json:"path_to_config"`
		PathToDB            string   `json:"path_to_db"`
		SessionCmd          string   `json:"session_cmd"`
		WorkColor           string   `json:"work_color"`
		ShortBreakColor     string   `json:"short_break_color"`
		LongBreakColor      string   `json:"long_break_color"`
		Tags                []string `json:"tags"`
		LongBreakInterval   int      `json:"long_break_interval"`
		Notify              bool     `json:"notify"`
		DarkTheme           bool     `json:"dark_theme"`
		TwentyFourHourClock bool     `json:"twenty_four_hour_clock"`
		PlaySoundOnBreak    bool     `json:"sound_on_break"`
		AutoStartBreak      bool     `json:"auto_start_break"`
		AutoStartWork       bool     `json:"auto_start_work"`
		Strict              bool     `json:"strict"`
	}
)

const (
	Work       SessType = "Work session"
	ShortBreak SessType = "Short break"
	LongBreak  SessType = "Long break"
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

const (
	defaultWorkColor       = "#B0DB43"
	defaultShortBreakColor = "#12EAEA"
	defaultLongBreakColor  = "#C492B1"
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
	configStrict              = "strict"
	configWorkColor           = "work_color"
	configShortBreakColor     = "short_break_color"
	configLongBreakColor      = "long_break_color"
)

var once sync.Once

var timerCfg = &TimerConfig{
	Message:  make(Message),
	Duration: make(Duration),
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
func prompt() error {
	var (
		workDur           int
		shortBreakDur     int
		longBreakDur      int
		longBreakInterval int
	)

	pterm.Fprintln(Stderr, pterm.Sprintf("%s\n", ascii))

	pterm.Fprintln(
		Stderr,
		pterm.Sprintf(
			"%s: your preferences will be saved to -> %s\n",
			pterm.Green("info"),
			timerCfg.PathToConfig,
		),
	)

	_ = putils.BulletListFromString(`Follow the prompts below to configure Focus for the first time.
Select your preferred value, or press ENTER to accept the defaults.
Edit the config file with 'focus edit-config' to change any settings.`, " ").
		Render()

	form := huh.NewForm(
		huh.NewGroup(
			huh.NewSelect[int]().
				Title("Work length").
				Options(
					huh.NewOption("25 minutes", 25).Selected(true),
					huh.NewOption("35 minutes", 35),
					huh.NewOption("50 minutes", 50),
					huh.NewOption("60 minutes", 60),
					huh.NewOption("90 minutes", 90),
				).
				Value(&workDur),
		),

		huh.NewGroup(
			huh.NewSelect[int]().
				Title("Short break length").
				Options(
					huh.NewOption("5 minutes", 5).Selected(true),
					huh.NewOption("10 minutes", 10),
					huh.NewOption("15 minutes", 15),
					huh.NewOption("20 minutes", 20),
					huh.NewOption("30 minutes", 30),
				).
				Value(&shortBreakDur),
		),

		huh.NewGroup(
			huh.NewSelect[int]().
				Title("Long break length").
				Options(
					huh.NewOption("15 minutes", 15).Selected(true),
					huh.NewOption("30 minutes", 30),
					huh.NewOption("45 minutes", 45),
					huh.NewOption("60 minutes", 60),
				).
				Value(&longBreakDur),
		),

		huh.NewGroup(
			huh.NewSelect[int]().
				Title("Long break interval").
				Options(
					huh.NewOption("4", 4).Selected(true),
					huh.NewOption("6", 6),
					huh.NewOption("8", 8),
				).
				Value(&longBreakInterval),
		),
	)

	err := form.Run()
	if err != nil {
		return err
	}

	viper.Set(configWorkDur, strconv.Itoa(workDur)+"m")
	viper.Set(configShortBreakDur, strconv.Itoa(shortBreakDur)+"m")
	viper.Set(configLongBreakDur, strconv.Itoa(longBreakDur)+"m")
	viper.Set(configLongBreakInterval, longBreakInterval)

	return nil
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

	if ctx.Bool("strict") {
		timerCfg.Strict = true
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
		timerCfg.Duration[Work] = parseTime(
			ctx.String("work"),
			configWorkDur,
			defaultWorkMins,
		)
	}

	if ctx.String("short-break") != "" {
		timerCfg.Duration[ShortBreak] = parseTime(
			ctx.String("short-break"),
			configShortBreakDur,
			defaultShortBreakMins,
		)
	}

	if ctx.String("long-break") != "" {
		timerCfg.Duration[LongBreak] = parseTime(
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
	timerCfg.Duration[Work] = workDur
	timerCfg.Duration[ShortBreak] = shortBreakDur
	timerCfg.Duration[LongBreak] = longBreakDur

	timerCfg.AutoStartBreak = viper.GetBool(configAutoStartBreak)
	timerCfg.AutoStartWork = viper.GetBool(configAutoStartWork)
	timerCfg.Strict = viper.GetBool(configStrict)
	timerCfg.Notify = viper.GetBool(configNotify)
	timerCfg.TwentyFourHourClock = viper.GetBool(configTwentyFourHourClock)
	timerCfg.PlaySoundOnBreak = viper.GetBool(configSoundOnBreak)
	timerCfg.AmbientSound = viper.GetString(configAmbientSound)
	timerCfg.SessionCmd = viper.GetString(configSessionCmd)
	timerCfg.BreakSound = viper.GetString(configBreakSound)
	timerCfg.WorkSound = viper.GetString(configWorkSound)
	timerCfg.WorkColor = viper.GetString(configWorkColor)
	timerCfg.ShortBreakColor = viper.GetString(configShortBreakColor)
	timerCfg.LongBreakColor = viper.GetString(configLongBreakColor)

	if viper.IsSet(configDarkTheme) {
		timerCfg.DarkTheme = viper.GetBool(configDarkTheme)
	} else {
		timerCfg.DarkTheme = true
	}

	timerCfg.Message[Work] = viper.GetString(configWorkMessage)
	timerCfg.Message[ShortBreak] = viper.GetString(
		configShortBreakMessage,
	)
	timerCfg.Message[LongBreak] = viper.GetString(
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
	err := prompt()
	if err != nil {
		return err
	}

	timerDefaults()

	err = viper.WriteConfigAs(timerCfg.PathToConfig)
	if err != nil {
		return err
	}

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
	viper.SetDefault(configStrict, false)
	viper.SetDefault(configWorkColor, defaultWorkColor)
	viper.SetDefault(configShortBreakColor, defaultShortBreakColor)
	viper.SetDefault(configLongBreakColor, defaultLongBreakColor)
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
