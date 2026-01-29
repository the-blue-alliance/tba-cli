package main

import (
	"os"

	"github.com/the-blue-alliance/tba-cli/cmd"
)

func main() {
	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}
