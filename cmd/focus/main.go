package main

import (
	"os"

	"github.com/ayoisaiah/focus"
	"github.com/pterm/pterm"
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
