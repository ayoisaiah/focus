package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/adrg/xdg"
	"github.com/pterm/pterm"
)

var (
	configDir      = "focus"
	configFileName = "config.yml"
	dbFileName     = "focus.db"
	pathToDB       string
	pathToConfig   string
)

func GetDir() string {
	return configDir
}

func InitializePaths() {
	focusEnv := strings.TrimSpace(os.Getenv("FOCUS_ENV"))
	if focusEnv != "" {
		configFileName = fmt.Sprintf("config_%s.yml", focusEnv)
		dbFileName = fmt.Sprintf("focus_%s.db", focusEnv)
	}

	var err error

	relPath := filepath.Join(configDir, configFileName)

	pathToConfig, err = xdg.ConfigFile(relPath)
	if err != nil {
		pterm.Error.Println(err)
		os.Exit(1)
	}

	pathToDB, err = xdg.DataFile(filepath.Join(configDir, dbFileName))
	if err != nil {
		pterm.Error.Println(err)
		os.Exit(1)
	}
}
