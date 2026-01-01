// Package handlers provides HTTP request handlers for the Yunt API.
package handlers

import (
	"context"
	"runtime"
	"time"

	"github.com/labstack/echo/v4"

	"yunt/internal/api"
	"yunt/internal/api/middleware"
	"yunt/internal/config"
	"yunt/internal/domain"
	"yunt/internal/repository"
	"yunt/internal/service"
)

// SystemHandler handles system management and administrative HTTP requests.
type SystemHandler struct {
	repo           repository.Repository
	authService    *service.AuthService
	messageService *service.MessageService
	config         *config.Config
	startTime      time.Time
	version        string
}

// SystemHandlerConfig contains configuration for creating a SystemHandler.
type SystemHandlerConfig struct {
	Repo           repository.Repository
	AuthService    *service.AuthService
	MessageService *service.MessageService
	Config         *config.Config
	Version        string
}

// NewSystemHandler creates a new SystemHandler.
func NewSystemHandler(cfg SystemHandlerConfig) *SystemHandler {
	return &SystemHandler{
		repo:           cfg.Repo,
		authService:    cfg.AuthService,
		messageService: cfg.MessageService,
		config:         cfg.Config,
		startTime:      time.Now(),
		version:        cfg.Version,
	}
}

// RegisterRoutes registers the system routes on the given group.
func (h *SystemHandler) RegisterRoutes(g *echo.Group) {
	system := g.Group("/system")

	// Public endpoints
	system.GET("/version", h.GetVersion)

	// Protected endpoints (require authentication)
	protected := system.Group("", middleware.Auth(h.authService))
	protected.GET("/stats", h.GetStats)

	// Admin-only endpoints
	admin := system.Group("", middleware.Auth(h.authService), middleware.RequireAdmin())
	admin.GET("/info", h.GetSystemInfo)
	admin.DELETE("/messages", h.DeleteAllMessages)
	admin.POST("/cleanup", h.Cleanup)
}

// VersionInfo contains version and build information.
type VersionInfo struct {
	Version   string `json:"version"`
	GoVersion string `json:"goVersion"`
	OS        string `json:"os"`
	Arch      string `json:"arch"`
}

// GetVersion returns version information.
// @Summary Get Version
// @Description Get application version information
// @Tags System
// @Produce json
// @Success 200 {object} api.Response{data=VersionInfo}
// @Router /system/version [get]
func (h *SystemHandler) GetVersion(c echo.Context) error {
	info := &VersionInfo{
		Version:   h.version,
		GoVersion: runtime.Version(),
		OS:        runtime.GOOS,
		Arch:      runtime.GOARCH,
	}
	return api.OK(c, info)
}

// SystemStats contains system statistics.
type SystemStats struct {
	// Users contains user statistics.
	Users UserStats `json:"users"`
	// Mailboxes contains mailbox statistics.
	Mailboxes MailboxStats `json:"mailboxes"`
	// Messages contains message statistics.
	Messages MessageStats `json:"messages"`
	// Uptime is the server uptime in seconds.
	Uptime int64 `json:"uptime"`
	// Timestamp is when the stats were collected.
	Timestamp time.Time `json:"timestamp"`
}

// UserStats contains user-related statistics.
type UserStats struct {
	// Total is the total number of users.
	Total int64 `json:"total"`
	// Active is the number of active users.
	Active int64 `json:"active"`
	// Inactive is the number of inactive users.
	Inactive int64 `json:"inactive"`
	// Pending is the number of pending users.
	Pending int64 `json:"pending"`
}

// MailboxStats contains mailbox-related statistics.
type MailboxStats struct {
	// Total is the total number of mailboxes.
	Total int64 `json:"total"`
	// TotalSize is the total size of all mailboxes in bytes.
	TotalSize int64 `json:"totalSize"`
}

// MessageStats contains message-related statistics.
type MessageStats struct {
	// Total is the total number of messages.
	Total int64 `json:"total"`
	// Unread is the number of unread messages.
	Unread int64 `json:"unread"`
	// TotalSize is the total size of all messages in bytes.
	TotalSize int64 `json:"totalSize"`
}

// GetStats returns system statistics.
// @Summary Get System Stats
// @Description Get system statistics including user, mailbox, and message counts
// @Tags System
// @Produce json
// @Security BearerAuth
// @Success 200 {object} api.Response{data=SystemStats}
// @Failure 401 {object} api.Response{error=api.ErrorDetail}
// @Failure 500 {object} api.Response{error=api.ErrorDetail}
// @Router /system/stats [get]
func (h *SystemHandler) GetStats(c echo.Context) error {
	ctx, cancel := context.WithTimeout(c.Request().Context(), 10*time.Second)
	defer cancel()

	stats := &SystemStats{
		Uptime:    int64(time.Since(h.startTime).Seconds()),
		Timestamp: time.Now().UTC(),
	}

	// Get user stats
	if h.repo != nil {
		userStats, err := h.getUserStats(ctx)
		if err != nil {
			return api.InternalServerError(c, "Failed to get user stats")
		}
		stats.Users = *userStats

		// Get mailbox stats
		mailboxStats, err := h.getMailboxStats(ctx)
		if err != nil {
			return api.InternalServerError(c, "Failed to get mailbox stats")
		}
		stats.Mailboxes = *mailboxStats

		// Get message stats
		messageStats, err := h.getMessageStats(ctx)
		if err != nil {
			return api.InternalServerError(c, "Failed to get message stats")
		}
		stats.Messages = *messageStats
	}

	return api.OK(c, stats)
}

// getUserStats retrieves user statistics from the repository.
func (h *SystemHandler) getUserStats(ctx context.Context) (*UserStats, error) {
	stats := &UserStats{}

	// Get total count
	total, err := h.repo.Users().Count(ctx, nil)
	if err != nil {
		return nil, err
	}
	stats.Total = total

	// Get counts by status
	statusCounts, err := h.repo.Users().CountByStatus(ctx)
	if err != nil {
		return nil, err
	}

	for status, count := range statusCounts {
		switch status {
		case domain.StatusActive:
			stats.Active = count
		case domain.StatusInactive:
			stats.Inactive = count
		case domain.StatusPending:
			stats.Pending = count
		}
	}

	return stats, nil
}

// getMailboxStats retrieves mailbox statistics from the repository.
func (h *SystemHandler) getMailboxStats(ctx context.Context) (*MailboxStats, error) {
	stats := &MailboxStats{}

	// Get total count
	total, err := h.repo.Mailboxes().Count(ctx, nil)
	if err != nil {
		return nil, err
	}
	stats.Total = total

	// Get total size from stats
	totalStats, err := h.repo.Mailboxes().GetTotalStats(ctx)
	if err != nil {
		// If error, just return without size
		return stats, nil
	}
	stats.TotalSize = totalStats.TotalSize

	return stats, nil
}

// getMessageStats retrieves message statistics from the repository.
func (h *SystemHandler) getMessageStats(ctx context.Context) (*MessageStats, error) {
	stats := &MessageStats{}

	// Get total count
	total, err := h.repo.Messages().Count(ctx, nil)
	if err != nil {
		return nil, err
	}
	stats.Total = total

	// Get total size
	totalSize, err := h.repo.Messages().GetTotalSize(ctx)
	if err != nil {
		// If error, just return without size
		return stats, nil
	}
	stats.TotalSize = totalSize

	return stats, nil
}

// SystemInfo contains detailed system information (admin only).
type SystemInfo struct {
	// Version is the application version.
	Version string `json:"version"`
	// Uptime is the server uptime in seconds.
	Uptime int64 `json:"uptime"`
	// StartTime is when the server was started.
	StartTime time.Time `json:"startTime"`
	// Runtime contains Go runtime information.
	Runtime RuntimeInfo `json:"runtime"`
	// Config contains non-sensitive configuration information.
	Config SystemConfigInfo `json:"config"`
	// Stats contains system statistics.
	Stats *SystemStats `json:"stats,omitempty"`
}

// SystemConfigInfo contains non-sensitive configuration values.
// Secrets and sensitive data are NOT included.
type SystemConfigInfo struct {
	// Server contains server configuration.
	Server ServerConfigInfo `json:"server"`
	// SMTP contains SMTP configuration.
	SMTP SMTPConfigInfo `json:"smtp"`
	// IMAP contains IMAP configuration.
	IMAP IMAPConfigInfo `json:"imap"`
	// API contains API configuration.
	API APIConfigInfo `json:"api"`
	// Database contains database configuration.
	Database DatabaseConfigInfo `json:"database"`
	// Storage contains storage configuration.
	Storage StorageConfigInfo `json:"storage"`
}

// ServerConfigInfo contains non-sensitive server configuration.
type ServerConfigInfo struct {
	Name            string `json:"name"`
	Domain          string `json:"domain"`
	GracefulTimeout string `json:"gracefulTimeout"`
}

// SMTPConfigInfo contains non-sensitive SMTP configuration.
type SMTPConfigInfo struct {
	Enabled        bool   `json:"enabled"`
	Host           string `json:"host"`
	Port           int    `json:"port"`
	TLSEnabled     bool   `json:"tlsEnabled"`
	MaxMessageSize int64  `json:"maxMessageSize"`
	MaxRecipients  int    `json:"maxRecipients"`
	AuthRequired   bool   `json:"authRequired"`
	AllowRelay     bool   `json:"allowRelay"`
}

// IMAPConfigInfo contains non-sensitive IMAP configuration.
type IMAPConfigInfo struct {
	Enabled    bool   `json:"enabled"`
	Host       string `json:"host"`
	Port       int    `json:"port"`
	TLSEnabled bool   `json:"tlsEnabled"`
}

// APIConfigInfo contains non-sensitive API configuration.
type APIConfigInfo struct {
	Enabled       bool     `json:"enabled"`
	Host          string   `json:"host"`
	Port          int      `json:"port"`
	TLSEnabled    bool     `json:"tlsEnabled"`
	EnableSwagger bool     `json:"enableSwagger"`
	CORSOrigins   []string `json:"corsOrigins"`
	RateLimit     int      `json:"rateLimit"`
}

// DatabaseConfigInfo contains non-sensitive database configuration.
type DatabaseConfigInfo struct {
	Driver       string `json:"driver"`
	Host         string `json:"host"`
	Port         int    `json:"port"`
	Name         string `json:"name"`
	MaxOpenConns int    `json:"maxOpenConns"`
	MaxIdleConns int    `json:"maxIdleConns"`
	AutoMigrate  bool   `json:"autoMigrate"`
}

// StorageConfigInfo contains non-sensitive storage configuration.
type StorageConfigInfo struct {
	Type           string `json:"type"`
	MaxMailboxSize int64  `json:"maxMailboxSize"`
	RetentionDays  int    `json:"retentionDays"`
}

// GetSystemInfo returns detailed system information (admin only).
// @Summary Get System Info
// @Description Get detailed system information including configuration (admin only)
// @Tags System
// @Produce json
// @Security BearerAuth
// @Success 200 {object} api.Response{data=SystemInfo}
// @Failure 401 {object} api.Response{error=api.ErrorDetail}
// @Failure 403 {object} api.Response{error=api.ErrorDetail}
// @Failure 500 {object} api.Response{error=api.ErrorDetail}
// @Router /system/info [get]
func (h *SystemHandler) GetSystemInfo(c echo.Context) error {
	ctx, cancel := context.WithTimeout(c.Request().Context(), 10*time.Second)
	defer cancel()

	info := &SystemInfo{
		Version:   h.version,
		Uptime:    int64(time.Since(h.startTime).Seconds()),
		StartTime: h.startTime,
		Runtime:   *GetRuntimeInfo(),
	}

	// Add non-sensitive config information
	if h.config != nil {
		info.Config = h.buildConfigInfo()
	}

	// Add stats
	if h.repo != nil {
		stats := &SystemStats{
			Uptime:    info.Uptime,
			Timestamp: time.Now().UTC(),
		}

		if userStats, err := h.getUserStats(ctx); err == nil {
			stats.Users = *userStats
		}
		if mailboxStats, err := h.getMailboxStats(ctx); err == nil {
			stats.Mailboxes = *mailboxStats
		}
		if messageStats, err := h.getMessageStats(ctx); err == nil {
			stats.Messages = *messageStats
		}

		info.Stats = stats
	}

	return api.OK(c, info)
}

// buildConfigInfo builds the non-sensitive configuration info.
// This method explicitly excludes all secrets and sensitive data.
func (h *SystemHandler) buildConfigInfo() SystemConfigInfo {
	cfg := h.config

	return SystemConfigInfo{
		Server: ServerConfigInfo{
			Name:            cfg.Server.Name,
			Domain:          cfg.Server.Domain,
			GracefulTimeout: cfg.Server.GracefulTimeout.String(),
		},
		SMTP: SMTPConfigInfo{
			Enabled:        cfg.SMTP.Enabled,
			Host:           cfg.SMTP.Host,
			Port:           cfg.SMTP.Port,
			TLSEnabled:     cfg.SMTP.TLS.Enabled,
			MaxMessageSize: cfg.SMTP.MaxMessageSize,
			MaxRecipients:  cfg.SMTP.MaxRecipients,
			AuthRequired:   cfg.SMTP.AuthRequired,
			AllowRelay:     cfg.SMTP.AllowRelay,
			// Note: RelayHost, RelayUsername, RelayPassword are NOT included
		},
		IMAP: IMAPConfigInfo{
			Enabled:    cfg.IMAP.Enabled,
			Host:       cfg.IMAP.Host,
			Port:       cfg.IMAP.Port,
			TLSEnabled: cfg.IMAP.TLS.Enabled,
		},
		API: APIConfigInfo{
			Enabled:       cfg.API.Enabled,
			Host:          cfg.API.Host,
			Port:          cfg.API.Port,
			TLSEnabled:    cfg.API.TLS.Enabled,
			EnableSwagger: cfg.API.EnableSwagger,
			CORSOrigins:   cfg.API.CORSAllowedOrigins,
			RateLimit:     cfg.API.RateLimit,
		},
		Database: DatabaseConfigInfo{
			Driver: cfg.Database.Driver,
			Host:   cfg.Database.Host,
			Port:   cfg.Database.Port,
			Name:   cfg.Database.Name,
			// Note: Username, Password, DSN are NOT included
			MaxOpenConns: cfg.Database.MaxOpenConns,
			MaxIdleConns: cfg.Database.MaxIdleConns,
			AutoMigrate:  cfg.Database.AutoMigrate,
		},
		Storage: StorageConfigInfo{
			Type:           cfg.Storage.Type,
			MaxMailboxSize: cfg.Storage.MaxMailboxSize,
			RetentionDays:  cfg.Storage.RetentionDays,
			// Note: Path is NOT included for security
		},
	}
}

// DeleteAllMessagesResponse contains the result of deleting all messages.
type DeleteAllMessagesResponse struct {
	// Deleted is the number of messages deleted.
	Deleted int64 `json:"deleted"`
	// Message is a human-readable result message.
	Message string `json:"message"`
}

// DeleteAllMessages deletes all messages from the system (admin only).
// @Summary Delete All Messages
// @Description Delete all messages from all mailboxes (admin only)
// @Tags System
// @Produce json
// @Security BearerAuth
// @Success 200 {object} api.Response{data=DeleteAllMessagesResponse}
// @Failure 401 {object} api.Response{error=api.ErrorDetail}
// @Failure 403 {object} api.Response{error=api.ErrorDetail}
// @Failure 500 {object} api.Response{error=api.ErrorDetail}
// @Router /system/messages [delete]
func (h *SystemHandler) DeleteAllMessages(c echo.Context) error {
	ctx, cancel := context.WithTimeout(c.Request().Context(), 60*time.Second)
	defer cancel()

	if h.repo == nil {
		return api.InternalServerError(c, "Repository not configured")
	}

	// Get all mailboxes
	mailboxes, err := h.repo.Mailboxes().List(ctx, nil, nil)
	if err != nil {
		return api.InternalServerError(c, "Failed to list mailboxes")
	}

	var totalDeleted int64

	// Delete messages from each mailbox
	for _, mailbox := range mailboxes.Items {
		deleted, err := h.repo.Messages().DeleteByMailbox(ctx, mailbox.ID)
		if err != nil {
			// Log error but continue with other mailboxes
			continue
		}
		totalDeleted += deleted

		// Reset mailbox statistics
		zeroCount := int64(0)
		zeroSize := int64(0)
		err = h.repo.Mailboxes().UpdateStats(ctx, mailbox.ID, &repository.MailboxStatsUpdate{
			MessageCount: &zeroCount,
			UnreadCount:  &zeroCount,
			TotalSize:    &zeroSize,
		})
		if err != nil {
			// Log error but continue
			continue
		}
	}

	return api.OK(c, &DeleteAllMessagesResponse{
		Deleted: totalDeleted,
		Message: "All messages deleted successfully",
	})
}

// CleanupRequest contains the request body for cleanup operations.
type CleanupRequest struct {
	// DeleteOldMessages deletes messages older than the specified days.
	DeleteOldMessages *int `json:"deleteOldMessages,omitempty"`
	// DeleteSpam deletes all spam messages.
	DeleteSpam bool `json:"deleteSpam,omitempty"`
	// RecalculateStats recalculates mailbox statistics.
	RecalculateStats bool `json:"recalculateStats,omitempty"`
}

// CleanupResponse contains the result of cleanup operations.
type CleanupResponse struct {
	// MessagesDeleted is the number of messages deleted.
	MessagesDeleted int64 `json:"messagesDeleted"`
	// SpamDeleted is the number of spam messages deleted.
	SpamDeleted int64 `json:"spamDeleted"`
	// StatsRecalculated is the number of mailboxes with recalculated stats.
	StatsRecalculated int64 `json:"statsRecalculated"`
	// Message is a human-readable result message.
	Message string `json:"message"`
}

// Cleanup performs database cleanup operations (admin only).
// @Summary Cleanup
// @Description Perform database cleanup operations (admin only)
// @Tags System
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param input body CleanupRequest true "Cleanup options"
// @Success 200 {object} api.Response{data=CleanupResponse}
// @Failure 400 {object} api.Response{error=api.ErrorDetail}
// @Failure 401 {object} api.Response{error=api.ErrorDetail}
// @Failure 403 {object} api.Response{error=api.ErrorDetail}
// @Failure 500 {object} api.Response{error=api.ErrorDetail}
// @Router /system/cleanup [post]
func (h *SystemHandler) Cleanup(c echo.Context) error {
	ctx, cancel := context.WithTimeout(c.Request().Context(), 120*time.Second)
	defer cancel()

	var req CleanupRequest
	if err := c.Bind(&req); err != nil {
		return api.BadRequest(c, "Invalid request body")
	}

	if h.repo == nil {
		return api.InternalServerError(c, "Repository not configured")
	}

	response := &CleanupResponse{}

	// Delete old messages
	if req.DeleteOldMessages != nil && *req.DeleteOldMessages > 0 {
		deleted, err := h.repo.Messages().DeleteOldMessages(ctx, *req.DeleteOldMessages)
		if err != nil {
			return api.InternalServerError(c, "Failed to delete old messages")
		}
		response.MessagesDeleted = deleted
	}

	// Delete spam
	if req.DeleteSpam {
		deleted, err := h.repo.Messages().DeleteSpam(ctx)
		if err != nil {
			return api.InternalServerError(c, "Failed to delete spam messages")
		}
		response.SpamDeleted = deleted
	}

	// Recalculate stats
	if req.RecalculateStats {
		mailboxes, err := h.repo.Mailboxes().List(ctx, nil, nil)
		if err != nil {
			return api.InternalServerError(c, "Failed to list mailboxes")
		}

		for _, mailbox := range mailboxes.Items {
			if err := h.repo.Mailboxes().RecalculateStats(ctx, mailbox.ID); err != nil {
				continue
			}
			response.StatsRecalculated++
		}
	}

	response.Message = "Cleanup completed successfully"
	return api.OK(c, response)
}
