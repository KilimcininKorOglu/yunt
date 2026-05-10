package imap

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/emersion/go-imap/v2"
	"github.com/emersion/go-imap/v2/imapserver"

	"yunt/internal/domain"
	"yunt/internal/repository"
)

// FetchHandler handles IMAP FETCH command operations.
type FetchHandler struct {
	repo           repository.Repository
	userID         domain.ID
	selectedMbox   *domain.Mailbox
	messageBuilder *MessageBuilder
}

// NewFetchHandler creates a new FetchHandler.
func NewFetchHandler(repo repository.Repository, userID domain.ID, selectedMbox *domain.Mailbox) *FetchHandler {
	return &FetchHandler{
		repo:           repo,
		userID:         userID,
		selectedMbox:   selectedMbox,
		messageBuilder: NewMessageBuilder(),
	}
}

// Fetch retrieves message data according to the specified options.
// It handles both UID and sequence number based fetches.
func (h *FetchHandler) Fetch(ctx context.Context, w *imapserver.FetchWriter, numSet imap.NumSet, options *imap.FetchOptions) error {
	if h.selectedMbox == nil {
		return &imap.Error{
			Type: imap.StatusResponseTypeNo,
			Text: "No mailbox selected",
		}
	}

	// Get messages from the selected mailbox
	messages, err := h.getMessagesForNumSet(ctx, numSet)
	if err != nil {
		return err
	}

	// Process each message
	for seqNum, msg := range messages {
		if err := h.fetchMessage(ctx, w, seqNum, msg, options); err != nil {
			return err
		}
	}

	return nil
}

// getMessagesForNumSet retrieves messages matching the given number set.
// The numSet can contain either sequence numbers or UIDs.
func (h *FetchHandler) getMessagesForNumSet(ctx context.Context, numSet imap.NumSet) (map[uint32]*domain.Message, error) {
	// Get all messages in the mailbox
	result, err := h.repo.Messages().ListByMailbox(ctx, h.selectedMbox.ID, nil)
	if err != nil {
		return nil, &imap.Error{
			Type: imap.StatusResponseTypeNo,
			Text: "Failed to list messages",
		}
	}

	messages := make(map[uint32]*domain.Message)

	// For now, we use sequence numbers (1-based index)
	// In a full implementation, we'd also need to handle UIDs
	for i, msg := range result.Items {
		seqNum := uint32(i + 1)
		uid := imap.UID(i + 1) // Simplified UID = sequence number

		// Check if this message matches the number set
		if numSetContains(numSet, seqNum, uid) {
			messages[seqNum] = msg
		}
	}

	return messages, nil
}

// numSetContains checks if a sequence number or UID is in the number set.
func numSetContains(numSet imap.NumSet, seqNum uint32, uid imap.UID) bool {
	switch ns := numSet.(type) {
	case imap.SeqSet:
		return ns.Contains(seqNum)
	case imap.UIDSet:
		return ns.Contains(uid)
	default:
		return false
	}
}

// fetchMessage fetches data for a single message and writes it to the response.
func (h *FetchHandler) fetchMessage(ctx context.Context, w *imapserver.FetchWriter, seqNum uint32, msg *domain.Message, options *imap.FetchOptions) error {
	respWriter := w.CreateMessage(seqNum)
	defer respWriter.Close()

	// Handle UID
	if options.UID {
		// For simplicity, we use the sequence number as UID
		// In a real implementation, you'd use actual UIDs stored with messages
		respWriter.WriteUID(imap.UID(seqNum))
	}

	// Handle FLAGS
	if options.Flags {
		flags := h.getMessageFlags(msg)
		respWriter.WriteFlags(flags)
	}

	// Handle INTERNALDATE
	if options.InternalDate {
		respWriter.WriteInternalDate(msg.ReceivedAt.Time)
	}

	// Handle RFC822.SIZE
	if options.RFC822Size {
		respWriter.WriteRFC822Size(msg.Size)
	}

	// Handle ENVELOPE
	if options.Envelope {
		envelope := h.buildEnvelope(msg)
		respWriter.WriteEnvelope(envelope)
	}

	// Handle BODYSTRUCTURE
	if options.BodyStructure != nil {
		bodyStructure, err := h.buildBodyStructure(ctx, msg, options.BodyStructure.Extended)
		if err != nil {
			return err
		}
		respWriter.WriteBodyStructure(bodyStructure)
	}

	// Handle BODY sections
	for _, section := range options.BodySection {
		if err := h.fetchBodySection(ctx, respWriter, msg, section); err != nil {
			return err
		}

		// Mark message as read if not PEEK and specifier is not header-only
		if !section.Peek && section.Specifier != imap.PartSpecifierHeader &&
			section.Specifier != imap.PartSpecifierMIME {
			h.markAsRead(ctx, msg)
		}
	}

	return nil
}

// getMessageFlags returns the IMAP flags for a message.
func (h *FetchHandler) getMessageFlags(msg *domain.Message) []imap.Flag {
	var flags []imap.Flag

	// \Seen flag
	if msg.Status == domain.MessageRead {
		flags = append(flags, imap.FlagSeen)
	}

	// \Flagged (starred)
	if msg.IsStarred {
		flags = append(flags, imap.FlagFlagged)
	}

	// \Answered - check if In-Reply-To or References exist
	// This is a simplified check; in reality, you'd track this explicitly
	if msg.InReplyTo != "" {
		flags = append(flags, imap.FlagAnswered)
	}

	// \Draft - check if in Drafts mailbox
	// For now, we don't have draft status

	return flags
}

// buildEnvelope builds the IMAP envelope structure for a message.
func (h *FetchHandler) buildEnvelope(msg *domain.Message) *imap.Envelope {
	envelope := &imap.Envelope{
		Subject:   msg.Subject,
		MessageID: msg.MessageID,
	}

	// Set date from SentAt if available, otherwise ReceivedAt
	if msg.SentAt != nil {
		envelope.Date = msg.SentAt.Time
	} else {
		envelope.Date = msg.ReceivedAt.Time
	}

	// Set From address
	if !msg.From.IsEmpty() {
		envelope.From = []imap.Address{domainToIMAPAddress(msg.From)}
	}

	// Set Sender (same as From if not specified)
	envelope.Sender = envelope.From

	// Set Reply-To
	if msg.ReplyTo != nil && !msg.ReplyTo.IsEmpty() {
		envelope.ReplyTo = []imap.Address{domainToIMAPAddress(*msg.ReplyTo)}
	} else {
		envelope.ReplyTo = envelope.From
	}

	// Set To addresses
	for _, addr := range msg.To {
		envelope.To = append(envelope.To, domainToIMAPAddress(addr))
	}

	// Set Cc addresses
	for _, addr := range msg.Cc {
		envelope.Cc = append(envelope.Cc, domainToIMAPAddress(addr))
	}

	// Set Bcc addresses
	for _, addr := range msg.Bcc {
		envelope.Bcc = append(envelope.Bcc, domainToIMAPAddress(addr))
	}

	// Set In-Reply-To
	if msg.InReplyTo != "" {
		envelope.InReplyTo = []string{msg.InReplyTo}
	}

	return envelope
}

// domainToIMAPAddress converts a domain.EmailAddress to imap.Address.
func domainToIMAPAddress(addr domain.EmailAddress) imap.Address {
	// Split email address into mailbox and host
	mailbox, host := splitEmailAddress(addr.Address)

	return imap.Address{
		Name:    addr.Name,
		Mailbox: mailbox,
		Host:    host,
	}
}

// splitEmailAddress splits an email address into mailbox and host parts.
func splitEmailAddress(email string) (mailbox, host string) {
	parts := strings.SplitN(email, "@", 2)
	if len(parts) == 2 {
		return parts[0], parts[1]
	}
	return email, ""
}

// buildBodyStructure builds the IMAP body structure for a message.
func (h *FetchHandler) buildBodyStructure(ctx context.Context, msg *domain.Message, extended bool) (imap.BodyStructure, error) {
	builder := NewBodyStructureBuilder()
	return builder.Build(ctx, h.repo, msg, extended)
}

// fetchBodySection fetches a specific body section of a message.
func (h *FetchHandler) fetchBodySection(ctx context.Context, w *imapserver.FetchResponseWriter, msg *domain.Message, section *imap.FetchItemBodySection) error {
	// Get the raw message data
	rawBody, err := h.repo.Messages().GetRawBody(ctx, msg.ID)
	if err != nil {
		// If raw body is not stored, reconstruct it from message fields
		rawBody = h.reconstructMessage(msg)
	}

	// Extract the requested section
	sectionData, err := h.extractSection(rawBody, msg, section)
	if err != nil {
		return err
	}

	// Apply partial fetch if specified
	if section.Partial != nil {
		sectionData = applyPartial(sectionData, section.Partial)
	}

	// Write the body section
	bodyWriter := w.WriteBodySection(section, int64(len(sectionData)))
	defer bodyWriter.Close()

	_, err = bodyWriter.Write(sectionData)
	return err
}

// extractSection extracts a specific section from the message.
func (h *FetchHandler) extractSection(rawBody []byte, _ *domain.Message, section *imap.FetchItemBodySection) ([]byte, error) {
	switch section.Specifier {
	case imap.PartSpecifierNone:
		// Entire message or specific part
		if len(section.Part) == 0 {
			// If header fields are specified, extract headers with filtering
			if len(section.HeaderFields) > 0 || len(section.HeaderFieldsNot) > 0 {
				return h.extractHeader(rawBody, section.HeaderFields, section.HeaderFieldsNot)
			}
			return rawBody, nil
		}
		return h.extractPart(rawBody, section.Part)

	case imap.PartSpecifierHeader:
		// Message header or part header
		if len(section.Part) == 0 {
			return h.extractHeader(rawBody, section.HeaderFields, section.HeaderFieldsNot)
		}
		return h.extractPartHeader(rawBody, section.Part, section.HeaderFields, section.HeaderFieldsNot)

	case imap.PartSpecifierText:
		// Message body (without header)
		if len(section.Part) == 0 {
			return h.extractText(rawBody)
		}
		return h.extractPartText(rawBody, section.Part)

	case imap.PartSpecifierMIME:
		// MIME header of a part
		return h.extractPartMIME(rawBody, section.Part)

	default:
		return rawBody, nil
	}
}

// extractHeader extracts headers from the message.
func (h *FetchHandler) extractHeader(rawBody []byte, fields, notFields []string) ([]byte, error) {
	// Find header/body separator
	headerEnd := bytes.Index(rawBody, []byte("\r\n\r\n"))
	if headerEnd == -1 {
		headerEnd = bytes.Index(rawBody, []byte("\n\n"))
		if headerEnd == -1 {
			// Entire message is headers
			headerEnd = len(rawBody)
		}
	}

	headerBytes := rawBody[:headerEnd]

	// If no field filtering, return all headers
	if len(fields) == 0 && len(notFields) == 0 {
		// Include the trailing CRLF
		if headerEnd+4 <= len(rawBody) {
			return append(headerBytes, '\r', '\n', '\r', '\n'), nil
		}
		return append(headerBytes, '\r', '\n'), nil
	}

	// Filter headers
	return filterHeaders(headerBytes, fields, notFields), nil
}

// filterHeaders filters headers based on included/excluded fields.
func filterHeaders(headerBytes []byte, fields, notFields []string) []byte {
	// Normalize field names to lowercase for comparison
	fieldsLower := make(map[string]bool)
	for _, f := range fields {
		fieldsLower[strings.ToLower(f)] = true
	}
	notFieldsLower := make(map[string]bool)
	for _, f := range notFields {
		notFieldsLower[strings.ToLower(f)] = true
	}

	var result bytes.Buffer
	lines := bytes.Split(headerBytes, []byte("\n"))
	var currentHeader string
	var currentValue bytes.Buffer
	includeCurrentHeader := false

	for _, line := range lines {
		// Check if this is a continuation line (starts with whitespace)
		if len(line) > 0 && (line[0] == ' ' || line[0] == '\t') {
			if includeCurrentHeader {
				currentValue.Write(line)
				currentValue.WriteByte('\n')
			}
			continue
		}

		// Write previous header if it should be included
		if includeCurrentHeader && currentValue.Len() > 0 {
			result.Write(currentValue.Bytes())
		}

		// Parse new header
		line = bytes.TrimRight(line, "\r")
		colonIdx := bytes.IndexByte(line, ':')
		if colonIdx == -1 {
			continue
		}

		currentHeader = strings.ToLower(string(bytes.TrimSpace(line[:colonIdx])))
		currentValue.Reset()
		currentValue.Write(line)
		currentValue.WriteByte('\r')
		currentValue.WriteByte('\n')

		// Determine if this header should be included
		if len(fields) > 0 {
			includeCurrentHeader = fieldsLower[currentHeader]
		} else if len(notFields) > 0 {
			includeCurrentHeader = !notFieldsLower[currentHeader]
		} else {
			includeCurrentHeader = true
		}
	}

	// Write last header if it should be included
	if includeCurrentHeader && currentValue.Len() > 0 {
		result.Write(currentValue.Bytes())
	}

	// Add final CRLF
	result.Write([]byte("\r\n"))

	return result.Bytes()
}

// extractText extracts the body (without header) from the message.
func (h *FetchHandler) extractText(rawBody []byte) ([]byte, error) {
	// Find header/body separator
	separator := []byte("\r\n\r\n")
	headerEnd := bytes.Index(rawBody, separator)
	if headerEnd == -1 {
		separator = []byte("\n\n")
		headerEnd = bytes.Index(rawBody, separator)
		if headerEnd == -1 {
			// No body
			return []byte{}, nil
		}
	}

	return rawBody[headerEnd+len(separator):], nil
}

// extractPart extracts a specific MIME part from the message.
func (h *FetchHandler) extractPart(rawBody []byte, partPath []int) ([]byte, error) {
	extractor := NewPartExtractor(rawBody)
	return extractor.ExtractPart(partPath)
}

// extractPartHeader extracts the header of a specific MIME part.
func (h *FetchHandler) extractPartHeader(rawBody []byte, partPath []int, fields, notFields []string) ([]byte, error) {
	extractor := NewPartExtractor(rawBody)
	partData, err := extractor.ExtractPart(partPath)
	if err != nil {
		return nil, err
	}
	return h.extractHeader(partData, fields, notFields)
}

// extractPartText extracts the body of a specific MIME part.
func (h *FetchHandler) extractPartText(rawBody []byte, partPath []int) ([]byte, error) {
	extractor := NewPartExtractor(rawBody)
	partData, err := extractor.ExtractPart(partPath)
	if err != nil {
		return nil, err
	}
	return h.extractText(partData)
}

// extractPartMIME extracts the MIME header of a specific part.
func (h *FetchHandler) extractPartMIME(rawBody []byte, partPath []int) ([]byte, error) {
	extractor := NewPartExtractor(rawBody)
	partData, err := extractor.ExtractPart(partPath)
	if err != nil {
		return nil, err
	}
	return h.extractHeader(partData, nil, nil)
}

// reconstructMessage reconstructs a raw RFC 822 message from domain.Message.
func (h *FetchHandler) reconstructMessage(msg *domain.Message) []byte {
	return h.messageBuilder.Build(msg)
}

// applyPartial applies partial fetch (offset and size) to the data.
func applyPartial(data []byte, partial *imap.SectionPartial) []byte {
	if partial == nil {
		return data
	}

	start := int(partial.Offset)
	if start >= len(data) {
		return []byte{}
	}

	end := len(data)
	if partial.Size > 0 {
		end = start + int(partial.Size)
		if end > len(data) {
			end = len(data)
		}
	}

	return data[start:end]
}

// markAsRead marks a message as read.
func (h *FetchHandler) markAsRead(ctx context.Context, msg *domain.Message) {
	if msg.Status != domain.MessageRead {
		_, _ = h.repo.Messages().MarkAsRead(ctx, msg.ID)
		msg.Status = domain.MessageRead
	}
}

// PartExtractor extracts MIME parts from a raw message.
type PartExtractor struct {
	rawBody []byte
}

// NewPartExtractor creates a new PartExtractor.
func NewPartExtractor(rawBody []byte) *PartExtractor {
	return &PartExtractor{rawBody: rawBody}
}

// ExtractPart extracts a MIME part by its path (e.g., [1, 2] for part 1.2).
func (e *PartExtractor) ExtractPart(partPath []int) ([]byte, error) {
	if len(partPath) == 0 {
		return e.rawBody, nil
	}

	return e.extractPartRecursive(e.rawBody, partPath, 0)
}

// extractPartRecursive recursively extracts a MIME part.
func (e *PartExtractor) extractPartRecursive(data []byte, partPath []int, depth int) ([]byte, error) {
	if depth >= len(partPath) {
		return data, nil
	}

	targetPart := partPath[depth]

	// Parse the Content-Type to get the boundary
	boundary := e.extractBoundary(data)
	if boundary == "" {
		// Not a multipart message; if targeting part 1, return the body
		if targetPart == 1 && depth == len(partPath)-1 {
			return e.extractBody(data)
		}
		return nil, fmt.Errorf("part %d not found", targetPart)
	}

	// Split by boundary
	parts := e.splitByBoundary(data, boundary)

	if targetPart < 1 || targetPart > len(parts) {
		return nil, fmt.Errorf("part %d not found (have %d parts)", targetPart, len(parts))
	}

	partData := parts[targetPart-1]

	// If there are more levels to descend, recurse
	if depth < len(partPath)-1 {
		return e.extractPartRecursive(partData, partPath, depth+1)
	}

	return partData, nil
}

// extractBoundary extracts the MIME boundary from headers.
func (e *PartExtractor) extractBoundary(data []byte) string {
	// Find Content-Type header
	headerEnd := bytes.Index(data, []byte("\r\n\r\n"))
	if headerEnd == -1 {
		headerEnd = bytes.Index(data, []byte("\n\n"))
		if headerEnd == -1 {
			return ""
		}
	}

	headers := string(data[:headerEnd])
	headersLower := strings.ToLower(headers)

	// Find content-type line
	ctStart := strings.Index(headersLower, "content-type:")
	if ctStart == -1 {
		return ""
	}

	// Extract the header value (may span multiple lines)
	ctLine := headers[ctStart:]
	ctEnd := strings.Index(ctLine, "\n")
	if ctEnd != -1 {
		// Check for continuation lines
		for ctEnd+1 < len(ctLine) && (ctLine[ctEnd+1] == ' ' || ctLine[ctEnd+1] == '\t') {
			nextEnd := strings.Index(ctLine[ctEnd+1:], "\n")
			if nextEnd == -1 {
				ctEnd = len(ctLine)
			} else {
				ctEnd = ctEnd + 1 + nextEnd
			}
		}
		ctLine = ctLine[:ctEnd]
	}

	// Find boundary parameter
	boundaryIdx := strings.Index(strings.ToLower(ctLine), "boundary=")
	if boundaryIdx == -1 {
		return ""
	}

	boundaryValue := ctLine[boundaryIdx+9:]
	boundaryValue = strings.TrimSpace(boundaryValue)

	// Remove quotes if present
	if len(boundaryValue) > 0 && boundaryValue[0] == '"' {
		endQuote := strings.Index(boundaryValue[1:], "\"")
		if endQuote != -1 {
			return boundaryValue[1 : endQuote+1]
		}
	}

	// Remove any trailing parameters
	if semiIdx := strings.Index(boundaryValue, ";"); semiIdx != -1 {
		boundaryValue = boundaryValue[:semiIdx]
	}
	if nlIdx := strings.Index(boundaryValue, "\r"); nlIdx != -1 {
		boundaryValue = boundaryValue[:nlIdx]
	}
	if nlIdx := strings.Index(boundaryValue, "\n"); nlIdx != -1 {
		boundaryValue = boundaryValue[:nlIdx]
	}

	return strings.TrimSpace(boundaryValue)
}

// extractBody extracts the body (after headers) from data.
func (e *PartExtractor) extractBody(data []byte) ([]byte, error) {
	separator := []byte("\r\n\r\n")
	headerEnd := bytes.Index(data, separator)
	if headerEnd == -1 {
		separator = []byte("\n\n")
		headerEnd = bytes.Index(data, separator)
		if headerEnd == -1 {
			return []byte{}, nil
		}
	}
	return data[headerEnd+len(separator):], nil
}

// splitByBoundary splits multipart content by the given boundary.
func (e *PartExtractor) splitByBoundary(data []byte, boundary string) [][]byte {
	delimiter := "--" + boundary
	closeDelimiter := "--" + boundary + "--"

	// Get body (after headers)
	body, err := e.extractBody(data)
	if err != nil {
		return nil
	}

	var parts [][]byte
	remaining := body

	for {
		// Find the next boundary
		delimIdx := bytes.Index(remaining, []byte(delimiter))
		if delimIdx == -1 {
			break
		}

		// Skip past the boundary line
		remaining = remaining[delimIdx+len(delimiter):]

		// Skip CRLF after boundary
		if len(remaining) > 0 && remaining[0] == '\r' {
			remaining = remaining[1:]
		}
		if len(remaining) > 0 && remaining[0] == '\n' {
			remaining = remaining[1:]
		}

		// Check if this is the closing delimiter
		if bytes.HasPrefix(remaining, []byte("--")) {
			break
		}

		// Find the end of this part (next boundary)
		nextDelimIdx := bytes.Index(remaining, []byte(delimiter))
		if nextDelimIdx == -1 {
			// Check for close delimiter
			closeIdx := bytes.Index(remaining, []byte(closeDelimiter))
			if closeIdx != -1 {
				partData := remaining[:closeIdx]
				partData = bytes.TrimSuffix(partData, []byte("\r\n"))
				partData = bytes.TrimSuffix(partData, []byte("\n"))
				parts = append(parts, partData)
			}
			break
		}

		partData := remaining[:nextDelimIdx]
		partData = bytes.TrimSuffix(partData, []byte("\r\n"))
		partData = bytes.TrimSuffix(partData, []byte("\n"))
		parts = append(parts, partData)

		remaining = remaining[nextDelimIdx:]
	}

	return parts
}

// MessageBuilder builds RFC 822 format messages from domain.Message.
type MessageBuilder struct{}

// NewMessageBuilder creates a new MessageBuilder.
func NewMessageBuilder() *MessageBuilder {
	return &MessageBuilder{}
}

// Build constructs an RFC 822 message from a domain.Message.
func (b *MessageBuilder) Build(msg *domain.Message) []byte {
	var buf bytes.Buffer

	// Write headers
	b.writeHeader(&buf, "Date", formatRFC822Date(msg.ReceivedAt.Time))
	b.writeHeader(&buf, "From", msg.From.String())
	b.writeHeader(&buf, "Subject", msg.Subject)

	if msg.MessageID != "" {
		b.writeHeader(&buf, "Message-ID", "<"+msg.MessageID+">")
	}

	if len(msg.To) > 0 {
		b.writeHeader(&buf, "To", formatAddressList(msg.To))
	}

	if len(msg.Cc) > 0 {
		b.writeHeader(&buf, "Cc", formatAddressList(msg.Cc))
	}

	if msg.ReplyTo != nil {
		b.writeHeader(&buf, "Reply-To", msg.ReplyTo.String())
	}

	if msg.InReplyTo != "" {
		b.writeHeader(&buf, "In-Reply-To", "<"+msg.InReplyTo+">")
	}

	if len(msg.References) > 0 {
		refs := make([]string, len(msg.References))
		for i, ref := range msg.References {
			refs[i] = "<" + ref + ">"
		}
		b.writeHeader(&buf, "References", strings.Join(refs, " "))
	}

	// Write custom headers
	for name, value := range msg.Headers {
		// Skip headers we've already written
		switch strings.ToLower(name) {
		case "date", "from", "subject", "message-id", "to", "cc", "reply-to", "in-reply-to", "references":
			continue
		}
		b.writeHeader(&buf, name, value)
	}

	// Determine content type
	if msg.HTMLBody != "" && msg.TextBody != "" {
		// Multipart alternative
		boundary := generateBoundary()
		b.writeHeader(&buf, "Content-Type", "multipart/alternative; boundary=\""+boundary+"\"")
		b.writeHeader(&buf, "MIME-Version", "1.0")

		buf.WriteString("\r\n")

		// Write text part
		buf.WriteString("--" + boundary + "\r\n")
		buf.WriteString("Content-Type: text/plain; charset=utf-8\r\n")
		buf.WriteString("Content-Transfer-Encoding: 8bit\r\n")
		buf.WriteString("\r\n")
		buf.WriteString(msg.TextBody)
		buf.WriteString("\r\n")

		// Write HTML part
		buf.WriteString("--" + boundary + "\r\n")
		buf.WriteString("Content-Type: text/html; charset=utf-8\r\n")
		buf.WriteString("Content-Transfer-Encoding: 8bit\r\n")
		buf.WriteString("\r\n")
		buf.WriteString(msg.HTMLBody)
		buf.WriteString("\r\n")

		buf.WriteString("--" + boundary + "--\r\n")
	} else if msg.HTMLBody != "" {
		b.writeHeader(&buf, "Content-Type", "text/html; charset=utf-8")
		b.writeHeader(&buf, "Content-Transfer-Encoding", "8bit")
		b.writeHeader(&buf, "MIME-Version", "1.0")
		buf.WriteString("\r\n")
		buf.WriteString(msg.HTMLBody)
	} else {
		b.writeHeader(&buf, "Content-Type", "text/plain; charset=utf-8")
		b.writeHeader(&buf, "Content-Transfer-Encoding", "8bit")
		b.writeHeader(&buf, "MIME-Version", "1.0")
		buf.WriteString("\r\n")
		buf.WriteString(msg.TextBody)
	}

	return buf.Bytes()
}

// writeHeader writes a single header line.
func (b *MessageBuilder) writeHeader(w io.Writer, name, value string) {
	fmt.Fprintf(w, "%s: %s\r\n", name, value)
}

// formatRFC822Date formats a time in RFC 822 format.
func formatRFC822Date(t time.Time) string {
	return t.Format(time.RFC1123Z)
}

// formatAddressList formats a list of email addresses.
func formatAddressList(addresses []domain.EmailAddress) string {
	parts := make([]string, len(addresses))
	for i, addr := range addresses {
		parts[i] = addr.String()
	}
	return strings.Join(parts, ", ")
}

// generateBoundary generates a MIME boundary string.
func generateBoundary() string {
	return fmt.Sprintf("=_Part_%d_%d", time.Now().UnixNano(), time.Now().UnixMicro())
}
