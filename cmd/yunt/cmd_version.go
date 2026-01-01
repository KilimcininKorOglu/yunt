package main

import (
	"encoding/json"
	"fmt"
	"runtime"

	"github.com/spf13/cobra"
)

var (
	// Version command flags
	versionOutputFormat string
	versionShort        bool
)

// versionCmd represents the version command.
var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Display version information",
	Long: `Display version information including build details.

Examples:
  # Show version information
  yunt version

  # Show short version only
  yunt version --short

  # Output as JSON
  yunt version --output json`,
	Run: runVersion,
}

func init() {
	versionCmd.Flags().StringVarP(&versionOutputFormat, "output", "o", "text", "output format (text, json)")
	versionCmd.Flags().BoolVarP(&versionShort, "short", "s", false, "show only the version number")
}

// VersionInfo contains version and build information.
type VersionInfo struct {
	Version   string `json:"version"`
	Commit    string `json:"commit"`
	BuildDate string `json:"buildDate"`
	GoVersion string `json:"goVersion"`
	OS        string `json:"os"`
	Arch      string `json:"arch"`
}

func runVersion(cmd *cobra.Command, args []string) {
	info := VersionInfo{
		Version:   version,
		Commit:    commit,
		BuildDate: buildDate,
		GoVersion: runtime.Version(),
		OS:        runtime.GOOS,
		Arch:      runtime.GOARCH,
	}

	if versionShort {
		fmt.Println(info.Version)
		return
	}

	switch versionOutputFormat {
	case "json":
		data, err := json.MarshalIndent(info, "", "  ")
		if err != nil {
			fmt.Printf("error: %v\n", err)
			return
		}
		fmt.Println(string(data))
	default:
		fmt.Printf("Yunt - Development Mail Server\n")
		fmt.Printf("Version:    %s\n", info.Version)
		fmt.Printf("Commit:     %s\n", info.Commit)
		fmt.Printf("Built:      %s\n", info.BuildDate)
		fmt.Printf("Go version: %s\n", info.GoVersion)
		fmt.Printf("OS/Arch:    %s/%s\n", info.OS, info.Arch)
	}
}
