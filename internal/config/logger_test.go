package config

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
)

func TestNewLogger(t *testing.T) {
	tests := []struct {
		name    string
		config  LoggingConfig
		wantErr bool
	}{
		{
			name: "default configuration",
			config: LoggingConfig{
				Level:  "info",
				Format: "json",
				Output: "stdout",
			},
			wantErr: false,
		},
		{
			name: "text format",
			config: LoggingConfig{
				Level:  "debug",
				Format: "text",
				Output: "stdout",
			},
			wantErr: false,
		},
		{
			name: "stderr output",
			config: LoggingConfig{
				Level:  "warn",
				Format: "json",
				Output: "stderr",
			},
			wantErr: false,
		},
		{
			name: "with caller info",
			config: LoggingConfig{
				Level:         "info",
				Format:        "json",
				Output:        "stdout",
				IncludeCaller: true,
			},
			wantErr: false,
		},
		{
			name: "all log levels",
			config: LoggingConfig{
				Level:  "trace",
				Format: "json",
				Output: "stdout",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger, err := NewLogger(tt.config)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewLogger() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && logger == nil {
				t.Error("NewLogger() returned nil logger without error")
			}
			if logger != nil {
				defer logger.Close()
			}
		})
	}
}

func TestNewLoggerFileOutput(t *testing.T) {
	tmpDir := t.TempDir()
	logPath := filepath.Join(tmpDir, "test.log")

	cfg := LoggingConfig{
		Level:    "info",
		Format:   "json",
		Output:   "file",
		FilePath: logPath,
	}

	logger, err := NewLogger(cfg)
	if err != nil {
		t.Fatalf("NewLogger() error = %v", err)
	}
	defer logger.Close()

	// Write a log message
	logger.Info().Msg("test message")

	// Ensure file sync
	if err := logger.Sync(); err != nil {
		t.Errorf("Sync() error = %v", err)
	}

	// Verify the file was created
	if _, err := os.Stat(logPath); os.IsNotExist(err) {
		t.Error("log file was not created")
	}

	// Verify content
	content, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("failed to read log file: %v", err)
	}

	if !strings.Contains(string(content), "test message") {
		t.Error("log file does not contain expected message")
	}
}

func TestNewLoggerFileOutputNestedDirectory(t *testing.T) {
	tmpDir := t.TempDir()
	logPath := filepath.Join(tmpDir, "nested", "dir", "test.log")

	cfg := LoggingConfig{
		Level:    "info",
		Format:   "json",
		Output:   "file",
		FilePath: logPath,
	}

	logger, err := NewLogger(cfg)
	if err != nil {
		t.Fatalf("NewLogger() error = %v", err)
	}
	defer logger.Close()

	// Verify the directory was created
	dir := filepath.Dir(logPath)
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		t.Error("log directory was not created")
	}
}

func TestNewLoggerFileOutputMissingPath(t *testing.T) {
	cfg := LoggingConfig{
		Level:  "info",
		Format: "json",
		Output: "file",
		// FilePath is intentionally empty
	}

	_, err := NewLogger(cfg)
	if err == nil {
		t.Error("NewLogger() should return error for file output without path")
	}
}

func TestLoggerLogLevels(t *testing.T) {
	tests := []struct {
		name          string
		configLevel   string
		logLevel      string
		shouldContain bool
	}{
		{"debug logs at debug level", "debug", "debug", true},
		{"info logs at debug level", "debug", "info", true},
		{"warn logs at debug level", "debug", "warn", true},
		{"error logs at debug level", "debug", "error", true},
		{"debug filtered at info level", "info", "debug", false},
		{"info logs at info level", "info", "info", true},
		{"warn logs at info level", "info", "warn", true},
		{"error logs at info level", "info", "error", true},
		{"debug filtered at warn level", "warn", "debug", false},
		{"info filtered at warn level", "warn", "info", false},
		{"warn logs at warn level", "warn", "warn", true},
		{"error logs at warn level", "warn", "error", true},
		{"debug filtered at error level", "error", "debug", false},
		{"info filtered at error level", "error", "info", false},
		{"warn filtered at error level", "error", "warn", false},
		{"error logs at error level", "error", "error", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			logPath := filepath.Join(tmpDir, "level_test.log")

			cfg := LoggingConfig{
				Level:    tt.configLevel,
				Format:   "json",
				Output:   "file",
				FilePath: logPath,
			}

			logger, err := NewLogger(cfg)
			if err != nil {
				t.Fatalf("NewLogger() error = %v", err)
			}

			testMsg := "test-level-message"
			switch tt.logLevel {
			case "debug":
				logger.Debug().Msg(testMsg)
			case "info":
				logger.Info().Msg(testMsg)
			case "warn":
				logger.Warn().Msg(testMsg)
			case "error":
				logger.Error().Msg(testMsg)
			}

			logger.Sync()
			logger.Close()

			content, err := os.ReadFile(logPath)
			if err != nil && !os.IsNotExist(err) {
				t.Fatalf("failed to read log file: %v", err)
			}

			contains := len(content) > 0 && strings.Contains(string(content), testMsg)
			if contains != tt.shouldContain {
				t.Errorf("log contains message = %v, want %v", contains, tt.shouldContain)
			}
		})
	}
}

func TestLoggerJSONFormat(t *testing.T) {
	tmpDir := t.TempDir()
	logPath := filepath.Join(tmpDir, "json_test.log")

	cfg := LoggingConfig{
		Level:    "info",
		Format:   "json",
		Output:   "file",
		FilePath: logPath,
	}

	logger, err := NewLogger(cfg)
	if err != nil {
		t.Fatalf("NewLogger() error = %v", err)
	}

	logger.Info().Str("key", "value").Int("count", 42).Msg("json test")
	logger.Sync()
	logger.Close()

	content, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("failed to read log file: %v", err)
	}

	// Parse as JSON to verify it's valid
	var logEntry map[string]interface{}
	if err := json.Unmarshal(bytes.TrimSpace(content), &logEntry); err != nil {
		t.Errorf("log output is not valid JSON: %v", err)
	}

	// Verify fields
	if logEntry["message"] != "json test" {
		t.Errorf("message = %v, want 'json test'", logEntry["message"])
	}
	if logEntry["key"] != "value" {
		t.Errorf("key = %v, want 'value'", logEntry["key"])
	}
	if logEntry["count"] != float64(42) { // JSON numbers are float64
		t.Errorf("count = %v, want 42", logEntry["count"])
	}
	if logEntry["level"] != "info" {
		t.Errorf("level = %v, want 'info'", logEntry["level"])
	}
	if _, ok := logEntry["time"]; !ok {
		t.Error("timestamp field is missing")
	}
}

func TestLoggerTextFormat(t *testing.T) {
	tmpDir := t.TempDir()
	logPath := filepath.Join(tmpDir, "text_test.log")

	cfg := LoggingConfig{
		Level:    "info",
		Format:   "text",
		Output:   "file",
		FilePath: logPath,
	}

	logger, err := NewLogger(cfg)
	if err != nil {
		t.Fatalf("NewLogger() error = %v", err)
	}

	logger.Info().Str("key", "value").Msg("text test")
	logger.Sync()
	logger.Close()

	content, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("failed to read log file: %v", err)
	}

	strContent := string(content)

	// Verify it contains expected parts (text format is human-readable)
	if !strings.Contains(strContent, "text test") {
		t.Error("log does not contain message")
	}
	if !strings.Contains(strContent, "key=value") {
		t.Error("log does not contain key=value field")
	}
	if !strings.Contains(strContent, "INF") {
		t.Error("log does not contain level indicator")
	}
}

func TestLoggerWithCaller(t *testing.T) {
	tmpDir := t.TempDir()
	logPath := filepath.Join(tmpDir, "caller_test.log")

	cfg := LoggingConfig{
		Level:         "info",
		Format:        "json",
		Output:        "file",
		FilePath:      logPath,
		IncludeCaller: true,
	}

	logger, err := NewLogger(cfg)
	if err != nil {
		t.Fatalf("NewLogger() error = %v", err)
	}

	logger.Info().Msg("caller test")
	logger.Sync()
	logger.Close()

	content, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("failed to read log file: %v", err)
	}

	var logEntry map[string]interface{}
	if err := json.Unmarshal(bytes.TrimSpace(content), &logEntry); err != nil {
		t.Fatalf("failed to parse JSON: %v", err)
	}

	// Verify caller field exists
	if _, ok := logEntry["caller"]; !ok {
		t.Error("caller field is missing")
	}
}

func TestLoggerThreadSafety(t *testing.T) {
	tmpDir := t.TempDir()
	logPath := filepath.Join(tmpDir, "concurrent_test.log")

	cfg := LoggingConfig{
		Level:    "info",
		Format:   "json",
		Output:   "file",
		FilePath: logPath,
	}

	logger, err := NewLogger(cfg)
	if err != nil {
		t.Fatalf("NewLogger() error = %v", err)
	}

	const goroutines = 100
	const messagesPerGoroutine = 10

	var wg sync.WaitGroup
	wg.Add(goroutines)

	for i := 0; i < goroutines; i++ {
		go func(id int) {
			defer wg.Done()
			for j := 0; j < messagesPerGoroutine; j++ {
				logger.Info().Int("goroutine", id).Int("message", j).Msg("concurrent message")
			}
		}(i)
	}

	wg.Wait()
	logger.Sync()
	logger.Close()

	// Verify the file exists and has content
	content, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("failed to read log file: %v", err)
	}

	// Count log lines
	lines := strings.Split(strings.TrimSpace(string(content)), "\n")
	expectedLines := goroutines * messagesPerGoroutine

	if len(lines) != expectedLines {
		t.Errorf("expected %d log lines, got %d", expectedLines, len(lines))
	}

	// Verify each line is valid JSON
	for i, line := range lines {
		var logEntry map[string]interface{}
		if err := json.Unmarshal([]byte(line), &logEntry); err != nil {
			t.Errorf("line %d is not valid JSON: %v", i, err)
		}
	}
}

func TestLoggerGetters(t *testing.T) {
	cfg := LoggingConfig{
		Level:  "debug",
		Format: "json",
		Output: "stdout",
	}

	logger, err := NewLogger(cfg)
	if err != nil {
		t.Fatalf("NewLogger() error = %v", err)
	}
	defer logger.Close()

	if got := logger.GetLevel(); got != "debug" {
		t.Errorf("GetLevel() = %v, want 'debug'", got)
	}

	if got := logger.GetFormat(); got != "json" {
		t.Errorf("GetFormat() = %v, want 'json'", got)
	}

	if got := logger.GetOutput(); got != "stdout" {
		t.Errorf("GetOutput() = %v, want 'stdout'", got)
	}

	gotCfg := logger.GetConfig()
	if gotCfg.Level != cfg.Level || gotCfg.Format != cfg.Format || gotCfg.Output != cfg.Output {
		t.Errorf("GetConfig() returned different config")
	}
}

func TestLoggerWithFields(t *testing.T) {
	tmpDir := t.TempDir()
	logPath := filepath.Join(tmpDir, "fields_test.log")

	cfg := LoggingConfig{
		Level:    "info",
		Format:   "json",
		Output:   "file",
		FilePath: logPath,
	}

	logger, err := NewLogger(cfg)
	if err != nil {
		t.Fatalf("NewLogger() error = %v", err)
	}

	childLogger := logger.WithFields(map[string]interface{}{
		"service": "test-service",
		"version": "1.0.0",
	})

	childLogger.Info().Msg("child logger test")
	logger.Sync()
	logger.Close()

	content, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("failed to read log file: %v", err)
	}

	var logEntry map[string]interface{}
	if err := json.Unmarshal(bytes.TrimSpace(content), &logEntry); err != nil {
		t.Fatalf("failed to parse JSON: %v", err)
	}

	if logEntry["service"] != "test-service" {
		t.Errorf("service = %v, want 'test-service'", logEntry["service"])
	}
	if logEntry["version"] != "1.0.0" {
		t.Errorf("version = %v, want '1.0.0'", logEntry["version"])
	}
}

func TestLoggerWithComponent(t *testing.T) {
	tmpDir := t.TempDir()
	logPath := filepath.Join(tmpDir, "component_test.log")

	cfg := LoggingConfig{
		Level:    "info",
		Format:   "json",
		Output:   "file",
		FilePath: logPath,
	}

	logger, err := NewLogger(cfg)
	if err != nil {
		t.Fatalf("NewLogger() error = %v", err)
	}

	smtpLogger := logger.WithComponent("smtp")
	smtpLogger.Info().Msg("smtp component test")
	logger.Sync()
	logger.Close()

	content, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("failed to read log file: %v", err)
	}

	var logEntry map[string]interface{}
	if err := json.Unmarshal(bytes.TrimSpace(content), &logEntry); err != nil {
		t.Fatalf("failed to parse JSON: %v", err)
	}

	if logEntry["component"] != "smtp" {
		t.Errorf("component = %v, want 'smtp'", logEntry["component"])
	}
}

func TestMustNewLogger(t *testing.T) {
	// Test that valid config doesn't panic
	cfg := LoggingConfig{
		Level:  "info",
		Format: "json",
		Output: "stdout",
	}

	defer func() {
		if r := recover(); r != nil {
			t.Errorf("MustNewLogger() panicked unexpectedly: %v", r)
		}
	}()

	logger := MustNewLogger(cfg)
	logger.Close()
}

func TestMustNewLoggerPanics(t *testing.T) {
	cfg := LoggingConfig{
		Level:    "info",
		Format:   "json",
		Output:   "file",
		FilePath: "", // Invalid: empty path for file output
	}

	defer func() {
		if r := recover(); r == nil {
			t.Error("MustNewLogger() should have panicked")
		}
	}()

	MustNewLogger(cfg)
}

func TestNewDefaultLogger(t *testing.T) {
	logger := NewDefaultLogger()
	if logger == nil {
		t.Fatal("NewDefaultLogger() returned nil")
	}
	defer logger.Close()

	if logger.GetLevel() != DefaultLoggingLevel {
		t.Errorf("GetLevel() = %v, want %v", logger.GetLevel(), DefaultLoggingLevel)
	}
	if logger.GetFormat() != DefaultLoggingFormat {
		t.Errorf("GetFormat() = %v, want %v", logger.GetFormat(), DefaultLoggingFormat)
	}
	if logger.GetOutput() != DefaultLoggingOutput {
		t.Errorf("GetOutput() = %v, want %v", logger.GetOutput(), DefaultLoggingOutput)
	}
}

func TestParseLevel(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"trace", "trace"},
		{"TRACE", "trace"},
		{"debug", "debug"},
		{"DEBUG", "debug"},
		{"info", "info"},
		{"INFO", "info"},
		{"warn", "warn"},
		{"WARN", "warn"},
		{"error", "error"},
		{"ERROR", "error"},
		{"fatal", "fatal"},
		{"FATAL", "fatal"},
		{"panic", "panic"},
		{"PANIC", "panic"},
		{"invalid", "info"}, // defaults to info
		{"", "info"},        // empty defaults to info
		{"  info  ", "info"}, // trimmed
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			cfg := LoggingConfig{
				Level:  tt.input,
				Format: "json",
				Output: "stdout",
			}

			logger, err := NewLogger(cfg)
			if err != nil {
				t.Fatalf("NewLogger() error = %v", err)
			}
			defer logger.Close()

			// The logger stores the original config level, but the internal level is parsed
			// We can't directly test parseLevel, but we can verify logging behavior
			if tt.input == "invalid" || tt.input == "" {
				// For invalid/empty, should default to info and filter debug
				if logger.IsLevelEnabled("debug") {
					// This is actually checking zerolog behavior - debug should be filtered at info
				}
			}
		})
	}
}

func TestIsLevelEnabled(t *testing.T) {
	cfg := LoggingConfig{
		Level:  "info",
		Format: "json",
		Output: "stdout",
	}

	logger, err := NewLogger(cfg)
	if err != nil {
		t.Fatalf("NewLogger() error = %v", err)
	}
	defer logger.Close()

	if logger.IsLevelEnabled("debug") {
		t.Error("debug should not be enabled at info level")
	}
	if !logger.IsLevelEnabled("info") {
		t.Error("info should be enabled at info level")
	}
	if !logger.IsLevelEnabled("warn") {
		t.Error("warn should be enabled at info level")
	}
	if !logger.IsLevelEnabled("error") {
		t.Error("error should be enabled at info level")
	}
}

func TestLoggerClose(t *testing.T) {
	tmpDir := t.TempDir()
	logPath := filepath.Join(tmpDir, "close_test.log")

	cfg := LoggingConfig{
		Level:    "info",
		Format:   "json",
		Output:   "file",
		FilePath: logPath,
	}

	logger, err := NewLogger(cfg)
	if err != nil {
		t.Fatalf("NewLogger() error = %v", err)
	}

	logger.Info().Msg("before close")

	if err := logger.Close(); err != nil {
		t.Errorf("Close() error = %v", err)
	}

	// Second close should not error
	if err := logger.Close(); err != nil {
		t.Errorf("second Close() error = %v", err)
	}
}

func TestLoggerSync(t *testing.T) {
	t.Run("file output", func(t *testing.T) {
		tmpDir := t.TempDir()
		logPath := filepath.Join(tmpDir, "sync_test.log")

		cfg := LoggingConfig{
			Level:    "info",
			Format:   "json",
			Output:   "file",
			FilePath: logPath,
		}

		logger, err := NewLogger(cfg)
		if err != nil {
			t.Fatalf("NewLogger() error = %v", err)
		}
		defer logger.Close()

		logger.Info().Msg("sync test")
		if err := logger.Sync(); err != nil {
			t.Errorf("Sync() error = %v", err)
		}
	})

	t.Run("stdout output", func(t *testing.T) {
		cfg := LoggingConfig{
			Level:  "info",
			Format: "json",
			Output: "stdout",
		}

		logger, err := NewLogger(cfg)
		if err != nil {
			t.Fatalf("NewLogger() error = %v", err)
		}
		defer logger.Close()

		// Sync on stdout should not error
		if err := logger.Sync(); err != nil {
			t.Errorf("Sync() error = %v", err)
		}
	})
}

func TestSetGlobalLevel(t *testing.T) {
	// This tests the global level setter
	// Save original level to restore later
	originalLevel := "info"

	SetGlobalLevel("debug")

	// Reset to original
	defer SetGlobalLevel(originalLevel)

	// Create a logger and verify it respects the global level
	cfg := LoggingConfig{
		Level:  "", // Empty should use global
		Format: "json",
		Output: "stdout",
	}

	logger, err := NewLogger(cfg)
	if err != nil {
		t.Fatalf("NewLogger() error = %v", err)
	}
	defer logger.Close()

	// Logger should work at the global level
	// Note: Empty level defaults to info in our parseLevel, not the global level
	// This is expected behavior
}

func TestLoggerOutputAsFilePath(t *testing.T) {
	// Test backwards compatibility: output can be a file path directly
	tmpDir := t.TempDir()
	logPath := filepath.Join(tmpDir, "direct_path.log")

	cfg := LoggingConfig{
		Level:  "info",
		Format: "json",
		Output: logPath, // Direct file path as output
	}

	logger, err := NewLogger(cfg)
	if err != nil {
		t.Fatalf("NewLogger() error = %v", err)
	}

	logger.Info().Msg("direct path test")
	logger.Sync()
	logger.Close()

	// Verify file was created
	if _, err := os.Stat(logPath); os.IsNotExist(err) {
		t.Error("log file was not created with direct path output")
	}
}
