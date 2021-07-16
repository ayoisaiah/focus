package main

import (
	"errors"
	"os"
	"path/filepath"

	_ "embed"

	cmd "github.com/ayoisaiah/focus/src"
	"github.com/pterm/pterm"
)

const configPath = ".config/focus"

//go:embed assets/focus-clock.png
var icon []byte

func init() {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		pterm.Error.Println(err)
		os.Exit(1)
	}

	pathToConfigDir := filepath.Join(homeDir, configPath)

	// Ensure the config directory exists
	err = os.MkdirAll(pathToConfigDir, 0750)
	if err != nil {
		pterm.Error.Println(err)
		os.Exit(1)
	}

	pathToIcon := filepath.Join(pathToConfigDir, "icon.png")
	_, err = os.Stat(pathToIcon)

	if err != nil && errors.Is(err, os.ErrNotExist) {
		_ = os.WriteFile(pathToIcon, icon, os.ModePerm)
	}
}

func run(args []string) error {
	return cmd.GetApp().Run(args)
}

func main() {
	err := run(os.Args)
	if err != nil {
		pterm.Error.Println(err)
		os.Exit(1)
	}
}
