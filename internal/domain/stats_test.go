package domain

import (
	"testing"
)

func TestFormatBytes(t *testing.T) {
	tests := []struct {
		name     string
		bytes    int64
		expected string
	}{
		{
			name:     "zero bytes",
			bytes:    0,
			expected: "0 B",
		},
		{
			name:     "small bytes",
			bytes:    512,
			expected: "512 B",
		},
		{
			name:     "one KB",
			bytes:    1024,
			expected: "1 KB",
		},
		{
			name:     "1.5 KB",
			bytes:    1536,
			expected: "1.5 KB",
		},
		{
			name:     "one MB",
			bytes:    1024 * 1024,
			expected: "1 MB",
		},
		{
			name:     "1.25 MB",
			bytes:    1310720,
			expected: "1.25 MB",
		},
		{
			name:     "one GB",
			bytes:    1024 * 1024 * 1024,
			expected: "1 GB",
		},
		{
			name:     "one TB",
			bytes:    1024 * 1024 * 1024 * 1024,
			expected: "1 TB",
		},
		{
			name:     "large KB value",
			bytes:    512000,
			expected: "500 KB",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FormatBytes(tt.bytes)
			if result != tt.expected {
				t.Errorf("FormatBytes(%d) = %s, want %s", tt.bytes, result, tt.expected)
			}
		})
	}
}

func TestMessageStats(t *testing.T) {
	stats := NewMessageStats()

	// Test empty stats
	if !stats.IsEmpty() {
		t.Error("new stats should be empty")
	}

	if stats.ReadPercentage() != 0 {
		t.Errorf("expected 0%% read, got %f", stats.ReadPercentage())
	}

	// Set some values
	stats.Count = 100
	stats.UnreadCount = 30
	stats.SpamCount = 10
	stats.TotalSize = 1024 * 1024

	if stats.IsEmpty() {
		t.Error("stats with count should not be empty")
	}

	// Test read percentage (70%)
	readPct := stats.ReadPercentage()
	if readPct != 70 {
		t.Errorf("expected 70%% read, got %f", readPct)
	}

	// Test unread percentage (30%)
	unreadPct := stats.UnreadPercentage()
	if unreadPct != 30 {
		t.Errorf("expected 30%% unread, got %f", unreadPct)
	}

	// Test spam percentage (10%)
	spamPct := stats.SpamPercentage()
	if spamPct != 10 {
		t.Errorf("expected 10%% spam, got %f", spamPct)
	}

	// Test average size
	avgSize := stats.AverageSize()
	expectedAvg := float64(1024*1024) / 100
	if avgSize != expectedAvg {
		t.Errorf("expected average size %f, got %f", expectedAvg, avgSize)
	}

	// Test formatted size
	formattedSize := stats.FormatTotalSize()
	if formattedSize != "1 MB" {
		t.Errorf("expected '1 MB', got '%s'", formattedSize)
	}
}

func TestStatsTimeRange(t *testing.T) {
	tests := []struct {
		timeRange StatsTimeRange
		valid     bool
	}{
		{StatsTimeRange24Hours, true},
		{StatsTimeRange7Days, true},
		{StatsTimeRange30Days, true},
		{StatsTimeRange90Days, true},
		{StatsTimeRangeAll, true},
		{StatsTimeRange("invalid"), false},
	}

	for _, tt := range tests {
		t.Run(string(tt.timeRange), func(t *testing.T) {
			if tt.timeRange.IsValid() != tt.valid {
				t.Errorf("%s.IsValid() = %v, want %v", tt.timeRange, tt.timeRange.IsValid(), tt.valid)
			}
		})
	}
}

func TestNewStats(t *testing.T) {
	stats := NewStats()

	if stats == nil {
		t.Fatal("NewStats() should not return nil")
	}

	if stats.GeneratedAt.IsZero() {
		t.Error("GeneratedAt should be set")
	}
}

func TestFormatStatsInt(t *testing.T) {
	tests := []struct {
		input    int64
		expected string
	}{
		{0, "0"},
		{1, "1"},
		{10, "10"},
		{100, "100"},
		{1000, "1000"},
		{-1, "-1"},
		{-100, "-100"},
	}

	for _, tt := range tests {
		result := formatStatsInt(tt.input)
		if result != tt.expected {
			t.Errorf("formatStatsInt(%d) = %s, want %s", tt.input, result, tt.expected)
		}
	}
}

func TestMessageStatsEdgeCases(t *testing.T) {
	stats := &MessageStats{}

	// Test with zero count
	if stats.ReadPercentage() != 0 {
		t.Error("read percentage should be 0 for empty stats")
	}
	if stats.UnreadPercentage() != 0 {
		t.Error("unread percentage should be 0 for empty stats")
	}
	if stats.SpamPercentage() != 0 {
		t.Error("spam percentage should be 0 for empty stats")
	}
	if stats.AverageSize() != 0 {
		t.Error("average size should be 0 for empty stats")
	}
}
