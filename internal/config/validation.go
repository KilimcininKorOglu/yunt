package config

import (
	"errors"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strings"
)

// ValidationError represents a configuration validation error.
type ValidationError struct {
	Field   string
	Message string
}

func (e *ValidationError) Error() string {
	return fmt.Sprintf("%s: %s", e.Field, e.Message)
}

// ValidationErrors is a collection of validation errors.
type ValidationErrors []ValidationError

func (e ValidationErrors) Error() string {
	if len(e) == 0 {
		return ""
	}

	var sb strings.Builder
	sb.WriteString("configuration validation failed:\n")
	for _, err := range e {
		sb.WriteString(fmt.Sprintf("  - %s: %s\n", err.Field, err.Message))
	}
	return sb.String()
}

// Validate validates the configuration and returns any errors.
func Validate(cfg *Config) error {
	var errs ValidationErrors

	errs = append(errs, validateServer(&cfg.Server)...)
	errs = append(errs, validateSMTP(&cfg.SMTP)...)
	errs = append(errs, validateIMAP(&cfg.IMAP)...)
	errs = append(errs, validateAPI(&cfg.API)...)
	errs = append(errs, validateDatabase(&cfg.Database)...)
	errs = append(errs, validateAuth(&cfg.Auth)...)
	errs = append(errs, validateLogging(&cfg.Logging)...)
	errs = append(errs, validateAdmin(&cfg.Admin)...)
	errs = append(errs, validateStorage(&cfg.Storage)...)

	if len(errs) > 0 {
		return errs
	}

	return nil
}

// validateServer validates server configuration.
func validateServer(cfg *ServerConfig) ValidationErrors {
	var errs ValidationErrors

	if cfg.Name == "" {
		errs = append(errs, ValidationError{Field: "server.name", Message: "server name is required"})
	}

	if cfg.Domain == "" {
		errs = append(errs, ValidationError{Field: "server.domain", Message: "server domain is required"})
	}

	if cfg.GracefulTimeout < 0 {
		errs = append(errs, ValidationError{Field: "server.gracefulTimeout", Message: "graceful timeout cannot be negative"})
	}

	return errs
}

// validateSMTP validates SMTP server configuration.
func validateSMTP(cfg *SMTPConfig) ValidationErrors {
	var errs ValidationErrors

	if !cfg.Enabled {
		return errs
	}

	if err := validateHost(cfg.Host); err != nil {
		errs = append(errs, ValidationError{Field: "smtp.host", Message: err.Error()})
	}

	if err := validatePort(cfg.Port); err != nil {
		errs = append(errs, ValidationError{Field: "smtp.port", Message: err.Error()})
	}

	if cfg.MaxMessageSize <= 0 {
		errs = append(errs, ValidationError{Field: "smtp.maxMessageSize", Message: "max message size must be positive"})
	}

	if cfg.MaxRecipients <= 0 {
		errs = append(errs, ValidationError{Field: "smtp.maxRecipients", Message: "max recipients must be positive"})
	}

	if cfg.ReadTimeout < 0 {
		errs = append(errs, ValidationError{Field: "smtp.readTimeout", Message: "read timeout cannot be negative"})
	}

	if cfg.WriteTimeout < 0 {
		errs = append(errs, ValidationError{Field: "smtp.writeTimeout", Message: "write timeout cannot be negative"})
	}

	errs = append(errs, validateTLS(&cfg.TLS, "smtp.tls")...)

	if cfg.AllowRelay {
		if cfg.RelayHost == "" {
			errs = append(errs, ValidationError{Field: "smtp.relayHost", Message: "relay host is required when relay is enabled"})
		}
		if err := validatePort(cfg.RelayPort); err != nil {
			errs = append(errs, ValidationError{Field: "smtp.relayPort", Message: err.Error()})
		}
	}

	return errs
}

// validateIMAP validates IMAP server configuration.
func validateIMAP(cfg *IMAPConfig) ValidationErrors {
	var errs ValidationErrors

	if !cfg.Enabled {
		return errs
	}

	if err := validateHost(cfg.Host); err != nil {
		errs = append(errs, ValidationError{Field: "imap.host", Message: err.Error()})
	}

	if err := validatePort(cfg.Port); err != nil {
		errs = append(errs, ValidationError{Field: "imap.port", Message: err.Error()})
	}

	if cfg.ReadTimeout < 0 {
		errs = append(errs, ValidationError{Field: "imap.readTimeout", Message: "read timeout cannot be negative"})
	}

	if cfg.WriteTimeout < 0 {
		errs = append(errs, ValidationError{Field: "imap.writeTimeout", Message: "write timeout cannot be negative"})
	}

	if cfg.IdleTimeout < 0 {
		errs = append(errs, ValidationError{Field: "imap.idleTimeout", Message: "idle timeout cannot be negative"})
	}

	errs = append(errs, validateTLS(&cfg.TLS, "imap.tls")...)

	return errs
}

// validateAPI validates API server configuration.
func validateAPI(cfg *APIConfig) ValidationErrors {
	var errs ValidationErrors

	if !cfg.Enabled {
		return errs
	}

	if err := validateHost(cfg.Host); err != nil {
		errs = append(errs, ValidationError{Field: "api.host", Message: err.Error()})
	}

	if err := validatePort(cfg.Port); err != nil {
		errs = append(errs, ValidationError{Field: "api.port", Message: err.Error()})
	}

	if cfg.ReadTimeout < 0 {
		errs = append(errs, ValidationError{Field: "api.readTimeout", Message: "read timeout cannot be negative"})
	}

	if cfg.WriteTimeout < 0 {
		errs = append(errs, ValidationError{Field: "api.writeTimeout", Message: "write timeout cannot be negative"})
	}

	if cfg.RateLimit < 0 {
		errs = append(errs, ValidationError{Field: "api.rateLimit", Message: "rate limit cannot be negative"})
	}

	errs = append(errs, validateTLS(&cfg.TLS, "api.tls")...)

	return errs
}

// validateDatabase validates database configuration.
func validateDatabase(cfg *DatabaseConfig) ValidationErrors {
	var errs ValidationErrors

	validDrivers := []string{"sqlite", "postgres", "mysql", "mongodb"}
	if !contains(validDrivers, cfg.Driver) {
		errs = append(errs, ValidationError{
			Field:   "database.driver",
			Message: fmt.Sprintf("invalid driver '%s', must be one of: %s", cfg.Driver, strings.Join(validDrivers, ", ")),
		})
	}

	if cfg.Driver != "sqlite" && cfg.DSN == "" {
		if cfg.Host == "" {
			errs = append(errs, ValidationError{Field: "database.host", Message: "database host is required"})
		}
		if cfg.Name == "" {
			errs = append(errs, ValidationError{Field: "database.name", Message: "database name is required"})
		}
	}

	if cfg.MaxOpenConns < 0 {
		errs = append(errs, ValidationError{Field: "database.maxOpenConns", Message: "max open connections cannot be negative"})
	}

	if cfg.MaxIdleConns < 0 {
		errs = append(errs, ValidationError{Field: "database.maxIdleConns", Message: "max idle connections cannot be negative"})
	}

	if cfg.MaxIdleConns > cfg.MaxOpenConns && cfg.MaxOpenConns > 0 {
		errs = append(errs, ValidationError{
			Field:   "database.maxIdleConns",
			Message: "max idle connections cannot exceed max open connections",
		})
	}

	if cfg.ConnMaxLifetime < 0 {
		errs = append(errs, ValidationError{Field: "database.connMaxLifetime", Message: "connection max lifetime cannot be negative"})
	}

	if cfg.ConnMaxIdleTime < 0 {
		errs = append(errs, ValidationError{Field: "database.connMaxIdleTime", Message: "connection max idle time cannot be negative"})
	}

	return errs
}

// validateAuth validates authentication configuration.
func validateAuth(cfg *AuthConfig) ValidationErrors {
	var errs ValidationErrors

	if cfg.JWTExpiration <= 0 {
		errs = append(errs, ValidationError{Field: "auth.jwtExpiration", Message: "JWT expiration must be positive"})
	}

	if cfg.RefreshExpiration <= 0 {
		errs = append(errs, ValidationError{Field: "auth.refreshExpiration", Message: "refresh expiration must be positive"})
	}

	if cfg.RefreshExpiration < cfg.JWTExpiration {
		errs = append(errs, ValidationError{
			Field:   "auth.refreshExpiration",
			Message: "refresh expiration should be greater than or equal to JWT expiration",
		})
	}

	if cfg.BCryptCost < 4 || cfg.BCryptCost > 31 {
		errs = append(errs, ValidationError{Field: "auth.bcryptCost", Message: "bcrypt cost must be between 4 and 31"})
	}

	if cfg.SessionTimeout < 0 {
		errs = append(errs, ValidationError{Field: "auth.sessionTimeout", Message: "session timeout cannot be negative"})
	}

	if cfg.MaxLoginAttempts < 0 {
		errs = append(errs, ValidationError{Field: "auth.maxLoginAttempts", Message: "max login attempts cannot be negative"})
	}

	if cfg.LockoutDuration < 0 {
		errs = append(errs, ValidationError{Field: "auth.lockoutDuration", Message: "lockout duration cannot be negative"})
	}

	return errs
}

// validateLogging validates logging configuration.
func validateLogging(cfg *LoggingConfig) ValidationErrors {
	var errs ValidationErrors

	validLevels := []string{"debug", "info", "warn", "error"}
	if !contains(validLevels, strings.ToLower(cfg.Level)) {
		errs = append(errs, ValidationError{
			Field:   "logging.level",
			Message: fmt.Sprintf("invalid log level '%s', must be one of: %s", cfg.Level, strings.Join(validLevels, ", ")),
		})
	}

	validFormats := []string{"json", "text"}
	if !contains(validFormats, strings.ToLower(cfg.Format)) {
		errs = append(errs, ValidationError{
			Field:   "logging.format",
			Message: fmt.Sprintf("invalid log format '%s', must be one of: %s", cfg.Format, strings.Join(validFormats, ", ")),
		})
	}

	if cfg.Output == "file" && cfg.FilePath == "" {
		errs = append(errs, ValidationError{Field: "logging.filePath", Message: "file path is required when output is 'file'"})
	}

	if cfg.MaxSize < 0 {
		errs = append(errs, ValidationError{Field: "logging.maxSize", Message: "max size cannot be negative"})
	}

	if cfg.MaxBackups < 0 {
		errs = append(errs, ValidationError{Field: "logging.maxBackups", Message: "max backups cannot be negative"})
	}

	if cfg.MaxAge < 0 {
		errs = append(errs, ValidationError{Field: "logging.maxAge", Message: "max age cannot be negative"})
	}

	return errs
}

// validateAdmin validates admin configuration.
func validateAdmin(cfg *AdminConfig) ValidationErrors {
	var errs ValidationErrors

	if !cfg.CreateOnStartup {
		return errs
	}

	if cfg.Username == "" {
		errs = append(errs, ValidationError{Field: "admin.username", Message: "admin username is required"})
	}

	if len(cfg.Username) < 3 {
		errs = append(errs, ValidationError{Field: "admin.username", Message: "admin username must be at least 3 characters"})
	}

	if cfg.Email == "" {
		errs = append(errs, ValidationError{Field: "admin.email", Message: "admin email is required"})
	}

	if !strings.Contains(cfg.Email, "@") {
		errs = append(errs, ValidationError{Field: "admin.email", Message: "admin email must be a valid email address"})
	}

	return errs
}

// validateStorage validates storage configuration.
func validateStorage(cfg *StorageConfig) ValidationErrors {
	var errs ValidationErrors

	validTypes := []string{"database", "filesystem"}
	if !contains(validTypes, strings.ToLower(cfg.Type)) {
		errs = append(errs, ValidationError{
			Field:   "storage.type",
			Message: fmt.Sprintf("invalid storage type '%s', must be one of: %s", cfg.Type, strings.Join(validTypes, ", ")),
		})
	}

	if strings.ToLower(cfg.Type) == "filesystem" {
		if cfg.Path == "" {
			errs = append(errs, ValidationError{Field: "storage.path", Message: "storage path is required for filesystem storage"})
		}
	}

	if cfg.MaxMailboxSize < 0 {
		errs = append(errs, ValidationError{Field: "storage.maxMailboxSize", Message: "max mailbox size cannot be negative"})
	}

	if cfg.RetentionDays < 0 {
		errs = append(errs, ValidationError{Field: "storage.retentionDays", Message: "retention days cannot be negative"})
	}

	return errs
}

// validateTLS validates TLS configuration.
func validateTLS(cfg *TLSConfig, prefix string) ValidationErrors {
	var errs ValidationErrors

	if !cfg.Enabled && !cfg.StartTLS {
		return errs
	}

	if cfg.Enabled || cfg.StartTLS {
		if cfg.CertFile != "" || cfg.KeyFile != "" {
			if cfg.CertFile == "" {
				errs = append(errs, ValidationError{Field: prefix + ".certFile", Message: "certificate file is required when TLS is enabled"})
			} else if !fileExists(cfg.CertFile) {
				errs = append(errs, ValidationError{Field: prefix + ".certFile", Message: fmt.Sprintf("certificate file not found: %s", cfg.CertFile)})
			}

			if cfg.KeyFile == "" {
				errs = append(errs, ValidationError{Field: prefix + ".keyFile", Message: "key file is required when TLS is enabled"})
			} else if !fileExists(cfg.KeyFile) {
				errs = append(errs, ValidationError{Field: prefix + ".keyFile", Message: fmt.Sprintf("key file not found: %s", cfg.KeyFile)})
			}
		}
	}

	return errs
}

// validateHost validates a host address.
func validateHost(host string) error {
	if host == "" {
		return errors.New("host is required")
	}

	if host == "0.0.0.0" || host == "::" || host == "localhost" {
		return nil
	}

	if net.ParseIP(host) != nil {
		return nil
	}

	if _, err := net.LookupHost(host); err != nil {
		return fmt.Errorf("invalid host: %s", host)
	}

	return nil
}

// validatePort validates a port number.
func validatePort(port int) error {
	if port < 1 || port > 65535 {
		return fmt.Errorf("port must be between 1 and 65535, got %d", port)
	}
	return nil
}

// contains checks if a string is in a slice.
func contains(slice []string, s string) bool {
	for _, v := range slice {
		if v == s {
			return true
		}
	}
	return false
}

// fileExists checks if a file exists.
func fileExists(path string) bool {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return false
	}

	info, err := os.Stat(absPath)
	if err != nil {
		return false
	}

	return !info.IsDir()
}
