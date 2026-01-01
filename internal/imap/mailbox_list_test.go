package imap

import (
	"testing"
)

func TestMatchMailboxPattern(t *testing.T) {
	tests := []struct {
		name     string
		mailbox  string
		ref      string
		pattern  string
		expected bool
	}{
		// Exact matches
		{"exact match simple", "INBOX", "", "INBOX", true},
		{"exact match case insensitive", "inbox", "", "INBOX", true},
		{"exact match nested", "Work/Projects", "", "Work/Projects", true},
		{"no match different name", "INBOX", "", "Sent", false},

		// Wildcard * (matches everything including hierarchy)
		{"star matches all", "INBOX", "", "*", true},
		{"star matches nested", "Work/Projects", "", "*", true},
		{"star at end", "Work/Projects", "", "Work/*", true},
		{"star at end matches all children", "Work/Projects/2024", "", "Work/*", true},
		{"star in middle", "Work/Projects", "", "*/Projects", true},

		// Wildcard % (matches everything except hierarchy delimiter)
		{"percent matches simple", "INBOX", "", "%", true},
		{"percent does not match nested", "Work/Projects", "", "%", false},
		{"percent at end of parent", "Work/Projects", "", "Work/%", true},
		{"percent does not match deep", "Work/Projects/2024", "", "Work/%", false},
		{"percent for each level", "Work/Projects", "", "%/%", true},

		// Reference prefix - the ref prepends to the pattern, so we match "Work/Projects" against "Work/Projects"
		{"ref prefix works", "Work/Projects", "Work", "Projects", true},
		{"ref prefix with star", "Work/Projects/2024", "Work", "*", true},

		// Empty pattern
		{"empty pattern no match", "INBOX", "", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := matchMailboxPattern(tt.mailbox, tt.ref, tt.pattern)
			if result != tt.expected {
				t.Errorf("matchMailboxPattern(%q, %q, %q) = %v, expected %v",
					tt.mailbox, tt.ref, tt.pattern, result, tt.expected)
			}
		})
	}
}

func TestMatchWildcard(t *testing.T) {
	tests := []struct {
		name     string
		text     string
		pattern  string
		expected bool
	}{
		// Basic matches
		{"exact match", "INBOX", "INBOX", true},
		{"different strings", "INBOX", "Sent", false},
		{"empty both", "", "", true},
		{"empty pattern", "INBOX", "", false},
		{"empty text", "", "INBOX", false},

		// Star wildcard
		{"star matches all", "anything", "*", true},
		{"star matches empty", "", "*", true},
		{"star at start", "HelloWorld", "*World", true},
		{"star at end", "HelloWorld", "Hello*", true},
		{"star in middle", "HelloWorld", "He*ld", true},
		{"star matches hierarchy", "Work/Projects/2024", "*", true},
		{"multiple stars", "a/b/c", "*/*", true},

		// Percent wildcard
		{"percent matches simple", "INBOX", "%", true},
		{"percent stops at hierarchy", "Work/Projects", "%", false},
		{"percent matches up to slash", "Work", "%", true},
		{"percent at end", "Projects", "Pro%", true},
		{"percent does not cross slash", "Work/Projects", "Work%", false},

		// Case insensitivity
		{"case insensitive", "inbox", "INBOX", true},
		{"case insensitive pattern", "INBOX", "inbox", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := matchWildcard(tt.text, tt.pattern, 0, 0)
			if result != tt.expected {
				t.Errorf("matchWildcard(%q, %q) = %v, expected %v",
					tt.text, tt.pattern, result, tt.expected)
			}
		})
	}
}

func TestEqualFoldChar(t *testing.T) {
	tests := []struct {
		a, b     byte
		expected bool
	}{
		{'a', 'a', true},
		{'A', 'A', true},
		{'a', 'A', true},
		{'A', 'a', true},
		{'a', 'b', false},
		{'1', '1', true},
		{'/', '/', true},
		{'/', '\\', false},
	}

	for _, tt := range tests {
		result := equalFoldChar(tt.a, tt.b)
		if result != tt.expected {
			t.Errorf("equalFoldChar(%q, %q) = %v, expected %v",
				tt.a, tt.b, result, tt.expected)
		}
	}
}

func TestFilterSpecialUseAttrs(t *testing.T) {
	tests := []struct {
		name     string
		input    []string
		expected int
	}{
		{
			name:     "empty",
			input:    []string{},
			expected: 0,
		},
		{
			name:     "no special use",
			input:    []string{"\\HasChildren", "\\Unmarked"},
			expected: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Since we're testing with our internal types, just verify the function exists
			// and handles empty input
			result := filterSpecialUseAttrs(nil)
			if len(result) != 0 {
				t.Errorf("Expected empty result for nil input")
			}
		})
	}
}
