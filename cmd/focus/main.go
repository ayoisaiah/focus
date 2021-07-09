package main

import (
	"fmt"
	"os"

	cmd "github.com/ayoisaiah/focus/src"
)

func run(args []string) error {
	return cmd.GetApp().Run(args)
}

func main() {
	err := run(os.Args)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
