package middleware

import (
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/labstack/echo/v4"

	"yunt/internal/config"
)

// RateLimitConfig holds configuration for the rate limiting middleware.
type RateLimitConfig struct {
	// Logger is the zerolog logger instance.
	Logger *config.Logger
	// RequestsPerMinute is the maximum number of requests allowed per minute for unauthenticated users.
	RequestsPerMinute int
	// AuthenticatedRequestsPerMinute is the maximum number of requests allowed per minute for authenticated users.
	AuthenticatedRequestsPerMinute int
	// BurstSize is the maximum burst size allowed (requests that can be made instantly).
	BurstSize int
	// AuthenticatedBurstSize is the maximum burst size for authenticated users.
	AuthenticatedBurstSize int
	// CleanupInterval is how often to clean up expired entries from the store.
	CleanupInterval time.Duration
	// SkipPaths is a list of paths that should not be rate limited.
	SkipPaths []string
	// KeyGenerator is a function to generate the rate limit key from the request.
	// Defaults to using client IP address.
	KeyGenerator func(c echo.Context) string
	// OnLimitReached is called when a rate limit is exceeded.
	OnLimitReached func(c echo.Context, key string)
}

// DefaultRateLimitConfig returns a default configuration for rate limiting.
func DefaultRateLimitConfig() RateLimitConfig {
	return RateLimitConfig{
		RequestsPerMinute:              60,  // 1 request per second for unauthenticated
		AuthenticatedRequestsPerMinute: 300, // 5 requests per second for authenticated
		BurstSize:                      10,
		AuthenticatedBurstSize:         30,
		CleanupInterval:                5 * time.Minute,
		SkipPaths:                      []string{"/health", "/healthz", "/ready"},
		KeyGenerator:                   defaultKeyGenerator,
	}
}

// defaultKeyGenerator generates a rate limit key based on the client's IP address.
func defaultKeyGenerator(c echo.Context) string {
	return c.RealIP()
}

// rateLimitEntry represents a single rate limit entry for a client.
type rateLimitEntry struct {
	tokens     float64
	lastAccess time.Time
}

// rateLimitStore is an in-memory store for rate limit entries.
type rateLimitStore struct {
	entries map[string]*rateLimitEntry
	mu      sync.RWMutex
}

// newRateLimitStore creates a new rate limit store.
func newRateLimitStore() *rateLimitStore {
	return &rateLimitStore{
		entries: make(map[string]*rateLimitEntry),
	}
}

// get retrieves the rate limit entry for a key.
func (s *rateLimitStore) get(key string) (*rateLimitEntry, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	entry, ok := s.entries[key]
	return entry, ok
}

// set stores a rate limit entry for a key.
func (s *rateLimitStore) set(key string, entry *rateLimitEntry) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.entries[key] = entry
}

// cleanup removes expired entries from the store.
func (s *rateLimitStore) cleanup(maxAge time.Duration) {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := time.Now()
	for key, entry := range s.entries {
		if now.Sub(entry.lastAccess) > maxAge {
			delete(s.entries, key)
		}
	}
}

// RateLimiter holds the state for the rate limiter middleware.
type RateLimiter struct {
	config RateLimitConfig
	store  *rateLimitStore
	stopCh chan struct{}
}

// NewRateLimiter creates a new rate limiter with the given configuration.
func NewRateLimiter(cfg RateLimitConfig) *RateLimiter {
	if cfg.RequestsPerMinute <= 0 {
		cfg.RequestsPerMinute = DefaultRateLimitConfig().RequestsPerMinute
	}
	if cfg.AuthenticatedRequestsPerMinute <= 0 {
		cfg.AuthenticatedRequestsPerMinute = DefaultRateLimitConfig().AuthenticatedRequestsPerMinute
	}
	if cfg.BurstSize <= 0 {
		cfg.BurstSize = DefaultRateLimitConfig().BurstSize
	}
	if cfg.AuthenticatedBurstSize <= 0 {
		cfg.AuthenticatedBurstSize = DefaultRateLimitConfig().AuthenticatedBurstSize
	}
	if cfg.CleanupInterval <= 0 {
		cfg.CleanupInterval = DefaultRateLimitConfig().CleanupInterval
	}
	if cfg.KeyGenerator == nil {
		cfg.KeyGenerator = defaultKeyGenerator
	}

	rl := &RateLimiter{
		config: cfg,
		store:  newRateLimitStore(),
		stopCh: make(chan struct{}),
	}

	// Start cleanup goroutine
	go rl.cleanupLoop()

	return rl
}

// cleanupLoop periodically cleans up expired entries.
func (rl *RateLimiter) cleanupLoop() {
	ticker := time.NewTicker(rl.config.CleanupInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			// Clean up entries older than 2x the cleanup interval
			rl.store.cleanup(rl.config.CleanupInterval * 2)
		case <-rl.stopCh:
			return
		}
	}
}

// Stop stops the rate limiter's cleanup goroutine.
func (rl *RateLimiter) Stop() {
	close(rl.stopCh)
}

// Middleware returns the rate limiting middleware function.
func (rl *RateLimiter) Middleware() echo.MiddlewareFunc {
	skipPaths := make(map[string]bool)
	for _, path := range rl.config.SkipPaths {
		skipPaths[path] = true
	}

	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			// Check if path should be skipped
			if skipPaths[c.Path()] {
				return next(c)
			}

			key := rl.config.KeyGenerator(c)
			isAuthenticated := IsAuthenticated(c)

			// Determine rate limit parameters based on authentication status
			var requestsPerMinute, burstSize int
			if isAuthenticated {
				requestsPerMinute = rl.config.AuthenticatedRequestsPerMinute
				burstSize = rl.config.AuthenticatedBurstSize
				// Include user ID in key for authenticated users
				if userID := GetUserID(c); userID != "" {
					key = key + ":" + string(userID)
				}
			} else {
				requestsPerMinute = rl.config.RequestsPerMinute
				burstSize = rl.config.BurstSize
			}

			// Calculate tokens per second
			tokensPerSecond := float64(requestsPerMinute) / 60.0

			now := time.Now()

			// Get or create entry
			entry, exists := rl.store.get(key)
			if !exists {
				entry = &rateLimitEntry{
					tokens:     float64(burstSize),
					lastAccess: now,
				}
			}

			// Calculate tokens to add based on time elapsed (token bucket algorithm)
			elapsed := now.Sub(entry.lastAccess).Seconds()
			entry.tokens += elapsed * tokensPerSecond
			if entry.tokens > float64(burstSize) {
				entry.tokens = float64(burstSize)
			}
			entry.lastAccess = now

			// Check if we have enough tokens
			if entry.tokens < 1.0 {
				// Rate limited
				rl.store.set(key, entry)

				// Calculate retry-after time
				retryAfter := int((1.0 - entry.tokens) / tokensPerSecond)
				if retryAfter < 1 {
					retryAfter = 1
				}

				// Log rate limit hit
				if rl.config.Logger != nil {
					rl.config.Logger.Warn().
						Str("ip", c.RealIP()).
						Str("method", c.Request().Method).
						Str("path", c.Request().URL.Path).
						Bool("authenticated", isAuthenticated).
						Int("retryAfter", retryAfter).
						Msg("Rate limit exceeded")
				}

				// Call the OnLimitReached callback if set
				if rl.config.OnLimitReached != nil {
					rl.config.OnLimitReached(c, key)
				}

				// Set Retry-After header
				c.Response().Header().Set("Retry-After", strconv.Itoa(retryAfter))
				c.Response().Header().Set("X-RateLimit-Limit", strconv.Itoa(requestsPerMinute))
				c.Response().Header().Set("X-RateLimit-Remaining", "0")
				c.Response().Header().Set("X-RateLimit-Reset", strconv.FormatInt(now.Add(time.Duration(retryAfter)*time.Second).Unix(), 10))

				return c.JSON(http.StatusTooManyRequests, RateLimitErrorResponse{
					Success:    false,
					Error:      "RATE_LIMIT_EXCEEDED",
					Message:    "Too many requests. Please try again later.",
					RetryAfter: retryAfter,
				})
			}

			// Consume one token
			entry.tokens -= 1.0
			rl.store.set(key, entry)

			// Set rate limit headers
			remaining := int(entry.tokens)
			if remaining < 0 {
				remaining = 0
			}
			c.Response().Header().Set("X-RateLimit-Limit", strconv.Itoa(requestsPerMinute))
			c.Response().Header().Set("X-RateLimit-Remaining", strconv.Itoa(remaining))
			c.Response().Header().Set("X-RateLimit-Reset", strconv.FormatInt(now.Add(time.Minute).Unix(), 10))

			return next(c)
		}
	}
}

// RateLimitErrorResponse represents a rate limit error response.
type RateLimitErrorResponse struct {
	Success    bool   `json:"success"`
	Error      string `json:"error"`
	Message    string `json:"message"`
	RetryAfter int    `json:"retryAfter"`
}

// RateLimit creates a rate limiting middleware with the given configuration.
func RateLimit(cfg RateLimitConfig) echo.MiddlewareFunc {
	limiter := NewRateLimiter(cfg)
	return limiter.Middleware()
}

// RateLimitWithDefaults creates a rate limiting middleware with default configuration.
func RateLimitWithDefaults() echo.MiddlewareFunc {
	return RateLimit(DefaultRateLimitConfig())
}

// RateLimitWithLogger creates a rate limiting middleware with the given logger and default settings.
func RateLimitWithLogger(logger *config.Logger) echo.MiddlewareFunc {
	cfg := DefaultRateLimitConfig()
	cfg.Logger = logger
	return RateLimit(cfg)
}
