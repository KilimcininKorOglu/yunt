package imap

import (
	"strings"
	"time"

	"github.com/emersion/go-imap/v2"

	"yunt/internal/domain"
)

// SearchCriteria represents the internal search criteria for IMAP SEARCH.
// It is converted from imap.SearchCriteria and used to query the repository.
type SearchCriteria struct {
	// All indicates to return all messages (no filtering).
	All bool

	// SeqNums filters by sequence number ranges.
	SeqNums []imap.SeqSet

	// UIDs filters by UID ranges.
	UIDs []imap.UIDSet

	// Since filters messages with INTERNALDATE on or after the specified date.
	Since *time.Time

	// Before filters messages with INTERNALDATE before the specified date.
	Before *time.Time

	// SentSince filters messages with Date header on or after the specified date.
	SentSince *time.Time

	// SentBefore filters messages with Date header before the specified date.
	SentBefore *time.Time

	// From filters messages with the specified string in the From header.
	From string

	// To filters messages with the specified string in the To header.
	To string

	// Cc filters messages with the specified string in the Cc header.
	Cc string

	// Bcc filters messages with the specified string in the Bcc header.
	Bcc string

	// Subject filters messages with the specified string in the Subject header.
	Subject string

	// Body filters messages with the specified string in the body.
	Body []string

	// Text filters messages with the specified string in headers or body.
	Text []string

	// Header filters messages with the specified header field containing value.
	Header []HeaderCriterion

	// Flags filters messages that have all specified flags set.
	Flags []imap.Flag

	// NotFlags filters messages that do not have any of the specified flags.
	NotFlags []imap.Flag

	// Larger filters messages larger than the specified size in bytes.
	Larger int64

	// Smaller filters messages smaller than the specified size in bytes.
	Smaller int64

	// Not represents negated search criteria (NOT operator).
	Not []*SearchCriteria

	// Or represents OR-combined criteria pairs.
	Or [][2]*SearchCriteria
}

// HeaderCriterion represents a header search criterion.
type HeaderCriterion struct {
	Key   string
	Value string
}

// SearchCriteriaParser parses IMAP SearchCriteria into internal SearchCriteria.
type SearchCriteriaParser struct{}

// NewSearchCriteriaParser creates a new SearchCriteriaParser.
func NewSearchCriteriaParser() *SearchCriteriaParser {
	return &SearchCriteriaParser{}
}

// Parse converts imap.SearchCriteria to internal SearchCriteria.
func (p *SearchCriteriaParser) Parse(criteria *imap.SearchCriteria) *SearchCriteria {
	if criteria == nil {
		return &SearchCriteria{All: true}
	}

	result := &SearchCriteria{}

	// Copy sequence number sets
	if len(criteria.SeqNum) > 0 {
		result.SeqNums = criteria.SeqNum
	}

	// Copy UID sets
	if len(criteria.UID) > 0 {
		result.UIDs = criteria.UID
	}

	// Date criteria
	if !criteria.Since.IsZero() {
		t := normalizeDate(criteria.Since)
		result.Since = &t
	}
	if !criteria.Before.IsZero() {
		t := normalizeDate(criteria.Before)
		result.Before = &t
	}
	if !criteria.SentSince.IsZero() {
		t := normalizeDate(criteria.SentSince)
		result.SentSince = &t
	}
	if !criteria.SentBefore.IsZero() {
		t := normalizeDate(criteria.SentBefore)
		result.SentBefore = &t
	}

	// Header criteria
	for _, hdr := range criteria.Header {
		p.processHeaderCriterion(result, hdr.Key, hdr.Value)
	}

	// Body and text searches
	result.Body = append(result.Body, criteria.Body...)
	result.Text = append(result.Text, criteria.Text...)

	// Flags
	result.Flags = append(result.Flags, criteria.Flag...)
	result.NotFlags = append(result.NotFlags, criteria.NotFlag...)

	// Size criteria
	result.Larger = criteria.Larger
	result.Smaller = criteria.Smaller

	// NOT criteria
	for _, notCriteria := range criteria.Not {
		parsed := p.Parse(&notCriteria)
		result.Not = append(result.Not, parsed)
	}

	// OR criteria
	for _, orPair := range criteria.Or {
		left := p.Parse(&orPair[0])
		right := p.Parse(&orPair[1])
		result.Or = append(result.Or, [2]*SearchCriteria{left, right})
	}

	// Check if this is essentially an "ALL" search
	if result.isEmpty() {
		result.All = true
	}

	return result
}

// processHeaderCriterion processes a header search criterion.
// Common headers are mapped to dedicated fields for efficiency.
func (p *SearchCriteriaParser) processHeaderCriterion(result *SearchCriteria, key, value string) {
	keyLower := strings.ToLower(key)

	switch keyLower {
	case "from":
		if value != "" {
			result.From = value
		}
	case "to":
		if value != "" {
			result.To = value
		}
	case "cc":
		if value != "" {
			result.Cc = value
		}
	case "bcc":
		if value != "" {
			result.Bcc = value
		}
	case "subject":
		if value != "" {
			result.Subject = value
		}
	default:
		// Generic header search
		result.Header = append(result.Header, HeaderCriterion{
			Key:   key,
			Value: value,
		})
	}
}

// isEmpty checks if the search criteria has no actual filtering conditions.
func (c *SearchCriteria) isEmpty() bool {
	return len(c.SeqNums) == 0 &&
		len(c.UIDs) == 0 &&
		c.Since == nil &&
		c.Before == nil &&
		c.SentSince == nil &&
		c.SentBefore == nil &&
		c.From == "" &&
		c.To == "" &&
		c.Cc == "" &&
		c.Bcc == "" &&
		c.Subject == "" &&
		len(c.Body) == 0 &&
		len(c.Text) == 0 &&
		len(c.Header) == 0 &&
		len(c.Flags) == 0 &&
		len(c.NotFlags) == 0 &&
		c.Larger == 0 &&
		c.Smaller == 0 &&
		len(c.Not) == 0 &&
		len(c.Or) == 0
}

// normalizeDate normalizes a date to midnight UTC for date comparisons.
// IMAP date criteria use only the date portion, not time.
func normalizeDate(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC)
}

// MessageMatcher checks if messages match search criteria.
type MessageMatcher struct {
	criteria *SearchCriteria
}

// NewMessageMatcher creates a new MessageMatcher for the given criteria.
func NewMessageMatcher(criteria *SearchCriteria) *MessageMatcher {
	return &MessageMatcher{criteria: criteria}
}

// Matches checks if a message matches the search criteria.
func (m *MessageMatcher) Matches(msg *domain.Message, seqNum uint32, uid imap.UID) bool {
	if m.criteria.All {
		return true
	}

	// Check sequence number constraints
	if len(m.criteria.SeqNums) > 0 {
		matched := false
		for _, seqSet := range m.criteria.SeqNums {
			if seqSet.Contains(seqNum) {
				matched = true
				break
			}
		}
		if !matched {
			return false
		}
	}

	// Check UID constraints
	if len(m.criteria.UIDs) > 0 {
		matched := false
		for _, uidSet := range m.criteria.UIDs {
			if uidSet.Contains(uid) {
				matched = true
				break
			}
		}
		if !matched {
			return false
		}
	}

	// Check date constraints (INTERNALDATE)
	if m.criteria.Since != nil {
		msgDate := normalizeDate(msg.ReceivedAt.Time)
		if msgDate.Before(*m.criteria.Since) {
			return false
		}
	}
	if m.criteria.Before != nil {
		msgDate := normalizeDate(msg.ReceivedAt.Time)
		if !msgDate.Before(*m.criteria.Before) {
			return false
		}
	}

	// Check sent date constraints (Date header)
	if m.criteria.SentSince != nil {
		if msg.SentAt == nil {
			return false
		}
		msgDate := normalizeDate(msg.SentAt.Time)
		if msgDate.Before(*m.criteria.SentSince) {
			return false
		}
	}
	if m.criteria.SentBefore != nil {
		if msg.SentAt == nil {
			return false
		}
		msgDate := normalizeDate(msg.SentAt.Time)
		if !msgDate.Before(*m.criteria.SentBefore) {
			return false
		}
	}

	// Check From
	if m.criteria.From != "" {
		if !containsIgnoreCase(msg.From.String(), m.criteria.From) {
			return false
		}
	}

	// Check To
	if m.criteria.To != "" {
		matched := false
		for _, addr := range msg.To {
			if containsIgnoreCase(addr.String(), m.criteria.To) {
				matched = true
				break
			}
		}
		if !matched {
			return false
		}
	}

	// Check Cc
	if m.criteria.Cc != "" {
		matched := false
		for _, addr := range msg.Cc {
			if containsIgnoreCase(addr.String(), m.criteria.Cc) {
				matched = true
				break
			}
		}
		if !matched {
			return false
		}
	}

	// Check Bcc
	if m.criteria.Bcc != "" {
		matched := false
		for _, addr := range msg.Bcc {
			if containsIgnoreCase(addr.String(), m.criteria.Bcc) {
				matched = true
				break
			}
		}
		if !matched {
			return false
		}
	}

	// Check Subject
	if m.criteria.Subject != "" {
		if !containsIgnoreCase(msg.Subject, m.criteria.Subject) {
			return false
		}
	}

	// Check Body
	for _, bodySearch := range m.criteria.Body {
		if !m.matchBody(msg, bodySearch) {
			return false
		}
	}

	// Check Text (headers + body)
	for _, textSearch := range m.criteria.Text {
		if !m.matchText(msg, textSearch) {
			return false
		}
	}

	// Check generic headers
	for _, hdrCrit := range m.criteria.Header {
		if !m.matchHeader(msg, hdrCrit) {
			return false
		}
	}

	// Check flags
	msgFlags := m.getMessageFlags(msg)
	for _, flag := range m.criteria.Flags {
		if !hasFlag(msgFlags, flag) {
			return false
		}
	}
	for _, flag := range m.criteria.NotFlags {
		if hasFlag(msgFlags, flag) {
			return false
		}
	}

	// Check size constraints
	if m.criteria.Larger > 0 && msg.Size <= m.criteria.Larger {
		return false
	}
	if m.criteria.Smaller > 0 && msg.Size >= m.criteria.Smaller {
		return false
	}

	// Check NOT criteria (all must NOT match)
	for _, notCriteria := range m.criteria.Not {
		notMatcher := NewMessageMatcher(notCriteria)
		if notMatcher.Matches(msg, seqNum, uid) {
			return false
		}
	}

	// Check OR criteria (at least one pair must have one side match)
	for _, orPair := range m.criteria.Or {
		leftMatcher := NewMessageMatcher(orPair[0])
		rightMatcher := NewMessageMatcher(orPair[1])
		if !leftMatcher.Matches(msg, seqNum, uid) && !rightMatcher.Matches(msg, seqNum, uid) {
			return false
		}
	}

	return true
}

// matchBody checks if the message body contains the search string.
func (m *MessageMatcher) matchBody(msg *domain.Message, search string) bool {
	// Search in text body
	if containsIgnoreCase(msg.TextBody, search) {
		return true
	}
	// Search in HTML body
	if containsIgnoreCase(msg.HTMLBody, search) {
		return true
	}
	return false
}

// matchText checks if the message (headers + body) contains the search string.
func (m *MessageMatcher) matchText(msg *domain.Message, search string) bool {
	// Check subject
	if containsIgnoreCase(msg.Subject, search) {
		return true
	}
	// Check from
	if containsIgnoreCase(msg.From.String(), search) {
		return true
	}
	// Check to addresses
	for _, addr := range msg.To {
		if containsIgnoreCase(addr.String(), search) {
			return true
		}
	}
	// Check cc addresses
	for _, addr := range msg.Cc {
		if containsIgnoreCase(addr.String(), search) {
			return true
		}
	}
	// Check custom headers
	for _, value := range msg.Headers {
		if containsIgnoreCase(value, search) {
			return true
		}
	}
	// Check body
	return m.matchBody(msg, search)
}

// matchHeader checks if a header matches the criterion.
func (m *MessageMatcher) matchHeader(msg *domain.Message, crit HeaderCriterion) bool {
	headerValue := msg.GetHeader(crit.Key)

	// If no value specified, just check if header exists
	if crit.Value == "" {
		return headerValue != ""
	}

	return containsIgnoreCase(headerValue, crit.Value)
}

// getMessageFlags returns the IMAP flags for a message using the canonical FlagSet.
func (m *MessageMatcher) getMessageFlags(msg *domain.Message) []imap.Flag {
	return NewFlagSetFromMessage(msg).ToSlice()
}

// containsIgnoreCase checks if s contains substr (case-insensitive).
func containsIgnoreCase(s, substr string) bool {
	return strings.Contains(strings.ToLower(s), strings.ToLower(substr))
}

// hasFlag checks if the flag is in the flags slice.
func hasFlag(flags []imap.Flag, flag imap.Flag) bool {
	for _, f := range flags {
		if f == flag {
			return true
		}
	}
	return false
}
