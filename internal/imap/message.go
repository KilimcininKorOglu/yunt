package imap

import (
	"bytes"
	"strings"
	"time"

	"github.com/emersion/go-imap/v2"

	"yunt/internal/domain"
)

// IMAPMessage wraps a domain.Message with IMAP-specific functionality.
type IMAPMessage struct {
	msg       *domain.Message
	seqNum   uint32
	uid      imap.UID
	rawBody  []byte
	flags    []imap.Flag
	flagsSet bool
}

// NewIMAPMessage creates a new IMAPMessage from a domain.Message.
func NewIMAPMessage(msg *domain.Message, seqNum uint32, uid imap.UID) *IMAPMessage {
	return &IMAPMessage{
		msg:    msg,
		seqNum: seqNum,
		uid:    uid,
	}
}

// Message returns the underlying domain.Message.
func (m *IMAPMessage) Message() *domain.Message {
	return m.msg
}

// SeqNum returns the sequence number.
func (m *IMAPMessage) SeqNum() uint32 {
	return m.seqNum
}

// UID returns the message UID.
func (m *IMAPMessage) UID() imap.UID {
	return m.uid
}

// SetRawBody sets the raw RFC 822 message body.
func (m *IMAPMessage) SetRawBody(data []byte) {
	m.rawBody = data
}

// RawBody returns the raw RFC 822 message body.
func (m *IMAPMessage) RawBody() []byte {
	return m.rawBody
}

// Flags returns the IMAP flags for this message.
func (m *IMAPMessage) Flags() []imap.Flag {
	if !m.flagsSet {
		m.flags = m.computeFlags()
		m.flagsSet = true
	}
	return m.flags
}

// computeFlags computes IMAP flags from the domain.Message.
func (m *IMAPMessage) computeFlags() []imap.Flag {
	var flags []imap.Flag

	// \Seen flag
	if m.msg.Status == domain.MessageRead {
		flags = append(flags, imap.FlagSeen)
	}

	// \Flagged (starred)
	if m.msg.IsStarred {
		flags = append(flags, imap.FlagFlagged)
	}

	// \Answered - check if message has been replied to
	// In a full implementation, this would be tracked explicitly
	if m.msg.InReplyTo != "" {
		flags = append(flags, imap.FlagAnswered)
	}

	// \Deleted - we don't currently track this separately
	// Messages in Trash mailbox could be considered deleted

	// \Draft - we don't currently track this separately
	// Messages in Drafts mailbox could be considered drafts

	return flags
}

// HasFlag checks if the message has a specific flag.
func (m *IMAPMessage) HasFlag(flag imap.Flag) bool {
	for _, f := range m.Flags() {
		if f == flag {
			return true
		}
	}
	return false
}

// Envelope returns the IMAP envelope for this message.
func (m *IMAPMessage) Envelope() *imap.Envelope {
	envelope := &imap.Envelope{
		Subject:   m.msg.Subject,
		MessageID: m.msg.MessageID,
	}

	// Set date
	if m.msg.SentAt != nil {
		envelope.Date = m.msg.SentAt.Time
	} else {
		envelope.Date = m.msg.ReceivedAt.Time
	}

	// Set From
	if !m.msg.From.IsEmpty() {
		envelope.From = []imap.Address{emailToIMAPAddress(m.msg.From)}
	}

	// Sender is same as From if not specified
	envelope.Sender = envelope.From

	// Reply-To
	if m.msg.ReplyTo != nil && !m.msg.ReplyTo.IsEmpty() {
		envelope.ReplyTo = []imap.Address{emailToIMAPAddress(*m.msg.ReplyTo)}
	} else {
		envelope.ReplyTo = envelope.From
	}

	// To
	for _, addr := range m.msg.To {
		envelope.To = append(envelope.To, emailToIMAPAddress(addr))
	}

	// Cc
	for _, addr := range m.msg.Cc {
		envelope.Cc = append(envelope.Cc, emailToIMAPAddress(addr))
	}

	// Bcc
	for _, addr := range m.msg.Bcc {
		envelope.Bcc = append(envelope.Bcc, emailToIMAPAddress(addr))
	}

	// In-Reply-To
	if m.msg.InReplyTo != "" {
		envelope.InReplyTo = []string{m.msg.InReplyTo}
	}

	return envelope
}

// emailToIMAPAddress converts a domain.EmailAddress to imap.Address.
func emailToIMAPAddress(addr domain.EmailAddress) imap.Address {
	mailbox, host := parseEmailAddress(addr.Address)
	return imap.Address{
		Name:    addr.Name,
		Mailbox: mailbox,
		Host:    host,
	}
}

// parseEmailAddress splits an email address into mailbox and host parts.
func parseEmailAddress(email string) (mailbox, host string) {
	idx := strings.LastIndex(email, "@")
	if idx == -1 {
		return email, ""
	}
	return email[:idx], email[idx+1:]
}

// InternalDate returns the internal date (when the message was received).
func (m *IMAPMessage) InternalDate() time.Time {
	return m.msg.ReceivedAt.Time
}

// Size returns the RFC822 size of the message.
func (m *IMAPMessage) Size() int64 {
	if m.rawBody != nil {
		return int64(len(m.rawBody))
	}
	return m.msg.Size
}

// MessageSequence manages sequence numbers and UIDs for messages in a mailbox.
type MessageSequence struct {
	messages   []*IMAPMessage
	uidMap     map[imap.UID]*IMAPMessage
	seqNumMap  map[uint32]*IMAPMessage
	uidValid   uint32
	nextUID    imap.UID
}

// NewMessageSequence creates a new MessageSequence.
func NewMessageSequence(uidValidity uint32) *MessageSequence {
	return &MessageSequence{
		messages:  make([]*IMAPMessage, 0),
		uidMap:    make(map[imap.UID]*IMAPMessage),
		seqNumMap: make(map[uint32]*IMAPMessage),
		uidValid:  uidValidity,
		nextUID:   1,
	}
}

// Add adds a message to the sequence.
func (s *MessageSequence) Add(msg *domain.Message) *IMAPMessage {
	seqNum := uint32(len(s.messages) + 1)
	uid := s.nextUID
	s.nextUID++

	imapMsg := NewIMAPMessage(msg, seqNum, uid)
	s.messages = append(s.messages, imapMsg)
	s.uidMap[uid] = imapMsg
	s.seqNumMap[seqNum] = imapMsg

	return imapMsg
}

// GetBySeqNum returns a message by sequence number.
func (s *MessageSequence) GetBySeqNum(seqNum uint32) *IMAPMessage {
	return s.seqNumMap[seqNum]
}

// GetByUID returns a message by UID.
func (s *MessageSequence) GetByUID(uid imap.UID) *IMAPMessage {
	return s.uidMap[uid]
}

// Count returns the number of messages.
func (s *MessageSequence) Count() uint32 {
	return uint32(len(s.messages))
}

// UIDValidity returns the UID validity value.
func (s *MessageSequence) UIDValidity() uint32 {
	return s.uidValid
}

// NextUID returns the next UID to be assigned.
func (s *MessageSequence) NextUID() imap.UID {
	return s.nextUID
}

// GetByNumSet returns messages matching the given number set.
func (s *MessageSequence) GetByNumSet(numSet imap.NumSet) []*IMAPMessage {
	var result []*IMAPMessage

	switch ns := numSet.(type) {
	case imap.SeqSet:
		for _, msg := range s.messages {
			if ns.Contains(msg.SeqNum()) {
				result = append(result, msg)
			}
		}
	case imap.UIDSet:
		for _, msg := range s.messages {
			if ns.Contains(msg.UID()) {
				result = append(result, msg)
			}
		}
	}

	return result
}

// FlagStore manages message flags.
type FlagStore struct {
	flags map[domain.ID]map[imap.Flag]bool
}

// NewFlagStore creates a new FlagStore.
func NewFlagStore() *FlagStore {
	return &FlagStore{
		flags: make(map[domain.ID]map[imap.Flag]bool),
	}
}

// SetFlags sets flags for a message.
func (s *FlagStore) SetFlags(msgID domain.ID, flags []imap.Flag) {
	s.flags[msgID] = make(map[imap.Flag]bool)
	for _, f := range flags {
		s.flags[msgID][f] = true
	}
}

// AddFlags adds flags to a message.
func (s *FlagStore) AddFlags(msgID domain.ID, flags []imap.Flag) {
	if s.flags[msgID] == nil {
		s.flags[msgID] = make(map[imap.Flag]bool)
	}
	for _, f := range flags {
		s.flags[msgID][f] = true
	}
}

// RemoveFlags removes flags from a message.
func (s *FlagStore) RemoveFlags(msgID domain.ID, flags []imap.Flag) {
	if s.flags[msgID] == nil {
		return
	}
	for _, f := range flags {
		delete(s.flags[msgID], f)
	}
}

// GetFlags returns flags for a message.
func (s *FlagStore) GetFlags(msgID domain.ID) []imap.Flag {
	flagMap := s.flags[msgID]
	if flagMap == nil {
		return nil
	}
	result := make([]imap.Flag, 0, len(flagMap))
	for f := range flagMap {
		result = append(result, f)
	}
	return result
}

// HasFlag checks if a message has a specific flag.
func (s *FlagStore) HasFlag(msgID domain.ID, flag imap.Flag) bool {
	if s.flags[msgID] == nil {
		return false
	}
	return s.flags[msgID][flag]
}

// RFC822 message reconstruction utilities

// reconstructRFC822Message reconstructs an RFC 822 message from domain.Message.
func reconstructRFC822Message(msg *domain.Message) []byte {
	var buf bytes.Buffer

	// Write headers
	writeRFC822Header(&buf, "Date", formatRFC822MessageDate(msg.ReceivedAt.Time))
	writeRFC822Header(&buf, "From", msg.From.String())
	writeRFC822Header(&buf, "Subject", msg.Subject)

	if msg.MessageID != "" {
		writeRFC822Header(&buf, "Message-ID", "<"+msg.MessageID+">")
	}

	if len(msg.To) > 0 {
		writeRFC822Header(&buf, "To", formatAddressListStr(msg.To))
	}

	if len(msg.Cc) > 0 {
		writeRFC822Header(&buf, "Cc", formatAddressListStr(msg.Cc))
	}

	if msg.ReplyTo != nil {
		writeRFC822Header(&buf, "Reply-To", msg.ReplyTo.String())
	}

	if msg.InReplyTo != "" {
		writeRFC822Header(&buf, "In-Reply-To", "<"+msg.InReplyTo+">")
	}

	// Write stored headers (excluding ones we already wrote)
	for name, value := range msg.Headers {
		switch strings.ToLower(name) {
		case "date", "from", "subject", "message-id", "to", "cc", "reply-to", "in-reply-to":
			continue
		}
		writeRFC822Header(&buf, name, value)
	}

	// Content-Type and body
	if msg.HTMLBody != "" && msg.TextBody != "" {
		boundary := "=_mixed_boundary_" + msg.ID.String()
		writeRFC822Header(&buf, "Content-Type", "multipart/alternative; boundary=\""+boundary+"\"")
		writeRFC822Header(&buf, "MIME-Version", "1.0")
		buf.WriteString("\r\n")

		// Text part
		buf.WriteString("--" + boundary + "\r\n")
		buf.WriteString("Content-Type: text/plain; charset=utf-8\r\n")
		buf.WriteString("Content-Transfer-Encoding: 8bit\r\n\r\n")
		buf.WriteString(msg.TextBody)
		buf.WriteString("\r\n")

		// HTML part
		buf.WriteString("--" + boundary + "\r\n")
		buf.WriteString("Content-Type: text/html; charset=utf-8\r\n")
		buf.WriteString("Content-Transfer-Encoding: 8bit\r\n\r\n")
		buf.WriteString(msg.HTMLBody)
		buf.WriteString("\r\n")

		buf.WriteString("--" + boundary + "--\r\n")
	} else if msg.HTMLBody != "" {
		writeRFC822Header(&buf, "Content-Type", "text/html; charset=utf-8")
		writeRFC822Header(&buf, "MIME-Version", "1.0")
		buf.WriteString("\r\n")
		buf.WriteString(msg.HTMLBody)
	} else {
		writeRFC822Header(&buf, "Content-Type", "text/plain; charset=utf-8")
		writeRFC822Header(&buf, "MIME-Version", "1.0")
		buf.WriteString("\r\n")
		buf.WriteString(msg.TextBody)
	}

	return buf.Bytes()
}

// writeRFC822Header writes a single RFC 822 header.
func writeRFC822Header(buf *bytes.Buffer, name, value string) {
	buf.WriteString(name)
	buf.WriteString(": ")
	buf.WriteString(value)
	buf.WriteString("\r\n")
}

// formatRFC822MessageDate formats a time for RFC 822 messages.
func formatRFC822MessageDate(t time.Time) string {
	return t.Format(time.RFC1123Z)
}

// formatAddressListStr formats a list of email addresses as a string.
func formatAddressListStr(addresses []domain.EmailAddress) string {
	parts := make([]string, len(addresses))
	for i, addr := range addresses {
		parts[i] = addr.String()
	}
	return strings.Join(parts, ", ")
}

// MessageIndex provides fast lookup for messages in a mailbox.
type MessageIndex struct {
	bySeqNum map[uint32]*IMAPMessage
	byUID    map[imap.UID]*IMAPMessage
	byID     map[domain.ID]*IMAPMessage
	ordered  []*IMAPMessage
}

// NewMessageIndex creates a new MessageIndex.
func NewMessageIndex() *MessageIndex {
	return &MessageIndex{
		bySeqNum: make(map[uint32]*IMAPMessage),
		byUID:    make(map[imap.UID]*IMAPMessage),
		byID:     make(map[domain.ID]*IMAPMessage),
		ordered:  make([]*IMAPMessage, 0),
	}
}

// Add adds a message to the index.
func (idx *MessageIndex) Add(msg *IMAPMessage) {
	idx.ordered = append(idx.ordered, msg)
	idx.bySeqNum[msg.SeqNum()] = msg
	idx.byUID[msg.UID()] = msg
	idx.byID[msg.Message().ID] = msg
}

// GetBySeqNum returns a message by sequence number.
func (idx *MessageIndex) GetBySeqNum(seqNum uint32) *IMAPMessage {
	return idx.bySeqNum[seqNum]
}

// GetByUID returns a message by UID.
func (idx *MessageIndex) GetByUID(uid imap.UID) *IMAPMessage {
	return idx.byUID[uid]
}

// GetByID returns a message by domain ID.
func (idx *MessageIndex) GetByID(id domain.ID) *IMAPMessage {
	return idx.byID[id]
}

// All returns all messages in order.
func (idx *MessageIndex) All() []*IMAPMessage {
	return idx.ordered
}

// Count returns the number of messages.
func (idx *MessageIndex) Count() int {
	return len(idx.ordered)
}

// Remove removes a message from the index.
func (idx *MessageIndex) Remove(msg *IMAPMessage) {
	delete(idx.bySeqNum, msg.SeqNum())
	delete(idx.byUID, msg.UID())
	delete(idx.byID, msg.Message().ID)

	// Remove from ordered slice
	for i, m := range idx.ordered {
		if m == msg {
			idx.ordered = append(idx.ordered[:i], idx.ordered[i+1:]...)
			break
		}
	}

	// Reindex sequence numbers
	for i, m := range idx.ordered {
		newSeqNum := uint32(i + 1)
		if m.seqNum != newSeqNum {
			delete(idx.bySeqNum, m.seqNum)
			m.seqNum = newSeqNum
			idx.bySeqNum[newSeqNum] = m
		}
	}
}
