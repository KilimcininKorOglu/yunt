package middleware

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/labstack/echo/v4"
)

func TestSecurity_SecurityHeaders(t *testing.T) {
	e := echo.New()

	cfg := DefaultSecurityConfig()
	e.Use(Security(cfg))
	e.GET("/api/test", func(c echo.Context) error {
		return c.String(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/api/test", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, rec.Code)
	}

	// Check security headers
	tests := []struct {
		header   string
		expected string
	}{
		{"X-Frame-Options", "DENY"},
		{"X-Content-Type-Options", "nosniff"},
		{"X-XSS-Protection", "1; mode=block"},
		{"Referrer-Policy", "strict-origin-when-cross-origin"},
		{"Strict-Transport-Security", "max-age=31536000; includeSubDomains"},
		{"Cache-Control", "no-store, no-cache, must-revalidate, proxy-revalidate"},
		{"Pragma", "no-cache"},
		{"Expires", "0"},
	}

	for _, tt := range tests {
		got := rec.Header().Get(tt.header)
		if got != tt.expected {
			t.Errorf("%s: expected %q, got %q", tt.header, tt.expected, got)
		}
	}

	// Check CSP header is set
	csp := rec.Header().Get("Content-Security-Policy")
	if csp == "" {
		t.Error("Content-Security-Policy header should be set")
	}

	// Check Permissions-Policy header is set
	pp := rec.Header().Get("Permissions-Policy")
	if pp == "" {
		t.Error("Permissions-Policy header should be set")
	}
}

func TestSecurity_CustomHeaders(t *testing.T) {
	e := echo.New()

	cfg := SecurityConfig{
		XFrameOptions:           "SAMEORIGIN",
		ContentSecurityPolicy:   "default-src 'none'",
		ReferrerPolicy:          "no-referrer",
		StrictTransportSecurity: "", // Disabled
	}
	e.Use(Security(cfg))
	e.GET("/api/test", func(c echo.Context) error {
		return c.String(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/api/test", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	// Check custom header values
	if got := rec.Header().Get("X-Frame-Options"); got != "SAMEORIGIN" {
		t.Errorf("X-Frame-Options: expected SAMEORIGIN, got %s", got)
	}
	if got := rec.Header().Get("Content-Security-Policy"); got != "default-src 'none'" {
		t.Errorf("Content-Security-Policy: expected default-src 'none', got %s", got)
	}
	if got := rec.Header().Get("Referrer-Policy"); got != "no-referrer" {
		t.Errorf("Referrer-Policy: expected no-referrer, got %s", got)
	}

	// HSTS should be empty when disabled
	if got := rec.Header().Get("Strict-Transport-Security"); got != "" {
		t.Errorf("Strict-Transport-Security: expected empty, got %s", got)
	}
}

func TestSecurity_SkipPaths(t *testing.T) {
	e := echo.New()

	cfg := SecurityConfig{
		XFrameOptions: "DENY",
		SkipPaths:     []string{"/health", "/public"},
	}
	e.Use(Security(cfg))
	e.GET("/health", func(c echo.Context) error {
		return c.String(http.StatusOK, "healthy")
	})
	e.GET("/api/test", func(c echo.Context) error {
		return c.String(http.StatusOK, "ok")
	})

	// Skip path should not have security headers
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if got := rec.Header().Get("X-Frame-Options"); got != "" {
		t.Errorf("Skip path should not have X-Frame-Options header, got %s", got)
	}

	// Regular path should have security headers
	req = httptest.NewRequest(http.MethodGet, "/api/test", nil)
	rec = httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if got := rec.Header().Get("X-Frame-Options"); got != "DENY" {
		t.Errorf("Regular path should have X-Frame-Options header, got %s", got)
	}
}

func TestSecurity_RequestSizeLimit_ContentLength(t *testing.T) {
	e := echo.New()

	cfg := SecurityConfig{
		MaxRequestBodySize: 100, // 100 bytes
	}
	e.Use(Security(cfg))
	e.POST("/api/upload", func(c echo.Context) error {
		body, _ := io.ReadAll(c.Request().Body)
		return c.String(http.StatusOK, string(body))
	})

	// Request with Content-Length exceeding limit should be rejected
	largeBody := strings.Repeat("x", 200)
	req := httptest.NewRequest(http.MethodPost, "/api/upload", strings.NewReader(largeBody))
	req.Header.Set("Content-Type", "text/plain")
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusRequestEntityTooLarge {
		t.Errorf("Expected status %d, got %d", http.StatusRequestEntityTooLarge, rec.Code)
	}
}

func TestSecurity_RequestSizeLimit_SmallBody(t *testing.T) {
	e := echo.New()

	cfg := SecurityConfig{
		MaxRequestBodySize: 100, // 100 bytes
	}
	e.Use(Security(cfg))
	e.POST("/api/upload", func(c echo.Context) error {
		body, _ := io.ReadAll(c.Request().Body)
		return c.String(http.StatusOK, string(body))
	})

	// Small request should succeed
	smallBody := "hello world"
	req := httptest.NewRequest(http.MethodPost, "/api/upload", strings.NewReader(smallBody))
	req.Header.Set("Content-Type", "text/plain")
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, rec.Code)
	}
	if rec.Body.String() != smallBody {
		t.Errorf("Expected body %q, got %q", smallBody, rec.Body.String())
	}
}

func TestSecurity_RequestSizeLimit_SkipPaths(t *testing.T) {
	e := echo.New()

	cfg := SecurityConfig{
		MaxRequestBodySize:   100,
		SkipRequestSizeLimit: []string{"/api/upload"},
	}
	e.Use(Security(cfg))
	e.POST("/api/upload", func(c echo.Context) error {
		body, _ := io.ReadAll(c.Request().Body)
		return c.String(http.StatusOK, string(body))
	})

	// Large request to skip path should succeed
	largeBody := strings.Repeat("x", 200)
	req := httptest.NewRequest(http.MethodPost, "/api/upload", strings.NewReader(largeBody))
	req.Header.Set("Content-Type", "text/plain")
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("Skip path should allow large body, got status %d", rec.Code)
	}
}

func TestSecurity_RequestSizeLimit_StreamingBody(t *testing.T) {
	e := echo.New()

	cfg := SecurityConfig{
		MaxRequestBodySize: 50, // 50 bytes
	}
	e.Use(Security(cfg))
	e.POST("/api/upload", func(c echo.Context) error {
		body, err := io.ReadAll(c.Request().Body)
		if err != nil {
			return err
		}
		return c.String(http.StatusOK, string(body))
	})

	// Simulate streaming body without Content-Length
	largeBody := strings.Repeat("x", 100)
	req := httptest.NewRequest(http.MethodPost, "/api/upload", strings.NewReader(largeBody))
	req.Header.Set("Content-Type", "text/plain")
	req.ContentLength = -1 // Unknown content length
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	// Should fail when body is read and exceeds limit
	if rec.Code == http.StatusOK && rec.Body.Len() > 50 {
		t.Error("Should have limited the body read")
	}
}

func TestSecurityWithDefaults(t *testing.T) {
	e := echo.New()
	e.Use(SecurityWithDefaults())
	e.GET("/api/test", func(c echo.Context) error {
		return c.String(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/api/test", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, rec.Code)
	}

	// Should have security headers
	if rec.Header().Get("X-Frame-Options") == "" {
		t.Error("X-Frame-Options header should be set")
	}
}

func TestSecurityHeaders_Only(t *testing.T) {
	e := echo.New()

	cfg := SecurityConfig{
		XFrameOptions:       "DENY",
		XContentTypeOptions: "nosniff",
	}
	e.Use(SecurityHeaders(cfg))
	e.GET("/api/test", func(c echo.Context) error {
		return c.String(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/api/test", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Header().Get("X-Frame-Options") != "DENY" {
		t.Error("X-Frame-Options should be set")
	}
	if rec.Header().Get("X-Content-Type-Options") != "nosniff" {
		t.Error("X-Content-Type-Options should be set")
	}
}

func TestRequestSizeLimit_Only(t *testing.T) {
	e := echo.New()

	e.Use(RequestSizeLimit(50))
	e.POST("/api/upload", func(c echo.Context) error {
		body, _ := io.ReadAll(c.Request().Body)
		return c.String(http.StatusOK, string(body))
	})

	// Large request should be rejected
	largeBody := strings.Repeat("x", 100)
	req := httptest.NewRequest(http.MethodPost, "/api/upload", strings.NewReader(largeBody))
	req.Header.Set("Content-Type", "text/plain")
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusRequestEntityTooLarge {
		t.Errorf("Expected status %d, got %d", http.StatusRequestEntityTooLarge, rec.Code)
	}

	// Small request should succeed
	smallBody := "hello"
	req = httptest.NewRequest(http.MethodPost, "/api/upload", strings.NewReader(smallBody))
	req.Header.Set("Content-Type", "text/plain")
	rec = httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, rec.Code)
	}
}

func TestRequestSizeLimit_WithSkipPaths(t *testing.T) {
	e := echo.New()

	e.Use(RequestSizeLimit(50, "/upload", "/files"))
	e.POST("/upload", func(c echo.Context) error {
		body, _ := io.ReadAll(c.Request().Body)
		return c.String(http.StatusOK, string(body))
	})
	e.POST("/api/data", func(c echo.Context) error {
		body, _ := io.ReadAll(c.Request().Body)
		return c.String(http.StatusOK, string(body))
	})

	// Skip path should allow large body
	largeBody := strings.Repeat("x", 100)
	req := httptest.NewRequest(http.MethodPost, "/upload", strings.NewReader(largeBody))
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("Skip path should allow large body, got %d", rec.Code)
	}

	// Non-skip path should reject large body
	req = httptest.NewRequest(http.MethodPost, "/api/data", strings.NewReader(largeBody))
	rec = httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusRequestEntityTooLarge {
		t.Errorf("Non-skip path should reject large body, got %d", rec.Code)
	}
}

func TestDefaultSecurityConfig(t *testing.T) {
	cfg := DefaultSecurityConfig()

	if cfg.MaxRequestBodySize <= 0 {
		t.Error("MaxRequestBodySize should be positive")
	}
	if cfg.XFrameOptions == "" {
		t.Error("XFrameOptions should not be empty")
	}
	if cfg.XContentTypeOptions == "" {
		t.Error("XContentTypeOptions should not be empty")
	}
	if cfg.XXSSProtection == "" {
		t.Error("XXSSProtection should not be empty")
	}
	if cfg.ReferrerPolicy == "" {
		t.Error("ReferrerPolicy should not be empty")
	}
	if cfg.ContentSecurityPolicy == "" {
		t.Error("ContentSecurityPolicy should not be empty")
	}
	if cfg.CacheControl == "" {
		t.Error("CacheControl should not be empty")
	}
}

func TestLimitedReader_Read(t *testing.T) {
	data := []byte("hello world, this is a test message")
	reader := bytes.NewReader(data)
	readCloser := io.NopCloser(reader)

	e := echo.New()
	c := e.NewContext(httptest.NewRequest(http.MethodPost, "/test", nil), httptest.NewRecorder())

	lr := &limitedReader{
		reader:  readCloser,
		limit:   10,
		context: c,
	}

	// Read should succeed for data within limit
	buf := make([]byte, 5)
	n, err := lr.Read(buf)
	if err != nil {
		t.Errorf("Read should succeed within limit: %v", err)
	}
	if n != 5 {
		t.Errorf("Expected to read 5 bytes, got %d", n)
	}

	// Continue reading
	n, err = lr.Read(buf)
	if err != nil {
		t.Errorf("Read should succeed within limit: %v", err)
	}

	// Reading more should fail
	n, err = lr.Read(buf)
	if err == nil {
		t.Error("Read should fail when exceeding limit")
	}
	if _, ok := err.(*RequestTooLargeError); !ok {
		t.Errorf("Expected RequestTooLargeError, got %T", err)
	}
}

func TestLimitedReader_Close(t *testing.T) {
	var closed bool
	reader := &mockReadCloser{
		data: []byte("test"),
		onClose: func() {
			closed = true
		},
	}

	e := echo.New()
	c := e.NewContext(httptest.NewRequest(http.MethodPost, "/test", nil), httptest.NewRecorder())

	lr := &limitedReader{
		reader:  reader,
		limit:   100,
		context: c,
	}

	err := lr.Close()
	if err != nil {
		t.Errorf("Close should not return error: %v", err)
	}
	if !closed {
		t.Error("Close should call underlying reader's Close")
	}
}

func TestRequestTooLargeError_Error(t *testing.T) {
	err := &RequestTooLargeError{
		MaxSize:   100,
		BytesRead: 150,
	}

	msg := err.Error()
	if !strings.Contains(msg, "100") {
		t.Error("Error message should contain max size")
	}
	if !strings.Contains(msg, "150") {
		t.Error("Error message should contain bytes read")
	}
}

// mockReadCloser is a mock io.ReadCloser for testing.
type mockReadCloser struct {
	data    []byte
	offset  int
	onClose func()
}

func (m *mockReadCloser) Read(p []byte) (int, error) {
	if m.offset >= len(m.data) {
		return 0, io.EOF
	}
	n := copy(p, m.data[m.offset:])
	m.offset += n
	return n, nil
}

func (m *mockReadCloser) Close() error {
	if m.onClose != nil {
		m.onClose()
	}
	return nil
}

func TestSecurity_NilBody(t *testing.T) {
	e := echo.New()

	cfg := SecurityConfig{
		MaxRequestBodySize: 100,
	}
	e.Use(Security(cfg))
	e.GET("/api/test", func(c echo.Context) error {
		return c.String(http.StatusOK, "ok")
	})

	// GET request with no body should work
	req := httptest.NewRequest(http.MethodGet, "/api/test", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, rec.Code)
	}
}
