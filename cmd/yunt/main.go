// Package main is the entry point for the Yunt mail server.
package main

import "os"

// Version information set via ldflags during build.
var (
	version   = "dev"
	commit    = "unknown"
	buildDate = "unknown"
)

func main() {
	if err := Execute(); err != nil {
		os.Exit(1)
	}
}
