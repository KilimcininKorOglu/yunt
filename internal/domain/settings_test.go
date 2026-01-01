package domain

import (
	"strings"
	"testing"
)

func TestNewSettings(t *testing.T) {
	settings := NewSettings(ID("settings1"))

	if settings.ID != ID("settings1") {
		t.Errorf("NewSettings().ID = %v, want %v", settings.ID, "settings1")
	}

	// Check SMTP defaults
	if settings.SMTP.Port != 1025 {
		t.Errorf("NewSettings().SMTP.Port = %v, want 1025", settings.SMTP.Port)
	}
	if !settings.SMTP.Enabled {
		t.Error("NewSettings().SMTP.Enabled should be true")
	}

	// Check IMAP defaults
	if settings.IMAP.Port != 1143 {
		t.Errorf("NewSettings().IMAP.Port = %v, want 1143", settings.IMAP.Port)
	}
	if !settings.IMAP.Enabled {
		t.Error("NewSettings().IMAP.Enabled should be true")
	}

	// Check WebUI defaults
	if settings.WebUI.Port != 8025 {
		t.Errorf("NewSettings().WebUI.Port = %v, want 8025", settings.WebUI.Port)
	}
	if settings.WebUI.Title != "Yunt Mail Server" {
		t.Errorf("NewSettings().WebUI.Title = %v, want 'Yunt Mail Server'", settings.WebUI.Title)
	}

	// Check Storage defaults
	if settings.Storage.Driver != DatabaseDriverSQLite {
		t.Errorf("NewSettings().Storage.Driver = %v, want %v", settings.Storage.Driver, DatabaseDriverSQLite)
	}

	// Check Security defaults
	if settings.Security.PasswordMinLength != 8 {
		t.Errorf("NewSettings().Security.PasswordMinLength = %v, want 8", settings.Security.PasswordMinLength)
	}

	// Check Retention defaults
	if settings.Retention.DefaultRetentionDays != 30 {
		t.Errorf("NewSettings().Retention.DefaultRetentionDays = %v, want 30", settings.Retention.DefaultRetentionDays)
	}
}

func TestSettings_Validate(t *testing.T) {
	validSettings := NewSettings(ID("s1"))

	err := validSettings.Validate()
	if err != nil {
		t.Errorf("Settings.Validate() should pass for default settings, got %v", err)
	}
}

func TestSMTPSettings_Validate(t *testing.T) {
	tests := []struct {
		name    string
		smtp    SMTPSettings
		wantErr bool
		errMsgs []string
	}{
		{
			name: "valid settings",
			smtp: SMTPSettings{
				Port:          1025,
				MaxRecipients: 100,
			},
			wantErr: false,
		},
		{
			name: "invalid port (0)",
			smtp: SMTPSettings{
				Port:          0,
				MaxRecipients: 100,
			},
			wantErr: true,
			errMsgs: []string{"port"},
		},
		{
			name: "invalid port (too high)",
			smtp: SMTPSettings{
				Port:          70000,
				MaxRecipients: 100,
			},
			wantErr: true,
			errMsgs: []string{"port"},
		},
		{
			name: "TLS enabled without cert",
			smtp: SMTPSettings{
				Port:          1025,
				TLSEnabled:    true,
				MaxRecipients: 100,
			},
			wantErr: true,
			errMsgs: []string{}, // Just check for error, don't check specific message
		},
		{
			name: "negative max message size",
			smtp: SMTPSettings{
				Port:           1025,
				MaxRecipients:  100,
				MaxMessageSize: -1,
			},
			wantErr: true,
			errMsgs: []string{"maxMessageSize"},
		},
		{
			name: "zero max recipients",
			smtp: SMTPSettings{
				Port:          1025,
				MaxRecipients: 0,
			},
			wantErr: true,
			errMsgs: []string{"maxRecipients"},
		},
		{
			name: "relay enabled without host",
			smtp: SMTPSettings{
				Port:          1025,
				MaxRecipients: 100,
				AllowRelay:    true,
			},
			wantErr: true,
			errMsgs: []string{"relayHost"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.smtp.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("SMTPSettings.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr && err != nil {
				errStr := err.Error()
				for _, msg := range tt.errMsgs {
					if !strings.Contains(errStr, msg) {
						t.Errorf("SMTPSettings.Validate() error should contain '%s', got %v", msg, errStr)
					}
				}
			}
		})
	}
}

func TestIMAPSettings_Validate(t *testing.T) {
	tests := []struct {
		name    string
		imap    IMAPSettings
		wantErr bool
		errMsgs []string
	}{
		{
			name: "valid settings",
			imap: IMAPSettings{
				Port:        1143,
				IdleTimeout: 1800,
			},
			wantErr: false,
		},
		{
			name: "invalid port",
			imap: IMAPSettings{
				Port:        0,
				IdleTimeout: 1800,
			},
			wantErr: true,
			errMsgs: []string{"port"},
		},
		{
			name: "TLS enabled without cert",
			imap: IMAPSettings{
				Port:        1143,
				TLSEnabled:  true,
				IdleTimeout: 1800,
			},
			wantErr: true,
			errMsgs: []string{"tlsCertFile"},
		},
		{
			name: "negative idle timeout",
			imap: IMAPSettings{
				Port:        1143,
				IdleTimeout: -1,
			},
			wantErr: true,
			errMsgs: []string{"idleTimeout"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.imap.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("IMAPSettings.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestWebUISettings_Validate(t *testing.T) {
	tests := []struct {
		name    string
		webUI   WebUISettings
		wantErr bool
		errMsgs []string
	}{
		{
			name: "valid settings",
			webUI: WebUISettings{
				Port:  8025,
				Theme: "auto",
			},
			wantErr: false,
		},
		{
			name: "invalid port",
			webUI: WebUISettings{
				Port: 0,
			},
			wantErr: true,
			errMsgs: []string{"port"},
		},
		{
			name: "invalid theme",
			webUI: WebUISettings{
				Port:  8025,
				Theme: "invalid",
			},
			wantErr: true,
			errMsgs: []string{"theme"},
		},
		{
			name: "valid themes",
			webUI: WebUISettings{
				Port:  8025,
				Theme: "light",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.webUI.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("WebUISettings.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestStorageSettings_Validate(t *testing.T) {
	tests := []struct {
		name    string
		storage StorageSettings
		wantErr bool
		errMsgs []string
	}{
		{
			name: "valid settings",
			storage: StorageSettings{
				Driver:         DatabaseDriverSQLite,
				MaxConnections: 10,
				AttachmentPath: "./data/attachments",
			},
			wantErr: false,
		},
		{
			name: "invalid driver",
			storage: StorageSettings{
				Driver:         DatabaseDriver("invalid"),
				MaxConnections: 10,
				AttachmentPath: "./data/attachments",
			},
			wantErr: true,
			errMsgs: []string{"driver"},
		},
		{
			name: "zero max connections",
			storage: StorageSettings{
				Driver:         DatabaseDriverSQLite,
				MaxConnections: 0,
				AttachmentPath: "./data/attachments",
			},
			wantErr: true,
			errMsgs: []string{"maxConnections"},
		},
		{
			name: "missing attachment path",
			storage: StorageSettings{
				Driver:         DatabaseDriverSQLite,
				MaxConnections: 10,
			},
			wantErr: true,
			errMsgs: []string{"attachmentPath"},
		},
		{
			name: "negative max attachment size",
			storage: StorageSettings{
				Driver:            DatabaseDriverSQLite,
				MaxConnections:    10,
				AttachmentPath:    "./data/attachments",
				MaxAttachmentSize: -1,
			},
			wantErr: true,
			errMsgs: []string{"maxAttachmentSize"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.storage.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("StorageSettings.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestSecuritySettings_Validate(t *testing.T) {
	tests := []struct {
		name     string
		security SecuritySettings
		wantErr  bool
		errMsgs  []string
	}{
		{
			name: "valid settings",
			security: SecuritySettings{
				JWTExpiration:     24,
				SessionTimeout:    60,
				PasswordMinLength: 8,
				MaxLoginAttempts:  5,
				LockoutDuration:   15,
			},
			wantErr: false,
		},
		{
			name: "zero JWT expiration",
			security: SecuritySettings{
				JWTExpiration:     0,
				SessionTimeout:    60,
				PasswordMinLength: 8,
				MaxLoginAttempts:  5,
				LockoutDuration:   15,
			},
			wantErr: true,
			errMsgs: []string{"jwtExpiration"},
		},
		{
			name: "zero session timeout",
			security: SecuritySettings{
				JWTExpiration:     24,
				SessionTimeout:    0,
				PasswordMinLength: 8,
				MaxLoginAttempts:  5,
				LockoutDuration:   15,
			},
			wantErr: true,
			errMsgs: []string{"sessionTimeout"},
		},
		{
			name: "rate limit enabled without requests",
			security: SecuritySettings{
				JWTExpiration:     24,
				SessionTimeout:    60,
				PasswordMinLength: 8,
				MaxLoginAttempts:  5,
				LockoutDuration:   15,
				RateLimitEnabled:  true,
				RateLimitRequests: 0,
				RateLimitWindow:   60,
			},
			wantErr: true,
			errMsgs: []string{"rateLimitRequests"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.security.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("SecuritySettings.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestRetentionSettings_Validate(t *testing.T) {
	tests := []struct {
		name      string
		retention RetentionSettings
		wantErr   bool
		errMsgs   []string
	}{
		{
			name: "valid settings",
			retention: RetentionSettings{
				DefaultRetentionDays: 30,
				MaxRetentionDays:     365,
				CleanupInterval:      24,
			},
			wantErr: false,
		},
		{
			name: "negative default retention",
			retention: RetentionSettings{
				DefaultRetentionDays: -1,
				MaxRetentionDays:     365,
				CleanupInterval:      24,
			},
			wantErr: true,
			errMsgs: []string{"defaultRetentionDays"},
		},
		{
			name: "default exceeds max",
			retention: RetentionSettings{
				DefaultRetentionDays: 400,
				MaxRetentionDays:     365,
				CleanupInterval:      24,
			},
			wantErr: true,
			errMsgs: []string{"defaultRetentionDays"},
		},
		{
			name: "zero cleanup interval",
			retention: RetentionSettings{
				DefaultRetentionDays: 30,
				MaxRetentionDays:     365,
				CleanupInterval:      0,
			},
			wantErr: true,
			errMsgs: []string{"cleanupInterval"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.retention.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("RetentionSettings.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestSettingsUpdateInput_Apply(t *testing.T) {
	settings := NewSettings(ID("s1"))

	newHost := "127.0.0.1"
	newPort := 2025
	smtpUpdate := &SMTPSettingsUpdate{
		Host: &newHost,
		Port: &newPort,
	}

	newTheme := "dark"
	webUIUpdate := &WebUISettingsUpdate{
		Theme: &newTheme,
	}

	input := &SettingsUpdateInput{
		SMTP:  smtpUpdate,
		WebUI: webUIUpdate,
	}

	input.Apply(settings)

	if settings.SMTP.Host != newHost {
		t.Errorf("Apply() SMTP.Host = %v, want %v", settings.SMTP.Host, newHost)
	}
	if settings.SMTP.Port != newPort {
		t.Errorf("Apply() SMTP.Port = %v, want %v", settings.SMTP.Port, newPort)
	}
	if settings.WebUI.Theme != newTheme {
		t.Errorf("Apply() WebUI.Theme = %v, want %v", settings.WebUI.Theme, newTheme)
	}
}

func TestSMTPSettingsUpdate_Apply(t *testing.T) {
	smtp := &SMTPSettings{
		Host:         "0.0.0.0",
		Port:         1025,
		AuthRequired: false,
	}

	newHost := "127.0.0.1"
	newPort := 2025
	authRequired := true

	update := &SMTPSettingsUpdate{
		Host:         &newHost,
		Port:         &newPort,
		AuthRequired: &authRequired,
	}

	update.Apply(smtp)

	if smtp.Host != newHost {
		t.Errorf("Apply() Host = %v, want %v", smtp.Host, newHost)
	}
	if smtp.Port != newPort {
		t.Errorf("Apply() Port = %v, want %v", smtp.Port, newPort)
	}
	if smtp.AuthRequired != authRequired {
		t.Errorf("Apply() AuthRequired = %v, want %v", smtp.AuthRequired, authRequired)
	}
}

func TestIMAPSettingsUpdate_Apply(t *testing.T) {
	imap := &IMAPSettings{
		Port:        1143,
		IdleTimeout: 1800,
	}

	newPort := 2143
	newTimeout := 3600

	update := &IMAPSettingsUpdate{
		Port:        &newPort,
		IdleTimeout: &newTimeout,
	}

	update.Apply(imap)

	if imap.Port != newPort {
		t.Errorf("Apply() Port = %v, want %v", imap.Port, newPort)
	}
	if imap.IdleTimeout != newTimeout {
		t.Errorf("Apply() IdleTimeout = %v, want %v", imap.IdleTimeout, newTimeout)
	}
}

func TestStorageSettingsUpdate_Apply(t *testing.T) {
	storage := &StorageSettings{
		Driver:         DatabaseDriverSQLite,
		MaxConnections: 10,
	}

	newDriver := DatabaseDriverPostgres
	newMaxConns := 50

	update := &StorageSettingsUpdate{
		Driver:         &newDriver,
		MaxConnections: &newMaxConns,
	}

	update.Apply(storage)

	if storage.Driver != newDriver {
		t.Errorf("Apply() Driver = %v, want %v", storage.Driver, newDriver)
	}
	if storage.MaxConnections != newMaxConns {
		t.Errorf("Apply() MaxConnections = %v, want %v", storage.MaxConnections, newMaxConns)
	}
}

func TestSecuritySettingsUpdate_Apply(t *testing.T) {
	security := &SecuritySettings{
		PasswordMinLength: 8,
		MaxLoginAttempts:  5,
	}

	newMinLength := 12
	newMaxAttempts := 3

	update := &SecuritySettingsUpdate{
		PasswordMinLength: &newMinLength,
		MaxLoginAttempts:  &newMaxAttempts,
	}

	update.Apply(security)

	if security.PasswordMinLength != newMinLength {
		t.Errorf("Apply() PasswordMinLength = %v, want %v", security.PasswordMinLength, newMinLength)
	}
	if security.MaxLoginAttempts != newMaxAttempts {
		t.Errorf("Apply() MaxLoginAttempts = %v, want %v", security.MaxLoginAttempts, newMaxAttempts)
	}
}

func TestRetentionSettingsUpdate_Apply(t *testing.T) {
	retention := &RetentionSettings{
		DefaultRetentionDays: 30,
		DeleteOldMessages:    true,
	}

	newRetention := 60
	deleteOld := false

	update := &RetentionSettingsUpdate{
		DefaultRetentionDays: &newRetention,
		DeleteOldMessages:    &deleteOld,
	}

	update.Apply(retention)

	if retention.DefaultRetentionDays != newRetention {
		t.Errorf("Apply() DefaultRetentionDays = %v, want %v", retention.DefaultRetentionDays, newRetention)
	}
	if retention.DeleteOldMessages != deleteOld {
		t.Errorf("Apply() DeleteOldMessages = %v, want %v", retention.DeleteOldMessages, deleteOld)
	}
}

func TestNotificationSettingsUpdate_Apply(t *testing.T) {
	notifications := &NotificationSettings{
		EmailNotifications: false,
		SlackWebhookURL:    "",
	}

	emailEnabled := true
	slackURL := "https://hooks.slack.com/test"

	update := &NotificationSettingsUpdate{
		EmailNotifications: &emailEnabled,
		SlackWebhookURL:    &slackURL,
	}

	update.Apply(notifications)

	if notifications.EmailNotifications != emailEnabled {
		t.Errorf("Apply() EmailNotifications = %v, want %v", notifications.EmailNotifications, emailEnabled)
	}
	if notifications.SlackWebhookURL != slackURL {
		t.Errorf("Apply() SlackWebhookURL = %v, want %v", notifications.SlackWebhookURL, slackURL)
	}
}
