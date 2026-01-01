package middleware

import (
	"fmt"
	"net/http"
	"runtime"
	"time"

	"github.com/labstack/echo/v4"

	"yunt/internal/config"
)

// RecoveryConfig holds configuration for the panic recovery middleware.
type RecoveryConfig struct {
	// Logger is the zerolog logger instance.
	Logger *config.Logger
	// StackSize is the maximum stack trace size to capture (default: 4KB).
	StackSize int
	// DisableStackAll disables capturing full goroutine stacks.
	DisableStackAll bool
	// DisablePrintStack disables printing stack trace in logs.
	DisablePrintStack bool
}

// DefaultRecoveryConfig returns a default configuration for recovery middleware.
func DefaultRecoveryConfig() RecoveryConfig {
	return RecoveryConfig{
		StackSize:         4 << 10, // 4KB
		DisableStackAll:   false,
		DisablePrintStack: false,
	}
}

// Recovery returns a middleware that recovers from panics.
// It logs the panic and stack trace, then returns an internal server error response.
func Recovery(cfg RecoveryConfig) echo.MiddlewareFunc {
	if cfg.StackSize == 0 {
		cfg.StackSize = 4 << 10 // 4KB default
	}

	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			defer func() {
				if r := recover(); r != nil {
					err, ok := r.(error)
					if !ok {
						err = fmt.Errorf("%v", r)
					}

					// Capture stack trace
					stack := make([]byte, cfg.StackSize)
					length := runtime.Stack(stack, cfg.DisableStackAll)
					stack = stack[:length]

					// Log the panic
					if cfg.Logger != nil {
						event := cfg.Logger.Error().
							Err(err).
							Str("requestId", c.Response().Header().Get(echo.HeaderXRequestID)).
							Str("method", c.Request().Method).
							Str("path", c.Request().URL.Path).
							Str("ip", c.RealIP())

						if !cfg.DisablePrintStack {
							event.Bytes("stack", stack)
						}

						event.Msg("Panic recovered")
					}

					// Return error response using inline structure to avoid import cycle
					_ = c.JSON(http.StatusInternalServerError, map[string]interface{}{
						"success": false,
						"error": map[string]interface{}{
							"code":    "INTERNAL_SERVER_ERROR",
							"message": "An unexpected error occurred",
						},
						"meta": map[string]interface{}{
							"timestamp": time.Now().UTC(),
							"requestId": c.Response().Header().Get(echo.HeaderXRequestID),
						},
					})
				}
			}()

			return next(c)
		}
	}
}

// RecoveryWithLogger creates a Recovery middleware with the given logger.
func RecoveryWithLogger(logger *config.Logger) echo.MiddlewareFunc {
	cfg := DefaultRecoveryConfig()
	cfg.Logger = logger
	return Recovery(cfg)
}
