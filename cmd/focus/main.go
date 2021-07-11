package main

import (
	"os"

	cmd "github.com/ayoisaiah/focus/src"
	"github.com/pterm/pterm"
)

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
