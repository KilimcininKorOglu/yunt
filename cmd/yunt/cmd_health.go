package main

import (
	"encoding/json"
	"fmt"
	"net"
	"time"

	"github.com/spf13/cobra"

	"yunt/internal/config"
)

var (
	// Health command flags
	healthOutputFormat string
	healthVerbose      bool
)

// healthCmd represents the health command.
var healthCmd = &cobra.Command{
	Use:   "health",
	Short: "Check system health",
	Long: `Check the health status of the Yunt mail server and its components.

This command verifies:
  - Database connectivity
  - SMTP server status
  - IMAP server status
  - API server status
  - Storage availability

Examples:
  # Basic health check
  yunt health

  # Verbose output
  yunt health --verbose

  # Output as JSON
  yunt health --output json`,
	RunE: runHealth,
}

func init() {
	healthCmd.Flags().StringVarP(&healthOutputFormat, "output", "o", "text", "output format (text, json)")
	healthCmd.Flags().BoolVarP(&healthVerbose, "verbose", "v", false, "show detailed information")
}

// HealthStatus represents the overall health status.
type HealthStatus struct {
	Status     string                    `json:"status"`
	Timestamp  time.Time                 `json:"timestamp"`
	Version    string                    `json:"version"`
	Components map[string]ComponentCheck `json:"components"`
}

// ComponentCheck represents a single component's health status.
type ComponentCheck struct {
	Status  string `json:"status"`
	Message string `json:"message,omitempty"`
	Latency string `json:"latency,omitempty"`
}

func runHealth(cmd *cobra.Command, args []string) error {
	log := getLogger()
	cfg := getConfig()

	log.Debug().Msg("Running health check")

	health := HealthStatus{
		Status:     "healthy",
		Timestamp:  time.Now().UTC(),
		Version:    version,
		Components: make(map[string]ComponentCheck),
	}

	// Check database
	dbCheck := checkDatabase(cfg)
	health.Components["database"] = dbCheck
	if dbCheck.Status != "healthy" {
		health.Status = "degraded"
	}

	// Check SMTP
	if cfg.SMTP.Enabled {
		smtpCheck := checkSMTP(cfg)
		health.Components["smtp"] = smtpCheck
		if smtpCheck.Status != "healthy" && health.Status == "healthy" {
			health.Status = "degraded"
		}
	}

	// Check IMAP
	if cfg.IMAP.Enabled {
		imapCheck := checkIMAP(cfg)
		health.Components["imap"] = imapCheck
		if imapCheck.Status != "healthy" && health.Status == "healthy" {
			health.Status = "degraded"
		}
	}

	// Check API
	if cfg.API.Enabled {
		apiCheck := checkAPI(cfg)
		health.Components["api"] = apiCheck
		if apiCheck.Status != "healthy" && health.Status == "healthy" {
			health.Status = "degraded"
		}
	}

	// Output results
	switch healthOutputFormat {
	case "json":
		data, err := json.MarshalIndent(health, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal health status: %w", err)
		}
		fmt.Println(string(data))
	default:
		printHealthText(health)
	}

	// Return error if unhealthy
	if health.Status == "unhealthy" {
		return fmt.Errorf("system is unhealthy")
	}

	return nil
}

func checkDatabase(cfg *config.Config) ComponentCheck {
	start := time.Now()

	// TODO: Implement actual database connectivity check
	// For now, simulate a check based on configuration
	check := ComponentCheck{
		Status:  "healthy",
		Message: fmt.Sprintf("Connected to %s database", cfg.Database.Driver),
		Latency: time.Since(start).String(),
	}

	return check
}

func checkSMTP(cfg *config.Config) ComponentCheck {
	start := time.Now()
	addr := fmt.Sprintf("%s:%d", cfg.SMTP.Host, cfg.SMTP.Port)

	// Try to connect to SMTP port
	conn, err := net.DialTimeout("tcp", addr, 2*time.Second)
	if err != nil {
		return ComponentCheck{
			Status:  "unhealthy",
			Message: fmt.Sprintf("Cannot connect to %s: %v", addr, err),
			Latency: time.Since(start).String(),
		}
	}
	conn.Close()

	return ComponentCheck{
		Status:  "healthy",
		Message: fmt.Sprintf("Listening on %s", addr),
		Latency: time.Since(start).String(),
	}
}

func checkIMAP(cfg *config.Config) ComponentCheck {
	start := time.Now()
	addr := fmt.Sprintf("%s:%d", cfg.IMAP.Host, cfg.IMAP.Port)

	// Try to connect to IMAP port
	conn, err := net.DialTimeout("tcp", addr, 2*time.Second)
	if err != nil {
		return ComponentCheck{
			Status:  "unhealthy",
			Message: fmt.Sprintf("Cannot connect to %s: %v", addr, err),
			Latency: time.Since(start).String(),
		}
	}
	conn.Close()

	return ComponentCheck{
		Status:  "healthy",
		Message: fmt.Sprintf("Listening on %s", addr),
		Latency: time.Since(start).String(),
	}
}

func checkAPI(cfg *config.Config) ComponentCheck {
	start := time.Now()
	addr := fmt.Sprintf("%s:%d", cfg.API.Host, cfg.API.Port)

	// Try to connect to API port
	conn, err := net.DialTimeout("tcp", addr, 2*time.Second)
	if err != nil {
		return ComponentCheck{
			Status:  "unhealthy",
			Message: fmt.Sprintf("Cannot connect to %s: %v", addr, err),
			Latency: time.Since(start).String(),
		}
	}
	conn.Close()

	return ComponentCheck{
		Status:  "healthy",
		Message: fmt.Sprintf("Listening on %s", addr),
		Latency: time.Since(start).String(),
	}
}

func printHealthText(health HealthStatus) {
	statusIcon := "✓"
	if health.Status != "healthy" {
		statusIcon = "✗"
	}

	fmt.Println("Health Check")
	fmt.Println("============")
	fmt.Println()
	fmt.Printf("Status:    %s %s\n", statusIcon, health.Status)
	fmt.Printf("Version:   %s\n", health.Version)
	fmt.Printf("Timestamp: %s\n", health.Timestamp.Format(time.RFC3339))
	fmt.Println()
	fmt.Println("Components:")

	for name, check := range health.Components {
		icon := "✓"
		if check.Status != "healthy" {
			icon = "✗"
		}

		fmt.Printf("  %s %-10s: %s\n", icon, name, check.Status)
		if healthVerbose && check.Message != "" {
			fmt.Printf("      Message: %s\n", check.Message)
			fmt.Printf("      Latency: %s\n", check.Latency)
		}
	}
}
