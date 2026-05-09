// Package main is the entry point for the Yunt mail server.
//
// @title           Yunt Mail Server API
// @version         1.0
// @description     REST API for the Yunt development mail server.
// @host            localhost:8025
// @BasePath        /api/v1
// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
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
