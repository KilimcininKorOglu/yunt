package parser

import (
	"bytes"
	"encoding/base64"
	"io"
	"strings"

	"yunt/internal/domain"
)

// AttachmentData represents the parsed data of an attachment.
type AttachmentData struct {
	// Filename is the name of the attachment file.
	Filename string

	// ContentType is the MIME content type.
	ContentType string

	// ContentID is the Content-ID for inline attachments.
	ContentID string

	// Disposition indicates if this is an attachment or inline.
	Disposition domain.AttachmentDisposition

	// IsInline indicates if this attachment is meant to be displayed inline.
	IsInline bool

	// Data is the decoded binary content of the attachment.
	Data []byte

	// Encoding is the original Content-Transfer-Encoding.
	Encoding string
}

// getHeaderCaseInsensitive looks up a header value case-insensitively.
func getHeaderCaseInsensitive(headers map[string]string, name string) string {
	// Try exact match first
	if v, ok := headers[name]; ok {
		return v
	}
	// Try case-insensitive match
	nameLower := strings.ToLower(name)
	for k, v := range headers {
		if strings.ToLower(k) == nameLower {
			return v
		}
	}
	return ""
}

// extractAttachment extracts attachment information from a MIME part.
func extractAttachment(headers map[string]string, body []byte) *AttachmentData {
	contentType := getHeaderCaseInsensitive(headers, "Content-Type")
	if contentType == "" {
		contentType = "application/octet-stream"
	}

	// Parse content type and parameters
	mediaType, params := parseMediaType(contentType)

	// Get filename from Content-Disposition or Content-Type
	filename := extractFilename(headers, params)
	if filename == "" {
		// Generate a default filename based on content type
		filename = generateDefaultFilename(mediaType)
	}

	// Get Content-ID for inline attachments
	contentID := getHeaderCaseInsensitive(headers, "Content-ID")
	contentID = strings.Trim(contentID, "<>")

	// Determine disposition
	disposition := domain.DispositionAttachment
	isInline := false
	dispHeader := getHeaderCaseInsensitive(headers, "Content-Disposition")
	if dispHeader != "" {
		dispType, _ := parseMediaType(dispHeader)
		if strings.EqualFold(dispType, "inline") {
			disposition = domain.DispositionInline
			isInline = true
		}
	}

	// If Content-ID is present without explicit disposition, treat as inline
	if contentID != "" && dispHeader == "" {
		disposition = domain.DispositionInline
		isInline = true
	}

	// Decode the content
	encoding := strings.ToLower(getHeaderCaseInsensitive(headers, "Content-Transfer-Encoding"))
	decodedData := decodeContent(body, encoding)

	return &AttachmentData{
		Filename:    sanitizeFilename(filename),
		ContentType: mediaType,
		ContentID:   contentID,
		Disposition: disposition,
		IsInline:    isInline,
		Data:        decodedData,
		Encoding:    encoding,
	}
}

// extractFilename extracts the filename from headers.
func extractFilename(headers map[string]string, contentTypeParams map[string]string) string {
	// First try Content-Disposition header
	if dispHeader := getHeaderCaseInsensitive(headers, "Content-Disposition"); dispHeader != "" {
		_, dispParams := parseMediaType(dispHeader)
		if name := dispParams["filename"]; name != "" {
			return decodeHeaderValue(name)
		}
		if name := dispParams["filename*"]; name != "" {
			return decodeEncodedWord(name)
		}
	}

	// Fall back to Content-Type name parameter
	if name := contentTypeParams["name"]; name != "" {
		return decodeHeaderValue(name)
	}

	return ""
}

// generateDefaultFilename generates a default filename based on content type.
func generateDefaultFilename(contentType string) string {
	switch {
	case strings.HasPrefix(contentType, "image/"):
		ext := strings.TrimPrefix(contentType, "image/")
		if ext == "jpeg" {
			ext = "jpg"
		}
		return "image." + ext
	case strings.HasPrefix(contentType, "text/plain"):
		return "document.txt"
	case strings.HasPrefix(contentType, "text/html"):
		return "document.html"
	case contentType == "application/pdf":
		return "document.pdf"
	default:
		return "attachment.bin"
	}
}

// sanitizeFilename removes or replaces problematic characters in filenames.
func sanitizeFilename(filename string) string {
	// Replace path separators and other dangerous characters
	replacer := strings.NewReplacer(
		"/", "_",
		"\\", "_",
		"..", "_",
		"\x00", "",
	)
	filename = replacer.Replace(filename)

	// Remove leading dots (hidden files)
	filename = strings.TrimLeft(filename, ".")

	// Limit length
	if len(filename) > 255 {
		ext := ""
		if idx := strings.LastIndex(filename, "."); idx != -1 && len(filename)-idx <= 10 {
			ext = filename[idx:]
			filename = filename[:idx]
		}
		filename = filename[:255-len(ext)] + ext
	}

	if filename == "" {
		return "attachment"
	}

	return filename
}

// decodeContent decodes content based on the transfer encoding.
func decodeContent(data []byte, encoding string) []byte {
	switch strings.ToLower(encoding) {
	case "base64":
		return decodeBase64(data)
	case "quoted-printable":
		return decodeQuotedPrintable(data)
	default:
		return data
	}
}

// decodeBase64 decodes base64-encoded content.
func decodeBase64(data []byte) []byte {
	// Remove whitespace that may be present in email bodies
	cleaned := make([]byte, 0, len(data))
	for _, b := range data {
		if !isWhitespace(b) {
			cleaned = append(cleaned, b)
		}
	}

	decoded := make([]byte, base64.StdEncoding.DecodedLen(len(cleaned)))
	n, err := base64.StdEncoding.Decode(decoded, cleaned)
	if err != nil {
		// Try with raw encoding (no padding)
		n, err = base64.RawStdEncoding.Decode(decoded, cleaned)
		if err != nil {
			return data // Return original if decoding fails
		}
	}
	return decoded[:n]
}

// decodeQuotedPrintable decodes quoted-printable encoded content.
func decodeQuotedPrintable(data []byte) []byte {
	reader := newQuotedPrintableReader(bytes.NewReader(data))
	decoded, err := io.ReadAll(reader)
	if err != nil {
		return data // Return original if decoding fails
	}
	return decoded
}

// quotedPrintableReader implements a quoted-printable decoder.
type quotedPrintableReader struct {
	r io.Reader
}

func newQuotedPrintableReader(r io.Reader) *quotedPrintableReader {
	return &quotedPrintableReader{r: r}
}

func (q *quotedPrintableReader) Read(p []byte) (int, error) {
	// Read all input first for simplicity
	input, err := io.ReadAll(q.r)
	if err != nil {
		return 0, err
	}

	var result bytes.Buffer
	i := 0
	for i < len(input) {
		b := input[i]
		switch {
		case b == '=':
			// Check for soft line break
			if i+1 < len(input) && (input[i+1] == '\r' || input[i+1] == '\n') {
				i++
				if i+1 < len(input) && input[i] == '\r' && input[i+1] == '\n' {
					i++
				}
				i++
				continue
			}
			// Decode hex pair
			if i+2 < len(input) {
				hi := unhex(input[i+1])
				lo := unhex(input[i+2])
				if hi >= 0 && lo >= 0 {
					result.WriteByte(byte(hi<<4 | lo))
					i += 3
					continue
				}
			}
			// Invalid sequence, keep as is
			result.WriteByte(b)
			i++
		case b == '_':
			// In headers, _ means space (RFC 2047)
			result.WriteByte(' ')
			i++
		default:
			result.WriteByte(b)
			i++
		}
	}

	n := copy(p, result.Bytes())
	return n, io.EOF
}

// unhex returns the value of a hex digit, or -1 if invalid.
func unhex(c byte) int {
	switch {
	case c >= '0' && c <= '9':
		return int(c - '0')
	case c >= 'A' && c <= 'F':
		return int(c - 'A' + 10)
	case c >= 'a' && c <= 'f':
		return int(c - 'a' + 10)
	default:
		return -1
	}
}

// isWhitespace returns true if the byte is a whitespace character.
func isWhitespace(b byte) bool {
	return b == ' ' || b == '\t' || b == '\r' || b == '\n'
}

// isAttachmentContentType returns true if the content type indicates an attachment.
func isAttachmentContentType(contentType string) bool {
	mediaType, _ := parseMediaType(contentType)
	mediaType = strings.ToLower(mediaType)

	// Text/plain and text/html are typically message bodies, not attachments
	// unless explicitly marked with Content-Disposition: attachment
	switch mediaType {
	case "text/plain", "text/html":
		return false
	case "multipart/mixed", "multipart/alternative", "multipart/related":
		return false
	default:
		return true
	}
}
