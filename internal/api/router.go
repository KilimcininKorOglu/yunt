package api

import (
	"net/http"
	"runtime"
	"time"

	"github.com/labstack/echo/v4"
	echoMiddleware "github.com/labstack/echo/v4/middleware"
	echoSwagger "github.com/swaggo/echo-swagger"

	"yunt/internal/api/middleware"
	"yunt/internal/config"

	_ "yunt/docs/swagger"
)

// RouterConfig contains configuration for the API router.
type RouterConfig struct {
	// Logger is the zerolog logger instance.
	Logger *config.Logger
	// CORSOrigins is a list of allowed CORS origins.
	CORSOrigins []string
	// EnableSwagger determines if Swagger documentation is enabled.
	EnableSwagger bool
}

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
}

// Health status constants.
const (
	HealthStatusHealthy   = "healthy"
	HealthStatusUnhealthy = "unhealthy"
	HealthStatusDegraded  = "degraded"
)

// startTime tracks when the server started for uptime calculation.
var startTime = time.Now()

// version holds the application version (set at build time).
var version = "dev"

// SetVersion sets the application version for health check responses.
func SetVersion(v string) {
	version = v
}

// Router wraps an Echo instance and provides access to route groups.
type Router struct {
	*echo.Echo
	v1 *echo.Group
}

// V1 returns the /api/v1 route group for registering handlers.
func (r *Router) V1() *echo.Group {
	return r.v1
}

// NewRouter creates and configures a new Echo router with middleware.
func NewRouter(cfg RouterConfig) *Router {
	e := echo.New()

	// Disable Echo's default banner and startup message
	e.HideBanner = true
	e.HidePort = true

	// Request ID middleware (should be first)
	e.Use(echoMiddleware.RequestID())

	// Recovery middleware
	e.Use(middleware.RecoveryWithLogger(cfg.Logger))

	// Security headers middleware (HSTS disabled — dev server runs on plain HTTP)
	secCfg := middleware.DefaultSecurityConfig()
	secCfg.Logger = cfg.Logger
	secCfg.StrictTransportSecurity = ""
	e.Use(middleware.Security(secCfg))

	// CORS middleware
	e.Use(middleware.CORSWithOrigins(cfg.CORSOrigins))

	// Logger middleware (skip health check endpoints to reduce noise)
	e.Use(middleware.LoggerWithConfig(cfg.Logger, "/health", "/healthz", "/ready"))

	// Rate limiting middleware
	e.Use(middleware.RateLimitWithLogger(cfg.Logger))

	// Create router wrapper
	router := &Router{Echo: e}

	// Register routes
	registerRoutes(router, cfg)

	return router
}

// registerRoutes sets up all API routes.
func registerRoutes(r *Router, cfg RouterConfig) {
	// Health check endpoints
	r.GET("/health", healthHandler)
	r.GET("/healthz", healthzHandler)
	r.GET("/ready", readyHandler)

	// API version group
	api := r.Group("/api")

	// API v1 group
	r.v1 = api.Group("/v1")

	// Version endpoint
	r.v1.GET("/version", versionHandler)

	// Swagger UI
	if cfg.EnableSwagger {
		r.GET("/swagger/*", echoSwagger.WrapHandler)
	}
}

// healthHandler returns detailed health information.
func healthHandler(c echo.Context) error {
	health := &HealthResponse{
		Status:    HealthStatusHealthy,
		Timestamp: time.Now().UTC(),
		Version:   version,
		Uptime:    int64(time.Since(startTime).Seconds()),
		Details: map[string]ComponentHealth{
			"api": {
				Status:  HealthStatusHealthy,
				Message: "API server is running",
			},
		},
	}

	return OK(c, health)
}

// healthzHandler is a simple liveness probe that returns 200 OK.
func healthzHandler(c echo.Context) error {
	return c.String(http.StatusOK, "OK")
}

// readyHandler is a readiness probe that checks if the server is ready to accept traffic.
func readyHandler(c echo.Context) error {
	// In a full implementation, this would check:
	// - Database connectivity
	// - External service dependencies
	// - Other critical components
	return c.String(http.StatusOK, "OK")
}

// VersionInfo contains version and build information.
type VersionInfo struct {
	Version   string `json:"version"`
	GoVersion string `json:"goVersion"`
	OS        string `json:"os"`
	Arch      string `json:"arch"`
}

// versionHandler returns version information.
func versionHandler(c echo.Context) error {
	info := &VersionInfo{
		Version:   version,
		GoVersion: runtime.Version(),
		OS:        runtime.GOOS,
		Arch:      runtime.GOARCH,
	}
	return OK(c, info)
}
