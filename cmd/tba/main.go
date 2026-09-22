package main

import (
	"fmt"
	"os"

	"github.com/the-blue-alliance/tba-cli/cmd"
	"github.com/the-blue-alliance/tba-cli/internal/clierr"
)

func main() {
	err := cmd.Execute()
	code := clierr.ExitCode(err)
	// A closed stdout is not worth a message: the reader has already gone.
	if err != nil && code != clierr.ExitBrokenPipe {
		fmt.Fprintln(os.Stderr, "Error:", err)
	}
	os.Exit(code)
}
