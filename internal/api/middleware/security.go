package middleware

import (
	"io"
	"net/http"
	"strconv"

	"github.com/labstack/echo/v4"

	"yunt/internal/config"
)

// SecurityConfig holds configuration for the security middleware.
type SecurityConfig struct {
	// Logger is the zerolog logger instance.
	Logger *config.Logger
	// MaxRequestBodySize is the maximum allowed request body size in bytes.
	// Default is 10MB. Set to 0 to disable the limit.
	MaxRequestBodySize int64
	// ContentSecurityPolicy sets the Content-Security-Policy header value.
	// Default is "default-src 'self'".
	ContentSecurityPolicy string
	// XFrameOptions sets the X-Frame-Options header value.
	// Default is "DENY".
	XFrameOptions string
	// XContentTypeOptions sets the X-Content-Type-Options header value.
	// Default is "nosniff".
	XContentTypeOptions string
	// XXSSProtection sets the X-XSS-Protection header value.
	// Default is "1; mode=block".
	XXSSProtection string
	// ReferrerPolicy sets the Referrer-Policy header value.
	// Default is "strict-origin-when-cross-origin".
	ReferrerPolicy string
	// StrictTransportSecurity sets the Strict-Transport-Security header value.
	// Default is "max-age=31536000; includeSubDomains".
	// Set to empty string to disable.
	StrictTransportSecurity string
	// PermissionsPolicy sets the Permissions-Policy header value.
	// Default is "geolocation=(), microphone=(), camera=()".
	PermissionsPolicy string
	// CacheControl sets the Cache-Control header for API responses.
	// Default is "no-store, no-cache, must-revalidate, proxy-revalidate".
	CacheControl string
	// SkipPaths is a list of paths that should skip security headers.
	SkipPaths []string
	// SkipRequestSizeLimit is a list of paths that should skip request size limits.
	// Useful for file upload endpoints.
	SkipRequestSizeLimit []string
}

// DefaultSecurityConfig returns a default configuration for security middleware.
func DefaultSecurityConfig() SecurityConfig {
	return SecurityConfig{
		MaxRequestBodySize:      10 * 1024 * 1024, // 10MB
		ContentSecurityPolicy:   "default-src 'self'; script-src 'self' 'unsafe-inline' 'unsafe-eval'; style-src 'self' 'unsafe-inline'; img-src 'self' data: blob:; font-src 'self' data:; connect-src 'self'",
		XFrameOptions:           "DENY",
		XContentTypeOptions:     "nosniff",
		XXSSProtection:          "1; mode=block",
		ReferrerPolicy:          "strict-origin-when-cross-origin",
		StrictTransportSecurity: "max-age=31536000; includeSubDomains",
		PermissionsPolicy:       "geolocation=(), microphone=(), camera=()",
		CacheControl:            "no-store, no-cache, must-revalidate, proxy-revalidate",
		SkipPaths:               []string{},
		SkipRequestSizeLimit:    []string{},
	}
}

// Security returns a middleware that adds security headers to responses and
// enforces request size limits.
func Security(cfg SecurityConfig) echo.MiddlewareFunc {
	// Set defaults if not provided
	if cfg.MaxRequestBodySize == 0 {
		cfg.MaxRequestBodySize = DefaultSecurityConfig().MaxRequestBodySize
	}
	if cfg.XFrameOptions == "" {
		cfg.XFrameOptions = DefaultSecurityConfig().XFrameOptions
	}
	if cfg.XContentTypeOptions == "" {
		cfg.XContentTypeOptions = DefaultSecurityConfig().XContentTypeOptions
	}
	if cfg.XXSSProtection == "" {
		cfg.XXSSProtection = DefaultSecurityConfig().XXSSProtection
	}
	if cfg.ReferrerPolicy == "" {
		cfg.ReferrerPolicy = DefaultSecurityConfig().ReferrerPolicy
	}
	if cfg.CacheControl == "" {
		cfg.CacheControl = DefaultSecurityConfig().CacheControl
	}

	skipPaths := make(map[string]bool)
	for _, path := range cfg.SkipPaths {
		skipPaths[path] = true
	}

	skipSizePaths := make(map[string]bool)
	for _, path := range cfg.SkipRequestSizeLimit {
		skipSizePaths[path] = true
	}

	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			req := c.Request()
			res := c.Response()
			path := c.Path()

			// Check request body size (skip if path is excluded)
			if cfg.MaxRequestBodySize > 0 && !skipSizePaths[path] {
				// Check Content-Length header first for early rejection
				if req.ContentLength > cfg.MaxRequestBodySize {
					if cfg.Logger != nil {
						cfg.Logger.Warn().
							Str("ip", c.RealIP()).
							Str("method", req.Method).
							Str("path", req.URL.Path).
							Int64("contentLength", req.ContentLength).
							Int64("maxSize", cfg.MaxRequestBodySize).
							Msg("Request body too large")
					}
					return c.JSON(http.StatusRequestEntityTooLarge, SecurityErrorResponse{
						Success: false,
						Error:   "REQUEST_ENTITY_TOO_LARGE",
						Message: "Request body exceeds maximum allowed size",
						MaxSize: cfg.MaxRequestBodySize,
					})
				}

				// Wrap the body reader to enforce size limit
				if req.Body != nil {
					req.Body = &limitedReader{
						reader:  req.Body,
						limit:   cfg.MaxRequestBodySize,
						logger:  cfg.Logger,
						context: c,
					}
				}
			}

			// Skip security headers for excluded paths
			if !skipPaths[path] {
				// Add security headers
				if cfg.XFrameOptions != "" {
					res.Header().Set("X-Frame-Options", cfg.XFrameOptions)
				}
				if cfg.XContentTypeOptions != "" {
					res.Header().Set("X-Content-Type-Options", cfg.XContentTypeOptions)
				}
				if cfg.XXSSProtection != "" {
					res.Header().Set("X-XSS-Protection", cfg.XXSSProtection)
				}
				if cfg.ReferrerPolicy != "" {
					res.Header().Set("Referrer-Policy", cfg.ReferrerPolicy)
				}
				if cfg.ContentSecurityPolicy != "" {
					res.Header().Set("Content-Security-Policy", cfg.ContentSecurityPolicy)
				}
				if cfg.StrictTransportSecurity != "" {
					res.Header().Set("Strict-Transport-Security", cfg.StrictTransportSecurity)
				}
				if cfg.PermissionsPolicy != "" {
					res.Header().Set("Permissions-Policy", cfg.PermissionsPolicy)
				}
				if cfg.CacheControl != "" {
					res.Header().Set("Cache-Control", cfg.CacheControl)
					res.Header().Set("Pragma", "no-cache")
					res.Header().Set("Expires", "0")
				}
			}

			return next(c)
		}
	}
}

// limitedReader wraps an io.ReadCloser to enforce a size limit.
type limitedReader struct {
	reader  io.ReadCloser
	limit   int64
	read    int64
	logger  *config.Logger
	context echo.Context
}

// Read reads from the underlying reader, enforcing the size limit.
func (lr *limitedReader) Read(p []byte) (int, error) {
	// Check if we've already exceeded the limit
	if lr.read >= lr.limit {
		if lr.logger != nil {
			lr.logger.Warn().
				Str("ip", lr.context.RealIP()).
				Str("method", lr.context.Request().Method).
				Str("path", lr.context.Request().URL.Path).
				Int64("bytesRead", lr.read).
				Int64("maxSize", lr.limit).
				Msg("Request body exceeded limit during read")
		}
		return 0, &RequestTooLargeError{MaxSize: lr.limit, BytesRead: lr.read}
	}

	// Calculate how much we can read
	remaining := lr.limit - lr.read
	if int64(len(p)) > remaining {
		p = p[:remaining+1] // Read one extra byte to detect overflow
	}

	n, err := lr.reader.Read(p)
	lr.read += int64(n)

	// Check if we've now exceeded the limit
	if lr.read > lr.limit {
		if lr.logger != nil {
			lr.logger.Warn().
				Str("ip", lr.context.RealIP()).
				Str("method", lr.context.Request().Method).
				Str("path", lr.context.Request().URL.Path).
				Int64("bytesRead", lr.read).
				Int64("maxSize", lr.limit).
				Msg("Request body exceeded limit during read")
		}
		return n, &RequestTooLargeError{MaxSize: lr.limit, BytesRead: lr.read}
	}

	return n, err
}

// Close closes the underlying reader.
func (lr *limitedReader) Close() error {
	return lr.reader.Close()
}

// RequestTooLargeError is returned when the request body exceeds the size limit.
type RequestTooLargeError struct {
	MaxSize   int64
	BytesRead int64
}

// Error implements the error interface.
func (e *RequestTooLargeError) Error() string {
	return "request body too large: read " + strconv.FormatInt(e.BytesRead, 10) +
		" bytes, max " + strconv.FormatInt(e.MaxSize, 10) + " bytes"
}

// SecurityErrorResponse represents a security-related error response.
type SecurityErrorResponse struct {
	Success bool   `json:"success"`
	Error   string `json:"error"`
	Message string `json:"message"`
	MaxSize int64  `json:"maxSize,omitempty"`
}

// SecurityWithDefaults creates a security middleware with default configuration.
func SecurityWithDefaults() echo.MiddlewareFunc {
	return Security(DefaultSecurityConfig())
}

// SecurityWithLogger creates a security middleware with the given logger and default settings.
func SecurityWithLogger(logger *config.Logger) echo.MiddlewareFunc {
	cfg := DefaultSecurityConfig()
	cfg.Logger = logger
	return Security(cfg)
}

// SecurityHeaders returns a middleware that only adds security headers without
// request size limiting. Useful when you want to separate concerns.
func SecurityHeaders(cfg SecurityConfig) echo.MiddlewareFunc {
	skipPaths := make(map[string]bool)
	for _, path := range cfg.SkipPaths {
		skipPaths[path] = true
	}

	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			res := c.Response()
			path := c.Path()

			// Skip security headers for excluded paths
			if skipPaths[path] {
				return next(c)
			}

			// Add security headers
			if cfg.XFrameOptions != "" {
				res.Header().Set("X-Frame-Options", cfg.XFrameOptions)
			}
			if cfg.XContentTypeOptions != "" {
				res.Header().Set("X-Content-Type-Options", cfg.XContentTypeOptions)
			}
			if cfg.XXSSProtection != "" {
				res.Header().Set("X-XSS-Protection", cfg.XXSSProtection)
			}
			if cfg.ReferrerPolicy != "" {
				res.Header().Set("Referrer-Policy", cfg.ReferrerPolicy)
			}
			if cfg.ContentSecurityPolicy != "" {
				res.Header().Set("Content-Security-Policy", cfg.ContentSecurityPolicy)
			}
			if cfg.StrictTransportSecurity != "" {
				res.Header().Set("Strict-Transport-Security", cfg.StrictTransportSecurity)
			}
			if cfg.PermissionsPolicy != "" {
				res.Header().Set("Permissions-Policy", cfg.PermissionsPolicy)
			}
			if cfg.CacheControl != "" {
				res.Header().Set("Cache-Control", cfg.CacheControl)
				res.Header().Set("Pragma", "no-cache")
				res.Header().Set("Expires", "0")
			}

			return next(c)
		}
	}
}

// RequestSizeLimit returns a middleware that only enforces request size limits
// without adding security headers. Useful when you want to separate concerns.
func RequestSizeLimit(maxSize int64, skipPaths ...string) echo.MiddlewareFunc {
	skipPathsMap := make(map[string]bool)
	for _, path := range skipPaths {
		skipPathsMap[path] = true
	}

	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			req := c.Request()
			path := c.Path()

			// Skip if path is excluded
			if skipPathsMap[path] {
				return next(c)
			}

			// Check Content-Length header first for early rejection
			if req.ContentLength > maxSize {
				return c.JSON(http.StatusRequestEntityTooLarge, SecurityErrorResponse{
					Success: false,
					Error:   "REQUEST_ENTITY_TOO_LARGE",
					Message: "Request body exceeds maximum allowed size",
					MaxSize: maxSize,
				})
			}

			// Wrap the body reader to enforce size limit
			if req.Body != nil {
				req.Body = &limitedReader{
					reader:  req.Body,
					limit:   maxSize,
					context: c,
				}
			}

			return next(c)
		}
	}
}
