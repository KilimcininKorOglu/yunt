// Package domain contains the core domain models and entities for the Yunt mail server.
package domain

// Stats represents aggregated statistics for the mail server.
// It provides a comprehensive overview of system-wide metrics.
type Stats struct {
	// Users contains user-related statistics.
	Users UserStats `json:"users"`

	// Mailboxes contains mailbox-related statistics.
	Mailboxes MailboxAggregateStats `json:"mailboxes"`

	// Messages contains message-related statistics.
	Messages MessageAggregateStats `json:"messages"`

	// Storage contains storage-related statistics.
	Storage StorageStats `json:"storage"`

	// GeneratedAt is the timestamp when these stats were calculated.
	GeneratedAt Timestamp `json:"generatedAt"`
}

// UserStats contains user-related statistics.
type UserStats struct {
	// TotalUsers is the total number of registered users.
	TotalUsers int64 `json:"totalUsers"`

	// ActiveUsers is the number of active users.
	ActiveUsers int64 `json:"activeUsers"`

	// InactiveUsers is the number of inactive users.
	InactiveUsers int64 `json:"inactiveUsers"`

	// PendingUsers is the number of pending users.
	PendingUsers int64 `json:"pendingUsers"`

	// AdminUsers is the number of admin users.
	AdminUsers int64 `json:"adminUsers"`
}

// MailboxAggregateStats contains mailbox-related aggregate statistics.
type MailboxAggregateStats struct {
	// TotalMailboxes is the total number of mailboxes.
	TotalMailboxes int64 `json:"totalMailboxes"`

	// ActiveMailboxes is the number of mailboxes with messages.
	ActiveMailboxes int64 `json:"activeMailboxes"`

	// EmptyMailboxes is the number of mailboxes without messages.
	EmptyMailboxes int64 `json:"emptyMailboxes"`

	// CatchAllMailboxes is the number of catch-all mailboxes.
	CatchAllMailboxes int64 `json:"catchAllMailboxes"`

	// DefaultMailboxes is the number of default mailboxes.
	DefaultMailboxes int64 `json:"defaultMailboxes"`

	// UniqueDomains is the number of unique email domains.
	UniqueDomains int64 `json:"uniqueDomains"`

	// AverageMessagesPerMailbox is the average messages per mailbox.
	AverageMessagesPerMailbox float64 `json:"averageMessagesPerMailbox"`
}

// MessageAggregateStats contains message-related aggregate statistics.
type MessageAggregateStats struct {
	// TotalMessages is the total number of messages.
	TotalMessages int64 `json:"totalMessages"`

	// UnreadMessages is the number of unread messages.
	UnreadMessages int64 `json:"unreadMessages"`

	// ReadMessages is the number of read messages.
	ReadMessages int64 `json:"readMessages"`

	// StarredMessages is the number of starred messages.
	StarredMessages int64 `json:"starredMessages"`

	// SpamMessages is the number of spam messages.
	SpamMessages int64 `json:"spamMessages"`

	// MessagesWithAttachments is the number of messages with attachments.
	MessagesWithAttachments int64 `json:"messagesWithAttachments"`

	// TotalAttachments is the total number of attachments.
	TotalAttachments int64 `json:"totalAttachments"`

	// AverageMessageSize is the average message size in bytes.
	AverageMessageSize float64 `json:"averageMessageSize"`

	// LargestMessage is the size of the largest message in bytes.
	LargestMessage int64 `json:"largestMessage"`
}

// StorageStats contains storage-related statistics.
type StorageStats struct {
	// TotalSize is the total size of all messages in bytes.
	TotalSize int64 `json:"totalSize"`

	// MessageStorageSize is the size used by messages in bytes.
	MessageStorageSize int64 `json:"messageStorageSize"`

	// AttachmentStorageSize is the size used by attachments in bytes.
	AttachmentStorageSize int64 `json:"attachmentStorageSize"`

	// AverageMailboxSize is the average mailbox size in bytes.
	AverageMailboxSize float64 `json:"averageMailboxSize"`

	// LargestMailboxSize is the size of the largest mailbox in bytes.
	LargestMailboxSize int64 `json:"largestMailboxSize"`
}

// MessageStats contains statistics for a specific message set.
// This is used for filtered queries, e.g., messages in a date range.
type MessageStats struct {
	// Count is the number of messages in the set.
	Count int64 `json:"count"`

	// UnreadCount is the number of unread messages.
	UnreadCount int64 `json:"unreadCount"`

	// StarredCount is the number of starred messages.
	StarredCount int64 `json:"starredCount"`

	// SpamCount is the number of spam messages.
	SpamCount int64 `json:"spamCount"`

	// TotalSize is the total size of messages in bytes.
	TotalSize int64 `json:"totalSize"`

	// AttachmentCount is the total number of attachments.
	AttachmentCount int64 `json:"attachmentCount"`

	// OldestMessage is the timestamp of the oldest message.
	OldestMessage *Timestamp `json:"oldestMessage,omitempty"`

	// NewestMessage is the timestamp of the newest message.
	NewestMessage *Timestamp `json:"newestMessage,omitempty"`
}

// DailyStats contains statistics for a specific day.
type DailyStats struct {
	// Date is the date for these statistics.
	Date string `json:"date"`

	// ReceivedCount is the number of messages received.
	ReceivedCount int64 `json:"receivedCount"`

	// TotalSize is the total size of messages received in bytes.
	TotalSize int64 `json:"totalSize"`

	// SpamCount is the number of spam messages received.
	SpamCount int64 `json:"spamCount"`

	// AttachmentCount is the number of attachments received.
	AttachmentCount int64 `json:"attachmentCount"`

	// UniqueRecipients is the number of unique recipient addresses.
	UniqueRecipients int64 `json:"uniqueRecipients"`

	// UniqueSenders is the number of unique sender addresses.
	UniqueSenders int64 `json:"uniqueSenders"`
}

// SenderStats contains statistics for a specific sender.
type SenderStats struct {
	// Address is the sender's email address.
	Address string `json:"address"`

	// Name is the sender's display name.
	Name string `json:"name,omitempty"`

	// MessageCount is the number of messages from this sender.
	MessageCount int64 `json:"messageCount"`

	// TotalSize is the total size of messages from this sender.
	TotalSize int64 `json:"totalSize"`

	// SpamCount is the number of spam messages from this sender.
	SpamCount int64 `json:"spamCount"`

	// FirstSeen is when the first message from this sender was received.
	FirstSeen *Timestamp `json:"firstSeen,omitempty"`

	// LastSeen is when the last message from this sender was received.
	LastSeen *Timestamp `json:"lastSeen,omitempty"`
}

// RecipientStats contains statistics for a specific recipient.
type RecipientStats struct {
	// Address is the recipient's email address.
	Address string `json:"address"`

	// Name is the recipient's display name.
	Name string `json:"name,omitempty"`

	// MessageCount is the number of messages to this recipient.
	MessageCount int64 `json:"messageCount"`

	// Type is the recipient type (to, cc, bcc).
	Type string `json:"type,omitempty"`
}

// ContentTypeStats contains statistics by content type.
type ContentTypeStats struct {
	// ContentType is the MIME content type.
	ContentType string `json:"contentType"`

	// Count is the number of items with this content type.
	Count int64 `json:"count"`

	// TotalSize is the total size of items in bytes.
	TotalSize int64 `json:"totalSize"`
}

// StatsFilter provides filtering options for statistics queries.
type StatsFilter struct {
	// UserID filters statistics for a specific user.
	UserID *ID `json:"userId,omitempty"`

	// MailboxID filters statistics for a specific mailbox.
	MailboxID *ID `json:"mailboxId,omitempty"`

	// MailboxIDs filters statistics for multiple mailboxes.
	MailboxIDs []ID `json:"mailboxIds,omitempty"`

	// DateFrom filters messages received on or after this date.
	DateFrom *Timestamp `json:"dateFrom,omitempty"`

	// DateTo filters messages received on or before this date.
	DateTo *Timestamp `json:"dateTo,omitempty"`

	// ExcludeSpam excludes spam messages from statistics.
	ExcludeSpam bool `json:"excludeSpam,omitempty"`
}

// StatsTimeRange represents a time range for historical statistics.
type StatsTimeRange string

const (
	// StatsTimeRange24Hours is the last 24 hours.
	StatsTimeRange24Hours StatsTimeRange = "24h"
	// StatsTimeRange7Days is the last 7 days.
	StatsTimeRange7Days StatsTimeRange = "7d"
	// StatsTimeRange30Days is the last 30 days.
	StatsTimeRange30Days StatsTimeRange = "30d"
	// StatsTimeRange90Days is the last 90 days.
	StatsTimeRange90Days StatsTimeRange = "90d"
	// StatsTimeRangeAll is all time.
	StatsTimeRangeAll StatsTimeRange = "all"
)

// IsValid returns true if the time range is a recognized value.
func (r StatsTimeRange) IsValid() bool {
	switch r {
	case StatsTimeRange24Hours, StatsTimeRange7Days, StatsTimeRange30Days,
		StatsTimeRange90Days, StatsTimeRangeAll:
		return true
	default:
		return false
	}
}

// TrendData contains data points for trend analysis.
type TrendData struct {
	// Labels contains the time labels for each data point.
	Labels []string `json:"labels"`

	// MessageCounts contains the message count for each time period.
	MessageCounts []int64 `json:"messageCounts"`

	// SizeTotals contains the total size for each time period in bytes.
	SizeTotals []int64 `json:"sizeTotals"`
}

// NewStats creates a new Stats instance with current timestamp.
func NewStats() *Stats {
	return &Stats{
		GeneratedAt: Now(),
	}
}

// NewMessageStats creates a new MessageStats instance.
func NewMessageStats() *MessageStats {
	return &MessageStats{}
}

// IsEmpty returns true if the MessageStats has no messages.
func (s *MessageStats) IsEmpty() bool {
	return s.Count == 0
}

// ReadPercentage returns the percentage of read messages.
func (s *MessageStats) ReadPercentage() float64 {
	if s.Count == 0 {
		return 0
	}
	readCount := s.Count - s.UnreadCount
	return float64(readCount) / float64(s.Count) * 100
}

// UnreadPercentage returns the percentage of unread messages.
func (s *MessageStats) UnreadPercentage() float64 {
	if s.Count == 0 {
		return 0
	}
	return float64(s.UnreadCount) / float64(s.Count) * 100
}

// SpamPercentage returns the percentage of spam messages.
func (s *MessageStats) SpamPercentage() float64 {
	if s.Count == 0 {
		return 0
	}
	return float64(s.SpamCount) / float64(s.Count) * 100
}

// AverageSize returns the average message size in bytes.
func (s *MessageStats) AverageSize() float64 {
	if s.Count == 0 {
		return 0
	}
	return float64(s.TotalSize) / float64(s.Count)
}

// FormatTotalSize returns the total size in a human-readable format.
func (s *MessageStats) FormatTotalSize() string {
	return FormatBytes(s.TotalSize)
}

// FormatBytes converts bytes to a human-readable string.
func FormatBytes(bytes int64) string {
	const (
		kb = 1024
		mb = kb * 1024
		gb = mb * 1024
		tb = gb * 1024
	)

	switch {
	case bytes >= tb:
		return formatStatsFloat(float64(bytes)/tb) + " TB"
	case bytes >= gb:
		return formatStatsFloat(float64(bytes)/gb) + " GB"
	case bytes >= mb:
		return formatStatsFloat(float64(bytes)/mb) + " MB"
	case bytes >= kb:
		return formatStatsFloat(float64(bytes)/kb) + " KB"
	default:
		return formatStatsInt(bytes) + " B"
	}
}

// formatStatsFloat formats a float with up to 2 decimal places, removing trailing zeros.
func formatStatsFloat(f float64) string {
	if f >= 100 {
		return formatStatsInt(int64(f))
	} else if f >= 10 {
		return formatStatsFloatWithPrecision(f, 1)
	}
	return formatStatsFloatWithPrecision(f, 2)
}

// formatStatsFloatWithPrecision formats a float with specified decimal places.
func formatStatsFloatWithPrecision(f float64, precision int) string {
	intPart := int64(f)
	var decPart int64
	var result string

	switch precision {
	case 1:
		decPart = int64((f - float64(intPart)) * 10)
		if decPart < 0 {
			decPart = -decPart
		}
		if decPart == 0 {
			return formatStatsInt(intPart)
		}
		result = formatStatsInt(intPart) + "." + formatStatsDigit(decPart)
	case 2:
		decPart = int64((f-float64(intPart))*100 + 0.5)
		if decPart < 0 {
			decPart = -decPart
		}
		if decPart == 0 {
			return formatStatsInt(intPart)
		}
		d1 := decPart / 10
		d2 := decPart % 10
		if d2 == 0 {
			result = formatStatsInt(intPart) + "." + formatStatsDigit(d1)
		} else {
			result = formatStatsInt(intPart) + "." + formatStatsDigit(d1) + formatStatsDigit(d2)
		}
	default:
		return formatStatsInt(intPart)
	}

	return result
}

// formatStatsDigit formats a single digit.
func formatStatsDigit(d int64) string {
	digits := "0123456789"
	if d >= 0 && d <= 9 {
		return string(digits[d])
	}
	return "0"
}

// formatStatsInt formats an integer as a string.
func formatStatsInt(i int64) string {
	if i == 0 {
		return "0"
	}

	neg := i < 0
	if neg {
		i = -i
	}

	var result []byte
	for i > 0 {
		result = append([]byte{byte('0' + i%10)}, result...)
		i /= 10
	}

	if neg {
		result = append([]byte{'-'}, result...)
	}

	return string(result)
}
