package parser

import (
	"bytes"
	"strings"
	"time"

	"yunt/internal/domain"
)

// ParsedMessage represents a fully parsed email message.
type ParsedMessage struct {
	// Headers contains all message headers.
	Headers map[string]string

	// From is the sender address.
	From domain.EmailAddress

	// To is the list of primary recipients.
	To []domain.EmailAddress

	// Cc is the list of CC recipients.
	Cc []domain.EmailAddress

	// Bcc is the list of BCC recipients.
	Bcc []domain.EmailAddress

	// ReplyTo is the reply-to address if set.
	ReplyTo *domain.EmailAddress

	// Subject is the email subject.
	Subject string

	// MessageID is the Message-ID header value.
	MessageID string

	// InReplyTo is the In-Reply-To header value.
	InReplyTo string

	// References contains message IDs from the References header.
	References []string

	// Date is the parsed Date header.
	Date *time.Time

	// TextBody is the plain text content.
	TextBody string

	// HTMLBody is the HTML content.
	HTMLBody string

	// Attachments contains all attachments including inline ones.
	Attachments []*AttachmentData

	// ContentType is the top-level content type.
	ContentType string

	// RawSize is the size of the raw message in bytes.
	RawSize int64
}

// Parser provides MIME message parsing functionality.
type Parser struct {
	// MaxAttachmentSize is the maximum allowed attachment size in bytes.
	// Zero means no limit.
	MaxAttachmentSize int64

	// MaxMessageSize is the maximum allowed message size in bytes.
	// Zero means no limit.
	MaxMessageSize int64

	// StrictMode enables strict parsing that may reject malformed messages.
	StrictMode bool
}

// NewParser creates a new Parser with default settings.
func NewParser() *Parser {
	return &Parser{
		MaxAttachmentSize: 0,
		MaxMessageSize:    0,
		StrictMode:        false,
	}
}

// Parse parses a raw email message and returns a ParsedMessage.
func (p *Parser) Parse(data []byte) (*ParsedMessage, error) {
	if len(data) == 0 {
		return nil, NewParseError("empty message data")
	}

	if p.MaxMessageSize > 0 && int64(len(data)) > p.MaxMessageSize {
		return nil, NewParseError("message exceeds maximum size")
	}

	msg := &ParsedMessage{
		Headers:     make(map[string]string),
		To:          make([]domain.EmailAddress, 0),
		Cc:          make([]domain.EmailAddress, 0),
		Bcc:         make([]domain.EmailAddress, 0),
		References:  make([]string, 0),
		Attachments: make([]*AttachmentData, 0),
		RawSize:     int64(len(data)),
	}

	// Split headers and body
	headers, body := splitHeadersBody(data)

	// Parse headers
	p.parseHeaders(msg, headers)

	// Parse body based on content type
	if err := p.parseBody(msg, body); err != nil {
		if p.StrictMode {
			return nil, err
		}
		// In non-strict mode, try to recover
		msg.TextBody = string(body)
	}

	return msg, nil
}

// parseHeaders extracts and parses all headers from the message.
func (p *Parser) parseHeaders(msg *ParsedMessage, headers []byte) {
	headerMap := parseHeaderBlock(headers)

	// Store all headers
	for name, value := range headerMap {
		msg.Headers[name] = value
	}

	// Helper function for case-insensitive header lookup
	getHeader := func(name string) string {
		// Try exact match first
		if v, ok := headerMap[name]; ok {
			return v
		}
		// Try case-insensitive match
		nameLower := strings.ToLower(name)
		for k, v := range headerMap {
			if strings.ToLower(k) == nameLower {
				return v
			}
		}
		return ""
	}

	// Parse specific headers
	msg.From = ParseAddress(getHeader("From"))
	msg.To = ParseAddressList(getHeader("To"))
	msg.Cc = ParseAddressList(getHeader("Cc"))
	msg.Bcc = ParseAddressList(getHeader("Bcc"))

	if replyTo := getHeader("Reply-To"); replyTo != "" {
		addr := ParseAddress(replyTo)
		if !addr.IsEmpty() {
			msg.ReplyTo = &addr
		}
	}

	msg.Subject = decodeHeaderValue(getHeader("Subject"))
	msg.MessageID = extractMessageID(getHeader("Message-ID"))
	msg.InReplyTo = extractMessageID(getHeader("In-Reply-To"))
	msg.References = extractMessageIDs(getHeader("References"))
	msg.ContentType = getHeader("Content-Type")

	// Parse date
	if dateStr := getHeader("Date"); dateStr != "" {
		if t, err := parseDate(dateStr); err == nil {
			msg.Date = &t
		}
	}
}

// parseBody parses the message body based on content type.
func (p *Parser) parseBody(msg *ParsedMessage, body []byte) error {
	contentType := msg.Headers["Content-Type"]
	if contentType == "" {
		contentType = "text/plain; charset=us-ascii"
	}

	mediaType, params := parseMediaType(contentType)
	mediaType = strings.ToLower(mediaType)

	switch {
	case strings.HasPrefix(mediaType, "multipart/"):
		boundary := params["boundary"]
		if boundary == "" {
			return NewParseError("multipart message missing boundary")
		}
		return p.parseMultipart(msg, body, boundary, mediaType)

	case mediaType == "text/plain":
		encoding := msg.Headers["Content-Transfer-Encoding"]
		msg.TextBody = decodeTextBody(body, encoding, params["charset"])
		return nil

	case mediaType == "text/html":
		encoding := msg.Headers["Content-Transfer-Encoding"]
		msg.HTMLBody = decodeTextBody(body, encoding, params["charset"])
		return nil

	default:
		// Non-text content at top level is treated as attachment
		attachment := extractAttachment(msg.Headers, body)
		if attachment != nil {
			msg.Attachments = append(msg.Attachments, attachment)
		}
		return nil
	}
}

// parseMultipart parses a multipart message.
func (p *Parser) parseMultipart(msg *ParsedMessage, body []byte, boundary, multipartType string) error {
	parts := splitMultipart(body, boundary)

	for _, part := range parts {
		if err := p.parsePart(msg, part, multipartType); err != nil {
			if p.StrictMode {
				return err
			}
			// Continue processing other parts in non-strict mode
		}
	}

	return nil
}

// parsePart parses a single MIME part.
func (p *Parser) parsePart(msg *ParsedMessage, part []byte, parentType string) error {
	headers, body := splitHeadersBody(part)
	headerMap := parseHeaderBlock(headers)

	contentType := headerMap["Content-Type"]
	if contentType == "" {
		contentType = "text/plain"
	}

	mediaType, params := parseMediaType(contentType)
	mediaType = strings.ToLower(mediaType)

	// Check for nested multipart
	if strings.HasPrefix(mediaType, "multipart/") {
		boundary := params["boundary"]
		if boundary == "" {
			return NewParseError("nested multipart missing boundary")
		}
		return p.parseMultipart(msg, body, boundary, mediaType)
	}

	// Determine if this part is an attachment
	disposition := strings.ToLower(headerMap["Content-Disposition"])
	isAttachment := strings.HasPrefix(disposition, "attachment")
	isInline := strings.HasPrefix(disposition, "inline")
	hasContentID := headerMap["Content-ID"] != ""

	// For multipart/alternative, we want text and html versions
	if parentType == "multipart/alternative" {
		encoding := headerMap["Content-Transfer-Encoding"]
		charset := params["charset"]

		switch mediaType {
		case "text/plain":
			if msg.TextBody == "" {
				msg.TextBody = decodeTextBody(body, encoding, charset)
			}
			return nil
		case "text/html":
			if msg.HTMLBody == "" {
				msg.HTMLBody = decodeTextBody(body, encoding, charset)
			}
			return nil
		}
	}

	// For multipart/related, track inline attachments but also extract text content
	if parentType == "multipart/related" {
		if mediaType == "text/html" && msg.HTMLBody == "" {
			encoding := headerMap["Content-Transfer-Encoding"]
			charset := params["charset"]
			msg.HTMLBody = decodeTextBody(body, encoding, charset)
			return nil
		}
	}

	// Handle text content
	if !isAttachment && !isInline && !hasContentID {
		encoding := headerMap["Content-Transfer-Encoding"]
		charset := params["charset"]

		switch mediaType {
		case "text/plain":
			if msg.TextBody == "" {
				msg.TextBody = decodeTextBody(body, encoding, charset)
			}
			return nil
		case "text/html":
			if msg.HTMLBody == "" {
				msg.HTMLBody = decodeTextBody(body, encoding, charset)
			}
			return nil
		}
	}

	// Handle attachments and inline content
	if isAttachment || isInline || hasContentID || isAttachmentContentType(contentType) {
		attachment := extractAttachment(headerMap, body)
		if attachment != nil {
			// Check attachment size limit
			if p.MaxAttachmentSize > 0 && int64(len(attachment.Data)) > p.MaxAttachmentSize {
				if p.StrictMode {
					return NewParseError("attachment exceeds maximum size")
				}
				// Skip oversized attachment in non-strict mode
				return nil
			}
			msg.Attachments = append(msg.Attachments, attachment)
		}
	}

	return nil
}

// splitHeadersBody splits message data into headers and body.
func splitHeadersBody(data []byte) (headers, body []byte) {
	// Find the blank line separating headers from body
	// Headers end with \r\n\r\n or \n\n
	if idx := bytes.Index(data, []byte("\r\n\r\n")); idx != -1 {
		return data[:idx], data[idx+4:]
	}
	if idx := bytes.Index(data, []byte("\n\n")); idx != -1 {
		return data[:idx], data[idx+2:]
	}
	// No body, all headers
	return data, nil
}

// parseHeaderBlock parses a header block into a map.
func parseHeaderBlock(headers []byte) map[string]string {
	result := make(map[string]string)
	if len(headers) == 0 {
		return result
	}

	// Normalize line endings
	normalized := bytes.ReplaceAll(headers, []byte("\r\n"), []byte("\n"))

	// Unfold headers (continuation lines start with whitespace)
	lines := bytes.Split(normalized, []byte("\n"))
	var currentHeader strings.Builder
	var currentName string

	for _, line := range lines {
		if len(line) == 0 {
			continue
		}

		// Check if this is a continuation line
		if (line[0] == ' ' || line[0] == '\t') && currentName != "" {
			// Continuation line - append to current header
			currentHeader.WriteByte(' ')
			currentHeader.WriteString(strings.TrimSpace(string(line)))
			continue
		}

		// Save previous header
		if currentName != "" {
			result[currentName] = currentHeader.String()
		}

		// Parse new header
		colonIdx := bytes.IndexByte(line, ':')
		if colonIdx == -1 {
			continue // Invalid header line
		}

		currentName = canonicalHeaderKey(string(line[:colonIdx]))
		currentHeader.Reset()
		currentHeader.WriteString(strings.TrimSpace(string(line[colonIdx+1:])))
	}

	// Save last header
	if currentName != "" {
		result[currentName] = currentHeader.String()
	}

	return result
}

// canonicalHeaderKey returns the canonical form of a header key.
func canonicalHeaderKey(key string) string {
	key = strings.TrimSpace(key)
	// Simple canonicalization: capitalize first letter and after hyphens
	var result strings.Builder
	capitalizeNext := true

	for _, r := range key {
		if r == '-' {
			result.WriteByte('-')
			capitalizeNext = true
		} else if capitalizeNext {
			result.WriteString(strings.ToUpper(string(r)))
			capitalizeNext = false
		} else {
			result.WriteString(strings.ToLower(string(r)))
		}
	}

	return result.String()
}

// splitMultipart splits a multipart body into its parts.
func splitMultipart(body []byte, boundary string) [][]byte {
	delimiter := "--" + boundary
	closeDelimiter := "--" + boundary + "--"

	// Normalize line endings
	body = bytes.ReplaceAll(body, []byte("\r\n"), []byte("\n"))

	var parts [][]byte
	lines := bytes.Split(body, []byte("\n"))
	var currentPart bytes.Buffer
	inPart := false

	for i := 0; i < len(lines); i++ {
		line := lines[i]
		lineStr := string(bytes.TrimRight(line, " \t"))

		// Check for close delimiter
		if lineStr == closeDelimiter {
			if inPart && currentPart.Len() > 0 {
				content := currentPart.Bytes()
				// Remove trailing newline if present
				if len(content) > 0 && content[len(content)-1] == '\n' {
					content = content[:len(content)-1]
				}
				partCopy := make([]byte, len(content))
				copy(partCopy, content)
				parts = append(parts, partCopy)
			}
			break
		}

		// Check for boundary delimiter
		if lineStr == delimiter {
			if inPart && currentPart.Len() > 0 {
				content := currentPart.Bytes()
				// Remove trailing newline if present
				if len(content) > 0 && content[len(content)-1] == '\n' {
					content = content[:len(content)-1]
				}
				partCopy := make([]byte, len(content))
				copy(partCopy, content)
				parts = append(parts, partCopy)
			}
			currentPart.Reset()
			inPart = true
			continue
		}

		if inPart {
			currentPart.Write(line)
			currentPart.WriteByte('\n')
		}
	}

	return parts
}

// parseMediaType parses a Content-Type or Content-Disposition header value.
func parseMediaType(value string) (mediaType string, params map[string]string) {
	params = make(map[string]string)

	// Split media type from parameters
	parts := strings.Split(value, ";")
	if len(parts) == 0 {
		return "", params
	}

	mediaType = strings.TrimSpace(parts[0])

	// Parse parameters
	for i := 1; i < len(parts); i++ {
		param := strings.TrimSpace(parts[i])
		if param == "" {
			continue
		}

		eqIdx := strings.Index(param, "=")
		if eqIdx == -1 {
			continue
		}

		name := strings.TrimSpace(param[:eqIdx])
		value := strings.TrimSpace(param[eqIdx+1:])

		// Remove surrounding quotes
		if len(value) >= 2 && value[0] == '"' && value[len(value)-1] == '"' {
			value = value[1 : len(value)-1]
		}

		params[strings.ToLower(name)] = value
	}

	return mediaType, params
}

// decodeTextBody decodes a text body with the given encoding and charset.
func decodeTextBody(body []byte, encoding, charset string) string {
	// First decode transfer encoding
	decoded := decodeContent(body, encoding)

	// Then convert charset if needed
	text := convertCharset(decoded, charset)

	// Normalize line endings
	text = strings.ReplaceAll(text, "\r\n", "\n")

	return strings.TrimSpace(text)
}

// convertCharset converts text from a given charset to UTF-8.
// For simplicity, we handle common charsets.
func convertCharset(data []byte, charset string) string {
	charset = strings.ToLower(charset)

	switch charset {
	case "", "us-ascii", "utf-8", "utf8":
		return string(data)
	case "iso-8859-1", "latin1", "windows-1252":
		// ISO-8859-1 is a subset of Unicode, so bytes map directly
		runes := make([]rune, len(data))
		for i, b := range data {
			runes[i] = rune(b)
		}
		return string(runes)
	default:
		// Unknown charset, try as UTF-8
		return string(data)
	}
}

// decodeHeaderValue decodes encoded words in header values (RFC 2047).
func decodeHeaderValue(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}

	// Handle RFC 2047 encoded words: =?charset?encoding?text?=
	var result strings.Builder
	i := 0

	for i < len(value) {
		// Look for encoded word
		if i+2 < len(value) && value[i:i+2] == "=?" {
			endIdx := strings.Index(value[i+2:], "?=")
			if endIdx != -1 {
				encodedWord := value[i : i+2+endIdx+2]
				decoded := decodeEncodedWord(encodedWord)
				result.WriteString(decoded)
				i += len(encodedWord)
				// Skip whitespace between encoded words
				for i < len(value) && (value[i] == ' ' || value[i] == '\t') {
					if i+2 < len(value) && value[i+1:i+3] == "=?" {
						i++
						break
					}
					result.WriteByte(value[i])
					i++
				}
				continue
			}
		}
		result.WriteByte(value[i])
		i++
	}

	return result.String()
}

// decodeEncodedWord decodes an RFC 2047 encoded word.
func decodeEncodedWord(word string) string {
	// Format: =?charset?encoding?encoded_text?=
	if !strings.HasPrefix(word, "=?") || !strings.HasSuffix(word, "?=") {
		return word
	}

	content := word[2 : len(word)-2]
	parts := strings.SplitN(content, "?", 3)
	if len(parts) != 3 {
		return word
	}

	charset := parts[0]
	encoding := strings.ToUpper(parts[1])
	encodedText := parts[2]

	var decoded []byte
	switch encoding {
	case "B":
		decoded = decodeBase64([]byte(encodedText))
	case "Q":
		decoded = decodeQuotedPrintable([]byte(encodedText))
	default:
		return word
	}

	return convertCharset(decoded, charset)
}

// extractMessageID extracts a message ID from angle brackets.
func extractMessageID(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	// Remove angle brackets if present
	value = strings.Trim(value, "<>")
	return value
}

// extractMessageIDs extracts multiple message IDs from a References header.
func extractMessageIDs(value string) []string {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}

	var ids []string
	// Message IDs are in angle brackets, separated by whitespace
	var current strings.Builder
	inBrackets := false

	for _, r := range value {
		switch r {
		case '<':
			inBrackets = true
		case '>':
			if inBrackets {
				if id := strings.TrimSpace(current.String()); id != "" {
					ids = append(ids, id)
				}
				current.Reset()
			}
			inBrackets = false
		default:
			if inBrackets {
				current.WriteRune(r)
			}
		}
	}

	return ids
}

// parseDate parses common date formats found in email headers.
func parseDate(value string) (time.Time, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return time.Time{}, NewParseError("empty date string")
	}

	// Clean up common issues in date strings
	// Remove day name prefix if present (e.g., "Mon, ")
	if idx := strings.Index(value, ", "); idx != -1 && idx < 5 {
		value = strings.TrimSpace(value[idx+2:])
	}

	// Common email date formats
	formats := []string{
		time.RFC1123Z,                      // "Mon, 02 Jan 2006 15:04:05 -0700"
		time.RFC1123,                       // "Mon, 02 Jan 2006 15:04:05 MST"
		"2 Jan 2006 15:04:05 -0700",        // Without day name
		"02 Jan 2006 15:04:05 -0700",       // With leading zero
		"2 Jan 2006 15:04:05 MST",          // With timezone name
		"02 Jan 2006 15:04:05 MST",         // With leading zero and timezone name
		"2006-01-02T15:04:05-07:00",        // ISO 8601
		"2006-01-02 15:04:05",              // Simple format
		"Mon, 2 Jan 2006 15:04:05 -0700",   // RFC 2822 variant
		"Mon, 02 Jan 2006 15:04:05 -0700",  // RFC 2822 variant with leading zero
		"2 Jan 06 15:04:05 -0700",          // Two-digit year
		"02 Jan 06 15:04:05 -0700",         // Two-digit year with leading zero
		"Jan 2, 2006 3:04:05 PM",           // US format
		"January 2, 2006 3:04:05 PM",       // US format with full month
		"2 Jan 2006 15:04:05 -0700 (MST)",  // With timezone in parentheses
		"02 Jan 2006 15:04:05 -0700 (MST)", // With leading zero
	}

	// Remove comments in parentheses (e.g., timezone names)
	if idx := strings.Index(value, "("); idx != -1 {
		endIdx := strings.Index(value, ")")
		if endIdx > idx {
			value = strings.TrimSpace(value[:idx]) + strings.TrimSpace(value[endIdx+1:])
		}
	}

	for _, format := range formats {
		if t, err := time.Parse(format, value); err == nil {
			return t, nil
		}
	}

	return time.Time{}, NewParseError("unable to parse date: " + value)
}

// ParseError represents a MIME parsing error.
type ParseError struct {
	Message string
}

// NewParseError creates a new ParseError.
func NewParseError(message string) *ParseError {
	return &ParseError{Message: message}
}

// Error implements the error interface.
func (e *ParseError) Error() string {
	return "mime parse error: " + e.Message
}

// ToMessage converts a ParsedMessage to a domain.Message.
func (pm *ParsedMessage) ToMessage(id, mailboxID domain.ID) *domain.Message {
	msg := domain.NewMessage(id, mailboxID)

	msg.MessageID = pm.MessageID
	msg.From = pm.From
	msg.To = pm.To
	msg.Cc = pm.Cc
	msg.Bcc = pm.Bcc
	msg.ReplyTo = pm.ReplyTo
	msg.Subject = pm.Subject
	msg.TextBody = pm.TextBody
	msg.HTMLBody = pm.HTMLBody
	msg.Headers = pm.Headers
	msg.InReplyTo = pm.InReplyTo
	msg.References = pm.References
	msg.Size = pm.RawSize
	msg.AttachmentCount = len(pm.Attachments)

	// Set content type
	if pm.HTMLBody != "" && pm.TextBody != "" {
		msg.ContentType = domain.ContentTypeMultipart
	} else if pm.HTMLBody != "" {
		msg.ContentType = domain.ContentTypeHTML
	} else {
		msg.ContentType = domain.ContentTypePlain
	}

	// Set sent date
	if pm.Date != nil {
		ts := domain.Timestamp{Time: *pm.Date}
		msg.SentAt = &ts
	}

	return msg
}

// GetInlineAttachments returns only inline attachments.
func (pm *ParsedMessage) GetInlineAttachments() []*AttachmentData {
	var inline []*AttachmentData
	for _, att := range pm.Attachments {
		if att.IsInline {
			inline = append(inline, att)
		}
	}
	return inline
}

// GetRegularAttachments returns only regular (non-inline) attachments.
func (pm *ParsedMessage) GetRegularAttachments() []*AttachmentData {
	var regular []*AttachmentData
	for _, att := range pm.Attachments {
		if !att.IsInline {
			regular = append(regular, att)
		}
	}
	return regular
}
