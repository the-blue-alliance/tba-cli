package main

import (
	"fmt"
	"os"

	"github.com/the-blue-alliance/tba-cli/cmd"
	"github.com/the-blue-alliance/tba-cli/internal/clierr"
)

func main() {
	err := cmd.Execute()
	// A closed stdout and a Ctrl-C are not worth a message: the reader has
	// already gone, and the person who interrupted knows they did. The exit
	// code still says which it was.
	if err != nil && !clierr.Silent(err) {
		fmt.Fprintln(os.Stderr, "Error:", err)
	}
	os.Exit(clierr.ExitCode(err))
}
