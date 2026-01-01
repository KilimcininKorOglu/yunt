package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/spf13/cobra"
)

var (
	// Serve command flags
	smtpPort int
	imapPort int
	apiPort  int
)

// serveCmd represents the serve command.
var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Start the Yunt mail server",
	Long: `Start the Yunt mail server with SMTP, IMAP, and API services.

By default, all services are started according to the configuration file.
You can override specific port settings using command-line flags.

Examples:
  # Start with default configuration
  yunt serve

  # Start with a custom configuration file
  yunt serve --config /path/to/yunt.yaml

  # Override specific ports
  yunt serve --smtp-port 2525 --api-port 8080

  # Start in foreground mode (default)
  yunt serve`,
	RunE: runServe,
}

func init() {
	// Serve command specific flags
	serveCmd.Flags().IntVar(&smtpPort, "smtp-port", 0, "override SMTP server port")
	serveCmd.Flags().IntVar(&imapPort, "imap-port", 0, "override IMAP server port")
	serveCmd.Flags().IntVar(&apiPort, "api-port", 0, "override API server port")
}

func runServe(cmd *cobra.Command, args []string) error {
	log := getLogger()
	cfg := getConfig()

	// Apply port overrides if specified
	if smtpPort > 0 {
		cfg.SMTP.Port = smtpPort
	}
	if imapPort > 0 {
		cfg.IMAP.Port = imapPort
	}
	if apiPort > 0 {
		cfg.API.Port = apiPort
	}

	log.Info().
		Str("version", version).
		Str("commit", commit).
		Msg("Starting Yunt mail server")

	// Log configuration summary
	log.Info().
		Str("server_name", cfg.Server.Name).
		Str("domain", cfg.Server.Domain).
		Msg("Server configuration loaded")

	// Log enabled services
	if cfg.SMTP.Enabled {
		log.Info().
			Str("host", cfg.SMTP.Host).
			Int("port", cfg.SMTP.Port).
			Bool("auth_required", cfg.SMTP.AuthRequired).
			Bool("relay_enabled", cfg.SMTP.AllowRelay).
			Msg("SMTP server configured")
	}

	if cfg.IMAP.Enabled {
		log.Info().
			Str("host", cfg.IMAP.Host).
			Int("port", cfg.IMAP.Port).
			Msg("IMAP server configured")
	}

	if cfg.API.Enabled {
		log.Info().
			Str("host", cfg.API.Host).
			Int("port", cfg.API.Port).
			Bool("swagger_enabled", cfg.API.EnableSwagger).
			Msg("API server configured")
	}

	log.Info().
		Str("driver", cfg.Database.Driver).
		Msg("Database configured")

	// Create context for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Setup signal handling for graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Start servers (placeholder for actual server implementation)
	// TODO: Implement actual server startup once services are available
	fmt.Println()
	fmt.Println("=================================================")
	fmt.Println("  Yunt - Development Mail Server")
	fmt.Printf("  Version: %s (commit: %s)\n", version, commit)
	fmt.Println("=================================================")
	fmt.Println()

	if cfg.SMTP.Enabled {
		fmt.Printf("  SMTP Server:  %s:%d\n", cfg.SMTP.Host, cfg.SMTP.Port)
	}
	if cfg.IMAP.Enabled {
		fmt.Printf("  IMAP Server:  %s:%d\n", cfg.IMAP.Host, cfg.IMAP.Port)
	}
	if cfg.API.Enabled {
		fmt.Printf("  API/Web UI:   http://%s:%d\n", cfg.API.Host, cfg.API.Port)
	}

	fmt.Println()
	fmt.Println("  Press Ctrl+C to stop")
	fmt.Println("=================================================")

	// Wait for shutdown signal
	select {
	case sig := <-sigChan:
		log.Info().Str("signal", sig.String()).Msg("Received shutdown signal")
	case <-ctx.Done():
		log.Info().Msg("Context cancelled")
	}

	// Begin graceful shutdown
	log.Info().Dur("timeout", cfg.Server.GracefulTimeout).Msg("Initiating graceful shutdown")

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), cfg.Server.GracefulTimeout)
	defer shutdownCancel()

	// TODO: Shutdown actual servers here
	// For now, simulate graceful shutdown
	select {
	case <-shutdownCtx.Done():
		if shutdownCtx.Err() == context.DeadlineExceeded {
			log.Warn().Msg("Graceful shutdown timed out, forcing exit")
		}
	case <-time.After(100 * time.Millisecond):
		// Quick shutdown for now since no actual servers
	}

	log.Info().Msg("Server stopped successfully")
	return nil
}
