package factory

import (
	"context"
	"strings"
	"testing"
	"time"

	"yunt/internal/config"
	"yunt/internal/domain"
)

// skipIfCGODisabled skips the test if CGO is not enabled.
// SQLite requires CGO to work, so tests that create actual SQLite connections
// must be skipped when running without CGO.
func skipIfCGODisabled(t *testing.T, err error) bool {
	t.Helper()
	if err != nil && strings.Contains(err.Error(), "CGO_ENABLED=0") {
		t.Skip("skipping test: SQLite requires CGO to be enabled")
		return true
	}
	return false
}

// TestNew tests the factory constructor.
func TestNew(t *testing.T) {
	tests := []struct {
		name    string
		config  *config.DatabaseConfig
		wantErr bool
		errMsg  string
	}{
		{
			name:    "nil config",
			config:  nil,
			wantErr: true,
			errMsg:  "database configuration is required",
		},
		{
			name: "valid sqlite config",
			config: &config.DatabaseConfig{
				Driver: "sqlite",
				DSN:    ":memory:",
			},
			wantErr: false,
		},
		{
			name: "valid sqlite config with empty DSN",
			config: &config.DatabaseConfig{
				Driver: "sqlite",
			},
			wantErr: false,
		},
		{
			name: "empty driver",
			config: &config.DatabaseConfig{
				Driver: "",
			},
			wantErr: true,
			errMsg:  "driver is required",
		},
		{
			name: "invalid driver",
			config: &config.DatabaseConfig{
				Driver: "oracle",
			},
			wantErr: true,
			errMsg:  "invalid driver",
		},
		{
			name: "postgres without connection info",
			config: &config.DatabaseConfig{
				Driver: "postgres",
			},
			wantErr: true,
			errMsg:  "either DSN or Host is required",
		},
		{
			name: "postgres with DSN",
			config: &config.DatabaseConfig{
				Driver: "postgres",
				DSN:    "postgres://user:pass@localhost/db",
			},
			wantErr: false,
		},
		{
			name: "postgres with host",
			config: &config.DatabaseConfig{
				Driver: "postgres",
				Host:   "localhost",
				Port:   5432,
			},
			wantErr: false,
		},
		{
			name: "mysql without connection info",
			config: &config.DatabaseConfig{
				Driver: "mysql",
			},
			wantErr: true,
			errMsg:  "either DSN or Host is required",
		},
		{
			name: "mysql with DSN",
			config: &config.DatabaseConfig{
				Driver: "mysql",
				DSN:    "user:pass@tcp(localhost:3306)/db",
			},
			wantErr: false,
		},
		{
			name: "mongodb without connection info",
			config: &config.DatabaseConfig{
				Driver: "mongodb",
			},
			wantErr: true,
			errMsg:  "either DSN or Host is required",
		},
		{
			name: "mongodb with DSN",
			config: &config.DatabaseConfig{
				Driver: "mongodb",
				DSN:    "mongodb://localhost:27017",
			},
			wantErr: false,
		},
		{
			name: "negative max open conns",
			config: &config.DatabaseConfig{
				Driver:       "sqlite",
				MaxOpenConns: -1,
			},
			wantErr: true,
			errMsg:  "maxOpenConns cannot be negative",
		},
		{
			name: "negative max idle conns",
			config: &config.DatabaseConfig{
				Driver:       "sqlite",
				MaxIdleConns: -1,
			},
			wantErr: true,
			errMsg:  "maxIdleConns cannot be negative",
		},
		{
			name: "max idle greater than max open",
			config: &config.DatabaseConfig{
				Driver:       "sqlite",
				MaxOpenConns: 5,
				MaxIdleConns: 10,
			},
			wantErr: true,
			errMsg:  "maxIdleConns (10) cannot be greater than maxOpenConns (5)",
		},
		{
			name: "valid connection pool settings",
			config: &config.DatabaseConfig{
				Driver:          "sqlite",
				MaxOpenConns:    10,
				MaxIdleConns:    5,
				ConnMaxLifetime: time.Hour,
				ConnMaxIdleTime: 30 * time.Minute,
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			factory, err := New(tt.config)

			if tt.wantErr {
				if err == nil {
					t.Error("expected error but got nil")
					return
				}
				if tt.errMsg != "" && !containsString(err.Error(), tt.errMsg) {
					t.Errorf("error message mismatch: got %q, want to contain %q", err.Error(), tt.errMsg)
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if factory == nil {
				t.Error("expected factory but got nil")
			}
		})
	}
}

// TestFactoryCreate tests repository creation.
func TestFactoryCreate(t *testing.T) {
	tests := []struct {
		name    string
		config  *config.DatabaseConfig
		wantErr bool
		errMsg  string
	}{
		{
			name: "create sqlite repository",
			config: &config.DatabaseConfig{
				Driver: "sqlite",
				DSN:    ":memory:",
			},
			wantErr: false,
		},
		// Note: postgres, mysql, mongodb factory methods are implemented,
		// but creating actual connections requires running database instances.
		// Testing actual connections should be done in integration tests.
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			factory, err := New(tt.config)
			if err != nil {
				t.Fatalf("failed to create factory: %v", err)
			}

			repo, err := factory.Create()

			if tt.wantErr {
				if err == nil {
					t.Error("expected error but got nil")
					if repo != nil {
						repo.Close()
					}
					return
				}
				if tt.errMsg != "" && !containsString(err.Error(), tt.errMsg) {
					t.Errorf("error message mismatch: got %q, want to contain %q", err.Error(), tt.errMsg)
				}
				return
			}

			// Skip SQLite tests if CGO is disabled
			if skipIfCGODisabled(t, err) {
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if repo == nil {
				t.Error("expected repository but got nil")
				return
			}

			// Verify it implements the interface by accessing sub-repositories
			if repo.Users() == nil {
				t.Error("Users() should not return nil")
			}
			if repo.Mailboxes() == nil {
				t.Error("Mailboxes() should not return nil")
			}
			if repo.Messages() == nil {
				t.Error("Messages() should not return nil")
			}
			if repo.Attachments() == nil {
				t.Error("Attachments() should not return nil")
			}
			if repo.Webhooks() == nil {
				t.Error("Webhooks() should not return nil")
			}
			if repo.Settings() == nil {
				t.Error("Settings() should not return nil")
			}

			// Cleanup
			if err := repo.Close(); err != nil {
				t.Errorf("failed to close repository: %v", err)
			}
		})
	}
}

// TestFactoryDriver tests the Driver method.
func TestFactoryDriver(t *testing.T) {
	tests := []struct {
		driver   string
		expected domain.DatabaseDriver
	}{
		{"sqlite", domain.DatabaseDriverSQLite},
		{"postgres", domain.DatabaseDriverPostgres},
		{"mysql", domain.DatabaseDriverMySQL},
		{"mongodb", domain.DatabaseDriverMongoDB},
	}

	for _, tt := range tests {
		t.Run(tt.driver, func(t *testing.T) {
			cfg := &config.DatabaseConfig{
				Driver: tt.driver,
			}
			// Add required fields for non-sqlite drivers
			if tt.driver != "sqlite" {
				cfg.DSN = "test://localhost"
			}

			factory, err := New(cfg)
			if err != nil {
				t.Fatalf("failed to create factory: %v", err)
			}

			if factory.Driver() != tt.expected {
				t.Errorf("Driver() = %v, want %v", factory.Driver(), tt.expected)
			}
		})
	}
}

// TestFactoryConfig tests the Config method.
func TestFactoryConfig(t *testing.T) {
	cfg := &config.DatabaseConfig{
		Driver: "sqlite",
		DSN:    ":memory:",
	}

	factory, err := New(cfg)
	if err != nil {
		t.Fatalf("failed to create factory: %v", err)
	}

	if factory.Config() != cfg {
		t.Error("Config() should return the same config instance")
	}

	if factory.Config().Driver != "sqlite" {
		t.Errorf("Config().Driver = %v, want sqlite", factory.Config().Driver)
	}
}

// TestSupportedDrivers tests the SupportedDrivers function.
func TestSupportedDrivers(t *testing.T) {
	drivers := SupportedDrivers()

	expectedDrivers := []domain.DatabaseDriver{
		domain.DatabaseDriverSQLite,
		domain.DatabaseDriverPostgres,
		domain.DatabaseDriverMySQL,
		domain.DatabaseDriverMongoDB,
	}

	if len(drivers) != len(expectedDrivers) {
		t.Errorf("SupportedDrivers() returned %d drivers, want %d", len(drivers), len(expectedDrivers))
	}

	for _, expected := range expectedDrivers {
		found := false
		for _, driver := range drivers {
			if driver == expected {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("SupportedDrivers() missing expected driver: %v", expected)
		}
	}
}

// TestIsDriverSupported tests the IsDriverSupported function.
func TestIsDriverSupported(t *testing.T) {
	tests := []struct {
		driver string
		want   bool
	}{
		{"sqlite", true},
		{"postgres", true},
		{"mysql", true},
		{"mongodb", true},
		{"oracle", false},
		{"mssql", false},
		{"", false},
		{"SQLITE", false}, // case sensitive
	}

	for _, tt := range tests {
		t.Run(tt.driver, func(t *testing.T) {
			if got := IsDriverSupported(tt.driver); got != tt.want {
				t.Errorf("IsDriverSupported(%q) = %v, want %v", tt.driver, got, tt.want)
			}
		})
	}
}

// TestImplementedDrivers tests the ImplementedDrivers function.
func TestImplementedDrivers(t *testing.T) {
	drivers := ImplementedDrivers()

	// All four drivers are now implemented
	expectedDrivers := []domain.DatabaseDriver{
		domain.DatabaseDriverSQLite,
		domain.DatabaseDriverPostgres,
		domain.DatabaseDriverMySQL,
		domain.DatabaseDriverMongoDB,
	}

	if len(drivers) != len(expectedDrivers) {
		t.Errorf("ImplementedDrivers() returned %d drivers, want %d", len(drivers), len(expectedDrivers))
	}

	for _, expected := range expectedDrivers {
		found := false
		for _, driver := range drivers {
			if driver == expected {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("ImplementedDrivers() missing expected driver: %v", expected)
		}
	}
}

// TestIsDriverImplemented tests the IsDriverImplemented function.
func TestIsDriverImplemented(t *testing.T) {
	tests := []struct {
		driver string
		want   bool
	}{
		{"sqlite", true},
		{"postgres", true},
		{"mysql", true},
		{"mongodb", true},
		{"oracle", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.driver, func(t *testing.T) {
			if got := IsDriverImplemented(tt.driver); got != tt.want {
				t.Errorf("IsDriverImplemented(%q) = %v, want %v", tt.driver, got, tt.want)
			}
		})
	}
}

// TestCreateSQLiteRepository tests SQLite repository creation in detail.
func TestCreateSQLiteRepository(t *testing.T) {
	cfg := &config.DatabaseConfig{
		Driver:       "sqlite",
		DSN:          ":memory:",
		MaxOpenConns: 1,
		MaxIdleConns: 1,
	}

	factory, err := New(cfg)
	if err != nil {
		t.Fatalf("failed to create factory: %v", err)
	}

	repo, err := factory.Create()
	if skipIfCGODisabled(t, err) {
		return
	}
	if err != nil {
		t.Fatalf("failed to create repository: %v", err)
	}
	defer repo.Close()

	// Test health check
	ctx := context.Background()
	if err := repo.Health(ctx); err != nil {
		t.Errorf("Health() failed: %v", err)
	}

	// Test that we can perform basic operations
	users := repo.Users()
	if users == nil {
		t.Fatal("Users() returned nil")
	}

	// Count should work
	count, err := users.Count(ctx, nil)
	if err != nil {
		t.Errorf("Count() failed: %v", err)
	}
	if count != 0 {
		t.Errorf("expected count 0, got %d", count)
	}
}

// TestValidateConfig tests the ValidateConfig function directly.
func TestValidateConfig(t *testing.T) {
	tests := []struct {
		name    string
		config  *config.DatabaseConfig
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid minimal sqlite",
			config: &config.DatabaseConfig{
				Driver: "sqlite",
			},
			wantErr: false,
		},
		{
			name: "valid complete config",
			config: &config.DatabaseConfig{
				Driver:          "sqlite",
				DSN:             ":memory:",
				MaxOpenConns:    10,
				MaxIdleConns:    5,
				ConnMaxLifetime: time.Hour,
				ConnMaxIdleTime: 30 * time.Minute,
			},
			wantErr: false,
		},
		{
			name: "zero max conns is valid",
			config: &config.DatabaseConfig{
				Driver:       "sqlite",
				MaxOpenConns: 0,
				MaxIdleConns: 0,
			},
			wantErr: false,
		},
		{
			name: "case sensitive driver",
			config: &config.DatabaseConfig{
				Driver: "SQLite",
			},
			wantErr: true,
			errMsg:  "invalid driver",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateConfig(tt.config)

			if tt.wantErr {
				if err == nil {
					t.Error("expected error but got nil")
					return
				}
				if tt.errMsg != "" && !containsString(err.Error(), tt.errMsg) {
					t.Errorf("error message mismatch: got %q, want to contain %q", err.Error(), tt.errMsg)
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

// TestFactoryWithOptions tests that options are applied.
func TestFactoryWithOptions(t *testing.T) {
	cfg := &config.DatabaseConfig{
		Driver: "sqlite",
		DSN:    ":memory:",
	}

	optionApplied := false
	testOption := func(f *Factory) {
		optionApplied = true
	}

	factory, err := New(cfg, testOption)
	if err != nil {
		t.Fatalf("failed to create factory: %v", err)
	}

	if factory == nil {
		t.Fatal("factory is nil")
	}

	if !optionApplied {
		t.Error("option was not applied")
	}
}

// TestUnsupportedDriverError tests that unsupported drivers return descriptive errors.
func TestUnsupportedDriverError(t *testing.T) {
	// This test verifies the error message format for unsupported drivers
	cfg := &config.DatabaseConfig{
		Driver: "cassandra",
	}

	_, err := New(cfg)
	if err == nil {
		t.Fatal("expected error for unsupported driver")
	}

	// Error should mention supported drivers
	errStr := err.Error()
	if !containsString(errStr, "sqlite") ||
		!containsString(errStr, "postgres") ||
		!containsString(errStr, "mysql") ||
		!containsString(errStr, "mongodb") {
		t.Errorf("error message should list supported drivers: %s", errStr)
	}
}

// TestFactoryReturnsInterface tests that Create returns interface not concrete type.
func TestFactoryReturnsInterface(t *testing.T) {
	cfg := &config.DatabaseConfig{
		Driver: "sqlite",
		DSN:    ":memory:",
	}

	factory, err := New(cfg)
	if err != nil {
		t.Fatalf("failed to create factory: %v", err)
	}

	// This verifies at compile time that Create returns repository.Repository
	// If it returned a concrete type, this would need explicit conversion
	repo, err := factory.Create()
	if skipIfCGODisabled(t, err) {
		return
	}
	if err != nil {
		t.Fatalf("failed to create repository: %v", err)
	}
	defer repo.Close()

	// Additional verification: we can assign to the interface type
	var _ interface {
		Health(context.Context) error
		Close() error
	} = repo
}

// TestMultipleRepositoryCreation tests creating multiple repositories.
func TestMultipleRepositoryCreation(t *testing.T) {
	cfg := &config.DatabaseConfig{
		Driver: "sqlite",
		DSN:    ":memory:",
	}

	factory, err := New(cfg)
	if err != nil {
		t.Fatalf("failed to create factory: %v", err)
	}

	// Create multiple repositories
	repos := make([]interface{ Close() error }, 3)
	for i := 0; i < 3; i++ {
		repo, err := factory.Create()
		if skipIfCGODisabled(t, err) {
			return
		}
		if err != nil {
			t.Fatalf("failed to create repository %d: %v", i, err)
		}
		repos[i] = repo
	}

	// Clean up
	for _, repo := range repos {
		if err := repo.Close(); err != nil {
			t.Errorf("failed to close repository: %v", err)
		}
	}
}

// containsString checks if s contains substr.
func containsString(s, substr string) bool {
	if len(substr) > len(s) {
		return false
	}
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
