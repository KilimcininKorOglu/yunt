// Package sqlite provides SQLite-specific implementation of the repository interfaces.
package sqlite

import (
	"context"
	"database/sql"
	"embed"
	"fmt"
	"io/fs"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"yunt/internal/repository"
)

//go:embed migrations/*.sql
var migrationsFS embed.FS

// Migration represents a single database migration.
type Migration struct {
	// Version is the migration version number (extracted from filename).
	Version int64

	// Name is the migration name (extracted from filename).
	Name string

	// UpSQL contains the SQL statements to apply the migration.
	UpSQL string

	// DownSQL contains the SQL statements to rollback the migration.
	DownSQL string
}

// Migrator handles database migrations for SQLite.
type Migrator struct {
	pool       *ConnectionPool
	migrations []Migration
}

// MigrationRecord represents a migration record in the schema_migrations table.
type MigrationRecord struct {
	Version   int64     `db:"version"`
	Name      string    `db:"name"`
	AppliedAt time.Time `db:"applied_at"`
}

// migrationFilePattern matches migration files like "001_initial_schema.sql".
var migrationFilePattern = regexp.MustCompile(`^(\d+)_(.+)\.sql$`)

// NewMigrator creates a new Migrator with the given connection pool.
func NewMigrator(pool *ConnectionPool) (*Migrator, error) {
	if pool == nil {
		return nil, fmt.Errorf("connection pool is required")
	}

	m := &Migrator{
		pool: pool,
	}

	if err := m.loadMigrations(); err != nil {
		return nil, fmt.Errorf("failed to load migrations: %w", err)
	}

	return m, nil
}

// loadMigrations loads all migration files from the embedded filesystem.
func (m *Migrator) loadMigrations() error {
	entries, err := fs.ReadDir(migrationsFS, "migrations")
	if err != nil {
		return fmt.Errorf("failed to read migrations directory: %w", err)
	}

	migrations := make([]Migration, 0, len(entries))

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		filename := entry.Name()
		matches := migrationFilePattern.FindStringSubmatch(filename)
		if matches == nil {
			continue
		}

		version, err := strconv.ParseInt(matches[1], 10, 64)
		if err != nil {
			return fmt.Errorf("invalid version in migration file %s: %w", filename, err)
		}

		name := matches[2]

		content, err := fs.ReadFile(migrationsFS, filepath.Join("migrations", filename))
		if err != nil {
			return fmt.Errorf("failed to read migration file %s: %w", filename, err)
		}

		upSQL, downSQL := parseMigrationContent(string(content))

		migrations = append(migrations, Migration{
			Version: version,
			Name:    name,
			UpSQL:   upSQL,
			DownSQL: downSQL,
		})
	}

	// Sort migrations by version
	sort.Slice(migrations, func(i, j int) bool {
		return migrations[i].Version < migrations[j].Version
	})

	m.migrations = migrations
	return nil
}

// parseMigrationContent parses migration SQL content and extracts up/down sections.
// Format expected:
//
//	-- +migrate Up
//	<up SQL statements>
//	-- +migrate Down
//	<down SQL statements>
//
// If no markers are found, the entire content is treated as up migration.
func parseMigrationContent(content string) (upSQL, downSQL string) {
	content = strings.TrimSpace(content)

	// Check for migration markers
	upMarker := "-- +migrate Up"
	downMarker := "-- +migrate Down"

	upIdx := strings.Index(content, upMarker)
	downIdx := strings.Index(content, downMarker)

	if upIdx == -1 && downIdx == -1 {
		// No markers found, treat entire content as up migration
		return content, ""
	}

	if upIdx != -1 && downIdx != -1 {
		// Both markers found
		upStart := upIdx + len(upMarker)
		upSQL = strings.TrimSpace(content[upStart:downIdx])
		downStart := downIdx + len(downMarker)
		downSQL = strings.TrimSpace(content[downStart:])
	} else if upIdx != -1 {
		// Only up marker found
		upStart := upIdx + len(upMarker)
		upSQL = strings.TrimSpace(content[upStart:])
	}

	return upSQL, downSQL
}

// ensureMigrationsTable creates the schema_migrations table if it doesn't exist.
func (m *Migrator) ensureMigrationsTable(ctx context.Context) error {
	query := `
		CREATE TABLE IF NOT EXISTS schema_migrations (
			version INTEGER PRIMARY KEY,
			name TEXT NOT NULL,
			applied_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		);
		CREATE INDEX IF NOT EXISTS idx_schema_migrations_applied_at ON schema_migrations(applied_at);
	`

	_, err := m.pool.ExecContext(ctx, query)
	if err != nil {
		return fmt.Errorf("failed to create schema_migrations table: %w", err)
	}

	return nil
}

// getAppliedVersions returns a set of applied migration versions.
func (m *Migrator) getAppliedVersions(ctx context.Context) (map[int64]bool, error) {
	query := `SELECT version FROM schema_migrations ORDER BY version`

	var versions []int64
	if err := m.pool.SelectContext(ctx, &versions, query); err != nil {
		return nil, fmt.Errorf("failed to get applied versions: %w", err)
	}

	applied := make(map[int64]bool, len(versions))
	for _, v := range versions {
		applied[v] = true
	}

	return applied, nil
}

// Migrate runs all pending migrations.
func (m *Migrator) Migrate(ctx context.Context) error {
	if err := m.ensureMigrationsTable(ctx); err != nil {
		return err
	}

	applied, err := m.getAppliedVersions(ctx)
	if err != nil {
		return err
	}

	for _, migration := range m.migrations {
		if applied[migration.Version] {
			continue
		}

		if err := m.applyMigration(ctx, migration); err != nil {
			return fmt.Errorf("failed to apply migration %d (%s): %w", migration.Version, migration.Name, err)
		}
	}

	return nil
}

// MigrateUp runs a specific number of pending migrations.
func (m *Migrator) MigrateUp(ctx context.Context, steps int) error {
	if err := m.ensureMigrationsTable(ctx); err != nil {
		return err
	}

	applied, err := m.getAppliedVersions(ctx)
	if err != nil {
		return err
	}

	count := 0
	for _, migration := range m.migrations {
		if count >= steps {
			break
		}

		if applied[migration.Version] {
			continue
		}

		if err := m.applyMigration(ctx, migration); err != nil {
			return fmt.Errorf("failed to apply migration %d (%s): %w", migration.Version, migration.Name, err)
		}

		count++
	}

	return nil
}

// MigrateDown rolls back a specific number of migrations.
func (m *Migrator) MigrateDown(ctx context.Context, steps int) error {
	if err := m.ensureMigrationsTable(ctx); err != nil {
		return err
	}

	applied, err := m.getAppliedVersions(ctx)
	if err != nil {
		return err
	}

	// Get migrations to rollback in reverse order
	var toRollback []Migration
	for i := len(m.migrations) - 1; i >= 0; i-- {
		if applied[m.migrations[i].Version] {
			toRollback = append(toRollback, m.migrations[i])
		}
	}

	count := 0
	for _, migration := range toRollback {
		if count >= steps {
			break
		}

		if migration.DownSQL == "" {
			return fmt.Errorf("migration %d (%s) does not have a down migration", migration.Version, migration.Name)
		}

		if err := m.rollbackMigration(ctx, migration); err != nil {
			return fmt.Errorf("failed to rollback migration %d (%s): %w", migration.Version, migration.Name, err)
		}

		count++
	}

	return nil
}

// MigrationVersion returns the current migration version.
func (m *Migrator) MigrationVersion(ctx context.Context) (int64, error) {
	if err := m.ensureMigrationsTable(ctx); err != nil {
		return 0, err
	}

	query := `SELECT COALESCE(MAX(version), 0) FROM schema_migrations`

	var version int64
	if err := m.pool.GetContext(ctx, &version, query); err != nil {
		return 0, fmt.Errorf("failed to get migration version: %w", err)
	}

	return version, nil
}

// MigrationStatus returns the status of all migrations.
func (m *Migrator) MigrationStatus(ctx context.Context) ([]repository.MigrationInfo, error) {
	if err := m.ensureMigrationsTable(ctx); err != nil {
		return nil, err
	}

	// Get all applied migrations
	query := `SELECT version, name, applied_at FROM schema_migrations ORDER BY version`
	var records []MigrationRecord
	if err := m.pool.SelectContext(ctx, &records, query); err != nil {
		return nil, fmt.Errorf("failed to get migration records: %w", err)
	}

	appliedMap := make(map[int64]MigrationRecord, len(records))
	for _, r := range records {
		appliedMap[r.Version] = r
	}

	// Build status list
	status := make([]repository.MigrationInfo, 0, len(m.migrations))
	for _, migration := range m.migrations {
		info := repository.MigrationInfo{
			Version: migration.Version,
			Name:    migration.Name,
			Applied: false,
		}

		if record, ok := appliedMap[migration.Version]; ok {
			info.Applied = true
			appliedAt := record.AppliedAt.Format(time.RFC3339)
			info.AppliedAt = &appliedAt
		}

		status = append(status, info)
	}

	return status, nil
}

// applyMigration applies a single migration.
func (m *Migrator) applyMigration(ctx context.Context, migration Migration) error {
	tx, err := m.pool.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() {
		if err != nil {
			tx.Rollback()
		}
	}()

	// Execute the up SQL
	if migration.UpSQL != "" {
		statements := splitStatements(migration.UpSQL)
		for _, stmt := range statements {
			if strings.TrimSpace(stmt) == "" {
				continue
			}
			if _, err = tx.ExecContext(ctx, stmt); err != nil {
				return fmt.Errorf("failed to execute statement: %w", err)
			}
		}
	}

	// Record the migration
	recordQuery := `INSERT INTO schema_migrations (version, name, applied_at) VALUES (?, ?, ?)`
	if _, err = tx.ExecContext(ctx, recordQuery, migration.Version, migration.Name, time.Now().UTC()); err != nil {
		return fmt.Errorf("failed to record migration: %w", err)
	}

	if err = tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// rollbackMigration rolls back a single migration.
func (m *Migrator) rollbackMigration(ctx context.Context, migration Migration) error {
	tx, err := m.pool.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() {
		if err != nil {
			tx.Rollback()
		}
	}()

	// Execute the down SQL
	if migration.DownSQL != "" {
		statements := splitStatements(migration.DownSQL)
		for _, stmt := range statements {
			if strings.TrimSpace(stmt) == "" {
				continue
			}
			if _, err = tx.ExecContext(ctx, stmt); err != nil {
				return fmt.Errorf("failed to execute statement: %w", err)
			}
		}
	}

	// Remove the migration record
	removeQuery := `DELETE FROM schema_migrations WHERE version = ?`
	if _, err = tx.ExecContext(ctx, removeQuery, migration.Version); err != nil {
		return fmt.Errorf("failed to remove migration record: %w", err)
	}

	if err = tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// MigrateToVersion migrates to a specific version.
// If the target version is higher than the current version, it applies up migrations.
// If the target version is lower, it rolls back migrations.
func (m *Migrator) MigrateToVersion(ctx context.Context, targetVersion int64) error {
	currentVersion, err := m.MigrationVersion(ctx)
	if err != nil {
		return err
	}

	if targetVersion == currentVersion {
		return nil
	}

	if targetVersion > currentVersion {
		// Apply migrations up to target version
		applied, err := m.getAppliedVersions(ctx)
		if err != nil {
			return err
		}

		for _, migration := range m.migrations {
			if migration.Version > targetVersion {
				break
			}
			if applied[migration.Version] {
				continue
			}
			if err := m.applyMigration(ctx, migration); err != nil {
				return fmt.Errorf("failed to apply migration %d (%s): %w", migration.Version, migration.Name, err)
			}
		}
	} else {
		// Rollback migrations down to target version
		applied, err := m.getAppliedVersions(ctx)
		if err != nil {
			return err
		}

		for i := len(m.migrations) - 1; i >= 0; i-- {
			migration := m.migrations[i]
			if migration.Version <= targetVersion {
				break
			}
			if !applied[migration.Version] {
				continue
			}
			if migration.DownSQL == "" {
				return fmt.Errorf("migration %d (%s) does not have a down migration", migration.Version, migration.Name)
			}
			if err := m.rollbackMigration(ctx, migration); err != nil {
				return fmt.Errorf("failed to rollback migration %d (%s): %w", migration.Version, migration.Name, err)
			}
		}
	}

	return nil
}

// Reset rolls back all migrations and re-applies them.
func (m *Migrator) Reset(ctx context.Context) error {
	// Get current version
	currentVersion, err := m.MigrationVersion(ctx)
	if err != nil {
		return err
	}

	// Rollback all migrations
	if currentVersion > 0 {
		if err := m.MigrateToVersion(ctx, 0); err != nil {
			return fmt.Errorf("failed to rollback migrations: %w", err)
		}
	}

	// Re-apply all migrations
	if err := m.Migrate(ctx); err != nil {
		return fmt.Errorf("failed to re-apply migrations: %w", err)
	}

	return nil
}

// IsPending returns true if there are pending migrations.
func (m *Migrator) IsPending(ctx context.Context) (bool, error) {
	if err := m.ensureMigrationsTable(ctx); err != nil {
		return false, err
	}

	applied, err := m.getAppliedVersions(ctx)
	if err != nil {
		return false, err
	}

	for _, migration := range m.migrations {
		if !applied[migration.Version] {
			return true, nil
		}
	}

	return false, nil
}

// GetPendingMigrations returns all pending migrations.
func (m *Migrator) GetPendingMigrations(ctx context.Context) ([]Migration, error) {
	if err := m.ensureMigrationsTable(ctx); err != nil {
		return nil, err
	}

	applied, err := m.getAppliedVersions(ctx)
	if err != nil {
		return nil, err
	}

	var pending []Migration
	for _, migration := range m.migrations {
		if !applied[migration.Version] {
			pending = append(pending, migration)
		}
	}

	return pending, nil
}

// splitStatements splits a SQL string into individual statements.
// It handles semicolons within strings and comments properly.
func splitStatements(sql string) []string {
	var statements []string
	var current strings.Builder
	inString := false
	stringChar := rune(0)
	inBlock := 0

	for i, char := range sql {
		switch {
		case !inString && (char == '\'' || char == '"'):
			inString = true
			stringChar = char
			current.WriteRune(char)
		case inString && char == stringChar:
			if i+1 < len(sql) && rune(sql[i+1]) == stringChar {
				current.WriteRune(char)
			} else {
				inString = false
				current.WriteRune(char)
			}
		case !inString && char == ';':
			if inBlock > 0 {
				current.WriteRune(char)
			} else {
				stmt := strings.TrimSpace(current.String())
				if stmt != "" {
					statements = append(statements, stmt)
				}
				current.Reset()
			}
		default:
			current.WriteRune(char)
			if !inString {
				built := current.String()
				upper := strings.ToUpper(built)
				if strings.HasSuffix(upper, " BEGIN") || strings.HasSuffix(upper, "\nBEGIN") || strings.HasSuffix(upper, "\tBEGIN") || upper == "BEGIN" {
					inBlock++
				}
				if inBlock > 0 && (strings.HasSuffix(upper, "\nEND") || strings.HasSuffix(upper, " END") || strings.HasSuffix(upper, "\tEND")) {
					inBlock--
				}
			}
		}
	}

	stmt := strings.TrimSpace(current.String())
	if stmt != "" {
		statements = append(statements, stmt)
	}

	return statements
}

// Ensure Migrator implements repository.Migrator
var _ repository.Migrator = (*Migrator)(nil)

// Unused import prevention
var _ = sql.ErrNoRows
