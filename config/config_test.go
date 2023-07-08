package config

import (
	"log"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/pterm/pterm"
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

	InitializePaths()

	pterm.DisableOutput()

	var err error

	code := m.Run()

	// Cleanup test directory
	err = os.RemoveAll(filepath.Dir(pathToConfig))
	if err != nil {
		log.Fatal(err)
	}

	os.Exit(code)
}
