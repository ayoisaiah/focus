package main

import (
	"embed"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/adrg/xdg"
	cmd "github.com/ayoisaiah/focus/src"
	"github.com/pterm/pterm"
)

const (
	configDir = "focus"
)

//go:embed static/*
var static embed.FS

func init() {
	_ = fs.WalkDir(static, "static", func(path string, d fs.DirEntry, err error) error {
		if !d.IsDir() {
			var b []byte

			b, err = fs.ReadFile(static, path)
			if err != nil {
				pterm.Error.Println(err)
				os.Exit(1)
			}

			relPath := filepath.Join(configDir, path)

			var pathToFile string

			pathToFile, err = xdg.DataFile(relPath)
			if err != nil {
				pterm.Error.Println(err)
				os.Exit(1)
			}

			if _, err = xdg.SearchDataFile(relPath); err != nil {
				_ = os.WriteFile(pathToFile, b, os.ModePerm)
			}
		}

		return err
	})
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
