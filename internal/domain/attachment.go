package domain

import (
	"path"
	"strings"
)

// Attachment represents a file attachment in an email message.
// Attachments are stored separately from messages for efficient storage
// and to support features like inline images and large file handling.
type Attachment struct {
	// ID is the unique identifier for the attachment.
	ID ID `json:"id"`

	// MessageID is the ID of the message this attachment belongs to.
	MessageID ID `json:"messageId"`

	// Filename is the original filename of the attachment.
	Filename string `json:"filename"`

	// ContentType is the MIME type of the attachment (e.g., "application/pdf").
	ContentType string `json:"contentType"`

	// Size is the size of the attachment in bytes.
	Size int64 `json:"size"`

	// ContentID is the Content-ID header for inline attachments.
	// This is used for referencing inline images in HTML emails (cid:xxx).
	ContentID string `json:"contentId,omitempty"`

	// Disposition indicates how the attachment should be presented.
	// "attachment" for regular attachments, "inline" for embedded content.
	Disposition AttachmentDisposition `json:"disposition"`

	// StoragePath is the internal storage path or key for the attachment data.
	// This field is not serialized to JSON for security.
	StoragePath string `json:"-"`

	// Checksum is a hash of the attachment content for integrity verification.
	// Typically MD5 or SHA256.
	Checksum string `json:"checksum,omitempty"`

	// IsInline returns true if this is an inline attachment (like embedded images).
	IsInline bool `json:"isInline"`

	// CreatedAt is the timestamp when the attachment was created.
	CreatedAt Timestamp `json:"createdAt"`
}

// AttachmentDisposition represents how an attachment should be presented.
type AttachmentDisposition string

const (
	// DispositionAttachment indicates a regular downloadable attachment.
	DispositionAttachment AttachmentDisposition = "attachment"
	// DispositionInline indicates an inline/embedded attachment.
	DispositionInline AttachmentDisposition = "inline"
)

// IsValid returns true if the disposition is a recognized value.
func (d AttachmentDisposition) IsValid() bool {
	switch d {
	case DispositionAttachment, DispositionInline:
		return true
	default:
		return false
	}
}

// String returns the string representation of the disposition.
func (d AttachmentDisposition) String() string {
	return string(d)
}

// NewAttachment creates a new Attachment with default values.
func NewAttachment(id, messageID ID, filename, contentType string, size int64) *Attachment {
	return &Attachment{
		ID:          id,
		MessageID:   messageID,
		Filename:    filename,
		ContentType: contentType,
		Size:        size,
		Disposition: DispositionAttachment,
		IsInline:    false,
		CreatedAt:   Now(),
	}
}

// NewInlineAttachment creates a new inline Attachment (for embedded images, etc.).
func NewInlineAttachment(id, messageID ID, filename, contentType, contentID string, size int64) *Attachment {
	return &Attachment{
		ID:          id,
		MessageID:   messageID,
		Filename:    filename,
		ContentType: contentType,
		ContentID:   contentID,
		Size:        size,
		Disposition: DispositionInline,
		IsInline:    true,
		CreatedAt:   Now(),
	}
}

// Validate checks if the attachment has valid field values.
func (a *Attachment) Validate() error {
	errs := NewValidationErrors()

	// Validate ID
	if a.ID.IsEmpty() {
		errs.Add("id", "id is required")
	}

	// Validate MessageID
	if a.MessageID.IsEmpty() {
		errs.Add("messageId", "message id is required")
	}

	// Validate Filename
	if a.Filename == "" {
		errs.Add("filename", "filename is required")
	} else if len(a.Filename) > 255 {
		errs.Add("filename", "filename must be at most 255 characters")
	} else if containsPathTraversal(a.Filename) {
		errs.Add("filename", "filename contains invalid characters")
	}

	// Validate ContentType
	if a.ContentType == "" {
		errs.Add("contentType", "content type is required")
	} else if !isValidMIMEType(a.ContentType) {
		errs.Add("contentType", "content type format is invalid")
	}

	// Validate Size
	if a.Size < 0 {
		errs.Add("size", "size cannot be negative")
	}

	// Validate Disposition
	if !a.Disposition.IsValid() {
		errs.Add("disposition", "invalid disposition")
	}

	// Validate ContentID is set for inline attachments
	if a.IsInline && a.ContentID == "" {
		errs.Add("contentId", "content id is required for inline attachments")
	}

	if errs.HasErrors() {
		return errs
	}
	return nil
}

// GetExtension returns the file extension of the attachment.
func (a *Attachment) GetExtension() string {
	ext := path.Ext(a.Filename)
	if ext != "" {
		return ext[1:] // Remove the leading dot
	}
	return ""
}

// GetBaseFilename returns the filename without the extension.
func (a *Attachment) GetBaseFilename() string {
	ext := path.Ext(a.Filename)
	if ext != "" {
		return a.Filename[:len(a.Filename)-len(ext)]
	}
	return a.Filename
}

// IsImage returns true if the attachment appears to be an image.
func (a *Attachment) IsImage() bool {
	return strings.HasPrefix(a.ContentType, "image/")
}

// IsPDF returns true if the attachment is a PDF.
func (a *Attachment) IsPDF() bool {
	return a.ContentType == "application/pdf"
}

// IsText returns true if the attachment is a text file.
func (a *Attachment) IsText() bool {
	return strings.HasPrefix(a.ContentType, "text/")
}

// IsArchive returns true if the attachment is an archive file.
func (a *Attachment) IsArchive() bool {
	switch a.ContentType {
	case "application/zip",
		"application/x-zip-compressed",
		"application/x-rar-compressed",
		"application/x-7z-compressed",
		"application/x-tar",
		"application/gzip",
		"application/x-gzip":
		return true
	default:
		return false
	}
}

// GetSizeFormatted returns a human-readable file size.
func (a *Attachment) GetSizeFormatted() string {
	return formatFileSize(a.Size)
}

// formatFileSize formats a file size in bytes to a human-readable string.
func formatFileSize(size int64) string {
	const (
		KB = 1024
		MB = KB * 1024
		GB = MB * 1024
	)

	switch {
	case size >= GB:
		return strings.TrimRight(strings.TrimRight(
			strings.Replace(string(rune(size/GB))+".", ".", ",", 1), "0"), ",") +
			" GB"
	case size >= MB:
		mb := float64(size) / float64(MB)
		if mb >= 10 {
			return strings.TrimSuffix(strings.TrimSuffix(
				formatFloat(mb, 0), ".0"), ".") + " MB"
		}
		return formatFloat(mb, 1) + " MB"
	case size >= KB:
		kb := float64(size) / float64(KB)
		if kb >= 10 {
			return strings.TrimSuffix(strings.TrimSuffix(
				formatFloat(kb, 0), ".0"), ".") + " KB"
		}
		return formatFloat(kb, 1) + " KB"
	default:
		return formatInt(size) + " B"
	}
}

// formatFloat formats a float with specified decimal places.
func formatFloat(f float64, decimals int) string {
	format := "%." + formatInt(int64(decimals)) + "f"
	return strings.Replace(
		strings.TrimRight(strings.TrimRight(
			sprintf(format, f), "0"), "."),
		".", ",", 1)
}

// formatInt formats an integer as a string.
func formatInt(i int64) string {
	return sprintf("%d", i)
}

// sprintf is a simple format function to avoid importing fmt in hot paths.
func sprintf(format string, args ...interface{}) string {
	// Simple implementation for our specific use cases
	result := format
	for _, arg := range args {
		switch v := arg.(type) {
		case int:
			result = strings.Replace(result, "%d", intToString(int64(v)), 1)
		case int64:
			result = strings.Replace(result, "%d", intToString(v), 1)
		case float64:
			// Handle float formatting
			if strings.Contains(result, "%.0f") {
				result = strings.Replace(result, "%.0f", intToString(int64(v)), 1)
			} else if strings.Contains(result, "%.1f") {
				result = strings.Replace(result, "%.1f", floatToString(v, 1), 1)
			} else if strings.Contains(result, "%.2f") {
				result = strings.Replace(result, "%.2f", floatToString(v, 2), 1)
			}
		}
	}
	return result
}

// intToString converts an int64 to a string without importing strconv.
func intToString(i int64) string {
	if i == 0 {
		return "0"
	}

	negative := i < 0
	if negative {
		i = -i
	}

	var digits []byte
	for i > 0 {
		digits = append([]byte{byte('0' + i%10)}, digits...)
		i /= 10
	}

	if negative {
		digits = append([]byte{'-'}, digits...)
	}

	return string(digits)
}

// floatToString converts a float64 to a string with specified decimal places.
func floatToString(f float64, decimals int) string {
	// Multiply by 10^decimals and round
	multiplier := 1.0
	for i := 0; i < decimals; i++ {
		multiplier *= 10
	}
	rounded := int64(f*multiplier + 0.5)

	intPart := rounded / int64(multiplier)
	fracPart := rounded % int64(multiplier)

	if decimals == 0 {
		return intToString(intPart)
	}

	fracStr := intToString(fracPart)
	// Pad with leading zeros if needed
	for len(fracStr) < decimals {
		fracStr = "0" + fracStr
	}

	return intToString(intPart) + "." + fracStr
}

// containsPathTraversal checks if a filename contains path traversal sequences.
func containsPathTraversal(filename string) bool {
	return strings.Contains(filename, "..") ||
		strings.Contains(filename, "/") ||
		strings.Contains(filename, "\\") ||
		strings.HasPrefix(filename, ".") ||
		strings.Contains(filename, "\x00")
}

// isValidMIMEType performs basic MIME type validation.
func isValidMIMEType(mimeType string) bool {
	// MIME type format: type/subtype
	parts := strings.Split(mimeType, "/")
	if len(parts) != 2 {
		return false
	}
	// Both parts should be non-empty and contain only valid characters
	for _, part := range parts {
		if part == "" {
			return false
		}
		for _, c := range part {
			if !((c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') ||
				(c >= '0' && c <= '9') || c == '-' || c == '+' || c == '.') {
				return false
			}
		}
	}
	return true
}

// AttachmentSummary represents a lightweight attachment summary for listings.
type AttachmentSummary struct {
	// ID is the unique identifier for the attachment.
	ID ID `json:"id"`

	// Filename is the original filename of the attachment.
	Filename string `json:"filename"`

	// ContentType is the MIME type of the attachment.
	ContentType string `json:"contentType"`

	// Size is the size of the attachment in bytes.
	Size int64 `json:"size"`

	// SizeFormatted is the human-readable file size.
	SizeFormatted string `json:"sizeFormatted"`

	// IsInline indicates if this is an inline attachment.
	IsInline bool `json:"isInline"`
}

// ToSummary converts an Attachment to an AttachmentSummary.
func (a *Attachment) ToSummary() *AttachmentSummary {
	return &AttachmentSummary{
		ID:            a.ID,
		Filename:      a.Filename,
		ContentType:   a.ContentType,
		Size:          a.Size,
		SizeFormatted: a.GetSizeFormatted(),
		IsInline:      a.IsInline,
	}
}

// AttachmentFilter represents filtering options for listing attachments.
type AttachmentFilter struct {
	// MessageID filters by message ID.
	MessageID *ID `json:"messageId,omitempty"`

	// IsInline filters by inline status.
	IsInline *bool `json:"isInline,omitempty"`

	// ContentType filters by content type (prefix match).
	ContentType string `json:"contentType,omitempty"`

	// MinSize filters attachments larger than this size in bytes.
	MinSize *int64 `json:"minSize,omitempty"`

	// MaxSize filters attachments smaller than this size in bytes.
	MaxSize *int64 `json:"maxSize,omitempty"`
}
