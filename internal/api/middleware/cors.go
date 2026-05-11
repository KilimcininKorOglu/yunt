package middleware

import (
	"net/http"
	"strings"

	"github.com/labstack/echo/v4"
)

// CORSConfig holds configuration for the CORS middleware.
type CORSConfig struct {
	// AllowOrigins is a list of origins that are allowed to access the API.
	// Use "*" to allow all origins (not recommended for production).
	AllowOrigins []string
	// AllowMethods is a list of allowed HTTP methods.
	AllowMethods []string
	// AllowHeaders is a list of allowed request headers.
	AllowHeaders []string
	// ExposeHeaders is a list of headers that browsers are allowed to access.
	ExposeHeaders []string
	// AllowCredentials indicates whether credentials are allowed.
	AllowCredentials bool
	// MaxAge indicates how long (in seconds) the results of a preflight request can be cached.
	MaxAge int
}

// DefaultCORSConfig returns a default CORS configuration.
func DefaultCORSConfig() CORSConfig {
	return CORSConfig{
		AllowOrigins: []string{"*"},
		AllowMethods: []string{
			http.MethodGet,
			http.MethodHead,
			http.MethodPut,
			http.MethodPatch,
			http.MethodPost,
			http.MethodDelete,
			http.MethodOptions,
		},
		AllowHeaders: []string{
			echo.HeaderAccept,
			"Accept-Language",
			echo.HeaderContentType,
			"Content-Language",
			echo.HeaderAuthorization,
			echo.HeaderXRequestedWith,
			echo.HeaderXRequestID,
		},
		ExposeHeaders: []string{
			echo.HeaderXRequestID,
			"X-Total-Count",
			"ETag",
		},
		AllowCredentials: false,
		MaxAge:           86400, // 24 hours
	}
}

// CORS returns a middleware that handles Cross-Origin Resource Sharing (CORS).
func CORS(cfg CORSConfig) echo.MiddlewareFunc {
	// Set defaults if not provided
	if len(cfg.AllowMethods) == 0 {
		cfg.AllowMethods = DefaultCORSConfig().AllowMethods
	}
	if len(cfg.AllowHeaders) == 0 {
		cfg.AllowHeaders = DefaultCORSConfig().AllowHeaders
	}
	if cfg.MaxAge == 0 {
		cfg.MaxAge = 86400
	}

	allowMethodsStr := strings.Join(cfg.AllowMethods, ", ")
	allowHeadersStr := strings.Join(cfg.AllowHeaders, ", ")
	exposeHeadersStr := strings.Join(cfg.ExposeHeaders, ", ")

	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			req := c.Request()
			res := c.Response()
			origin := req.Header.Get(echo.HeaderOrigin)

			// Check if origin is allowed
			allowOrigin := ""
			for _, o := range cfg.AllowOrigins {
				if o == "*" {
					allowOrigin = "*"
					break
				}
				if o == origin {
					allowOrigin = origin
					break
				}
			}

			// If origin is not allowed, proceed without CORS headers
			if allowOrigin == "" {
				return next(c)
			}

			// Set CORS headers
			res.Header().Set(echo.HeaderAccessControlAllowOrigin, allowOrigin)

			if cfg.AllowCredentials {
				res.Header().Set(echo.HeaderAccessControlAllowCredentials, "true")
			}

			if exposeHeadersStr != "" {
				res.Header().Set(echo.HeaderAccessControlExposeHeaders, exposeHeadersStr)
			}

			// Handle preflight requests
			if req.Method == http.MethodOptions {
				res.Header().Set(echo.HeaderAccessControlAllowMethods, allowMethodsStr)
				res.Header().Set(echo.HeaderAccessControlAllowHeaders, allowHeadersStr)

				if cfg.MaxAge > 0 {
					res.Header().Set(echo.HeaderAccessControlMaxAge, itoa(cfg.MaxAge))
				}

				return c.NoContent(http.StatusNoContent)
			}

			return next(c)
		}
	}
}

// CORSWithOrigins creates a CORS middleware with the given allowed origins.
func CORSWithOrigins(origins []string) echo.MiddlewareFunc {
	cfg := DefaultCORSConfig()
	cfg.AllowOrigins = origins
	return CORS(cfg)
}

// itoa converts an integer to a string without importing strconv.
func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	if n < 0 {
		return "-" + itoa(-n)
	}
	var digits []byte
	for n > 0 {
		digits = append([]byte{byte('0' + n%10)}, digits...)
		n /= 10
	}
	return string(digits)
}
