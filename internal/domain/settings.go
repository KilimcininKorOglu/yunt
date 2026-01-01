package domain

import (
	"strings"
)

// Settings represents the global application settings.
// These settings control the behavior of the mail server including
// SMTP, IMAP, Web UI, and storage configurations.
type Settings struct {
	// ID is the unique identifier for the settings (typically singleton).
	ID ID `json:"id"`

	// SMTP contains SMTP server configuration.
	SMTP SMTPSettings `json:"smtp"`

	// IMAP contains IMAP server configuration.
	IMAP IMAPSettings `json:"imap"`

	// WebUI contains Web UI configuration.
	WebUI WebUISettings `json:"webUI"`

	// Storage contains storage configuration.
	Storage StorageSettings `json:"storage"`

	// Security contains security-related settings.
	Security SecuritySettings `json:"security"`

	// Retention contains message retention settings.
	Retention RetentionSettings `json:"retention"`

	// Notifications contains notification settings.
	Notifications NotificationSettings `json:"notifications"`

	// UpdatedAt is the timestamp when settings were last updated.
	UpdatedAt Timestamp `json:"updatedAt"`
}

// SMTPSettings contains SMTP server configuration.
type SMTPSettings struct {
	// Host is the hostname/IP to bind the SMTP server.
	Host string `json:"host"`

	// Port is the port number for the SMTP server.
	Port int `json:"port"`

	// TLSEnabled enables TLS/STARTTLS support.
	TLSEnabled bool `json:"tlsEnabled"`

	// TLSCertFile is the path to the TLS certificate file.
	TLSCertFile string `json:"tlsCertFile,omitempty"`

	// TLSKeyFile is the path to the TLS private key file.
	TLSKeyFile string `json:"tlsKeyFile,omitempty"`

	// AuthRequired requires authentication for sending mail.
	AuthRequired bool `json:"authRequired"`

	// MaxMessageSize is the maximum message size in bytes (0 = unlimited).
	MaxMessageSize int64 `json:"maxMessageSize"`

	// MaxRecipients is the maximum number of recipients per message.
	MaxRecipients int `json:"maxRecipients"`

	// AllowRelay enables mail relay to external servers.
	AllowRelay bool `json:"allowRelay"`

	// RelayHost is the upstream SMTP server for relaying.
	RelayHost string `json:"relayHost,omitempty"`

	// RelayPort is the port for the relay server.
	RelayPort int `json:"relayPort,omitempty"`

	// RelayUsername is the username for relay authentication.
	RelayUsername string `json:"relayUsername,omitempty"`

	// RelayPassword is the password for relay authentication.
	// This field is not serialized to JSON for security.
	RelayPassword string `json:"-"`

	// Enabled indicates if the SMTP server is enabled.
	Enabled bool `json:"enabled"`
}

// IMAPSettings contains IMAP server configuration.
type IMAPSettings struct {
	// Host is the hostname/IP to bind the IMAP server.
	Host string `json:"host"`

	// Port is the port number for the IMAP server.
	Port int `json:"port"`

	// TLSEnabled enables TLS support.
	TLSEnabled bool `json:"tlsEnabled"`

	// TLSCertFile is the path to the TLS certificate file.
	TLSCertFile string `json:"tlsCertFile,omitempty"`

	// TLSKeyFile is the path to the TLS private key file.
	TLSKeyFile string `json:"tlsKeyFile,omitempty"`

	// IdleTimeout is the IMAP IDLE timeout in seconds.
	IdleTimeout int `json:"idleTimeout"`

	// Enabled indicates if the IMAP server is enabled.
	Enabled bool `json:"enabled"`
}

// WebUISettings contains Web UI configuration.
type WebUISettings struct {
	// Host is the hostname/IP to bind the web server.
	Host string `json:"host"`

	// Port is the port number for the web server.
	Port int `json:"port"`

	// TLSEnabled enables HTTPS.
	TLSEnabled bool `json:"tlsEnabled"`

	// TLSCertFile is the path to the TLS certificate file.
	TLSCertFile string `json:"tlsCertFile,omitempty"`

	// TLSKeyFile is the path to the TLS private key file.
	TLSKeyFile string `json:"tlsKeyFile,omitempty"`

	// BasePath is the base URL path for the web UI.
	BasePath string `json:"basePath"`

	// Title is the custom title for the web UI.
	Title string `json:"title"`

	// Theme is the UI theme ("light", "dark", "auto").
	Theme string `json:"theme"`

	// AllowRegistration allows new user registration.
	AllowRegistration bool `json:"allowRegistration"`

	// Enabled indicates if the web UI is enabled.
	Enabled bool `json:"enabled"`
}

// StorageSettings contains storage configuration.
type StorageSettings struct {
	// Driver is the database driver to use.
	Driver DatabaseDriver `json:"driver"`

	// DSN is the database connection string.
	// This field is not serialized to JSON for security.
	DSN string `json:"-"`

	// MaxConnections is the maximum number of database connections.
	MaxConnections int `json:"maxConnections"`

	// AttachmentPath is the directory for storing attachments.
	AttachmentPath string `json:"attachmentPath"`

	// MaxAttachmentSize is the maximum attachment size in bytes.
	MaxAttachmentSize int64 `json:"maxAttachmentSize"`
}

// SecuritySettings contains security-related configuration.
type SecuritySettings struct {
	// JWTSecret is the secret key for JWT token signing.
	// This field is not serialized to JSON for security.
	JWTSecret string `json:"-"`

	// JWTExpiration is the JWT token expiration time in hours.
	JWTExpiration int `json:"jwtExpiration"`

	// SessionTimeout is the session timeout in minutes.
	SessionTimeout int `json:"sessionTimeout"`

	// PasswordMinLength is the minimum password length.
	PasswordMinLength int `json:"passwordMinLength"`

	// PasswordRequireNumbers requires numbers in passwords.
	PasswordRequireNumbers bool `json:"passwordRequireNumbers"`

	// PasswordRequireSpecial requires special characters in passwords.
	PasswordRequireSpecial bool `json:"passwordRequireSpecial"`

	// MaxLoginAttempts is the maximum failed login attempts before lockout.
	MaxLoginAttempts int `json:"maxLoginAttempts"`

	// LockoutDuration is the account lockout duration in minutes.
	LockoutDuration int `json:"lockoutDuration"`

	// RateLimitEnabled enables rate limiting.
	RateLimitEnabled bool `json:"rateLimitEnabled"`

	// RateLimitRequests is the maximum requests per window.
	RateLimitRequests int `json:"rateLimitRequests"`

	// RateLimitWindow is the rate limit window in seconds.
	RateLimitWindow int `json:"rateLimitWindow"`
}

// RetentionSettings contains message retention configuration.
type RetentionSettings struct {
	// DefaultRetentionDays is the default message retention period in days (0 = forever).
	DefaultRetentionDays int `json:"defaultRetentionDays"`

	// MaxRetentionDays is the maximum allowed retention period.
	MaxRetentionDays int `json:"maxRetentionDays"`

	// DeleteOldMessages enables automatic deletion of old messages.
	DeleteOldMessages bool `json:"deleteOldMessages"`

	// CleanupInterval is the cleanup job interval in hours.
	CleanupInterval int `json:"cleanupInterval"`
}

// NotificationSettings contains notification configuration.
type NotificationSettings struct {
	// EmailNotifications enables email notifications.
	EmailNotifications bool `json:"emailNotifications"`

	// WebhookNotifications enables webhook notifications.
	WebhookNotifications bool `json:"webhookNotifications"`

	// SlackWebhookURL is the Slack webhook URL for notifications.
	SlackWebhookURL string `json:"slackWebhookUrl,omitempty"`

	// DiscordWebhookURL is the Discord webhook URL for notifications.
	DiscordWebhookURL string `json:"discordWebhookUrl,omitempty"`
}

// NewSettings creates new Settings with default values.
func NewSettings(id ID) *Settings {
	return &Settings{
		ID: id,
		SMTP: SMTPSettings{
			Host:           "0.0.0.0",
			Port:           1025,
			TLSEnabled:     false,
			AuthRequired:   false,
			MaxMessageSize: 10 * 1024 * 1024, // 10MB
			MaxRecipients:  100,
			AllowRelay:     false,
			Enabled:        true,
		},
		IMAP: IMAPSettings{
			Host:        "0.0.0.0",
			Port:        1143,
			TLSEnabled:  false,
			IdleTimeout: 1800, // 30 minutes
			Enabled:     true,
		},
		WebUI: WebUISettings{
			Host:              "0.0.0.0",
			Port:              8025,
			TLSEnabled:        false,
			BasePath:          "/",
			Title:             "Yunt Mail Server",
			Theme:             "auto",
			AllowRegistration: false,
			Enabled:           true,
		},
		Storage: StorageSettings{
			Driver:            DatabaseDriverSQLite,
			MaxConnections:    10,
			AttachmentPath:    "./data/attachments",
			MaxAttachmentSize: 25 * 1024 * 1024, // 25MB
		},
		Security: SecuritySettings{
			JWTExpiration:          24, // 24 hours
			SessionTimeout:         60, // 60 minutes
			PasswordMinLength:      8,
			PasswordRequireNumbers: false,
			PasswordRequireSpecial: false,
			MaxLoginAttempts:       5,
			LockoutDuration:        15, // 15 minutes
			RateLimitEnabled:       true,
			RateLimitRequests:      100,
			RateLimitWindow:        60, // 1 minute
		},
		Retention: RetentionSettings{
			DefaultRetentionDays: 30,
			MaxRetentionDays:     365,
			DeleteOldMessages:    true,
			CleanupInterval:      24, // Daily
		},
		Notifications: NotificationSettings{
			EmailNotifications:   false,
			WebhookNotifications: true,
		},
		UpdatedAt: Now(),
	}
}

// Validate checks if the settings have valid values.
func (s *Settings) Validate() error {
	errs := NewValidationErrors()

	// Validate SMTP settings
	if err := s.SMTP.Validate(); err != nil {
		if ve, ok := err.(*ValidationErrors); ok {
			for _, e := range ve.Errors {
				errs.Add("smtp."+e.Field, e.Message)
			}
		}
	}

	// Validate IMAP settings
	if err := s.IMAP.Validate(); err != nil {
		if ve, ok := err.(*ValidationErrors); ok {
			for _, e := range ve.Errors {
				errs.Add("imap."+e.Field, e.Message)
			}
		}
	}

	// Validate WebUI settings
	if err := s.WebUI.Validate(); err != nil {
		if ve, ok := err.(*ValidationErrors); ok {
			for _, e := range ve.Errors {
				errs.Add("webUI."+e.Field, e.Message)
			}
		}
	}

	// Validate Storage settings
	if err := s.Storage.Validate(); err != nil {
		if ve, ok := err.(*ValidationErrors); ok {
			for _, e := range ve.Errors {
				errs.Add("storage."+e.Field, e.Message)
			}
		}
	}

	// Validate Security settings
	if err := s.Security.Validate(); err != nil {
		if ve, ok := err.(*ValidationErrors); ok {
			for _, e := range ve.Errors {
				errs.Add("security."+e.Field, e.Message)
			}
		}
	}

	// Validate Retention settings
	if err := s.Retention.Validate(); err != nil {
		if ve, ok := err.(*ValidationErrors); ok {
			for _, e := range ve.Errors {
				errs.Add("retention."+e.Field, e.Message)
			}
		}
	}

	if errs.HasErrors() {
		return errs
	}
	return nil
}

// Validate checks if the SMTP settings are valid.
func (s *SMTPSettings) Validate() error {
	errs := NewValidationErrors()

	if s.Port < 1 || s.Port > 65535 {
		errs.Add("port", "port must be between 1 and 65535")
	}

	if s.TLSEnabled {
		if s.TLSCertFile == "" {
			errs.Add("tlsCertFile", "TLS certificate file is required when TLS is enabled")
		}
		if s.TLSKeyFile == "" {
			errs.Add("tlsKeyFile", "TLS key file is required when TLS is enabled")
		}
	}

	if s.MaxMessageSize < 0 {
		errs.Add("maxMessageSize", "max message size cannot be negative")
	}

	if s.MaxRecipients < 1 {
		errs.Add("maxRecipients", "max recipients must be at least 1")
	}

	if s.AllowRelay && s.RelayHost == "" {
		errs.Add("relayHost", "relay host is required when relay is enabled")
	}

	if errs.HasErrors() {
		return errs
	}
	return nil
}

// Validate checks if the IMAP settings are valid.
func (s *IMAPSettings) Validate() error {
	errs := NewValidationErrors()

	if s.Port < 1 || s.Port > 65535 {
		errs.Add("port", "port must be between 1 and 65535")
	}

	if s.TLSEnabled {
		if s.TLSCertFile == "" {
			errs.Add("tlsCertFile", "TLS certificate file is required when TLS is enabled")
		}
		if s.TLSKeyFile == "" {
			errs.Add("tlsKeyFile", "TLS key file is required when TLS is enabled")
		}
	}

	if s.IdleTimeout < 0 {
		errs.Add("idleTimeout", "idle timeout cannot be negative")
	}

	if errs.HasErrors() {
		return errs
	}
	return nil
}

// Validate checks if the WebUI settings are valid.
func (s *WebUISettings) Validate() error {
	errs := NewValidationErrors()

	if s.Port < 1 || s.Port > 65535 {
		errs.Add("port", "port must be between 1 and 65535")
	}

	if s.TLSEnabled {
		if s.TLSCertFile == "" {
			errs.Add("tlsCertFile", "TLS certificate file is required when TLS is enabled")
		}
		if s.TLSKeyFile == "" {
			errs.Add("tlsKeyFile", "TLS key file is required when TLS is enabled")
		}
	}

	if s.Theme != "" && s.Theme != "light" && s.Theme != "dark" && s.Theme != "auto" {
		errs.Add("theme", "theme must be 'light', 'dark', or 'auto'")
	}

	if errs.HasErrors() {
		return errs
	}
	return nil
}

// Validate checks if the Storage settings are valid.
func (s *StorageSettings) Validate() error {
	errs := NewValidationErrors()

	if !s.Driver.IsValid() {
		errs.Add("driver", "invalid database driver")
	}

	if s.MaxConnections < 1 {
		errs.Add("maxConnections", "max connections must be at least 1")
	}

	if s.AttachmentPath == "" {
		errs.Add("attachmentPath", "attachment path is required")
	}

	if s.MaxAttachmentSize < 0 {
		errs.Add("maxAttachmentSize", "max attachment size cannot be negative")
	}

	if errs.HasErrors() {
		return errs
	}
	return nil
}

// Validate checks if the Security settings are valid.
func (s *SecuritySettings) Validate() error {
	errs := NewValidationErrors()

	if s.JWTExpiration < 1 {
		errs.Add("jwtExpiration", "JWT expiration must be at least 1 hour")
	}

	if s.SessionTimeout < 1 {
		errs.Add("sessionTimeout", "session timeout must be at least 1 minute")
	}

	if s.PasswordMinLength < 1 {
		errs.Add("passwordMinLength", "password min length must be at least 1")
	}

	if s.MaxLoginAttempts < 1 {
		errs.Add("maxLoginAttempts", "max login attempts must be at least 1")
	}

	if s.LockoutDuration < 1 {
		errs.Add("lockoutDuration", "lockout duration must be at least 1 minute")
	}

	if s.RateLimitEnabled {
		if s.RateLimitRequests < 1 {
			errs.Add("rateLimitRequests", "rate limit requests must be at least 1")
		}
		if s.RateLimitWindow < 1 {
			errs.Add("rateLimitWindow", "rate limit window must be at least 1 second")
		}
	}

	if errs.HasErrors() {
		return errs
	}
	return nil
}

// Validate checks if the Retention settings are valid.
func (s *RetentionSettings) Validate() error {
	errs := NewValidationErrors()

	if s.DefaultRetentionDays < 0 {
		errs.Add("defaultRetentionDays", "default retention days cannot be negative")
	}

	if s.MaxRetentionDays < 0 {
		errs.Add("maxRetentionDays", "max retention days cannot be negative")
	}

	if s.DefaultRetentionDays > s.MaxRetentionDays && s.MaxRetentionDays > 0 {
		errs.Add("defaultRetentionDays", "default retention days cannot exceed max retention days")
	}

	if s.CleanupInterval < 1 {
		errs.Add("cleanupInterval", "cleanup interval must be at least 1 hour")
	}

	if errs.HasErrors() {
		return errs
	}
	return nil
}

// SettingsUpdateInput represents the input for updating settings.
type SettingsUpdateInput struct {
	// SMTP contains updated SMTP settings.
	SMTP *SMTPSettingsUpdate `json:"smtp,omitempty"`

	// IMAP contains updated IMAP settings.
	IMAP *IMAPSettingsUpdate `json:"imap,omitempty"`

	// WebUI contains updated Web UI settings.
	WebUI *WebUISettingsUpdate `json:"webUI,omitempty"`

	// Storage contains updated storage settings.
	Storage *StorageSettingsUpdate `json:"storage,omitempty"`

	// Security contains updated security settings.
	Security *SecuritySettingsUpdate `json:"security,omitempty"`

	// Retention contains updated retention settings.
	Retention *RetentionSettingsUpdate `json:"retention,omitempty"`

	// Notifications contains updated notification settings.
	Notifications *NotificationSettingsUpdate `json:"notifications,omitempty"`
}

// SMTPSettingsUpdate represents updateable SMTP settings.
type SMTPSettingsUpdate struct {
	Host           *string `json:"host,omitempty"`
	Port           *int    `json:"port,omitempty"`
	TLSEnabled     *bool   `json:"tlsEnabled,omitempty"`
	TLSCertFile    *string `json:"tlsCertFile,omitempty"`
	TLSKeyFile     *string `json:"tlsKeyFile,omitempty"`
	AuthRequired   *bool   `json:"authRequired,omitempty"`
	MaxMessageSize *int64  `json:"maxMessageSize,omitempty"`
	MaxRecipients  *int    `json:"maxRecipients,omitempty"`
	AllowRelay     *bool   `json:"allowRelay,omitempty"`
	RelayHost      *string `json:"relayHost,omitempty"`
	RelayPort      *int    `json:"relayPort,omitempty"`
	RelayUsername  *string `json:"relayUsername,omitempty"`
	RelayPassword  *string `json:"relayPassword,omitempty"`
	Enabled        *bool   `json:"enabled,omitempty"`
}

// IMAPSettingsUpdate represents updateable IMAP settings.
type IMAPSettingsUpdate struct {
	Host        *string `json:"host,omitempty"`
	Port        *int    `json:"port,omitempty"`
	TLSEnabled  *bool   `json:"tlsEnabled,omitempty"`
	TLSCertFile *string `json:"tlsCertFile,omitempty"`
	TLSKeyFile  *string `json:"tlsKeyFile,omitempty"`
	IdleTimeout *int    `json:"idleTimeout,omitempty"`
	Enabled     *bool   `json:"enabled,omitempty"`
}

// WebUISettingsUpdate represents updateable Web UI settings.
type WebUISettingsUpdate struct {
	Host              *string `json:"host,omitempty"`
	Port              *int    `json:"port,omitempty"`
	TLSEnabled        *bool   `json:"tlsEnabled,omitempty"`
	TLSCertFile       *string `json:"tlsCertFile,omitempty"`
	TLSKeyFile        *string `json:"tlsKeyFile,omitempty"`
	BasePath          *string `json:"basePath,omitempty"`
	Title             *string `json:"title,omitempty"`
	Theme             *string `json:"theme,omitempty"`
	AllowRegistration *bool   `json:"allowRegistration,omitempty"`
	Enabled           *bool   `json:"enabled,omitempty"`
}

// StorageSettingsUpdate represents updateable storage settings.
type StorageSettingsUpdate struct {
	Driver            *DatabaseDriver `json:"driver,omitempty"`
	DSN               *string         `json:"dsn,omitempty"`
	MaxConnections    *int            `json:"maxConnections,omitempty"`
	AttachmentPath    *string         `json:"attachmentPath,omitempty"`
	MaxAttachmentSize *int64          `json:"maxAttachmentSize,omitempty"`
}

// SecuritySettingsUpdate represents updateable security settings.
type SecuritySettingsUpdate struct {
	JWTSecret              *string `json:"jwtSecret,omitempty"`
	JWTExpiration          *int    `json:"jwtExpiration,omitempty"`
	SessionTimeout         *int    `json:"sessionTimeout,omitempty"`
	PasswordMinLength      *int    `json:"passwordMinLength,omitempty"`
	PasswordRequireNumbers *bool   `json:"passwordRequireNumbers,omitempty"`
	PasswordRequireSpecial *bool   `json:"passwordRequireSpecial,omitempty"`
	MaxLoginAttempts       *int    `json:"maxLoginAttempts,omitempty"`
	LockoutDuration        *int    `json:"lockoutDuration,omitempty"`
	RateLimitEnabled       *bool   `json:"rateLimitEnabled,omitempty"`
	RateLimitRequests      *int    `json:"rateLimitRequests,omitempty"`
	RateLimitWindow        *int    `json:"rateLimitWindow,omitempty"`
}

// RetentionSettingsUpdate represents updateable retention settings.
type RetentionSettingsUpdate struct {
	DefaultRetentionDays *int  `json:"defaultRetentionDays,omitempty"`
	MaxRetentionDays     *int  `json:"maxRetentionDays,omitempty"`
	DeleteOldMessages    *bool `json:"deleteOldMessages,omitempty"`
	CleanupInterval      *int  `json:"cleanupInterval,omitempty"`
}

// NotificationSettingsUpdate represents updateable notification settings.
type NotificationSettingsUpdate struct {
	EmailNotifications   *bool   `json:"emailNotifications,omitempty"`
	WebhookNotifications *bool   `json:"webhookNotifications,omitempty"`
	SlackWebhookURL      *string `json:"slackWebhookUrl,omitempty"`
	DiscordWebhookURL    *string `json:"discordWebhookUrl,omitempty"`
}

// Apply applies the update to the given settings.
func (i *SettingsUpdateInput) Apply(settings *Settings) {
	if i.SMTP != nil {
		i.SMTP.Apply(&settings.SMTP)
	}
	if i.IMAP != nil {
		i.IMAP.Apply(&settings.IMAP)
	}
	if i.WebUI != nil {
		i.WebUI.Apply(&settings.WebUI)
	}
	if i.Storage != nil {
		i.Storage.Apply(&settings.Storage)
	}
	if i.Security != nil {
		i.Security.Apply(&settings.Security)
	}
	if i.Retention != nil {
		i.Retention.Apply(&settings.Retention)
	}
	if i.Notifications != nil {
		i.Notifications.Apply(&settings.Notifications)
	}
	settings.UpdatedAt = Now()
}

// Apply applies SMTP settings updates.
func (u *SMTPSettingsUpdate) Apply(s *SMTPSettings) {
	if u.Host != nil {
		s.Host = strings.TrimSpace(*u.Host)
	}
	if u.Port != nil {
		s.Port = *u.Port
	}
	if u.TLSEnabled != nil {
		s.TLSEnabled = *u.TLSEnabled
	}
	if u.TLSCertFile != nil {
		s.TLSCertFile = strings.TrimSpace(*u.TLSCertFile)
	}
	if u.TLSKeyFile != nil {
		s.TLSKeyFile = strings.TrimSpace(*u.TLSKeyFile)
	}
	if u.AuthRequired != nil {
		s.AuthRequired = *u.AuthRequired
	}
	if u.MaxMessageSize != nil {
		s.MaxMessageSize = *u.MaxMessageSize
	}
	if u.MaxRecipients != nil {
		s.MaxRecipients = *u.MaxRecipients
	}
	if u.AllowRelay != nil {
		s.AllowRelay = *u.AllowRelay
	}
	if u.RelayHost != nil {
		s.RelayHost = strings.TrimSpace(*u.RelayHost)
	}
	if u.RelayPort != nil {
		s.RelayPort = *u.RelayPort
	}
	if u.RelayUsername != nil {
		s.RelayUsername = strings.TrimSpace(*u.RelayUsername)
	}
	if u.RelayPassword != nil {
		s.RelayPassword = *u.RelayPassword
	}
	if u.Enabled != nil {
		s.Enabled = *u.Enabled
	}
}

// Apply applies IMAP settings updates.
func (u *IMAPSettingsUpdate) Apply(s *IMAPSettings) {
	if u.Host != nil {
		s.Host = strings.TrimSpace(*u.Host)
	}
	if u.Port != nil {
		s.Port = *u.Port
	}
	if u.TLSEnabled != nil {
		s.TLSEnabled = *u.TLSEnabled
	}
	if u.TLSCertFile != nil {
		s.TLSCertFile = strings.TrimSpace(*u.TLSCertFile)
	}
	if u.TLSKeyFile != nil {
		s.TLSKeyFile = strings.TrimSpace(*u.TLSKeyFile)
	}
	if u.IdleTimeout != nil {
		s.IdleTimeout = *u.IdleTimeout
	}
	if u.Enabled != nil {
		s.Enabled = *u.Enabled
	}
}

// Apply applies Web UI settings updates.
func (u *WebUISettingsUpdate) Apply(s *WebUISettings) {
	if u.Host != nil {
		s.Host = strings.TrimSpace(*u.Host)
	}
	if u.Port != nil {
		s.Port = *u.Port
	}
	if u.TLSEnabled != nil {
		s.TLSEnabled = *u.TLSEnabled
	}
	if u.TLSCertFile != nil {
		s.TLSCertFile = strings.TrimSpace(*u.TLSCertFile)
	}
	if u.TLSKeyFile != nil {
		s.TLSKeyFile = strings.TrimSpace(*u.TLSKeyFile)
	}
	if u.BasePath != nil {
		s.BasePath = strings.TrimSpace(*u.BasePath)
	}
	if u.Title != nil {
		s.Title = strings.TrimSpace(*u.Title)
	}
	if u.Theme != nil {
		s.Theme = strings.TrimSpace(*u.Theme)
	}
	if u.AllowRegistration != nil {
		s.AllowRegistration = *u.AllowRegistration
	}
	if u.Enabled != nil {
		s.Enabled = *u.Enabled
	}
}

// Apply applies Storage settings updates.
func (u *StorageSettingsUpdate) Apply(s *StorageSettings) {
	if u.Driver != nil {
		s.Driver = *u.Driver
	}
	if u.DSN != nil {
		s.DSN = *u.DSN
	}
	if u.MaxConnections != nil {
		s.MaxConnections = *u.MaxConnections
	}
	if u.AttachmentPath != nil {
		s.AttachmentPath = strings.TrimSpace(*u.AttachmentPath)
	}
	if u.MaxAttachmentSize != nil {
		s.MaxAttachmentSize = *u.MaxAttachmentSize
	}
}

// Apply applies Security settings updates.
func (u *SecuritySettingsUpdate) Apply(s *SecuritySettings) {
	if u.JWTSecret != nil {
		s.JWTSecret = *u.JWTSecret
	}
	if u.JWTExpiration != nil {
		s.JWTExpiration = *u.JWTExpiration
	}
	if u.SessionTimeout != nil {
		s.SessionTimeout = *u.SessionTimeout
	}
	if u.PasswordMinLength != nil {
		s.PasswordMinLength = *u.PasswordMinLength
	}
	if u.PasswordRequireNumbers != nil {
		s.PasswordRequireNumbers = *u.PasswordRequireNumbers
	}
	if u.PasswordRequireSpecial != nil {
		s.PasswordRequireSpecial = *u.PasswordRequireSpecial
	}
	if u.MaxLoginAttempts != nil {
		s.MaxLoginAttempts = *u.MaxLoginAttempts
	}
	if u.LockoutDuration != nil {
		s.LockoutDuration = *u.LockoutDuration
	}
	if u.RateLimitEnabled != nil {
		s.RateLimitEnabled = *u.RateLimitEnabled
	}
	if u.RateLimitRequests != nil {
		s.RateLimitRequests = *u.RateLimitRequests
	}
	if u.RateLimitWindow != nil {
		s.RateLimitWindow = *u.RateLimitWindow
	}
}

// Apply applies Retention settings updates.
func (u *RetentionSettingsUpdate) Apply(s *RetentionSettings) {
	if u.DefaultRetentionDays != nil {
		s.DefaultRetentionDays = *u.DefaultRetentionDays
	}
	if u.MaxRetentionDays != nil {
		s.MaxRetentionDays = *u.MaxRetentionDays
	}
	if u.DeleteOldMessages != nil {
		s.DeleteOldMessages = *u.DeleteOldMessages
	}
	if u.CleanupInterval != nil {
		s.CleanupInterval = *u.CleanupInterval
	}
}

// Apply applies Notification settings updates.
func (u *NotificationSettingsUpdate) Apply(s *NotificationSettings) {
	if u.EmailNotifications != nil {
		s.EmailNotifications = *u.EmailNotifications
	}
	if u.WebhookNotifications != nil {
		s.WebhookNotifications = *u.WebhookNotifications
	}
	if u.SlackWebhookURL != nil {
		s.SlackWebhookURL = strings.TrimSpace(*u.SlackWebhookURL)
	}
	if u.DiscordWebhookURL != nil {
		s.DiscordWebhookURL = strings.TrimSpace(*u.DiscordWebhookURL)
	}
}
