package main

import (
	"os"

	"github.com/pterm/pterm"

	"github.com/ayoisaiah/focus/app"
	"github.com/ayoisaiah/focus/config"
)

func run(args []string) error {
	return app.Get().Run(args)
}

func main() {
	config.InitializePaths()

	err := run(os.Args)
	if err != nil {
		pterm.Error.Println(err)
		os.Exit(1)
	}
}
