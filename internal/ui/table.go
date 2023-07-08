package ui

import (
	"fmt"
	"io"

	"github.com/pterm/pterm"
)

func PrintTable(data [][]string, writer io.Writer) {
	table := pterm.DefaultTable
	table.Boxed = true

	str, err := table.WithHasHeader().WithData(data).Srender()
	if err != nil {
		pterm.Error.Printfln("Failed to output session table: %s", err.Error())
		return
	}

	fmt.Fprintln(writer, str)
}
