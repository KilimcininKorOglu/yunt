// Package main is the entry point for the Yunt mail server.
package main

import (
	"fmt"
	"os"
)

// Version information set via ldflags during build.
var (
	version   = "dev"
	commit    = "unknown"
	buildDate = "unknown"
)

func main() {
	fmt.Println("Yunt - Development Mail Server")
	fmt.Printf("Version: %s, Commit: %s, Built: %s\n", version, commit, buildDate)
	os.Exit(0)
}
