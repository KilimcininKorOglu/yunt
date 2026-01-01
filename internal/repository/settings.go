package repository

import (
	"context"

	"yunt/internal/domain"
)

// SettingsRepository provides data access operations for Settings entities.
// Settings are typically a singleton resource with a single global configuration.
type SettingsRepository interface {
	// Get retrieves the current settings.
	// Returns default settings if none have been saved.
	Get(ctx context.Context) (*domain.Settings, error)

	// GetByID retrieves settings by their ID.
	// This is useful when multiple settings configurations exist (e.g., per-environment).
	// Returns domain.ErrNotFound if settings with the ID do not exist.
	GetByID(ctx context.Context, id domain.ID) (*domain.Settings, error)

	// Save saves or updates the settings.
	// If settings do not exist, they are created.
	// If settings already exist, they are updated.
	Save(ctx context.Context, settings *domain.Settings) error

	// Update applies partial updates to settings.
	// Only the specified fields are updated.
	// Returns domain.ErrNotFound if settings do not exist.
	Update(ctx context.Context, id domain.ID, input *domain.SettingsUpdateInput) error

	// Reset resets settings to their default values.
	Reset(ctx context.Context) error

	// Exists checks if settings have been saved.
	Exists(ctx context.Context) (bool, error)

	// GetSMTP retrieves only the SMTP settings.
	GetSMTP(ctx context.Context) (*domain.SMTPSettings, error)

	// UpdateSMTP updates only the SMTP settings.
	UpdateSMTP(ctx context.Context, update *domain.SMTPSettingsUpdate) error

	// GetIMAP retrieves only the IMAP settings.
	GetIMAP(ctx context.Context) (*domain.IMAPSettings, error)

	// UpdateIMAP updates only the IMAP settings.
	UpdateIMAP(ctx context.Context, update *domain.IMAPSettingsUpdate) error

	// GetWebUI retrieves only the Web UI settings.
	GetWebUI(ctx context.Context) (*domain.WebUISettings, error)

	// UpdateWebUI updates only the Web UI settings.
	UpdateWebUI(ctx context.Context, update *domain.WebUISettingsUpdate) error

	// GetStorage retrieves only the storage settings.
	GetStorage(ctx context.Context) (*domain.StorageSettings, error)

	// UpdateStorage updates only the storage settings.
	UpdateStorage(ctx context.Context, update *domain.StorageSettingsUpdate) error

	// GetSecurity retrieves only the security settings.
	GetSecurity(ctx context.Context) (*domain.SecuritySettings, error)

	// UpdateSecurity updates only the security settings.
	UpdateSecurity(ctx context.Context, update *domain.SecuritySettingsUpdate) error

	// GetRetention retrieves only the retention settings.
	GetRetention(ctx context.Context) (*domain.RetentionSettings, error)

	// UpdateRetention updates only the retention settings.
	UpdateRetention(ctx context.Context, update *domain.RetentionSettingsUpdate) error

	// GetNotifications retrieves only the notification settings.
	GetNotifications(ctx context.Context) (*domain.NotificationSettings, error)

	// UpdateNotifications updates only the notification settings.
	UpdateNotifications(ctx context.Context, update *domain.NotificationSettingsUpdate) error

	// GetSettingValue retrieves a specific setting value by path.
	// The path uses dot notation (e.g., "smtp.port", "security.jwtExpiration").
	// Returns nil if the setting is not found or the path is invalid.
	GetSettingValue(ctx context.Context, path string) (interface{}, error)

	// SetSettingValue sets a specific setting value by path.
	// The path uses dot notation (e.g., "smtp.port", "security.jwtExpiration").
	// Returns an error if the path is invalid or the value type is incorrect.
	SetSettingValue(ctx context.Context, path string, value interface{}) error

	// GetHistory retrieves the history of settings changes.
	// This is useful for auditing and rollback scenarios.
	GetHistory(ctx context.Context, opts *ListOptions) (*ListResult[*SettingsChange], error)

	// GetHistoryByField retrieves the history of changes for a specific field.
	GetHistoryByField(ctx context.Context, fieldPath string, opts *ListOptions) (*ListResult[*SettingsChange], error)

	// Revert reverts settings to a specific historical version.
	// Returns domain.ErrNotFound if the version does not exist.
	Revert(ctx context.Context, changeID domain.ID) error

	// Export exports settings in a portable format (e.g., for backup or migration).
	Export(ctx context.Context) (*SettingsExport, error)

	// Import imports settings from a portable format.
	// merge indicates whether to merge with existing settings or replace them.
	Import(ctx context.Context, data *SettingsExport, merge bool) error

	// Validate validates the current settings.
	// Returns a list of validation errors if any settings are invalid.
	Validate(ctx context.Context) ([]*SettingsValidationError, error)

	// GetDatabaseInfo retrieves information about the database connection.
	GetDatabaseInfo(ctx context.Context) (*DatabaseInfo, error)

	// TestSMTPConnection tests the SMTP relay connection.
	TestSMTPConnection(ctx context.Context) error

	// TestDatabaseConnection tests the database connection.
	TestDatabaseConnection(ctx context.Context) error
}

// SettingsChange represents a historical change to settings.
type SettingsChange struct {
	// ID is the unique identifier for this change.
	ID domain.ID

	// FieldPath is the path to the changed field (dot notation).
	FieldPath string

	// OldValue is the previous value (as JSON).
	OldValue string

	// NewValue is the new value (as JSON).
	NewValue string

	// ChangedBy is the user ID who made the change (if applicable).
	ChangedBy *domain.ID

	// ChangedAt is when the change was made.
	ChangedAt domain.Timestamp

	// Reason is an optional description of why the change was made.
	Reason string
}

// SettingsExport represents exported settings for backup or migration.
type SettingsExport struct {
	// Version is the export format version.
	Version string

	// ExportedAt is when the export was created.
	ExportedAt domain.Timestamp

	// Settings is the serialized settings data.
	Settings *domain.Settings

	// Checksum is a hash of the settings for integrity verification.
	Checksum string
}

// SettingsValidationError represents a validation error in settings.
type SettingsValidationError struct {
	// Path is the path to the invalid setting (dot notation).
	Path string

	// Message describes why validation failed.
	Message string

	// Severity indicates how critical the error is.
	Severity ValidationSeverity
}

// ValidationSeverity represents the severity of a validation error.
type ValidationSeverity string

const (
	// ValidationSeverityError indicates a critical error that prevents operation.
	ValidationSeverityError ValidationSeverity = "error"

	// ValidationSeverityWarning indicates a non-critical issue.
	ValidationSeverityWarning ValidationSeverity = "warning"

	// ValidationSeverityInfo indicates informational feedback.
	ValidationSeverityInfo ValidationSeverity = "info"
)

// DatabaseInfo contains information about the database connection.
type DatabaseInfo struct {
	// Driver is the database driver being used.
	Driver domain.DatabaseDriver

	// Version is the database server version.
	Version string

	// Host is the database server host.
	Host string

	// Database is the database name.
	Database string

	// ConnectionCount is the current number of open connections.
	ConnectionCount int

	// MaxConnections is the maximum allowed connections.
	MaxConnections int

	// PoolStats contains connection pool statistics (if applicable).
	PoolStats *ConnectionPoolStats

	// Size is the total database size in bytes.
	Size int64

	// TableStats contains statistics for each table.
	TableStats []TableStats
}

// ConnectionPoolStats contains database connection pool statistics.
type ConnectionPoolStats struct {
	// OpenConnections is the number of open connections.
	OpenConnections int

	// InUse is the number of connections currently in use.
	InUse int

	// Idle is the number of idle connections.
	Idle int

	// WaitCount is the total number of connections waited for.
	WaitCount int64

	// WaitDuration is the total time waited for connections.
	WaitDuration int64

	// MaxIdleClosed is connections closed due to max idle count.
	MaxIdleClosed int64

	// MaxLifetimeClosed is connections closed due to max lifetime.
	MaxLifetimeClosed int64
}

// TableStats contains statistics for a database table.
type TableStats struct {
	// Name is the table name.
	Name string

	// RowCount is the number of rows in the table.
	RowCount int64

	// Size is the table size in bytes.
	Size int64

	// IndexSize is the index size in bytes.
	IndexSize int64
}
