package main

import (
	"os"

	focus "github.com/ayoisaiah/focus/src"
)

func run(args []string) error {
	return focus.GetApp().Run(args)
}

func main() {
	err := run(os.Args)
	if err != nil {
		os.Exit(1)
	}
}
