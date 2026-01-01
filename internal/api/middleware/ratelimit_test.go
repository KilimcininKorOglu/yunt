package middleware

import (
	"net/http"
	"net/http/httptest"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/labstack/echo/v4"

	"yunt/internal/domain"
)

func TestRateLimit_BasicFunctionality(t *testing.T) {
	e := echo.New()

	cfg := RateLimitConfig{
		RequestsPerMinute:              60, // 1 per second
		AuthenticatedRequestsPerMinute: 120,
		BurstSize:                      3, // Allow 3 requests initially
		AuthenticatedBurstSize:         6,
		SkipPaths:                      []string{"/health"},
	}

	limiter := NewRateLimiter(cfg)
	defer limiter.Stop()

	e.Use(limiter.Middleware())
	e.GET("/api/test", func(c echo.Context) error {
		return c.String(http.StatusOK, "ok")
	})
	e.GET("/health", func(c echo.Context) error {
		return c.String(http.StatusOK, "healthy")
	})

	// First 3 requests should succeed (burst size)
	for i := 0; i < 3; i++ {
		req := httptest.NewRequest(http.MethodGet, "/api/test", nil)
		req.RemoteAddr = "192.168.1.1:12345"
		rec := httptest.NewRecorder()
		e.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("Request %d: expected status %d, got %d", i+1, http.StatusOK, rec.Code)
		}
	}

	// 4th request should be rate limited
	req := httptest.NewRequest(http.MethodGet, "/api/test", nil)
	req.RemoteAddr = "192.168.1.1:12345"
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusTooManyRequests {
		t.Errorf("4th request: expected status %d, got %d", http.StatusTooManyRequests, rec.Code)
	}

	// Check Retry-After header is set
	retryAfter := rec.Header().Get("Retry-After")
	if retryAfter == "" {
		t.Error("Retry-After header should be set")
	}

	// Check X-RateLimit headers
	if rec.Header().Get("X-RateLimit-Limit") == "" {
		t.Error("X-RateLimit-Limit header should be set")
	}
	if rec.Header().Get("X-RateLimit-Remaining") != "0" {
		t.Errorf("X-RateLimit-Remaining should be 0, got %s", rec.Header().Get("X-RateLimit-Remaining"))
	}

	// Health check should not be rate limited
	req = httptest.NewRequest(http.MethodGet, "/health", nil)
	req.RemoteAddr = "192.168.1.1:12345"
	rec = httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("Health check should not be rate limited, got status %d", rec.Code)
	}
}

func TestRateLimit_DifferentIPs(t *testing.T) {
	e := echo.New()

	cfg := RateLimitConfig{
		RequestsPerMinute: 60,
		BurstSize:         2,
	}

	limiter := NewRateLimiter(cfg)
	defer limiter.Stop()

	e.Use(limiter.Middleware())
	e.GET("/api/test", func(c echo.Context) error {
		return c.String(http.StatusOK, "ok")
	})

	// Requests from different IPs should have separate limits
	for _, ip := range []string{"192.168.1.1:12345", "192.168.1.2:12345"} {
		// Both should get 2 requests (burst size)
		for i := 0; i < 2; i++ {
			req := httptest.NewRequest(http.MethodGet, "/api/test", nil)
			req.RemoteAddr = ip
			rec := httptest.NewRecorder()
			e.ServeHTTP(rec, req)

			if rec.Code != http.StatusOK {
				t.Errorf("IP %s, Request %d: expected status %d, got %d", ip, i+1, http.StatusOK, rec.Code)
			}
		}
	}
}

func TestRateLimit_AuthenticatedUsers(t *testing.T) {
	e := echo.New()

	cfg := RateLimitConfig{
		RequestsPerMinute:              60,
		AuthenticatedRequestsPerMinute: 120,
		BurstSize:                      2,
		AuthenticatedBurstSize:         5,
	}

	limiter := NewRateLimiter(cfg)
	defer limiter.Stop()

	e.Use(limiter.Middleware())
	e.GET("/api/test", func(c echo.Context) error {
		return c.String(http.StatusOK, "ok")
	})

	// Authenticated user gets higher burst size
	for i := 0; i < 5; i++ {
		req := httptest.NewRequest(http.MethodGet, "/api/test", nil)
		req.RemoteAddr = "192.168.1.100:12345"
		rec := httptest.NewRecorder()

		// Simulate authenticated request by setting claims
		c := e.NewContext(req, rec)
		c.Set("claims", &domain.TokenClaims{
			UserID:   domain.ID("user-123"),
			Username: "testuser",
		})
		c.Set("userId", domain.ID("user-123"))

		// Process through middleware and handler
		handler := limiter.Middleware()(func(c echo.Context) error {
			return c.String(http.StatusOK, "ok")
		})
		err := handler(c)

		if err != nil {
			t.Errorf("Request %d: unexpected error: %v", i+1, err)
		}
		if rec.Code != http.StatusOK {
			t.Errorf("Authenticated request %d: expected status %d, got %d", i+1, http.StatusOK, rec.Code)
		}
	}

	// 6th request should be rate limited
	req := httptest.NewRequest(http.MethodGet, "/api/test", nil)
	req.RemoteAddr = "192.168.1.100:12345"
	rec := httptest.NewRecorder()

	c := e.NewContext(req, rec)
	c.Set("claims", &domain.TokenClaims{
		UserID:   domain.ID("user-123"),
		Username: "testuser",
	})
	c.Set("userId", domain.ID("user-123"))

	handler := limiter.Middleware()(func(c echo.Context) error {
		return c.String(http.StatusOK, "ok")
	})
	_ = handler(c)

	if rec.Code != http.StatusTooManyRequests {
		t.Errorf("6th authenticated request: expected status %d, got %d", http.StatusTooManyRequests, rec.Code)
	}
}

func TestRateLimit_TokenRefill(t *testing.T) {
	e := echo.New()

	cfg := RateLimitConfig{
		RequestsPerMinute: 60, // 1 token per second
		BurstSize:         1,  // Start with 1 token
	}

	limiter := NewRateLimiter(cfg)
	defer limiter.Stop()

	e.Use(limiter.Middleware())
	e.GET("/api/test", func(c echo.Context) error {
		return c.String(http.StatusOK, "ok")
	})

	// First request should succeed
	req := httptest.NewRequest(http.MethodGet, "/api/test", nil)
	req.RemoteAddr = "192.168.1.50:12345"
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("First request: expected status %d, got %d", http.StatusOK, rec.Code)
	}

	// Second request immediately should be rate limited
	req = httptest.NewRequest(http.MethodGet, "/api/test", nil)
	req.RemoteAddr = "192.168.1.50:12345"
	rec = httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusTooManyRequests {
		t.Errorf("Second request: expected status %d, got %d", http.StatusTooManyRequests, rec.Code)
	}

	// Wait for token refill (slightly more than 1 second for 1 token)
	time.Sleep(1100 * time.Millisecond)

	// Request should succeed again
	req = httptest.NewRequest(http.MethodGet, "/api/test", nil)
	req.RemoteAddr = "192.168.1.50:12345"
	rec = httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("Request after refill: expected status %d, got %d", http.StatusOK, rec.Code)
	}
}

func TestRateLimit_RateLimitHeaders(t *testing.T) {
	e := echo.New()

	cfg := RateLimitConfig{
		RequestsPerMinute: 60,
		BurstSize:         5,
	}

	limiter := NewRateLimiter(cfg)
	defer limiter.Stop()

	e.Use(limiter.Middleware())
	e.GET("/api/test", func(c echo.Context) error {
		return c.String(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/api/test", nil)
	req.RemoteAddr = "192.168.1.200:12345"
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	// Check rate limit headers
	limit := rec.Header().Get("X-RateLimit-Limit")
	if limit != "60" {
		t.Errorf("X-RateLimit-Limit: expected 60, got %s", limit)
	}

	remaining := rec.Header().Get("X-RateLimit-Remaining")
	remainingInt, _ := strconv.Atoi(remaining)
	if remainingInt != 4 { // 5 burst - 1 request = 4
		t.Errorf("X-RateLimit-Remaining: expected 4, got %s", remaining)
	}

	reset := rec.Header().Get("X-RateLimit-Reset")
	if reset == "" {
		t.Error("X-RateLimit-Reset header should be set")
	}
}

func TestRateLimit_SkipPaths(t *testing.T) {
	e := echo.New()

	cfg := RateLimitConfig{
		RequestsPerMinute: 60,
		BurstSize:         1,
		SkipPaths:         []string{"/health", "/healthz", "/ready"},
	}

	limiter := NewRateLimiter(cfg)
	defer limiter.Stop()

	e.Use(limiter.Middleware())
	e.GET("/health", func(c echo.Context) error {
		return c.String(http.StatusOK, "healthy")
	})
	e.GET("/api/test", func(c echo.Context) error {
		return c.String(http.StatusOK, "ok")
	})

	// Make many requests to health endpoint - should never be rate limited
	for i := 0; i < 10; i++ {
		req := httptest.NewRequest(http.MethodGet, "/health", nil)
		req.RemoteAddr = "192.168.1.99:12345"
		rec := httptest.NewRecorder()
		e.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("Health request %d: expected status %d, got %d", i+1, http.StatusOK, rec.Code)
		}
	}
}

func TestRateLimit_CustomKeyGenerator(t *testing.T) {
	e := echo.New()

	cfg := RateLimitConfig{
		RequestsPerMinute: 60,
		BurstSize:         2,
		KeyGenerator: func(c echo.Context) string {
			// Use custom header as key
			return c.Request().Header.Get("X-API-Key")
		},
	}

	limiter := NewRateLimiter(cfg)
	defer limiter.Stop()

	e.Use(limiter.Middleware())
	e.GET("/api/test", func(c echo.Context) error {
		return c.String(http.StatusOK, "ok")
	})

	// Different API keys should have separate limits
	for _, apiKey := range []string{"key1", "key2"} {
		for i := 0; i < 2; i++ {
			req := httptest.NewRequest(http.MethodGet, "/api/test", nil)
			req.Header.Set("X-API-Key", apiKey)
			rec := httptest.NewRecorder()
			e.ServeHTTP(rec, req)

			if rec.Code != http.StatusOK {
				t.Errorf("API Key %s, Request %d: expected status %d, got %d", apiKey, i+1, http.StatusOK, rec.Code)
			}
		}
	}
}

func TestRateLimit_OnLimitReachedCallback(t *testing.T) {
	e := echo.New()

	var callbackCalled bool
	var callbackKey string
	var callbackMu sync.Mutex

	cfg := RateLimitConfig{
		RequestsPerMinute: 60,
		BurstSize:         1,
		OnLimitReached: func(c echo.Context, key string) {
			callbackMu.Lock()
			callbackCalled = true
			callbackKey = key
			callbackMu.Unlock()
		},
	}

	limiter := NewRateLimiter(cfg)
	defer limiter.Stop()

	e.Use(limiter.Middleware())
	e.GET("/api/test", func(c echo.Context) error {
		return c.String(http.StatusOK, "ok")
	})

	// First request uses the burst
	req := httptest.NewRequest(http.MethodGet, "/api/test", nil)
	req.RemoteAddr = "192.168.1.77:12345"
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	// Second request should trigger callback
	req = httptest.NewRequest(http.MethodGet, "/api/test", nil)
	req.RemoteAddr = "192.168.1.77:12345"
	rec = httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	callbackMu.Lock()
	if !callbackCalled {
		t.Error("OnLimitReached callback should have been called")
	}
	if callbackKey == "" {
		t.Error("OnLimitReached callback should have received a key")
	}
	callbackMu.Unlock()
}

func TestRateLimit_ConcurrentRequests(t *testing.T) {
	e := echo.New()

	cfg := RateLimitConfig{
		RequestsPerMinute: 60,
		BurstSize:         10,
	}

	limiter := NewRateLimiter(cfg)
	defer limiter.Stop()

	e.Use(limiter.Middleware())
	e.GET("/api/test", func(c echo.Context) error {
		return c.String(http.StatusOK, "ok")
	})

	var wg sync.WaitGroup
	successCount := 0
	limitedCount := 0
	var mu sync.Mutex

	// Make 20 concurrent requests from the same IP
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()

			req := httptest.NewRequest(http.MethodGet, "/api/test", nil)
			req.RemoteAddr = "192.168.1.88:12345"
			rec := httptest.NewRecorder()
			e.ServeHTTP(rec, req)

			mu.Lock()
			if rec.Code == http.StatusOK {
				successCount++
			} else if rec.Code == http.StatusTooManyRequests {
				limitedCount++
			}
			mu.Unlock()
		}()
	}

	wg.Wait()

	// Should have roughly 10 successful (burst size) and 10 limited
	if successCount == 0 {
		t.Error("Should have some successful requests")
	}
	if limitedCount == 0 {
		t.Error("Should have some rate limited requests")
	}
	if successCount+limitedCount != 20 {
		t.Errorf("Total requests should be 20, got %d", successCount+limitedCount)
	}
}

func TestRateLimit_DefaultConfig(t *testing.T) {
	cfg := DefaultRateLimitConfig()

	if cfg.RequestsPerMinute <= 0 {
		t.Error("RequestsPerMinute should be positive")
	}
	if cfg.AuthenticatedRequestsPerMinute <= 0 {
		t.Error("AuthenticatedRequestsPerMinute should be positive")
	}
	if cfg.BurstSize <= 0 {
		t.Error("BurstSize should be positive")
	}
	if cfg.AuthenticatedBurstSize <= 0 {
		t.Error("AuthenticatedBurstSize should be positive")
	}
	if cfg.CleanupInterval <= 0 {
		t.Error("CleanupInterval should be positive")
	}
	if cfg.KeyGenerator == nil {
		t.Error("KeyGenerator should not be nil")
	}
}

func TestRateLimitWithDefaults(t *testing.T) {
	e := echo.New()
	e.Use(RateLimitWithDefaults())
	e.GET("/api/test", func(c echo.Context) error {
		return c.String(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/api/test", nil)
	req.RemoteAddr = "192.168.1.111:12345"
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, rec.Code)
	}

	// Check that rate limit headers are set
	if rec.Header().Get("X-RateLimit-Limit") == "" {
		t.Error("X-RateLimit-Limit header should be set")
	}
}

func TestRateLimitStore_Cleanup(t *testing.T) {
	store := newRateLimitStore()

	// Add some entries
	now := time.Now()
	store.set("recent", &rateLimitEntry{tokens: 5, lastAccess: now})
	store.set("old", &rateLimitEntry{tokens: 5, lastAccess: now.Add(-10 * time.Minute)})

	// Cleanup entries older than 5 minutes
	store.cleanup(5 * time.Minute)

	// Recent entry should still exist
	if _, ok := store.get("recent"); !ok {
		t.Error("Recent entry should still exist after cleanup")
	}

	// Old entry should be removed
	if _, ok := store.get("old"); ok {
		t.Error("Old entry should be removed after cleanup")
	}
}
