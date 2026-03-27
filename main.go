package main

import (
	"os"

	"github.com/wcrg/lissh/cmd"
)

func main() {
	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}
