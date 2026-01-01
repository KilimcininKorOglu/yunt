// Package smtp provides the SMTP server implementation for Yunt mail server.
package smtp

import (
	"context"
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/emersion/go-smtp"
	"github.com/rs/zerolog"
)

// RateLimitConfig holds configuration for rate limiting.
type RateLimitConfig struct {
	// Enabled determines if rate limiting is active.
	Enabled bool

	// MessagesPerHour is the maximum number of messages per IP per hour.
	MessagesPerHour int

	// ConnectionsPerMinute is the maximum number of new connections per IP per minute.
	ConnectionsPerMinute int

	// MaxConcurrentConnections is the maximum number of concurrent connections per IP.
	MaxConcurrentConnections int

	// MaxGlobalConnections is the maximum total concurrent connections.
	MaxGlobalConnections int

	// RecipientsPerMessage is the maximum number of recipients per single message.
	RecipientsPerMessage int

	// MessagesPerConnection is the maximum number of messages per single connection.
	MessagesPerConnection int

	// CleanupInterval is how often to clean up expired rate limit entries.
	CleanupInterval time.Duration
}

// DefaultRateLimitConfig returns a RateLimitConfig with sensible defaults.
func DefaultRateLimitConfig() *RateLimitConfig {
	return &RateLimitConfig{
		Enabled:                  true,
		MessagesPerHour:          100,
		ConnectionsPerMinute:     20,
		MaxConcurrentConnections: 10,
		MaxGlobalConnections:     1000,
		RecipientsPerMessage:     100,
		MessagesPerConnection:    50,
		CleanupInterval:          5 * time.Minute,
	}
}

// RateLimiter tracks rate limits for SMTP connections.
type RateLimiter struct {
	config *RateLimitConfig
	logger zerolog.Logger

	mu sync.RWMutex

	// messageCount tracks messages per IP in the current hour window.
	messageCount map[string]*rateLimitEntry

	// connectionRate tracks connection attempts per IP in the current minute window.
	connectionRate map[string]*rateLimitEntry

	// concurrentConnections tracks active connections per IP.
	concurrentConnections map[string]int

	// globalConnections tracks total active connections.
	globalConnections int

	// messagesPerConnection tracks messages sent in current connection.
	messagesPerConnection map[string]int

	// stopChan signals the cleanup goroutine to stop.
	stopChan chan struct{}
}

// rateLimitEntry holds rate limit data for a single IP.
type rateLimitEntry struct {
	count     int
	windowEnd time.Time
}

// NewRateLimiter creates a new RateLimiter with the given configuration.
func NewRateLimiter(config *RateLimitConfig, logger zerolog.Logger) *RateLimiter {
	if config == nil {
		config = DefaultRateLimitConfig()
	}

	rl := &RateLimiter{
		config:                config,
		logger:                logger.With().Str("component", "ratelimit").Logger(),
		messageCount:          make(map[string]*rateLimitEntry),
		connectionRate:        make(map[string]*rateLimitEntry),
		concurrentConnections: make(map[string]int),
		messagesPerConnection: make(map[string]int),
		stopChan:              make(chan struct{}),
	}

	// Start cleanup goroutine
	if config.CleanupInterval > 0 {
		go rl.cleanupLoop()
	}

	return rl
}

// Stop stops the rate limiter's cleanup goroutine.
func (rl *RateLimiter) Stop() {
	close(rl.stopChan)
}

// cleanupLoop periodically cleans up expired rate limit entries.
func (rl *RateLimiter) cleanupLoop() {
	ticker := time.NewTicker(rl.config.CleanupInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			rl.cleanup()
		case <-rl.stopChan:
			return
		}
	}
}

// cleanup removes expired rate limit entries.
func (rl *RateLimiter) cleanup() {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()

	// Clean up message counts
	for ip, entry := range rl.messageCount {
		if now.After(entry.windowEnd) {
			delete(rl.messageCount, ip)
		}
	}

	// Clean up connection rates
	for ip, entry := range rl.connectionRate {
		if now.After(entry.windowEnd) {
			delete(rl.connectionRate, ip)
		}
	}

	rl.logger.Debug().
		Int("messageEntries", len(rl.messageCount)).
		Int("connectionEntries", len(rl.connectionRate)).
		Int("concurrentConnections", rl.globalConnections).
		Msg("rate limit cleanup completed")
}

// extractIP extracts the IP address from a remote address string (host:port).
func extractIP(remoteAddr string) string {
	host, _, err := net.SplitHostPort(remoteAddr)
	if err != nil {
		// If parsing fails, return the original address
		return remoteAddr
	}
	return host
}

// CheckConnection checks if a new connection from the given address should be allowed.
// Returns nil if allowed, or an SMTP error if rate limited.
func (rl *RateLimiter) CheckConnection(ctx context.Context, remoteAddr string) error {
	if !rl.config.Enabled {
		return nil
	}

	ip := extractIP(remoteAddr)

	rl.mu.Lock()
	defer rl.mu.Unlock()

	// Check global connection limit
	if rl.config.MaxGlobalConnections > 0 && rl.globalConnections >= rl.config.MaxGlobalConnections {
		rl.logger.Warn().
			Str("ip", ip).
			Int("globalConnections", rl.globalConnections).
			Int("maxGlobalConnections", rl.config.MaxGlobalConnections).
			Msg("global connection limit exceeded")

		return &smtp.SMTPError{
			Code:         421,
			EnhancedCode: smtp.EnhancedCode{4, 7, 0},
			Message:      "service temporarily unavailable, too many connections",
		}
	}

	// Check per-IP concurrent connection limit
	if rl.config.MaxConcurrentConnections > 0 {
		current := rl.concurrentConnections[ip]
		if current >= rl.config.MaxConcurrentConnections {
			rl.logger.Warn().
				Str("ip", ip).
				Int("currentConnections", current).
				Int("maxConcurrentConnections", rl.config.MaxConcurrentConnections).
				Msg("concurrent connection limit exceeded")

			return &smtp.SMTPError{
				Code:         421,
				EnhancedCode: smtp.EnhancedCode{4, 7, 0},
				Message:      fmt.Sprintf("too many concurrent connections from %s", ip),
			}
		}
	}

	// Check connection rate limit
	if rl.config.ConnectionsPerMinute > 0 {
		now := time.Now()
		entry, exists := rl.connectionRate[ip]

		if !exists || now.After(entry.windowEnd) {
			// New window
			rl.connectionRate[ip] = &rateLimitEntry{
				count:     1,
				windowEnd: now.Add(time.Minute),
			}
		} else {
			// Existing window
			if entry.count >= rl.config.ConnectionsPerMinute {
				rl.logger.Warn().
					Str("ip", ip).
					Int("connectionCount", entry.count).
					Int("connectionsPerMinute", rl.config.ConnectionsPerMinute).
					Time("windowEnd", entry.windowEnd).
					Msg("connection rate limit exceeded")

				return &smtp.SMTPError{
					Code:         421,
					EnhancedCode: smtp.EnhancedCode{4, 7, 0},
					Message:      "connection rate limit exceeded, please try again later",
				}
			}
			entry.count++
		}
	}

	return nil
}

// OnConnectionOpened should be called when a new connection is opened.
func (rl *RateLimiter) OnConnectionOpened(remoteAddr string) {
	if !rl.config.Enabled {
		return
	}

	ip := extractIP(remoteAddr)

	rl.mu.Lock()
	defer rl.mu.Unlock()

	rl.globalConnections++
	rl.concurrentConnections[ip]++

	rl.logger.Debug().
		Str("ip", ip).
		Int("ipConnections", rl.concurrentConnections[ip]).
		Int("globalConnections", rl.globalConnections).
		Msg("connection opened")
}

// OnConnectionClosed should be called when a connection is closed.
func (rl *RateLimiter) OnConnectionClosed(remoteAddr string) {
	if !rl.config.Enabled {
		return
	}

	ip := extractIP(remoteAddr)

	rl.mu.Lock()
	defer rl.mu.Unlock()

	if rl.globalConnections > 0 {
		rl.globalConnections--
	}

	if count, exists := rl.concurrentConnections[ip]; exists {
		if count <= 1 {
			delete(rl.concurrentConnections, ip)
		} else {
			rl.concurrentConnections[ip] = count - 1
		}
	}

	// Clean up messages per connection tracking
	delete(rl.messagesPerConnection, remoteAddr)

	rl.logger.Debug().
		Str("ip", ip).
		Int("ipConnections", rl.concurrentConnections[ip]).
		Int("globalConnections", rl.globalConnections).
		Msg("connection closed")
}

// CheckMessage checks if sending a message from the given address should be allowed.
// Returns nil if allowed, or an SMTP error if rate limited.
func (rl *RateLimiter) CheckMessage(ctx context.Context, remoteAddr string) error {
	if !rl.config.Enabled {
		return nil
	}

	ip := extractIP(remoteAddr)

	rl.mu.Lock()
	defer rl.mu.Unlock()

	// Check messages per connection
	if rl.config.MessagesPerConnection > 0 {
		current := rl.messagesPerConnection[remoteAddr]
		if current >= rl.config.MessagesPerConnection {
			rl.logger.Warn().
				Str("ip", ip).
				Str("remoteAddr", remoteAddr).
				Int("messagesInConnection", current).
				Int("messagesPerConnection", rl.config.MessagesPerConnection).
				Msg("messages per connection limit exceeded")

			return &smtp.SMTPError{
				Code:         452,
				EnhancedCode: smtp.EnhancedCode{4, 7, 1},
				Message:      "too many messages in this connection, please reconnect",
			}
		}
	}

	// Check hourly message rate
	if rl.config.MessagesPerHour > 0 {
		now := time.Now()
		entry, exists := rl.messageCount[ip]

		if !exists || now.After(entry.windowEnd) {
			// New window - will be incremented in OnMessageSent
		} else {
			if entry.count >= rl.config.MessagesPerHour {
				rl.logger.Warn().
					Str("ip", ip).
					Int("messageCount", entry.count).
					Int("messagesPerHour", rl.config.MessagesPerHour).
					Time("windowEnd", entry.windowEnd).
					Msg("hourly message rate limit exceeded")

				return &smtp.SMTPError{
					Code:         452,
					EnhancedCode: smtp.EnhancedCode{4, 7, 1},
					Message:      "message rate limit exceeded, please try again later",
				}
			}
		}
	}

	return nil
}

// OnMessageSent should be called when a message is successfully sent.
func (rl *RateLimiter) OnMessageSent(remoteAddr string) {
	if !rl.config.Enabled {
		return
	}

	ip := extractIP(remoteAddr)

	rl.mu.Lock()
	defer rl.mu.Unlock()

	// Increment messages per connection
	rl.messagesPerConnection[remoteAddr]++

	// Increment hourly message count
	now := time.Now()
	entry, exists := rl.messageCount[ip]

	if !exists || now.After(entry.windowEnd) {
		// New window
		rl.messageCount[ip] = &rateLimitEntry{
			count:     1,
			windowEnd: now.Add(time.Hour),
		}
	} else {
		entry.count++
	}

	rl.logger.Debug().
		Str("ip", ip).
		Int("messagesInConnection", rl.messagesPerConnection[remoteAddr]).
		Int("messagesInHour", rl.messageCount[ip].count).
		Msg("message sent")
}

// CheckRecipients checks if the number of recipients is within limits.
// Returns nil if allowed, or an SMTP error if limit exceeded.
func (rl *RateLimiter) CheckRecipients(recipientCount int) error {
	if !rl.config.Enabled {
		return nil
	}

	if rl.config.RecipientsPerMessage > 0 && recipientCount > rl.config.RecipientsPerMessage {
		return &smtp.SMTPError{
			Code:         452,
			EnhancedCode: smtp.EnhancedCode{4, 5, 3},
			Message:      fmt.Sprintf("too many recipients, maximum is %d", rl.config.RecipientsPerMessage),
		}
	}

	return nil
}

// GetStats returns current rate limiter statistics.
func (rl *RateLimiter) GetStats() RateLimitStats {
	rl.mu.RLock()
	defer rl.mu.RUnlock()

	return RateLimitStats{
		GlobalConnections:     rl.globalConnections,
		UniqueIPsWithMessages: len(rl.messageCount),
		UniqueIPsConnected:    len(rl.concurrentConnections),
	}
}

// RateLimitStats holds rate limiter statistics.
type RateLimitStats struct {
	GlobalConnections     int
	UniqueIPsWithMessages int
	UniqueIPsConnected    int
}

// IsEnabled returns whether rate limiting is enabled.
func (rl *RateLimiter) IsEnabled() bool {
	return rl.config.Enabled
}

// Config returns the rate limiter configuration.
func (rl *RateLimiter) Config() *RateLimitConfig {
	return rl.config
}
