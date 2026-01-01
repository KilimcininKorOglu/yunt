package imap

import (
	"bytes"
	"context"
	"strings"

	"github.com/emersion/go-imap/v2"

	"yunt/internal/domain"
	"yunt/internal/repository"
)

// BodyStructureBuilder builds IMAP BODYSTRUCTURE responses.
type BodyStructureBuilder struct{}

// NewBodyStructureBuilder creates a new BodyStructureBuilder.
func NewBodyStructureBuilder() *BodyStructureBuilder {
	return &BodyStructureBuilder{}
}

// Build constructs a BodyStructure from a domain.Message.
func (b *BodyStructureBuilder) Build(ctx context.Context, repo repository.Repository, msg *domain.Message, extended bool) (imap.BodyStructure, error) {
	// Try to get raw body for accurate structure parsing
	rawBody, err := repo.Messages().GetRawBody(ctx, msg.ID)
	if err == nil && len(rawBody) > 0 {
		return b.buildFromRaw(rawBody, msg, extended)
	}

	// Fall back to building from message fields
	return b.buildFromMessage(ctx, repo, msg, extended)
}

// buildFromRaw builds BodyStructure by parsing the raw message.
func (b *BodyStructureBuilder) buildFromRaw(rawBody []byte, msg *domain.Message, extended bool) (imap.BodyStructure, error) {
	// Extract Content-Type header
	contentType := b.extractContentType(rawBody)

	if strings.HasPrefix(contentType, "multipart/") {
		return b.buildMultipartFromRaw(rawBody, contentType, extended)
	}

	return b.buildSinglePartFromRaw(rawBody, contentType, msg, extended)
}

// extractContentType extracts the Content-Type header from raw message.
func (b *BodyStructureBuilder) extractContentType(rawBody []byte) string {
	headerEnd := bytes.Index(rawBody, []byte("\r\n\r\n"))
	if headerEnd == -1 {
		headerEnd = bytes.Index(rawBody, []byte("\n\n"))
		if headerEnd == -1 {
			return "text/plain"
		}
	}

	headers := rawBody[:headerEnd]
	headersLower := bytes.ToLower(headers)

	ctStart := bytes.Index(headersLower, []byte("content-type:"))
	if ctStart == -1 {
		return "text/plain"
	}

	ctLine := headers[ctStart+13:]
	ctEnd := bytes.Index(ctLine, []byte("\n"))
	if ctEnd != -1 {
		ctLine = ctLine[:ctEnd]
	}

	ctLine = bytes.TrimSpace(ctLine)
	ctLine = bytes.TrimSuffix(ctLine, []byte("\r"))

	// Extract just the media type (before parameters)
	if semiIdx := bytes.Index(ctLine, []byte(";")); semiIdx != -1 {
		return string(bytes.TrimSpace(ctLine[:semiIdx]))
	}

	return string(ctLine)
}

// buildMultipartFromRaw builds a multipart BodyStructure from raw message.
func (b *BodyStructureBuilder) buildMultipartFromRaw(rawBody []byte, contentType string, extended bool) (imap.BodyStructure, error) {
	boundary := b.extractBoundary(rawBody)
	if boundary == "" {
		// Fallback to single part if no boundary found
		return b.buildSinglePartFromRaw(rawBody, contentType, nil, extended)
	}

	parts := b.splitByBoundary(rawBody, boundary)

	// Get subtype (e.g., "mixed" from "multipart/mixed")
	subtype := "mixed"
	if idx := strings.Index(contentType, "/"); idx != -1 {
		subtype = strings.ToLower(strings.TrimSpace(contentType[idx+1:]))
	}

	multipart := &imap.BodyStructureMultiPart{
		Subtype:  strings.ToUpper(subtype),
		Children: make([]imap.BodyStructure, 0, len(parts)),
	}

	for _, part := range parts {
		partCT := b.extractContentType(part)
		var child imap.BodyStructure
		var err error

		if strings.HasPrefix(partCT, "multipart/") {
			child, err = b.buildMultipartFromRaw(part, partCT, extended)
		} else {
			child, err = b.buildSinglePartFromRaw(part, partCT, nil, extended)
		}

		if err != nil {
			continue
		}
		multipart.Children = append(multipart.Children, child)
	}

	if extended {
		multipart.Extended = &imap.BodyStructureMultiPartExt{
			Params: b.extractContentTypeParams(rawBody),
		}
	}

	return multipart, nil
}

// buildSinglePartFromRaw builds a single-part BodyStructure from raw data.
func (b *BodyStructureBuilder) buildSinglePartFromRaw(rawBody []byte, contentType string, msg *domain.Message, extended bool) (imap.BodyStructure, error) {
	// Parse content type
	mediaType := strings.ToLower(contentType)
	typeStr := "text"
	subtype := "plain"

	if idx := strings.Index(mediaType, "/"); idx != -1 {
		typeStr = strings.TrimSpace(mediaType[:idx])
		subtype = strings.TrimSpace(mediaType[idx+1:])
	}

	// Extract body for size calculation
	body := b.extractBody(rawBody)

	singlePart := &imap.BodyStructureSinglePart{
		Type:     strings.ToUpper(typeStr),
		Subtype:  strings.ToUpper(subtype),
		Params:   b.extractContentTypeParams(rawBody),
		ID:       b.extractHeaderValue(rawBody, "Content-ID"),
		Encoding: strings.ToUpper(b.extractHeaderValue(rawBody, "Content-Transfer-Encoding")),
		Size:     uint32(len(body)),
	}

	if singlePart.Encoding == "" {
		singlePart.Encoding = "7BIT"
	}

	// Add text-specific fields
	if strings.ToLower(typeStr) == "text" {
		lines := b.countLines(body)
		singlePart.Text = &imap.BodyStructureText{
			NumLines: int64(lines),
		}
	}

	if extended {
		singlePart.Extended = &imap.BodyStructureSinglePartExt{
			Disposition: b.extractDisposition(rawBody),
		}
	}

	return singlePart, nil
}

// buildFromMessage builds BodyStructure from domain.Message fields.
func (b *BodyStructureBuilder) buildFromMessage(ctx context.Context, repo repository.Repository, msg *domain.Message, extended bool) (imap.BodyStructure, error) {
	// Check if we need multipart structure
	hasAttachments := msg.AttachmentCount > 0
	hasBothBodies := msg.HTMLBody != "" && msg.TextBody != ""

	if hasAttachments || hasBothBodies {
		return b.buildMultipartFromMessage(ctx, repo, msg, extended)
	}

	return b.buildSinglePartFromMessage(msg, extended)
}

// buildSinglePartFromMessage builds a single-part structure from message fields.
func (b *BodyStructureBuilder) buildSinglePartFromMessage(msg *domain.Message, extended bool) (imap.BodyStructure, error) {
	var typeStr, subtype, body string

	if msg.HTMLBody != "" {
		typeStr = "TEXT"
		subtype = "HTML"
		body = msg.HTMLBody
	} else {
		typeStr = "TEXT"
		subtype = "PLAIN"
		body = msg.TextBody
	}

	singlePart := &imap.BodyStructureSinglePart{
		Type:     typeStr,
		Subtype:  subtype,
		Params:   map[string]string{"charset": "utf-8"},
		Encoding: "8BIT",
		Size:     uint32(len(body)),
		Text: &imap.BodyStructureText{
			NumLines: int64(strings.Count(body, "\n") + 1),
		},
	}

	return singlePart, nil
}

// buildMultipartFromMessage builds a multipart structure from message fields.
func (b *BodyStructureBuilder) buildMultipartFromMessage(ctx context.Context, repo repository.Repository, msg *domain.Message, extended bool) (imap.BodyStructure, error) {
	var children []imap.BodyStructure

	// Check if we have both text and HTML (alternative)
	if msg.HTMLBody != "" && msg.TextBody != "" {
		// Create alternative part containing text and HTML
		alternative := &imap.BodyStructureMultiPart{
			Subtype: "ALTERNATIVE",
			Children: []imap.BodyStructure{
				&imap.BodyStructureSinglePart{
					Type:     "TEXT",
					Subtype:  "PLAIN",
					Params:   map[string]string{"charset": "utf-8"},
					Encoding: "8BIT",
					Size:     uint32(len(msg.TextBody)),
					Text: &imap.BodyStructureText{
						NumLines: int64(strings.Count(msg.TextBody, "\n") + 1),
					},
				},
				&imap.BodyStructureSinglePart{
					Type:     "TEXT",
					Subtype:  "HTML",
					Params:   map[string]string{"charset": "utf-8"},
					Encoding: "8BIT",
					Size:     uint32(len(msg.HTMLBody)),
					Text: &imap.BodyStructureText{
						NumLines: int64(strings.Count(msg.HTMLBody, "\n") + 1),
					},
				},
			},
		}
		children = append(children, alternative)
	} else if msg.HTMLBody != "" {
		children = append(children, &imap.BodyStructureSinglePart{
			Type:     "TEXT",
			Subtype:  "HTML",
			Params:   map[string]string{"charset": "utf-8"},
			Encoding: "8BIT",
			Size:     uint32(len(msg.HTMLBody)),
			Text: &imap.BodyStructureText{
				NumLines: int64(strings.Count(msg.HTMLBody, "\n") + 1),
			},
		})
	} else if msg.TextBody != "" {
		children = append(children, &imap.BodyStructureSinglePart{
			Type:     "TEXT",
			Subtype:  "PLAIN",
			Params:   map[string]string{"charset": "utf-8"},
			Encoding: "8BIT",
			Size:     uint32(len(msg.TextBody)),
			Text: &imap.BodyStructureText{
				NumLines: int64(strings.Count(msg.TextBody, "\n") + 1),
			},
		})
	}

	// Add attachments
	if msg.AttachmentCount > 0 {
		attachments, err := b.getAttachments(ctx, repo, msg.ID)
		if err == nil {
			for _, att := range attachments {
				children = append(children, b.attachmentToBodyStructure(att, extended))
			}
		}
	}

	subtype := "MIXED"
	if msg.AttachmentCount == 0 && msg.HTMLBody != "" && msg.TextBody != "" {
		// Just alternative content, no attachments
		subtype = "ALTERNATIVE"
	}

	multipart := &imap.BodyStructureMultiPart{
		Subtype:  subtype,
		Children: children,
	}

	return multipart, nil
}

// getAttachments retrieves attachments for a message.
func (b *BodyStructureBuilder) getAttachments(ctx context.Context, repo repository.Repository, msgID domain.ID) ([]*domain.Attachment, error) {
	// The attachment repository should be accessed through the main repository
	attachments, err := repo.Attachments().ListByMessage(ctx, msgID)
	if err != nil {
		return nil, err
	}
	return attachments, nil
}

// attachmentToBodyStructure converts an attachment to BodyStructure.
func (b *BodyStructureBuilder) attachmentToBodyStructure(att *domain.Attachment, extended bool) imap.BodyStructure {
	// Parse content type
	typeStr := "APPLICATION"
	subtype := "OCTET-STREAM"

	if att.ContentType != "" {
		parts := strings.SplitN(att.ContentType, "/", 2)
		if len(parts) == 2 {
			typeStr = strings.ToUpper(parts[0])
			subtype = strings.ToUpper(parts[1])
		}
	}

	singlePart := &imap.BodyStructureSinglePart{
		Type:     typeStr,
		Subtype:  subtype,
		Params:   map[string]string{"name": att.Filename},
		ID:       att.ContentID,
		Encoding: "BASE64",
		Size:     uint32(att.Size),
	}

	if extended {
		disposition := "attachment"
		if att.IsInline {
			disposition = "inline"
		}
		singlePart.Extended = &imap.BodyStructureSinglePartExt{
			Disposition: &imap.BodyStructureDisposition{
				Value:  disposition,
				Params: map[string]string{"filename": att.Filename},
			},
		}
	}

	return singlePart
}

// extractBoundary extracts the MIME boundary from Content-Type header.
func (b *BodyStructureBuilder) extractBoundary(rawBody []byte) string {
	headerEnd := bytes.Index(rawBody, []byte("\r\n\r\n"))
	if headerEnd == -1 {
		headerEnd = bytes.Index(rawBody, []byte("\n\n"))
		if headerEnd == -1 {
			return ""
		}
	}

	headers := string(rawBody[:headerEnd])
	headersLower := strings.ToLower(headers)

	ctStart := strings.Index(headersLower, "content-type:")
	if ctStart == -1 {
		return ""
	}

	ctLine := headers[ctStart:]
	// Find the end of the header (next non-continuation line)
	lines := strings.Split(ctLine, "\n")
	var ctValue strings.Builder
	for i, line := range lines {
		if i == 0 {
			// First line, remove "content-type:" prefix
			line = line[13:]
		} else if len(line) > 0 && (line[0] == ' ' || line[0] == '\t') {
			// Continuation line
		} else {
			break
		}
		ctValue.WriteString(strings.TrimRight(line, "\r"))
	}

	// Find boundary parameter
	ctContent := ctValue.String()
	boundaryIdx := strings.Index(strings.ToLower(ctContent), "boundary=")
	if boundaryIdx == -1 {
		return ""
	}

	boundaryValue := ctContent[boundaryIdx+9:]
	boundaryValue = strings.TrimSpace(boundaryValue)

	// Handle quoted boundary
	if len(boundaryValue) > 0 && boundaryValue[0] == '"' {
		endQuote := strings.Index(boundaryValue[1:], "\"")
		if endQuote != -1 {
			return boundaryValue[1 : endQuote+1]
		}
	}

	// Unquoted boundary - ends at semicolon, space, or end of string
	for i, c := range boundaryValue {
		if c == ';' || c == ' ' || c == '\t' || c == '\r' || c == '\n' {
			return boundaryValue[:i]
		}
	}

	return boundaryValue
}

// splitByBoundary splits multipart content by boundary.
func (b *BodyStructureBuilder) splitByBoundary(rawBody []byte, boundary string) [][]byte {
	delimiter := []byte("--" + boundary)
	closeDelimiter := []byte("--" + boundary + "--")

	// Find body start
	headerEnd := bytes.Index(rawBody, []byte("\r\n\r\n"))
	if headerEnd == -1 {
		headerEnd = bytes.Index(rawBody, []byte("\n\n"))
		if headerEnd == -1 {
			return nil
		}
		headerEnd += 2
	} else {
		headerEnd += 4
	}

	body := rawBody[headerEnd:]
	var parts [][]byte

	// Find first delimiter
	firstDelim := bytes.Index(body, delimiter)
	if firstDelim == -1 {
		return nil
	}

	remaining := body[firstDelim+len(delimiter):]

	for len(remaining) > 0 {
		// Skip CRLF after delimiter
		if len(remaining) > 0 && remaining[0] == '\r' {
			remaining = remaining[1:]
		}
		if len(remaining) > 0 && remaining[0] == '\n' {
			remaining = remaining[1:]
		}

		// Check for close delimiter
		if bytes.HasPrefix(remaining, []byte("--")) {
			break
		}

		// Find next delimiter
		nextDelim := bytes.Index(remaining, delimiter)
		if nextDelim == -1 {
			// Check for close delimiter
			closeIdx := bytes.Index(remaining, closeDelimiter)
			if closeIdx != -1 {
				part := remaining[:closeIdx]
				part = bytes.TrimSuffix(part, []byte("\r\n"))
				part = bytes.TrimSuffix(part, []byte("\n"))
				parts = append(parts, part)
			}
			break
		}

		part := remaining[:nextDelim]
		part = bytes.TrimSuffix(part, []byte("\r\n"))
		part = bytes.TrimSuffix(part, []byte("\n"))
		parts = append(parts, part)

		remaining = remaining[nextDelim+len(delimiter):]
	}

	return parts
}

// extractContentTypeParams extracts parameters from Content-Type header.
func (b *BodyStructureBuilder) extractContentTypeParams(rawBody []byte) map[string]string {
	params := make(map[string]string)

	headerEnd := bytes.Index(rawBody, []byte("\r\n\r\n"))
	if headerEnd == -1 {
		headerEnd = bytes.Index(rawBody, []byte("\n\n"))
		if headerEnd == -1 {
			return params
		}
	}

	headers := string(rawBody[:headerEnd])
	headersLower := strings.ToLower(headers)

	ctStart := strings.Index(headersLower, "content-type:")
	if ctStart == -1 {
		return params
	}

	ctLine := headers[ctStart+13:]
	// Find end of header
	endIdx := strings.Index(ctLine, "\n")
	if endIdx != -1 {
		// Check for continuation
		for endIdx+1 < len(ctLine) && (ctLine[endIdx+1] == ' ' || ctLine[endIdx+1] == '\t') {
			nextEnd := strings.Index(ctLine[endIdx+1:], "\n")
			if nextEnd == -1 {
				endIdx = len(ctLine)
			} else {
				endIdx = endIdx + 1 + nextEnd
			}
		}
		ctLine = ctLine[:endIdx]
	}

	// Parse parameters
	parts := strings.Split(ctLine, ";")
	for i := 1; i < len(parts); i++ {
		part := strings.TrimSpace(parts[i])
		eqIdx := strings.Index(part, "=")
		if eqIdx == -1 {
			continue
		}

		name := strings.TrimSpace(part[:eqIdx])
		value := strings.TrimSpace(part[eqIdx+1:])

		// Remove quotes
		if len(value) >= 2 && value[0] == '"' && value[len(value)-1] == '"' {
			value = value[1 : len(value)-1]
		}

		params[strings.ToLower(name)] = value
	}

	return params
}

// extractHeaderValue extracts a header value by name.
func (b *BodyStructureBuilder) extractHeaderValue(rawBody []byte, headerName string) string {
	headerEnd := bytes.Index(rawBody, []byte("\r\n\r\n"))
	if headerEnd == -1 {
		headerEnd = bytes.Index(rawBody, []byte("\n\n"))
		if headerEnd == -1 {
			return ""
		}
	}

	headers := string(rawBody[:headerEnd])
	headersLower := strings.ToLower(headers)
	headerNameLower := strings.ToLower(headerName) + ":"

	start := strings.Index(headersLower, headerNameLower)
	if start == -1 {
		return ""
	}

	value := headers[start+len(headerNameLower):]
	endIdx := strings.Index(value, "\n")
	if endIdx != -1 {
		value = value[:endIdx]
	}

	return strings.TrimSpace(strings.TrimSuffix(value, "\r"))
}

// extractDisposition extracts Content-Disposition from headers.
func (b *BodyStructureBuilder) extractDisposition(rawBody []byte) *imap.BodyStructureDisposition {
	headerEnd := bytes.Index(rawBody, []byte("\r\n\r\n"))
	if headerEnd == -1 {
		headerEnd = bytes.Index(rawBody, []byte("\n\n"))
		if headerEnd == -1 {
			return nil
		}
	}

	headers := string(rawBody[:headerEnd])
	headersLower := strings.ToLower(headers)

	cdStart := strings.Index(headersLower, "content-disposition:")
	if cdStart == -1 {
		return nil
	}

	cdLine := headers[cdStart+20:]
	endIdx := strings.Index(cdLine, "\n")
	if endIdx != -1 {
		cdLine = cdLine[:endIdx]
	}
	cdLine = strings.TrimSpace(strings.TrimSuffix(cdLine, "\r"))

	// Parse disposition type and parameters
	parts := strings.SplitN(cdLine, ";", 2)
	disposition := &imap.BodyStructureDisposition{
		Value:  strings.TrimSpace(parts[0]),
		Params: make(map[string]string),
	}

	if len(parts) > 1 {
		// Parse parameters
		paramParts := strings.Split(parts[1], ";")
		for _, param := range paramParts {
			param = strings.TrimSpace(param)
			eqIdx := strings.Index(param, "=")
			if eqIdx == -1 {
				continue
			}
			name := strings.TrimSpace(param[:eqIdx])
			value := strings.TrimSpace(param[eqIdx+1:])

			// Remove quotes
			if len(value) >= 2 && value[0] == '"' && value[len(value)-1] == '"' {
				value = value[1 : len(value)-1]
			}

			disposition.Params[strings.ToLower(name)] = value
		}
	}

	return disposition
}

// extractBody extracts the body (after headers) from raw data.
func (b *BodyStructureBuilder) extractBody(rawBody []byte) []byte {
	separator := []byte("\r\n\r\n")
	headerEnd := bytes.Index(rawBody, separator)
	if headerEnd == -1 {
		separator = []byte("\n\n")
		headerEnd = bytes.Index(rawBody, separator)
		if headerEnd == -1 {
			return []byte{}
		}
	}
	return rawBody[headerEnd+len(separator):]
}

// countLines counts the number of lines in data.
func (b *BodyStructureBuilder) countLines(data []byte) int {
	if len(data) == 0 {
		return 0
	}
	count := 1
	for _, c := range data {
		if c == '\n' {
			count++
		}
	}
	return count
}
