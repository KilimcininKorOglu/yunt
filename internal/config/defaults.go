package config

import "time"

// Default configuration values.
const (
	// Server defaults.
	DefaultServerName           = "localhost"
	DefaultServerDomain         = "localhost"
	DefaultServerGracefulTimeout = 30 * time.Second

	// SMTP defaults.
	DefaultSMTPEnabled        = true
	DefaultSMTPHost           = "0.0.0.0"
	DefaultSMTPPort           = 1025
	DefaultSMTPMaxMessageSize = 10 * 1024 * 1024 // 10MB
	DefaultSMTPMaxRecipients  = 100
	DefaultSMTPReadTimeout    = 60 * time.Second
	DefaultSMTPWriteTimeout   = 60 * time.Second
	DefaultSMTPAuthRequired         = false
	DefaultSMTPAllowRelay           = false
	DefaultSMTPRelayPort            = 587
	DefaultSMTPRelayUseTLS          = false
	DefaultSMTPRelayUseSTARTTLS     = true
	DefaultSMTPRelayTimeout         = 30 * time.Second
	DefaultSMTPRelayRetryCount      = 3
	DefaultSMTPRelayInsecureSkipVerify = false

	// IMAP defaults.
	DefaultIMAPEnabled      = true
	DefaultIMAPHost         = "0.0.0.0"
	DefaultIMAPPort         = 1143
	DefaultIMAPReadTimeout  = 60 * time.Second
	DefaultIMAPWriteTimeout = 60 * time.Second
	DefaultIMAPIdleTimeout  = 30 * time.Minute

	// API defaults.
	DefaultAPIEnabled       = true
	DefaultAPIHost          = "0.0.0.0"
	DefaultAPIPort          = 8025
	DefaultAPIReadTimeout   = 30 * time.Second
	DefaultAPIWriteTimeout  = 30 * time.Second
	DefaultAPIRateLimit     = 100
	DefaultAPIEnableSwagger = true

	// Database defaults.
	DefaultDatabaseDriver         = "sqlite"
	DefaultDatabaseDSN            = "yunt.db"
	DefaultDatabaseHost           = "localhost"
	DefaultDatabasePort           = 5432
	DefaultDatabaseName           = "yunt"
	DefaultDatabaseSSLMode        = "disable"
	DefaultDatabaseMaxOpenConns   = 25
	DefaultDatabaseMaxIdleConns   = 5
	DefaultDatabaseConnMaxLifetime = 5 * time.Minute
	DefaultDatabaseConnMaxIdleTime = 5 * time.Minute
	DefaultDatabaseAutoMigrate    = true

	// Auth defaults.
	DefaultAuthJWTExpiration     = 24 * time.Hour
	DefaultAuthRefreshExpiration = 7 * 24 * time.Hour
	DefaultAuthBCryptCost        = 10
	DefaultAuthSessionTimeout    = 24 * time.Hour
	DefaultAuthMaxLoginAttempts  = 5
	DefaultAuthLockoutDuration   = 15 * time.Minute

	// Logging defaults.
	DefaultLoggingLevel         = "info"
	DefaultLoggingFormat        = "text"
	DefaultLoggingOutput        = "stdout"
	DefaultLoggingMaxSize       = 100
	DefaultLoggingMaxBackups    = 3
	DefaultLoggingMaxAge        = 28
	DefaultLoggingCompress      = true
	DefaultLoggingIncludeCaller = false

	// Admin defaults.
	DefaultAdminUsername        = "admin"
	DefaultAdminEmail           = "admin@localhost"
	DefaultAdminCreateOnStartup = true

	// Storage defaults.
	DefaultStorageType           = "database"
	DefaultStoragePath           = "./data/mail"
	DefaultStorageMaxMailboxSize = 0 // unlimited
	DefaultStorageRetentionDays  = 0 // unlimited
)

// Default returns a Config with all default values applied.
func Default() *Config {
	return &Config{
		Server: ServerConfig{
			Name:            DefaultServerName,
			Domain:          DefaultServerDomain,
			GracefulTimeout: DefaultServerGracefulTimeout,
		},
		SMTP: SMTPConfig{
			Enabled:                 DefaultSMTPEnabled,
			Host:                    DefaultSMTPHost,
			Port:                    DefaultSMTPPort,
			MaxMessageSize:          DefaultSMTPMaxMessageSize,
			MaxRecipients:           DefaultSMTPMaxRecipients,
			ReadTimeout:             DefaultSMTPReadTimeout,
			WriteTimeout:            DefaultSMTPWriteTimeout,
			AuthRequired:            DefaultSMTPAuthRequired,
			AllowRelay:              DefaultSMTPAllowRelay,
			RelayPort:               DefaultSMTPRelayPort,
			RelayUseTLS:             DefaultSMTPRelayUseTLS,
			RelayUseSTARTTLS:        DefaultSMTPRelayUseSTARTTLS,
			RelayTimeout:            DefaultSMTPRelayTimeout,
			RelayRetryCount:         DefaultSMTPRelayRetryCount,
			RelayInsecureSkipVerify: DefaultSMTPRelayInsecureSkipVerify,
			TLS: TLSConfig{
				Enabled:  false,
				StartTLS: true,
			},
		},
		IMAP: IMAPConfig{
			Enabled:      DefaultIMAPEnabled,
			Host:         DefaultIMAPHost,
			Port:         DefaultIMAPPort,
			ReadTimeout:  DefaultIMAPReadTimeout,
			WriteTimeout: DefaultIMAPWriteTimeout,
			IdleTimeout:  DefaultIMAPIdleTimeout,
			TLS: TLSConfig{
				Enabled:  false,
				StartTLS: true,
			},
		},
		API: APIConfig{
			Enabled:            DefaultAPIEnabled,
			Host:               DefaultAPIHost,
			Port:               DefaultAPIPort,
			ReadTimeout:        DefaultAPIReadTimeout,
			WriteTimeout:       DefaultAPIWriteTimeout,
			RateLimit:          DefaultAPIRateLimit,
			EnableSwagger:      DefaultAPIEnableSwagger,
			CORSAllowedOrigins: []string{"*"},
			TLS: TLSConfig{
				Enabled: false,
			},
		},
		Database: DatabaseConfig{
			Driver:          DefaultDatabaseDriver,
			DSN:             DefaultDatabaseDSN,
			Host:            DefaultDatabaseHost,
			Port:            DefaultDatabasePort,
			Name:            DefaultDatabaseName,
			SSLMode:         DefaultDatabaseSSLMode,
			MaxOpenConns:    DefaultDatabaseMaxOpenConns,
			MaxIdleConns:    DefaultDatabaseMaxIdleConns,
			ConnMaxLifetime: DefaultDatabaseConnMaxLifetime,
			ConnMaxIdleTime: DefaultDatabaseConnMaxIdleTime,
			AutoMigrate:     DefaultDatabaseAutoMigrate,
		},
		Auth: AuthConfig{
			JWTExpiration:     DefaultAuthJWTExpiration,
			RefreshExpiration: DefaultAuthRefreshExpiration,
			BCryptCost:        DefaultAuthBCryptCost,
			SessionTimeout:    DefaultAuthSessionTimeout,
			MaxLoginAttempts:  DefaultAuthMaxLoginAttempts,
			LockoutDuration:   DefaultAuthLockoutDuration,
		},
		Logging: LoggingConfig{
			Level:         DefaultLoggingLevel,
			Format:        DefaultLoggingFormat,
			Output:        DefaultLoggingOutput,
			MaxSize:       DefaultLoggingMaxSize,
			MaxBackups:    DefaultLoggingMaxBackups,
			MaxAge:        DefaultLoggingMaxAge,
			Compress:      DefaultLoggingCompress,
			IncludeCaller: DefaultLoggingIncludeCaller,
		},
		Admin: AdminConfig{
			Username:        DefaultAdminUsername,
			Email:           DefaultAdminEmail,
			CreateOnStartup: DefaultAdminCreateOnStartup,
		},
		Storage: StorageConfig{
			Type:           DefaultStorageType,
			Path:           DefaultStoragePath,
			MaxMailboxSize: DefaultStorageMaxMailboxSize,
			RetentionDays:  DefaultStorageRetentionDays,
		},
	}
}
