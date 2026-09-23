package main

import (
	"os"

	"github.com/the-blue-alliance/tba-cli/cmd"
	"github.com/the-blue-alliance/tba-cli/internal/clierr"
)

func main() {
	// cmd.Run does the reporting, so that the one line saying what went wrong
	// is printed above the usage hint rather than below it. All that is left
	// here is the exit code.
	os.Exit(clierr.ExitCode(cmd.Execute()))
}
