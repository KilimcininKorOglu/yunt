package main

import (
	"context"
	"fmt"
	"time"

	"github.com/spf13/cobra"

	"yunt/internal/repository"
	"yunt/internal/repository/postgres"
	"yunt/internal/repository/sqlite"
)

var (
	migrateForce   bool
	migrateVersion int
	migrateDryRun  bool
)

var migrateCmd = &cobra.Command{
	Use:   "migrate",
	Short: "Manage database migrations",
	Long: `Manage database migrations for the Yunt mail server.

Use subcommands to run, rollback, or check migration status.`,
}

var migrateUpCmd = &cobra.Command{
	Use:   "up",
	Short: "Run pending migrations",
	Long: `Run all pending database migrations.

Examples:
  yunt migrate up
  yunt migrate up --dry-run`,
	RunE: runMigrateUp,
}

var migrateDownCmd = &cobra.Command{
	Use:   "down",
	Short: "Rollback migrations",
	Long: `Rollback database migrations.

Examples:
  yunt migrate down
  yunt migrate down --version 5`,
	RunE: runMigrateDown,
}

var migrateStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show migration status",
	Long: `Show the current status of database migrations.

Examples:
  yunt migrate status`,
	RunE: runMigrateStatus,
}

var migrateResetCmd = &cobra.Command{
	Use:   "reset",
	Short: "Reset database (rollback all and re-run)",
	Long: `Reset the database by rolling back all migrations and re-running them.

WARNING: This will delete all data in the database!

Examples:
  yunt migrate reset --force`,
	RunE: runMigrateReset,
}

func init() {
	migrateCmd.AddCommand(migrateUpCmd)
	migrateCmd.AddCommand(migrateDownCmd)
	migrateCmd.AddCommand(migrateStatusCmd)
	migrateCmd.AddCommand(migrateResetCmd)

	migrateUpCmd.Flags().BoolVar(&migrateDryRun, "dry-run", false, "preview changes without applying")
	migrateUpCmd.Flags().BoolVar(&migrateForce, "force", false, "force run even if database is locked")

	migrateDownCmd.Flags().BoolVar(&migrateDryRun, "dry-run", false, "preview changes without applying")
	migrateDownCmd.Flags().IntVar(&migrateVersion, "version", 0, "target migration version to rollback to")

	migrateResetCmd.Flags().BoolVar(&migrateForce, "force", false, "confirm database reset (required)")
}

func getMigrator() (repository.Migrator, func(), error) {
	repo, err := initRepo()
	if err != nil {
		return nil, nil, err
	}

	cleanup := func() { repo.Close() }

	switch r := repo.(type) {
	case *sqlite.Repository:
		m := r.Migrator()
		if m == nil {
			cleanup()
			return nil, nil, fmt.Errorf("SQLite migrator not available")
		}
		return m, cleanup, nil
	case *postgres.Repository:
		m := r.Migrator()
		if m == nil {
			cleanup()
			return nil, nil, fmt.Errorf("PostgreSQL migrator not available")
		}
		return m, cleanup, nil
	default:
		cleanup()
		return nil, nil, fmt.Errorf("migration CLI not supported for this database driver; use auto-migrate in config instead")
	}
}

func runMigrateUp(cmd *cobra.Command, args []string) error {
	log := getLogger()
	cfg := getConfig()

	log.Info().
		Str("driver", cfg.Database.Driver).
		Bool("dry_run", migrateDryRun).
		Msg("Running database migrations")

	if migrateDryRun {
		migrator, cleanup, err := getMigrator()
		if err != nil {
			return err
		}
		defer cleanup()

		ctx := context.Background()
		statuses, err := migrator.MigrationStatus(ctx)
		if err != nil {
			return fmt.Errorf("failed to get migration status: %w", err)
		}

		pending := 0
		for _, s := range statuses {
			if !s.Applied {
				fmt.Printf("  Would apply: %03d_%s\n", s.Version, s.Name)
				pending++
			}
		}
		if pending == 0 {
			fmt.Println("No pending migrations")
		} else {
			fmt.Printf("\n%d migration(s) would be applied\n", pending)
		}
		return nil
	}

	migrator, cleanup, err := getMigrator()
	if err != nil {
		return err
	}
	defer cleanup()

	ctx := context.Background()
	if err := migrator.Migrate(ctx); err != nil {
		return fmt.Errorf("migration failed: %w", err)
	}

	fmt.Println("All migrations applied successfully")
	log.Info().Msg("Migrations completed")
	return nil
}

func runMigrateDown(cmd *cobra.Command, args []string) error {
	log := getLogger()
	cfg := getConfig()

	log.Info().
		Str("driver", cfg.Database.Driver).
		Int("target_version", migrateVersion).
		Bool("dry_run", migrateDryRun).
		Msg("Rolling back migrations")

	migrator, cleanup, err := getMigrator()
	if err != nil {
		return err
	}
	defer cleanup()

	ctx := context.Background()

	if migrateDryRun {
		version, err := migrator.MigrationVersion(ctx)
		if err != nil {
			return fmt.Errorf("failed to get current version: %w", err)
		}
		if migrateVersion > 0 {
			fmt.Printf("Would rollback from version %d to version %d\n", version, migrateVersion)
		} else {
			fmt.Printf("Would rollback version %d\n", version)
		}
		return nil
	}

	steps := 1
	if migrateVersion > 0 {
		current, err := migrator.MigrationVersion(ctx)
		if err != nil {
			return fmt.Errorf("failed to get current version: %w", err)
		}
		steps = int(current) - migrateVersion
		if steps <= 0 {
			fmt.Println("Already at or below target version")
			return nil
		}
	}

	if err := migrator.MigrateDown(ctx, steps); err != nil {
		return fmt.Errorf("rollback failed: %w", err)
	}

	version, _ := migrator.MigrationVersion(ctx)
	fmt.Printf("Rolled back to version %d\n", version)
	log.Info().Int64("version", version).Msg("Rollback completed")
	return nil
}

func runMigrateStatus(cmd *cobra.Command, args []string) error {
	log := getLogger()
	cfg := getConfig()

	log.Info().
		Str("driver", cfg.Database.Driver).
		Msg("Checking migration status")

	migrator, cleanup, err := getMigrator()
	if err != nil {
		return err
	}
	defer cleanup()

	ctx := context.Background()
	statuses, err := migrator.MigrationStatus(ctx)
	if err != nil {
		return fmt.Errorf("failed to get migration status: %w", err)
	}

	version, _ := migrator.MigrationVersion(ctx)

	fmt.Println("Migration Status")
	fmt.Println("================")
	fmt.Printf("Driver:          %s\n", cfg.Database.Driver)
	fmt.Printf("Current version: %d\n", version)
	fmt.Println()

	if len(statuses) == 0 {
		fmt.Println("No migrations found")
		return nil
	}

	fmt.Printf("%-8s %-30s %-10s %s\n", "VERSION", "NAME", "STATUS", "APPLIED AT")
	fmt.Printf("%-8s %-30s %-10s %s\n", "-------", "----", "------", "----------")

	applied, pending := 0, 0
	for _, s := range statuses {
		status := "pending"
		appliedAt := ""
		if s.Applied {
			status = "applied"
			applied++
			if s.AppliedAt != nil {
				appliedAt = *s.AppliedAt
			}
		} else {
			pending++
		}
		fmt.Printf("%-8d %-30s %-10s %s\n", s.Version, s.Name, status, appliedAt)
	}

	fmt.Printf("\nTotal: %d | Applied: %d | Pending: %d\n", len(statuses), applied, pending)
	return nil
}

func runMigrateReset(cmd *cobra.Command, args []string) error {
	log := getLogger()
	cfg := getConfig()

	if !migrateForce {
		return fmt.Errorf("database reset requires --force flag to confirm")
	}

	log.Warn().
		Str("driver", cfg.Database.Driver).
		Msg("Resetting database — all data will be deleted!")

	migrator, cleanup, err := getMigrator()
	if err != nil {
		return err
	}
	defer cleanup()

	ctx := context.Background()

	fmt.Println("Rolling back all migrations...")
	version, _ := migrator.MigrationVersion(ctx)
	if version > 0 {
		if err := migrator.MigrateDown(ctx, int(version)); err != nil {
			return fmt.Errorf("rollback failed: %w", err)
		}
	}

	fmt.Println("Re-running all migrations...")
	if err := migrator.Migrate(ctx); err != nil {
		return fmt.Errorf("migration failed: %w", err)
	}

	newVersion, _ := migrator.MigrationVersion(ctx)
	fmt.Printf("Database reset completed (version: %d)\n", newVersion)
	log.Info().Int64("version", newVersion).Msg("Database reset completed")
	return nil
}

// migrateTimeout returns a context with a reasonable timeout for migration operations.
func migrateTimeout() (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), 5*time.Minute)
}
