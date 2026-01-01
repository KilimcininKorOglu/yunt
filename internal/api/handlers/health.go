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
	repo      repository.Repository
	startTime time.Time
	version   string
}

// NewHealthHandler creates a new HealthHandler.
func NewHealthHandler(repo repository.Repository, version string) *HealthHandler {
	return &HealthHandler{
		repo:      repo,
		startTime: time.Now(),
		version:   version,
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
// @Description Get detailed health status including database connectivity
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

	// Check API server health
	health.Details["api"] = ComponentHealth{
		Status:  HealthStatusHealthy,
		Message: "API server is running",
	}

	// Check database health
	dbHealth := h.checkDatabaseHealth(ctx)
	health.Details["database"] = dbHealth

	// If any component is unhealthy, set overall status
	if dbHealth.Status == HealthStatusUnhealthy {
		health.Status = HealthStatusUnhealthy
		return api.Error(c, 503, api.CodeServiceUnavailable, "Service unavailable", health)
	}

	if dbHealth.Status == HealthStatusDegraded {
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
// @Description Check if the server is ready to accept traffic
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

	return c.String(200, "OK")
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
