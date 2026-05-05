package main

import (
	"fmt"
	"os"

	"github.com/engram/cmd"
)

// Version information set at build time
var (
	Version   = "dev"
	Commit    = "none"
	BuildDate = "unknown"
)

func main() {
	if err := cmd.Execute(Version, Commit, BuildDate); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
