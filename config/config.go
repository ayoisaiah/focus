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
	statusFileName = "status.json"
	dbFilePath     string
	configFilePath string
	statusFilePath string
)

func GetDir() string {
	return configDir
}

func GetDBFilePath() string {
	return dbFilePath
}

func GetStatusFilePath() string {
	return statusFilePath
}

func InitializePaths() {
	focusEnv := strings.TrimSpace(os.Getenv("FOCUS_ENV"))
	if focusEnv != "" {
		configFileName = fmt.Sprintf("config_%s.yml", focusEnv)
		dbFileName = fmt.Sprintf("focus_%s.db", focusEnv)
		statusFileName = fmt.Sprintf("status_%s.json", focusEnv)
	}

	var err error

	relPath := filepath.Join(configDir, configFileName)

	configFilePath, err = xdg.ConfigFile(relPath)
	if err != nil {
		pterm.Error.Println(err)
		os.Exit(1)
	}

	dataDir, err := xdg.DataFile(configDir)
	if err != nil {
		pterm.Error.Println(err)
		os.Exit(1)
	}

	dbFilePath = filepath.Join(dataDir, dbFileName)

	statusFilePath = filepath.Join(dataDir, statusFileName)
}
