// Package repository provides the data access layer interfaces for the Yunt mail server.
// This file contains search-related types and utilities for IMAP SEARCH functionality.
package repository

import (
	"time"

	"yunt/internal/domain"
)

// IMAPSearchCriteria represents criteria for IMAP SEARCH operations.
// This provides a high-level interface for IMAP search that can be
// converted to MessageFilter for database queries.
type IMAPSearchCriteria struct {
	// MailboxID is the ID of the mailbox to search within.
	MailboxID domain.ID

	// All indicates to return all messages (no filtering).
	All bool

	// Flags filters messages that have all specified flags set.
	Flags []string

	// NotFlags filters messages that do not have any of the specified flags.
	NotFlags []string

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

	// Larger filters messages larger than the specified size in bytes.
	Larger int64

	// Smaller filters messages smaller than the specified size in bytes.
	Smaller int64
}

// ToMessageFilter converts IMAPSearchCriteria to MessageFilter for database queries.
// Note: Some criteria (Body, Text, complex NOT/OR) may require post-filtering.
func (c *IMAPSearchCriteria) ToMessageFilter() *MessageFilter {
	filter := &MessageFilter{
		MailboxID: &c.MailboxID,
	}

	// Date filters
	if c.Since != nil {
		ts := domain.Timestamp{Time: *c.Since}
		filter.ReceivedAfter = &ts
	}
	if c.Before != nil {
		ts := domain.Timestamp{Time: *c.Before}
		filter.ReceivedBefore = &ts
	}
	if c.SentSince != nil {
		ts := domain.Timestamp{Time: *c.SentSince}
		filter.SentAfter = &ts
	}
	if c.SentBefore != nil {
		ts := domain.Timestamp{Time: *c.SentBefore}
		filter.SentBefore = &ts
	}

	// Size filters
	if c.Larger > 0 {
		filter.MinSize = &c.Larger
	}
	if c.Smaller > 0 {
		filter.MaxSize = &c.Smaller
	}

	// Header filters
	if c.From != "" {
		filter.FromAddressContains = c.From
	}
	if c.To != "" {
		filter.ToAddressContains = c.To
	}
	if c.Subject != "" {
		filter.SubjectContains = c.Subject
	}

	// Flag filters
	for _, flag := range c.Flags {
		switch flag {
		case "\\Seen":
			status := domain.MessageRead
			filter.Status = &status
		case "\\Flagged":
			starred := true
			filter.IsStarred = &starred
		case "$Junk", "\\Junk":
			spam := true
			filter.IsSpam = &spam
		}
	}

	for _, flag := range c.NotFlags {
		switch flag {
		case "\\Seen":
			status := domain.MessageUnread
			filter.Status = &status
		case "\\Flagged":
			starred := false
			filter.IsStarred = &starred
		case "$Junk", "\\Junk":
			spam := false
			filter.IsSpam = &spam
		}
	}

	// Body search - requires full-text search
	if len(c.Body) > 0 {
		// Use the first body search term as the main search
		filter.BodyContains = c.Body[0]
	}

	// Text search - searches in both headers and body
	if len(c.Text) > 0 {
		// Use the first text search term as the main search
		filter.Search = c.Text[0]
	}

	return filter
}

// NeedsPostFiltering returns true if the criteria requires post-filtering
// after the initial database query.
func (c *IMAPSearchCriteria) NeedsPostFiltering() bool {
	// Multiple body or text searches need post-filtering
	if len(c.Body) > 1 || len(c.Text) > 1 {
		return true
	}

	// Cc and Bcc filters need post-filtering (not in standard MessageFilter)
	if c.Cc != "" || c.Bcc != "" {
		return true
	}

	return false
}

// SearchResult represents the result of an IMAP search operation.
type SearchResult struct {
	// MessageIDs contains the IDs of matching messages.
	MessageIDs []domain.ID

	// SequenceNumbers contains the sequence numbers of matching messages.
	SequenceNumbers []uint32

	// UIDs contains the UIDs of matching messages.
	UIDs []uint32

	// Total is the total count of matching messages.
	Total int64
}
