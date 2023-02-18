package main

import (
	"os"

	"github.com/pterm/pterm"

	"github.com/ayoisaiah/focus"
)

func run(args []string) error {
	return focus.GetApp().Run(args)
}

func main() {
	err := run(os.Args)
	if err != nil {
		pterm.Error.Println(err)
		os.Exit(1)
	}
}
