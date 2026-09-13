package main

import (
	"fmt"
	"os"

	"github.com/EliasLd/goweeb/internal/tui"
)

func main() {
	if err := tui.Start(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
