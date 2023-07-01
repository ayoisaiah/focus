package config

import (
	"log"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/adrg/xdg"
	"github.com/pterm/pterm"
)

var (
	pathToConfig string
	pathToDB     string
)

func init() {
	//nolint:dogsled // necessary for testing setup
	_, filename, _, _ := runtime.Caller(0)

	dir := path.Join(path.Dir(filename), "..")

	err := os.Chdir(dir)
	if err != nil {
		log.Fatal(err)
	}
}

func TestMain(m *testing.M) {
	// replace focus directory to avoid overriding configuration
	configDir = "focus_test"

	pterm.DisableOutput()

	var err error

	pathToDB, err = xdg.DataFile(filepath.Join(configDir, dbFileName))
	if err != nil {
		log.Fatal(err)
	}

	pathToConfig, err = xdg.ConfigFile(
		filepath.Join(configDir, configFileName),
	)
	if err != nil {
		log.Fatal(err)
	}

	code := m.Run()

	// Remove test config directory
	err = os.RemoveAll(filepath.Dir(pathToConfig))
	if err != nil {
		log.Fatal(err)
	}

	os.Exit(code)
}
