// Package config provides configuration management for the Yunt mail server.
// This file implements structured logging using zerolog.
package config

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/rs/zerolog"
	"gopkg.in/natefinch/lumberjack.v2"
)

// Logger wraps zerolog.Logger with additional configuration capabilities.
type Logger struct {
	zerolog.Logger
	config  LoggingConfig
	writer  io.Writer
	file    *os.File
	mu      sync.RWMutex
	closers []io.Closer
}

// logLevel maps string log levels to zerolog levels.
var logLevel = map[string]zerolog.Level{
	"trace": zerolog.TraceLevel,
	"debug": zerolog.DebugLevel,
	"info":  zerolog.InfoLevel,
	"warn":  zerolog.WarnLevel,
	"error": zerolog.ErrorLevel,
	"fatal": zerolog.FatalLevel,
	"panic": zerolog.PanicLevel,
}

// NewLogger creates a new logger instance based on the provided configuration.
// It initializes the logger with the specified level, format, and output destination.
// The returned logger is thread-safe and can be used concurrently.
func NewLogger(cfg LoggingConfig) (*Logger, error) {
	l := &Logger{
		config:  cfg,
		closers: make([]io.Closer, 0),
	}

	// Set up the writer based on output configuration
	writer, err := l.setupWriter(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to setup logger writer: %w", err)
	}
	l.writer = writer

	// Configure the logger format
	var logWriter io.Writer
	if strings.ToLower(cfg.Format) == "text" {
		logWriter = zerolog.ConsoleWriter{
			Out:        writer,
			TimeFormat: time.RFC3339,
			NoColor:    !isTerminal(writer),
		}
	} else {
		// Default to JSON format
		logWriter = writer
	}

	// Create the zerolog logger
	logger := zerolog.New(logWriter).With().Timestamp()

	// Add caller information if configured
	if cfg.IncludeCaller {
		logger = logger.Caller()
	}

	l.Logger = logger.Logger()

	// Set the log level
	level := parseLevel(cfg.Level)
	l.Logger = l.Logger.Level(level)

	return l, nil
}

// setupWriter configures the output destination based on the configuration.
func (l *Logger) setupWriter(cfg LoggingConfig) (io.Writer, error) {
	output := strings.ToLower(cfg.Output)

	switch output {
	case "stdout", "":
		return os.Stdout, nil
	case "stderr":
		return os.Stderr, nil
	case "file":
		return l.setupFileWriter(cfg)
	default:
		// Treat as file path for backwards compatibility
		if output != "" {
			cfg.FilePath = output
			return l.setupFileWriter(cfg)
		}
		return os.Stdout, nil
	}
}

// setupFileWriter creates a file writer for log output with rotation support.
func (l *Logger) setupFileWriter(cfg LoggingConfig) (io.Writer, error) {
	if cfg.FilePath == "" {
		return nil, fmt.Errorf("file path is required when output is 'file'")
	}

	// Ensure the directory exists
	dir := filepath.Dir(cfg.FilePath)
	if dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return nil, fmt.Errorf("failed to create log directory %s: %w", dir, err)
		}
	}

	maxSize := cfg.MaxSize
	if maxSize <= 0 {
		maxSize = 100
	}
	maxBackups := cfg.MaxBackups
	if maxBackups <= 0 {
		maxBackups = 5
	}
	maxAge := cfg.MaxAge
	if maxAge <= 0 {
		maxAge = 30
	}

	rotator := &lumberjack.Logger{
		Filename:   cfg.FilePath,
		MaxSize:    maxSize,
		MaxBackups: maxBackups,
		MaxAge:     maxAge,
		Compress:   true,
	}

	l.closers = append(l.closers, rotator)

	return rotator, nil
}

// parseLevel converts a string log level to zerolog.Level.
func parseLevel(level string) zerolog.Level {
	level = strings.ToLower(strings.TrimSpace(level))
	if lvl, ok := logLevel[level]; ok {
		return lvl
	}
	// Default to info level
	return zerolog.InfoLevel
}

// isTerminal checks if the writer is a terminal (for colored output).
func isTerminal(w io.Writer) bool {
	if f, ok := w.(*os.File); ok {
		fi, err := f.Stat()
		if err != nil {
			return false
		}
		return (fi.Mode() & os.ModeCharDevice) != 0
	}
	return false
}

// Close closes any open file handles used by the logger.
func (l *Logger) Close() error {
	l.mu.Lock()
	defer l.mu.Unlock()

	var errs []error
	for _, closer := range l.closers {
		if err := closer.Close(); err != nil {
			errs = append(errs, err)
		}
	}
	l.closers = nil

	if len(errs) > 0 {
		return fmt.Errorf("errors closing logger: %v", errs)
	}
	return nil
}

// GetLevel returns the current log level as a string.
func (l *Logger) GetLevel() string {
	l.mu.RLock()
	defer l.mu.RUnlock()
	return l.config.Level
}

// GetFormat returns the current log format.
func (l *Logger) GetFormat() string {
	l.mu.RLock()
	defer l.mu.RUnlock()
	return l.config.Format
}

// GetOutput returns the current output destination.
func (l *Logger) GetOutput() string {
	l.mu.RLock()
	defer l.mu.RUnlock()
	return l.config.Output
}

// GetConfig returns a copy of the logging configuration.
func (l *Logger) GetConfig() LoggingConfig {
	l.mu.RLock()
	defer l.mu.RUnlock()
	return l.config
}

// With creates a child logger with additional context fields.
func (l *Logger) With() zerolog.Context {
	return l.Logger.With()
}

// WithFields creates a child logger with multiple fields.
func (l *Logger) WithFields(fields map[string]interface{}) *Logger {
	l.mu.RLock()
	defer l.mu.RUnlock()

	ctx := l.Logger.With()
	for k, v := range fields {
		ctx = ctx.Interface(k, v)
	}

	return &Logger{
		Logger:  ctx.Logger(),
		config:  l.config,
		writer:  l.writer,
		file:    l.file,
		closers: nil, // Child loggers don't own the closers
	}
}

// WithComponent creates a child logger with a component field.
func (l *Logger) WithComponent(component string) *Logger {
	l.mu.RLock()
	defer l.mu.RUnlock()

	return &Logger{
		Logger:  l.Logger.With().Str("component", component).Logger(),
		config:  l.config,
		writer:  l.writer,
		file:    l.file,
		closers: nil,
	}
}

// MustNewLogger creates a new logger and panics on error.
func MustNewLogger(cfg LoggingConfig) *Logger {
	logger, err := NewLogger(cfg)
	if err != nil {
		panic(fmt.Sprintf("failed to create logger: %v", err))
	}
	return logger
}

// NewDefaultLogger creates a logger with default settings.
func NewDefaultLogger() *Logger {
	cfg := LoggingConfig{
		Level:         DefaultLoggingLevel,
		Format:        DefaultLoggingFormat,
		Output:        DefaultLoggingOutput,
		IncludeCaller: DefaultLoggingIncludeCaller,
	}
	logger, _ := NewLogger(cfg)
	return logger
}

// SetGlobalLevel sets the global zerolog level.
// This affects all loggers that don't have an explicit level set.
func SetGlobalLevel(level string) {
	zerolog.SetGlobalLevel(parseLevel(level))
}

// IsLevelEnabled checks if a log level is enabled for this logger.
func (l *Logger) IsLevelEnabled(level string) bool {
	lvl := parseLevel(level)
	return l.Logger.GetLevel() <= lvl
}

// Sync flushes any buffered log entries. For file output, this syncs the file.
func (l *Logger) Sync() error {
	l.mu.Lock()
	defer l.mu.Unlock()

	if l.file != nil {
		return l.file.Sync()
	}
	return nil
}
