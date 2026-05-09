package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"yunt/internal/config"
	"yunt/internal/repository"
	"yunt/internal/repository/factory"
)

var (
	// cfgFile is the path to the configuration file.
	cfgFile string

	// cfg holds the loaded configuration.
	cfg *config.Config

	// logger is the application logger.
	logger *config.Logger
)

// rootCmd represents the base command when called without any subcommands.
var rootCmd = &cobra.Command{
	Use:   "yunt",
	Short: "Yunt - Development Mail Server",
	Long: `Yunt is a lightweight, powerful mail server written in Go, 
designed for developers and test environments.

The name comes from the Gokturk Turkish word for "horse" - just as 
mounted couriers carried letters, Yunt safely delivers your emails.

Features:
  - SMTP Server: Mail capture and relay support
  - IMAP Server: Mail client support (Thunderbird, etc.)
  - Web UI: Modern admin panel
  - REST API: Full-featured API for integration
  - Multi-user Support: Team collaboration with isolated mailboxes
  - Multi-database: SQLite, PostgreSQL, MySQL, MongoDB`,
	SilenceUsage:  true,
	SilenceErrors: true,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		// Skip config loading for version and completion commands
		if cmd.Name() == "version" || cmd.Name() == "completion" {
			return nil
		}

		// Load configuration
		var err error
		cfg, err = config.Load(cfgFile)
		if err != nil {
			return fmt.Errorf("failed to load configuration: %w", err)
		}

		// Initialize logger
		logger, err = config.NewLogger(cfg.Logging)
		if err != nil {
			return fmt.Errorf("failed to initialize logger: %w", err)
		}

		return nil
	},
	PersistentPostRun: func(cmd *cobra.Command, args []string) {
		// Close logger if initialized
		if logger != nil {
			_ = logger.Close()
		}
	},
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() error {
	return rootCmd.Execute()
}

func init() {
	// Persistent flags available to all subcommands
	rootCmd.PersistentFlags().StringVarP(&cfgFile, "config", "c", "", "config file (default: ./yunt.yaml or /etc/yunt/yunt.yaml)")

	// Add subcommands
	rootCmd.AddCommand(serveCmd)
	rootCmd.AddCommand(versionCmd)
	rootCmd.AddCommand(migrateCmd)
	rootCmd.AddCommand(userCmd)
	rootCmd.AddCommand(messagesCmd)
	rootCmd.AddCommand(healthCmd)
}

// getConfig returns the loaded configuration.
// Panics if configuration is not loaded.
func getConfig() *config.Config {
	if cfg == nil {
		fmt.Fprintln(os.Stderr, "error: configuration not loaded")
		os.Exit(1)
	}
	return cfg
}

// getLogger returns the application logger.
// Returns a default logger if not initialized.
func getLogger() *config.Logger {
	if logger == nil {
		return config.NewDefaultLogger()
	}
	return logger
}

// initRepo creates a repository instance from the current configuration.
func initRepo() (repository.Repository, error) {
	c := getConfig()
	f, err := factory.New(&c.Database)
	if err != nil {
		return nil, fmt.Errorf("failed to create repository factory: %w", err)
	}
	repo, err := f.Create()
	if err != nil {
		return nil, fmt.Errorf("failed to create repository: %w", err)
	}
	return repo, nil
}
