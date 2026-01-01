// Package parser provides MIME message parsing functionality for the Yunt mail server.
// It handles parsing of raw email data including headers, bodies, and attachments.
package parser

import (
	"strings"
	"unicode"

	"yunt/internal/domain"
)

// ParseAddress parses a single email address string into an EmailAddress.
// It handles formats like:
// - "user@example.com"
// - "Name <user@example.com>"
// - "<user@example.com>"
// - "\"Name With Quotes\" <user@example.com>"
func ParseAddress(input string) domain.EmailAddress {
	input = strings.TrimSpace(input)
	if input == "" {
		return domain.EmailAddress{}
	}

	// Check for angle bracket format: "Name <address>" or "<address>"
	if idx := strings.LastIndex(input, "<"); idx != -1 {
		endIdx := strings.LastIndex(input, ">")
		if endIdx > idx {
			address := strings.TrimSpace(input[idx+1 : endIdx])
			name := strings.TrimSpace(input[:idx])

			// Remove surrounding quotes from name
			name = unquoteString(name)

			return domain.EmailAddress{
				Name:    name,
				Address: address,
			}
		}
	}

	// No angle brackets, treat the whole thing as address
	return domain.EmailAddress{
		Address: input,
	}
}

// ParseAddressList parses a comma-separated list of email addresses.
// It handles complex cases with quoted strings containing commas.
func ParseAddressList(input string) []domain.EmailAddress {
	input = strings.TrimSpace(input)
	if input == "" {
		return nil
	}

	var addresses []domain.EmailAddress
	var current strings.Builder
	inQuotes := false
	inAngle := false
	escaped := false

	for i := 0; i < len(input); i++ {
		c := input[i]

		if escaped {
			current.WriteByte(c)
			escaped = false
			continue
		}

		switch c {
		case '\\':
			escaped = true
			current.WriteByte(c)
		case '"':
			inQuotes = !inQuotes
			current.WriteByte(c)
		case '<':
			if !inQuotes {
				inAngle = true
			}
			current.WriteByte(c)
		case '>':
			if !inQuotes {
				inAngle = false
			}
			current.WriteByte(c)
		case ',':
			if !inQuotes && !inAngle {
				if addr := ParseAddress(current.String()); !addr.IsEmpty() {
					addresses = append(addresses, addr)
				}
				current.Reset()
			} else {
				current.WriteByte(c)
			}
		default:
			current.WriteByte(c)
		}
	}

	// Don't forget the last address
	if addr := ParseAddress(current.String()); !addr.IsEmpty() {
		addresses = append(addresses, addr)
	}

	return addresses
}

// FormatAddress formats an EmailAddress back to a string representation.
func FormatAddress(addr domain.EmailAddress) string {
	if addr.IsEmpty() {
		return ""
	}
	if addr.Name == "" {
		return addr.Address
	}

	// Check if name needs quoting
	if needsQuoting(addr.Name) {
		return "\"" + escapeString(addr.Name) + "\" <" + addr.Address + ">"
	}
	return addr.Name + " <" + addr.Address + ">"
}

// FormatAddressList formats a list of EmailAddresses to a comma-separated string.
func FormatAddressList(addresses []domain.EmailAddress) string {
	if len(addresses) == 0 {
		return ""
	}

	var parts []string
	for _, addr := range addresses {
		if formatted := FormatAddress(addr); formatted != "" {
			parts = append(parts, formatted)
		}
	}
	return strings.Join(parts, ", ")
}

// unquoteString removes surrounding quotes and handles escape sequences.
func unquoteString(s string) string {
	s = strings.TrimSpace(s)
	if len(s) < 2 {
		return s
	}

	// Remove surrounding double quotes
	if s[0] == '"' && s[len(s)-1] == '"' {
		s = s[1 : len(s)-1]
	}

	// Handle escape sequences
	var result strings.Builder
	escaped := false
	for _, r := range s {
		if escaped {
			result.WriteRune(r)
			escaped = false
			continue
		}
		if r == '\\' {
			escaped = true
			continue
		}
		result.WriteRune(r)
	}
	return result.String()
}

// needsQuoting returns true if the string contains characters that require quoting.
func needsQuoting(s string) bool {
	for _, r := range s {
		// Special characters that require quoting in email addresses
		switch r {
		case '"', '\\', ',', '<', '>', '@', '(', ')', '[', ']', ';', ':':
			return true
		}
		// Non-ASCII or control characters
		if r > 127 || unicode.IsControl(r) {
			return true
		}
	}
	return false
}

// escapeString escapes special characters in a string for use in quoted strings.
func escapeString(s string) string {
	var result strings.Builder
	for _, r := range s {
		switch r {
		case '"', '\\':
			result.WriteByte('\\')
			result.WriteRune(r)
		default:
			result.WriteRune(r)
		}
	}
	return result.String()
}

// ExtractDomain extracts the domain part from an email address.
func ExtractDomain(address string) string {
	if idx := strings.LastIndex(address, "@"); idx != -1 {
		return strings.ToLower(address[idx+1:])
	}
	return ""
}

// ExtractLocalPart extracts the local part (before @) from an email address.
func ExtractLocalPart(address string) string {
	if idx := strings.LastIndex(address, "@"); idx != -1 {
		return address[:idx]
	}
	return address
}

// NormalizeAddress normalizes an email address by lowercasing the domain part.
func NormalizeAddress(address string) string {
	if idx := strings.LastIndex(address, "@"); idx != -1 {
		return address[:idx+1] + strings.ToLower(address[idx+1:])
	}
	return address
}

// IsValidAddress performs basic validation on an email address format.
// Note: This is not a full RFC 5321 validation, just basic format checking.
func IsValidAddress(address string) bool {
	address = strings.TrimSpace(address)
	if address == "" {
		return false
	}

	// Must contain exactly one @
	atIdx := strings.Index(address, "@")
	if atIdx == -1 || atIdx == 0 || atIdx == len(address)-1 {
		return false
	}
	if strings.Count(address, "@") > 1 {
		return false
	}

	local := address[:atIdx]
	domain := address[atIdx+1:]

	// Local part checks
	if local == "" || len(local) > 64 {
		return false
	}

	// Domain part checks
	if domain == "" || len(domain) > 255 {
		return false
	}

	// Domain must have at least one dot
	if !strings.Contains(domain, ".") {
		return false
	}

	// Domain cannot start or end with dot or hyphen
	if domain[0] == '.' || domain[0] == '-' ||
		domain[len(domain)-1] == '.' || domain[len(domain)-1] == '-' {
		return false
	}

	return true
}
