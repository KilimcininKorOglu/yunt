// Package handlers provides HTTP request handlers for the Yunt API.
package handlers

import (
	"context"
	"runtime"
	"time"

	"github.com/labstack/echo/v4"

	"yunt/internal/api"
	"yunt/internal/repository"
)

// ServiceChecker is an interface for checking if a service is running.
// This is implemented by both smtp.Server and imap.Server.
type ServiceChecker interface {
	// IsRunning returns true if the service is running.
	IsRunning() bool
}

// HealthStatus constants define the possible health states.
const (
	HealthStatusHealthy   = "healthy"
	HealthStatusUnhealthy = "unhealthy"
	HealthStatusDegraded  = "degraded"
)

// HealthResponse represents the health check response.
type HealthResponse struct {
	// Status is the overall health status.
	Status string `json:"status"`
	// Timestamp is when the health check was performed.
	Timestamp time.Time `json:"timestamp"`
	// Version is the application version.
	Version string `json:"version,omitempty"`
	// Uptime is the server uptime in seconds.
	Uptime int64 `json:"uptime,omitempty"`
	// Details contains component-specific health information.
	Details map[string]ComponentHealth `json:"details,omitempty"`
}

// ComponentHealth represents the health status of a component.
type ComponentHealth struct {
	// Status is the component health status.
	Status string `json:"status"`
	// Message provides additional context.
	Message string `json:"message,omitempty"`
	// Latency is the response time in milliseconds (for applicable components).
	Latency *int64 `json:"latency,omitempty"`
}

// HealthHandler handles health check HTTP requests.
type HealthHandler struct {
	repo        repository.Repository
	smtpServer  ServiceChecker
	imapServer  ServiceChecker
	smtpEnabled bool
	imapEnabled bool
	startTime   time.Time
	version     string
}

// HealthHandlerConfig contains configuration for creating a HealthHandler.
type HealthHandlerConfig struct {
	// Repo is the database repository for health checks.
	Repo repository.Repository
	// SmtpServer is the SMTP server instance.
	SmtpServer ServiceChecker
	// ImapServer is the IMAP server instance.
	ImapServer ServiceChecker
	// SmtpEnabled indicates if SMTP is configured to be enabled.
	SmtpEnabled bool
	// ImapEnabled indicates if IMAP is configured to be enabled.
	ImapEnabled bool
	// Version is the application version.
	Version string
}

// NewHealthHandler creates a new HealthHandler.
func NewHealthHandler(repo repository.Repository, version string) *HealthHandler {
	return &HealthHandler{
		repo:      repo,
		startTime: time.Now(),
		version:   version,
	}
}

// NewHealthHandlerWithConfig creates a new HealthHandler with full configuration.
func NewHealthHandlerWithConfig(cfg HealthHandlerConfig) *HealthHandler {
	return &HealthHandler{
		repo:        cfg.Repo,
		smtpServer:  cfg.SmtpServer,
		imapServer:  cfg.ImapServer,
		smtpEnabled: cfg.SmtpEnabled,
		imapEnabled: cfg.ImapEnabled,
		startTime:   time.Now(),
		version:     cfg.Version,
	}
}

// RegisterRoutes registers the health check routes on the given Echo instance.
// Health endpoints are typically registered at the root level, not under /api.
func (h *HealthHandler) RegisterRoutes(e *echo.Echo) {
	e.GET("/health", h.Health)
	e.GET("/healthz", h.Healthz)
	e.GET("/ready", h.Ready)
}

// Health returns detailed health information including database connectivity.
// @Summary Health Check
// @Description Get detailed health status including database, SMTP, and IMAP connectivity
// @Tags Health
// @Produce json
// @Success 200 {object} api.Response{data=HealthResponse}
// @Failure 503 {object} api.Response{error=api.ErrorDetail}
// @Router /health [get]
func (h *HealthHandler) Health(c echo.Context) error {
	ctx, cancel := context.WithTimeout(c.Request().Context(), 5*time.Second)
	defer cancel()

	health := &HealthResponse{
		Status:    HealthStatusHealthy,
		Timestamp: time.Now().UTC(),
		Version:   h.version,
		Uptime:    int64(time.Since(h.startTime).Seconds()),
		Details:   make(map[string]ComponentHealth),
	}

	// Track if any component is unhealthy or degraded
	hasUnhealthy := false
	hasDegraded := false

	// Check API server health
	health.Details["api"] = ComponentHealth{
		Status:  HealthStatusHealthy,
		Message: "API server is running",
	}

	// Check database health
	dbHealth := h.checkDatabaseHealth(ctx)
	health.Details["database"] = dbHealth
	if dbHealth.Status == HealthStatusUnhealthy {
		hasUnhealthy = true
	} else if dbHealth.Status == HealthStatusDegraded {
		hasDegraded = true
	}

	// Check SMTP server health
	smtpHealth := h.checkSmtpHealth()
	health.Details["smtp"] = smtpHealth
	if smtpHealth.Status == HealthStatusUnhealthy {
		hasUnhealthy = true
	} else if smtpHealth.Status == HealthStatusDegraded {
		hasDegraded = true
	}

	// Check IMAP server health
	imapHealth := h.checkImapHealth()
	health.Details["imap"] = imapHealth
	if imapHealth.Status == HealthStatusUnhealthy {
		hasUnhealthy = true
	} else if imapHealth.Status == HealthStatusDegraded {
		hasDegraded = true
	}

	// Set overall status based on component health
	if hasUnhealthy {
		health.Status = HealthStatusUnhealthy
		return api.Error(c, 503, api.CodeServiceUnavailable, "Service unavailable", health)
	}

	if hasDegraded {
		health.Status = HealthStatusDegraded
	}

	return api.OK(c, health)
}

// Healthz is a simple liveness probe that returns 200 OK.
// @Summary Liveness Probe
// @Description Simple liveness check that returns OK if the server is running
// @Tags Health
// @Produce plain
// @Success 200 {string} string "OK"
// @Router /healthz [get]
func (h *HealthHandler) Healthz(c echo.Context) error {
	return c.String(200, "OK")
}

// Ready is a readiness probe that checks if the server is ready to accept traffic.
// @Summary Readiness Probe
// @Description Check if the server is ready to accept traffic (database, SMTP, IMAP)
// @Tags Health
// @Produce plain
// @Success 200 {string} string "OK"
// @Failure 503 {string} string "Service Unavailable"
// @Router /ready [get]
func (h *HealthHandler) Ready(c echo.Context) error {
	ctx, cancel := context.WithTimeout(c.Request().Context(), 3*time.Second)
	defer cancel()

	// Check database connectivity for readiness
	if h.repo != nil {
		if err := h.repo.Health(ctx); err != nil {
			return c.String(503, "Service Unavailable")
		}
	}

	// Check SMTP server readiness if enabled
	if h.smtpEnabled && h.smtpServer != nil {
		if !h.smtpServer.IsRunning() {
			return c.String(503, "Service Unavailable")
		}
	}

	// Check IMAP server readiness if enabled
	if h.imapEnabled && h.imapServer != nil {
		if !h.imapServer.IsRunning() {
			return c.String(503, "Service Unavailable")
		}
	}

	return c.String(200, "OK")
}

// checkSmtpHealth checks the SMTP server health and returns its status.
func (h *HealthHandler) checkSmtpHealth() ComponentHealth {
	// If SMTP is not enabled in config, it's not unhealthy
	if !h.smtpEnabled {
		return ComponentHealth{
			Status:  HealthStatusHealthy,
			Message: "SMTP server is disabled",
		}
	}

	// If SMTP server is not configured, it might be a startup issue
	if h.smtpServer == nil {
		return ComponentHealth{
			Status:  HealthStatusUnhealthy,
			Message: "SMTP server not configured",
		}
	}

	// Check if SMTP server is running
	if !h.smtpServer.IsRunning() {
		return ComponentHealth{
			Status:  HealthStatusUnhealthy,
			Message: "SMTP server is not running",
		}
	}

	return ComponentHealth{
		Status:  HealthStatusHealthy,
		Message: "SMTP server is running",
	}
}

// checkImapHealth checks the IMAP server health and returns its status.
func (h *HealthHandler) checkImapHealth() ComponentHealth {
	// If IMAP is not enabled in config, it's not unhealthy
	if !h.imapEnabled {
		return ComponentHealth{
			Status:  HealthStatusHealthy,
			Message: "IMAP server is disabled",
		}
	}

	// If IMAP server is not configured, it might be a startup issue
	if h.imapServer == nil {
		return ComponentHealth{
			Status:  HealthStatusUnhealthy,
			Message: "IMAP server not configured",
		}
	}

	// Check if IMAP server is running
	if !h.imapServer.IsRunning() {
		return ComponentHealth{
			Status:  HealthStatusUnhealthy,
			Message: "IMAP server is not running",
		}
	}

	return ComponentHealth{
		Status:  HealthStatusHealthy,
		Message: "IMAP server is running",
	}
}

// checkDatabaseHealth checks the database connectivity and returns its health status.
func (h *HealthHandler) checkDatabaseHealth(ctx context.Context) ComponentHealth {
	if h.repo == nil {
		return ComponentHealth{
			Status:  HealthStatusUnhealthy,
			Message: "Database repository not configured",
		}
	}

	start := time.Now()
	err := h.repo.Health(ctx)
	latency := time.Since(start).Milliseconds()

	if err != nil {
		return ComponentHealth{
			Status:  HealthStatusUnhealthy,
			Message: "Database connection failed: " + err.Error(),
			Latency: &latency,
		}
	}

	// Consider high latency as degraded
	if latency > 1000 {
		return ComponentHealth{
			Status:  HealthStatusDegraded,
			Message: "Database connection slow",
			Latency: &latency,
		}
	}

	return ComponentHealth{
		Status:  HealthStatusHealthy,
		Message: "Database connection OK",
		Latency: &latency,
	}
}

// RuntimeInfo contains Go runtime information.
type RuntimeInfo struct {
	// GoVersion is the Go runtime version.
	GoVersion string `json:"goVersion"`
	// NumGoroutine is the number of goroutines.
	NumGoroutine int `json:"numGoroutine"`
	// NumCPU is the number of CPUs available.
	NumCPU int `json:"numCpu"`
	// OS is the operating system.
	OS string `json:"os"`
	// Arch is the architecture.
	Arch string `json:"arch"`
}

// GetRuntimeInfo returns current runtime information.
func GetRuntimeInfo() *RuntimeInfo {
	return &RuntimeInfo{
		GoVersion:    runtime.Version(),
		NumGoroutine: runtime.NumGoroutine(),
		NumCPU:       runtime.NumCPU(),
		OS:           runtime.GOOS,
		Arch:         runtime.GOARCH,
	}
}
