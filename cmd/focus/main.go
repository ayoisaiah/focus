package main

import (
	"os"
	"path/filepath"

	_ "embed"

	"github.com/adrg/xdg"
	cmd "github.com/ayoisaiah/focus/src"
	"github.com/pterm/pterm"
)

const (
	configDir = "focus"
)

//go:embed assets/focus-clock.png
var icon []byte

func init() {
	relPath := filepath.Join(configDir, "icon.png")

	pathToIcon, err := xdg.DataFile(relPath)
	if err != nil {
		pterm.Error.Println(err)
		os.Exit(1)
	}

	// copy the application icon to the data folder
	// if it doesn't exist already
	if _, err := xdg.SearchDataFile(relPath); err != nil {
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
