package main

import (
	"fmt"
	"os"
	"strings"

	appcmds "github.com/go-go-golems/goja-github-actions/cmd/goja-gha/cmds"
)

func main() {
	if err := appcmds.Execute(); err != nil {
		message := appcmds.FormatCLIError(err)
		if strings.Contains(message, "\n") {
			fmt.Fprintf(os.Stderr, "Error:\n%s\n", message)
		} else {
			fmt.Fprintf(os.Stderr, "Error: %s\n", message)
		}
		os.Exit(1)
	}
}
