// Package config provides configuration management for the Yunt mail server.
// It supports YAML configuration files with environment variable overrides.
package config

import (
	"strings"
	"time"
)

// Config represents the complete configuration for the Yunt mail server.
type Config struct {
	// Server contains general server settings.
	Server ServerConfig `yaml:"server" envPrefix:"YUNT_SERVER_"`

	// SMTP contains SMTP server configuration.
	SMTP SMTPConfig `yaml:"smtp" envPrefix:"YUNT_SMTP_"`

	// IMAP contains IMAP server configuration.
	IMAP IMAPConfig `yaml:"imap" envPrefix:"YUNT_IMAP_"`

	// API contains REST API and Web UI configuration.
	API APIConfig `yaml:"api" envPrefix:"YUNT_API_"`

	// Database contains database connection settings.
	Database DatabaseConfig `yaml:"database" envPrefix:"YUNT_DATABASE_"`

	// Auth contains authentication settings.
	Auth AuthConfig `yaml:"auth" envPrefix:"YUNT_AUTH_"`

	// Logging contains logging configuration.
	Logging LoggingConfig `yaml:"logging" envPrefix:"YUNT_LOGGING_"`

	// Admin contains default admin user settings.
	Admin AdminConfig `yaml:"admin" envPrefix:"YUNT_ADMIN_"`

	// Storage contains mail storage settings.
	Storage StorageConfig `yaml:"storage" envPrefix:"YUNT_STORAGE_"`
}

// ServerConfig contains general server settings.
type ServerConfig struct {
	// Name is the server hostname used in SMTP HELO/EHLO.
	Name string `yaml:"name" env:"YUNT_SERVER_NAME"`

	// Domain is the primary mail domain.
	Domain string `yaml:"domain" env:"YUNT_SERVER_DOMAIN"`

	// LocalDomains lists all domains considered local for internal delivery.
	// Messages to these domains are delivered directly without relay.
	// The primary Domain is always included automatically.
	LocalDomains []string `yaml:"localDomains" env:"YUNT_SERVER_LOCAL_DOMAINS"`

	// GracefulTimeout is the duration to wait for graceful shutdown.
	GracefulTimeout time.Duration `yaml:"gracefulTimeout" env:"YUNT_SERVER_GRACEFUL_TIMEOUT"`
}

// IsLocalDomain checks if the given domain is in the local domains list.
func (c *ServerConfig) IsLocalDomain(domain string) bool {
	domain = strings.ToLower(domain)
	for _, d := range c.LocalDomains {
		if strings.ToLower(d) == domain {
			return true
		}
	}
	return false
}

// NormalizeLocalDomains ensures the primary Domain is always in LocalDomains.
func (c *ServerConfig) NormalizeLocalDomains() {
	if c.Domain != "" {
		found := false
		for _, d := range c.LocalDomains {
			if strings.EqualFold(d, c.Domain) {
				found = true
				break
			}
		}
		if !found {
			c.LocalDomains = append(c.LocalDomains, c.Domain)
		}
	}
	if len(c.LocalDomains) == 0 {
		c.LocalDomains = []string{"localhost"}
	}
}

// SMTPConfig contains SMTP server configuration.
type SMTPConfig struct {
	// Enabled determines if the SMTP server should start.
	Enabled bool `yaml:"enabled" env:"YUNT_SMTP_ENABLED"`

	// Host is the address to bind the SMTP server to.
	Host string `yaml:"host" env:"YUNT_SMTP_HOST"`

	// Port is the port number for the SMTP server.
	Port int `yaml:"port" env:"YUNT_SMTP_PORT"`

	// TLS contains TLS configuration for SMTP.
	TLS TLSConfig `yaml:"tls" envPrefix:"YUNT_SMTP_TLS_"`

	// MaxMessageSize is the maximum message size in bytes.
	MaxMessageSize int64 `yaml:"maxMessageSize" env:"YUNT_SMTP_MAX_MESSAGE_SIZE"`

	// MaxAttachmentSize is the maximum attachment size in bytes (0 = unlimited).
	MaxAttachmentSize int64 `yaml:"maxAttachmentSize" env:"YUNT_SMTP_MAX_ATTACHMENT_SIZE"`

	// MaxRecipients is the maximum number of recipients per message.
	MaxRecipients int `yaml:"maxRecipients" env:"YUNT_SMTP_MAX_RECIPIENTS"`

	// ReadTimeout is the read timeout for SMTP connections.
	ReadTimeout time.Duration `yaml:"readTimeout" env:"YUNT_SMTP_READ_TIMEOUT"`

	// WriteTimeout is the write timeout for SMTP connections.
	WriteTimeout time.Duration `yaml:"writeTimeout" env:"YUNT_SMTP_WRITE_TIMEOUT"`

	// AuthRequired determines if authentication is required for SMTP.
	AuthRequired bool `yaml:"authRequired" env:"YUNT_SMTP_AUTH_REQUIRED"`

	// AllowRelay determines if the server allows relaying to external domains.
	AllowRelay bool `yaml:"allowRelay" env:"YUNT_SMTP_ALLOW_RELAY"`

	// RelayHost is the external SMTP server for relaying (if enabled).
	RelayHost string `yaml:"relayHost" env:"YUNT_SMTP_RELAY_HOST"`

	// RelayPort is the port for the relay server.
	RelayPort int `yaml:"relayPort" env:"YUNT_SMTP_RELAY_PORT"`

	// RelayUsername is the username for relay authentication.
	RelayUsername string `yaml:"relayUsername" env:"YUNT_SMTP_RELAY_USERNAME"`

	// RelayPassword is the password for relay authentication.
	RelayPassword string `yaml:"relayPassword" env:"YUNT_SMTP_RELAY_PASSWORD"`

	// RelayUseTLS enables implicit TLS for the relay connection.
	RelayUseTLS bool `yaml:"relayUseTLS" env:"YUNT_SMTP_RELAY_USE_TLS"`

	// RelayUseSTARTTLS enables STARTTLS upgrade for the relay connection.
	RelayUseSTARTTLS bool `yaml:"relayUseSTARTTLS" env:"YUNT_SMTP_RELAY_USE_STARTTLS"`

	// RelayAllowedDomains is a list of domains allowed for relay (comma-separated).
	RelayAllowedDomains []string `yaml:"relayAllowedDomains" env:"YUNT_SMTP_RELAY_ALLOWED_DOMAINS"`

	// RelayTimeout is the timeout for relay operations.
	RelayTimeout time.Duration `yaml:"relayTimeout" env:"YUNT_SMTP_RELAY_TIMEOUT"`

	// RelayRetryCount is the number of retry attempts for failed relay.
	RelayRetryCount int `yaml:"relayRetryCount" env:"YUNT_SMTP_RELAY_RETRY_COUNT"`

	// RelayInsecureSkipVerify skips TLS certificate verification for relay.
	RelayInsecureSkipVerify bool `yaml:"relayInsecureSkipVerify" env:"YUNT_SMTP_RELAY_INSECURE_SKIP_VERIFY"`
}

// IMAPConfig contains IMAP server configuration.
type IMAPConfig struct {
	// Enabled determines if the IMAP server should start.
	Enabled bool `yaml:"enabled" env:"YUNT_IMAP_ENABLED"`

	// Host is the address to bind the IMAP server to.
	Host string `yaml:"host" env:"YUNT_IMAP_HOST"`

	// Port is the port number for the IMAP server.
	Port int `yaml:"port" env:"YUNT_IMAP_PORT"`

	// TLS contains TLS configuration for IMAP.
	TLS TLSConfig `yaml:"tls" envPrefix:"YUNT_IMAP_TLS_"`

	// ReadTimeout is the read timeout for IMAP connections.
	ReadTimeout time.Duration `yaml:"readTimeout" env:"YUNT_IMAP_READ_TIMEOUT"`

	// WriteTimeout is the write timeout for IMAP connections.
	WriteTimeout time.Duration `yaml:"writeTimeout" env:"YUNT_IMAP_WRITE_TIMEOUT"`

	// IdleTimeout is the timeout for IMAP IDLE connections.
	IdleTimeout time.Duration `yaml:"idleTimeout" env:"YUNT_IMAP_IDLE_TIMEOUT"`
}

// APIConfig contains REST API and Web UI configuration.
type APIConfig struct {
	// Enabled determines if the API server should start.
	Enabled bool `yaml:"enabled" env:"YUNT_API_ENABLED"`

	// Host is the address to bind the API server to.
	Host string `yaml:"host" env:"YUNT_API_HOST"`

	// Port is the port number for the API server.
	Port int `yaml:"port" env:"YUNT_API_PORT"`

	// TLS contains TLS configuration for the API.
	TLS TLSConfig `yaml:"tls" envPrefix:"YUNT_API_TLS_"`

	// ReadTimeout is the read timeout for API connections.
	ReadTimeout time.Duration `yaml:"readTimeout" env:"YUNT_API_READ_TIMEOUT"`

	// WriteTimeout is the write timeout for API connections.
	WriteTimeout time.Duration `yaml:"writeTimeout" env:"YUNT_API_WRITE_TIMEOUT"`

	// CORSAllowedOrigins is a list of allowed CORS origins.
	CORSAllowedOrigins []string `yaml:"corsAllowedOrigins" env:"YUNT_API_CORS_ALLOWED_ORIGINS"`

	// EnableRateLimit determines if rate limiting is enabled. Disabled by default.
	EnableRateLimit bool `yaml:"enableRateLimit" env:"YUNT_API_ENABLE_RATE_LIMIT"`

	// RateLimit is the number of requests per minute per IP (when rate limiting is enabled).
	RateLimit int `yaml:"rateLimit" env:"YUNT_API_RATE_LIMIT"`

	// EnableSwagger determines if Swagger documentation is enabled.
	EnableSwagger bool `yaml:"enableSwagger" env:"YUNT_API_ENABLE_SWAGGER"`

	// EnableMetrics determines if Prometheus /metrics endpoint is enabled.
	EnableMetrics bool `yaml:"enableMetrics" env:"YUNT_API_ENABLE_METRICS"`
}

// DatabaseConfig contains database connection settings.
type DatabaseConfig struct {
	// Driver is the database driver (sqlite, postgres, mysql, mongodb).
	Driver string `yaml:"driver" env:"YUNT_DATABASE_DRIVER"`

	// DSN is the data source name for the database connection.
	DSN string `yaml:"dsn" env:"YUNT_DATABASE_DSN"`

	// Host is the database host (for non-DSN configurations).
	Host string `yaml:"host" env:"YUNT_DATABASE_HOST"`

	// Port is the database port.
	Port int `yaml:"port" env:"YUNT_DATABASE_PORT"`

	// Name is the database name.
	Name string `yaml:"name" env:"YUNT_DATABASE_NAME"`

	// Username is the database username.
	Username string `yaml:"username" env:"YUNT_DATABASE_USERNAME"`

	// Password is the database password.
	Password string `yaml:"password" env:"YUNT_DATABASE_PASSWORD"`

	// SSLMode is the SSL mode for PostgreSQL connections.
	SSLMode string `yaml:"sslMode" env:"YUNT_DATABASE_SSL_MODE"`

	// MaxOpenConns is the maximum number of open connections.
	MaxOpenConns int `yaml:"maxOpenConns" env:"YUNT_DATABASE_MAX_OPEN_CONNS"`

	// MaxIdleConns is the maximum number of idle connections.
	MaxIdleConns int `yaml:"maxIdleConns" env:"YUNT_DATABASE_MAX_IDLE_CONNS"`

	// ConnMaxLifetime is the maximum connection lifetime.
	ConnMaxLifetime time.Duration `yaml:"connMaxLifetime" env:"YUNT_DATABASE_CONN_MAX_LIFETIME"`

	// ConnMaxIdleTime is the maximum connection idle time.
	ConnMaxIdleTime time.Duration `yaml:"connMaxIdleTime" env:"YUNT_DATABASE_CONN_MAX_IDLE_TIME"`

	// AutoMigrate determines if database migrations run automatically.
	AutoMigrate bool `yaml:"autoMigrate" env:"YUNT_DATABASE_AUTO_MIGRATE"`
}

// AuthConfig contains authentication settings.
type AuthConfig struct {
	// JWTSecret is the secret key for JWT token signing.
	JWTSecret string `yaml:"jwtSecret" env:"YUNT_AUTH_JWT_SECRET"`

	// JWTExpiration is the JWT token expiration duration.
	JWTExpiration time.Duration `yaml:"jwtExpiration" env:"YUNT_AUTH_JWT_EXPIRATION"`

	// RefreshExpiration is the refresh token expiration duration.
	RefreshExpiration time.Duration `yaml:"refreshExpiration" env:"YUNT_AUTH_REFRESH_EXPIRATION"`

	// BCryptCost is the bcrypt hashing cost.
	BCryptCost int `yaml:"bcryptCost" env:"YUNT_AUTH_BCRYPT_COST"`

	// SessionTimeout is the session timeout duration.
	SessionTimeout time.Duration `yaml:"sessionTimeout" env:"YUNT_AUTH_SESSION_TIMEOUT"`

	// MaxLoginAttempts is the maximum number of failed login attempts.
	MaxLoginAttempts int `yaml:"maxLoginAttempts" env:"YUNT_AUTH_MAX_LOGIN_ATTEMPTS"`

	// LockoutDuration is the account lockout duration after max attempts.
	LockoutDuration time.Duration `yaml:"lockoutDuration" env:"YUNT_AUTH_LOCKOUT_DURATION"`
}

// LoggingConfig contains logging configuration.
type LoggingConfig struct {
	// Level is the minimum log level (debug, info, warn, error).
	Level string `yaml:"level" env:"YUNT_LOGGING_LEVEL"`

	// Format is the log format (json, text).
	Format string `yaml:"format" env:"YUNT_LOGGING_FORMAT"`

	// Output is the log output destination (stdout, stderr, file path).
	Output string `yaml:"output" env:"YUNT_LOGGING_OUTPUT"`

	// FilePath is the log file path when output is "file".
	FilePath string `yaml:"filePath" env:"YUNT_LOGGING_FILE_PATH"`

	// MaxSize is the maximum log file size in megabytes.
	MaxSize int `yaml:"maxSize" env:"YUNT_LOGGING_MAX_SIZE"`

	// MaxBackups is the maximum number of old log files to retain.
	MaxBackups int `yaml:"maxBackups" env:"YUNT_LOGGING_MAX_BACKUPS"`

	// MaxAge is the maximum number of days to retain old log files.
	MaxAge int `yaml:"maxAge" env:"YUNT_LOGGING_MAX_AGE"`

	// Compress determines if old log files should be compressed.
	Compress bool `yaml:"compress" env:"YUNT_LOGGING_COMPRESS"`

	// IncludeCaller determines if caller info is included in logs.
	IncludeCaller bool `yaml:"includeCaller" env:"YUNT_LOGGING_INCLUDE_CALLER"`
}

// AdminConfig contains default admin user settings.
type AdminConfig struct {
	// Username is the default admin username.
	Username string `yaml:"username" env:"YUNT_ADMIN_USERNAME"`

	// Password is the default admin password.
	Password string `yaml:"password" env:"YUNT_ADMIN_PASSWORD"`

	// Email is the default admin email address.
	Email string `yaml:"email" env:"YUNT_ADMIN_EMAIL"`

	// CreateOnStartup determines if the admin user is created on startup.
	CreateOnStartup bool `yaml:"createOnStartup" env:"YUNT_ADMIN_CREATE_ON_STARTUP"`
}

// StorageConfig contains mail storage settings.
type StorageConfig struct {
	// Type is the storage type: "db" (default), "filesystem", "s3".
	Type string `yaml:"type" env:"YUNT_STORAGE_TYPE"`

	// Path is the filesystem storage path (when type is "filesystem").
	Path string `yaml:"path" env:"YUNT_STORAGE_PATH"`

	// S3Bucket is the S3 bucket name (when type is "s3").
	S3Bucket string `yaml:"s3Bucket" env:"YUNT_STORAGE_S3_BUCKET"`

	// S3Region is the AWS region (when type is "s3").
	S3Region string `yaml:"s3Region" env:"YUNT_STORAGE_S3_REGION"`

	// S3Endpoint is a custom S3 endpoint for S3-compatible services like MinIO.
	S3Endpoint string `yaml:"s3Endpoint" env:"YUNT_STORAGE_S3_ENDPOINT"`

	// S3AccessKey is the AWS access key (when type is "s3").
	S3AccessKey string `yaml:"s3AccessKey" env:"YUNT_STORAGE_S3_ACCESS_KEY"`

	// S3SecretKey is the AWS secret key (when type is "s3").
	S3SecretKey string `yaml:"s3SecretKey" env:"YUNT_STORAGE_S3_SECRET_KEY"`

	// S3Prefix is the key prefix for S3 objects.
	S3Prefix string `yaml:"s3Prefix" env:"YUNT_STORAGE_S3_PREFIX"`

	// MaxMailboxSize is the maximum mailbox size in bytes (0 for unlimited).
	MaxMailboxSize int64 `yaml:"maxMailboxSize" env:"YUNT_STORAGE_MAX_MAILBOX_SIZE"`

	// RetentionDays is the number of days to retain messages (0 for unlimited).
	RetentionDays int `yaml:"retentionDays" env:"YUNT_STORAGE_RETENTION_DAYS"`
}

// TLSConfig contains TLS configuration settings.
type TLSConfig struct {
	// Enabled determines if TLS is enabled.
	Enabled bool `yaml:"enabled" env:"ENABLED"`

	// CertFile is the path to the TLS certificate file.
	CertFile string `yaml:"certFile" env:"CERT_FILE"`

	// KeyFile is the path to the TLS private key file.
	KeyFile string `yaml:"keyFile" env:"KEY_FILE"`

	// StartTLS determines if STARTTLS is supported (for SMTP/IMAP).
	StartTLS bool `yaml:"startTLS" env:"START_TLS"`
}
