package api

import (
	"context"
	"net/http"
	"testing"
	"time"

	"yunt/internal/config"
)

func setupTestServer() *Server {
	cfg := config.APIConfig{
		Enabled:            true,
		Host:               "127.0.0.1",
		Port:               0, // Let the OS assign a port
		ReadTimeout:        30 * time.Second,
		WriteTimeout:       30 * time.Second,
		CORSAllowedOrigins: []string{"*"},
	}

	logger := config.NewDefaultLogger()
	return New(cfg, WithLogger(logger))
}

func TestNew(t *testing.T) {
	cfg := config.APIConfig{
		Enabled:            true,
		Host:               "127.0.0.1",
		Port:               8080,
		ReadTimeout:        30 * time.Second,
		WriteTimeout:       30 * time.Second,
		CORSAllowedOrigins: []string{"*"},
	}

	server := New(cfg)

	if server == nil {
		t.Fatal("New returned nil")
	}

	if server.Echo() == nil {
		t.Error("Echo instance should not be nil")
	}

	if server.logger == nil {
		t.Error("Logger should be set to default when not provided")
	}
}

func TestNewWithLogger(t *testing.T) {
	cfg := config.APIConfig{
		Enabled: true,
		Host:    "127.0.0.1",
		Port:    8080,
	}

	logger := config.NewDefaultLogger()
	server := New(cfg, WithLogger(logger))

	if server.logger != logger {
		t.Error("Logger was not set correctly")
	}
}

func TestServerAddress(t *testing.T) {
	cfg := config.APIConfig{
		Host: "127.0.0.1",
		Port: 8080,
	}

	server := New(cfg)
	expected := "127.0.0.1:8080"

	if server.Address() != expected {
		t.Errorf("expected address %s, got %s", expected, server.Address())
	}
}

func TestServerStartAndShutdown(t *testing.T) {
	cfg := config.APIConfig{
		Enabled:            true,
		Host:               "127.0.0.1",
		Port:               18025, // Use high port to avoid conflicts
		ReadTimeout:        30 * time.Second,
		WriteTimeout:       30 * time.Second,
		CORSAllowedOrigins: []string{"*"},
	}

	logger := config.NewDefaultLogger()
	server := New(cfg, WithLogger(logger))

	// Create a context that we can cancel
	ctx, cancel := context.WithCancel(context.Background())

	// Start server in a goroutine
	errChan := make(chan error, 1)
	go func() {
		errChan <- server.StartWithContext(ctx)
	}()

	// Wait a bit for the server to start
	time.Sleep(100 * time.Millisecond)

	// Make a request to verify the server is running
	resp, err := http.Get("http://127.0.0.1:18025/healthz")
	if err != nil {
		t.Fatalf("failed to make request: %v", err)
	}
	resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, resp.StatusCode)
	}

	// Cancel the context to trigger shutdown
	cancel()

	// Wait for shutdown to complete
	select {
	case err := <-errChan:
		if err != nil {
			t.Errorf("StartWithContext returned error: %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Error("shutdown timed out")
	}
}

func TestServerGracefulShutdown(t *testing.T) {
	cfg := config.APIConfig{
		Enabled:            true,
		Host:               "127.0.0.1",
		Port:               18026,
		ReadTimeout:        30 * time.Second,
		WriteTimeout:       30 * time.Second,
		CORSAllowedOrigins: []string{"*"},
	}

	logger := config.NewDefaultLogger()
	server := New(cfg, WithLogger(logger))

	// Start server in a goroutine
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	errChan := make(chan error, 1)
	go func() {
		errChan <- server.StartWithContext(ctx)
	}()

	// Wait for server to start
	time.Sleep(100 * time.Millisecond)

	// Shutdown via the Shutdown method
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutdownCancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		t.Errorf("Shutdown returned error: %v", err)
	}
}

func TestServerIsRunning(t *testing.T) {
	cfg := config.APIConfig{
		Enabled: true,
		Host:    "127.0.0.1",
		Port:    8080,
	}

	server := New(cfg)

	// Server should not be running before Start is called
	if server.IsRunning() {
		t.Error("Server should not be running before Start is called")
	}
}
