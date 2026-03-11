package main

import (
	"fmt"
	"os"

	appcmds "github.com/go-go-golems/goja-github-actions/cmd/goja-gha/cmds"
)

func main() {
	if err := appcmds.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
