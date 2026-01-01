// Package middleware provides HTTP middleware for the Yunt API server.
package middleware

import (
	"time"

	"github.com/labstack/echo/v4"
	"github.com/rs/zerolog"

	"yunt/internal/config"
)

// LoggerConfig holds configuration for the request logger middleware.
type LoggerConfig struct {
	// Logger is the zerolog logger instance.
	Logger *config.Logger
	// SkipPaths contains paths that should not be logged (e.g., health checks).
	SkipPaths []string
}

// Logger returns a middleware that logs HTTP requests.
// It captures request method, path, status code, latency, and other metadata.
func Logger(cfg LoggerConfig) echo.MiddlewareFunc {
	skipPaths := make(map[string]bool)
	for _, path := range cfg.SkipPaths {
		skipPaths[path] = true
	}

	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			// Skip logging for certain paths
			if skipPaths[c.Path()] {
				return next(c)
			}

			req := c.Request()
			res := c.Response()

			// Record start time
			start := time.Now()

			// Process request
			err := next(c)
			if err != nil {
				c.Error(err)
			}

			// Calculate latency
			latency := time.Since(start)

			// Get request ID
			requestID := res.Header().Get(echo.HeaderXRequestID)

			// Determine log level based on status code
			status := res.Status
			var event *zerolog.Event
			switch {
			case status >= 500:
				event = cfg.Logger.Error()
			case status >= 400:
				event = cfg.Logger.Warn()
			default:
				event = cfg.Logger.Info()
			}

			// Build log entry
			event.
				Str("requestId", requestID).
				Str("method", req.Method).
				Str("path", req.URL.Path).
				Str("query", req.URL.RawQuery).
				Int("status", status).
				Int64("size", res.Size).
				Dur("latency", latency).
				Str("ip", c.RealIP()).
				Str("userAgent", req.UserAgent())

			// Add error info if present
			if err != nil {
				event.Err(err)
			}

			event.Msg("HTTP request")

			return nil
		}
	}
}

// LoggerWithConfig creates a Logger middleware with the given configuration.
func LoggerWithConfig(logger *config.Logger, skipPaths ...string) echo.MiddlewareFunc {
	return Logger(LoggerConfig{
		Logger:    logger,
		SkipPaths: skipPaths,
	})
}
