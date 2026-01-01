package imap

import (
	"strings"

	"github.com/emersion/go-imap/v2"

	"yunt/internal/domain"
)

// MailboxHierarchySeparator is the character used to separate mailbox hierarchy levels.
const MailboxHierarchySeparator = "/"

// SystemMailboxNames defines the standard system mailbox names.
// These mailboxes are created automatically for each user and cannot be deleted.
var SystemMailboxNames = []string{
	"INBOX",
	"Sent",
	"Drafts",
	"Trash",
	"Spam",
}

// SystemMailboxNameMap provides O(1) lookup for system mailbox names.
var SystemMailboxNameMap = map[string]bool{
	"INBOX":  true,
	"Sent":   true,
	"Drafts": true,
	"Trash":  true,
	"Spam":   true,
}

// IsSystemMailbox returns true if the mailbox name is a system mailbox.
// The comparison is case-insensitive for INBOX (per RFC 3501) and case-sensitive for others.
func IsSystemMailbox(name string) bool {
	// INBOX is case-insensitive per RFC 3501
	if strings.EqualFold(name, "INBOX") {
		return true
	}
	return SystemMailboxNameMap[name]
}

// NormalizeMailboxName normalizes a mailbox name according to IMAP conventions.
// INBOX is always returned as uppercase, other names are returned as-is.
func NormalizeMailboxName(name string) string {
	if strings.EqualFold(name, "INBOX") {
		return "INBOX"
	}
	return name
}

// MailboxInfo contains information about a mailbox for IMAP responses.
type MailboxInfo struct {
	// Name is the mailbox name.
	Name string

	// Delimiter is the hierarchy delimiter.
	Delimiter string

	// Attributes are the mailbox attributes (flags).
	Attributes []imap.MailboxAttr

	// IsSystem indicates if this is a system mailbox.
	IsSystem bool
}

// NewMailboxInfo creates a new MailboxInfo from a domain Mailbox.
func NewMailboxInfo(mailbox *domain.Mailbox) *MailboxInfo {
	name := NormalizeMailboxName(mailbox.Name)
	attrs := getMailboxAttributes(name, mailbox)

	return &MailboxInfo{
		Name:       name,
		Delimiter:  MailboxHierarchySeparator,
		Attributes: attrs,
		IsSystem:   IsSystemMailbox(name),
	}
}

// ToIMAPListData converts MailboxInfo to IMAP ListData.
func (m *MailboxInfo) ToIMAPListData() *imap.ListData {
	return &imap.ListData{
		Mailbox: m.Name,
		Delim:   rune(m.Delimiter[0]),
		Attrs:   m.Attributes,
	}
}

// getMailboxAttributes returns the appropriate IMAP attributes for a mailbox.
func getMailboxAttributes(name string, mailbox *domain.Mailbox) []imap.MailboxAttr {
	var attrs []imap.MailboxAttr

	// Add special-use attributes for system mailboxes
	switch {
	case strings.EqualFold(name, "INBOX"):
		// INBOX has no special attribute, it's implied
	case name == "Sent":
		attrs = append(attrs, imap.MailboxAttrSent)
	case name == "Drafts":
		attrs = append(attrs, imap.MailboxAttrDrafts)
	case name == "Trash":
		attrs = append(attrs, imap.MailboxAttrTrash)
	case name == "Spam":
		attrs = append(attrs, imap.MailboxAttrJunk)
	}

	// Check if mailbox has children
	// This would require a separate query in practice; for now, we don't set it
	// Parent mailboxes can be marked with \HasChildren or \HasNoChildren

	return attrs
}

// MailboxStatus contains status information for a mailbox.
type MailboxStatus struct {
	// Name is the mailbox name.
	Name string

	// Messages is the total number of messages.
	Messages uint32

	// Recent is the number of recent messages.
	Recent uint32

	// Unseen is the number of unseen (unread) messages.
	Unseen uint32

	// UIDNext is the predicted next UID.
	UIDNext imap.UID

	// UIDValidity is the UID validity value.
	UIDValidity uint32

	// Size is the total size of all messages in bytes.
	Size int64
}

// NewMailboxStatus creates a MailboxStatus from a domain Mailbox.
func NewMailboxStatus(mailbox *domain.Mailbox) *MailboxStatus {
	return &MailboxStatus{
		Name:        NormalizeMailboxName(mailbox.Name),
		Messages:    uint32(mailbox.MessageCount),
		Recent:      0, // Recent flag handling is not implemented yet
		Unseen:      uint32(mailbox.UnreadCount),
		UIDNext:     imap.UID(mailbox.MessageCount + 1), // Simplified: next UID is count + 1
		UIDValidity: generateUIDValidity(mailbox),
		Size:        mailbox.TotalSize,
	}
}

// ToIMAPStatusData converts MailboxStatus to IMAP StatusData.
func (s *MailboxStatus) ToIMAPStatusData(options *imap.StatusOptions) *imap.StatusData {
	data := &imap.StatusData{
		Mailbox: s.Name,
	}

	if options == nil {
		// Return all status items if no options specified
		numMessages := s.Messages
		data.NumMessages = &numMessages
		unseen := s.Unseen
		data.NumUnseen = &unseen
		uidNext := s.UIDNext
		data.UIDNext = uidNext
		uidValidity := s.UIDValidity
		data.UIDValidity = uidValidity
		return data
	}

	// Return only requested items
	if options.NumMessages {
		numMessages := s.Messages
		data.NumMessages = &numMessages
	}
	if options.NumUnseen {
		unseen := s.Unseen
		data.NumUnseen = &unseen
	}
	if options.UIDNext {
		data.UIDNext = s.UIDNext
	}
	if options.UIDValidity {
		data.UIDValidity = s.UIDValidity
	}

	return data
}

// generateUIDValidity generates a UID validity value for a mailbox.
// This is based on the mailbox creation time for consistency.
func generateUIDValidity(mailbox *domain.Mailbox) uint32 {
	// Use the lower 32 bits of the creation timestamp as UID validity
	// This ensures the value is stable but changes if the mailbox is recreated
	return uint32(mailbox.CreatedAt.Unix() & 0xFFFFFFFF)
}

// SelectData contains the result of selecting a mailbox.
type SelectData struct {
	// Flags are the flags applicable to messages in this mailbox.
	Flags []imap.Flag

	// PermanentFlags are the flags that can be permanently stored.
	PermanentFlags []imap.Flag

	// NumMessages is the total number of messages.
	NumMessages uint32

	// NumRecent is the number of recent messages.
	NumRecent uint32

	// FirstUnseen is the sequence number of the first unseen message.
	FirstUnseen uint32

	// UIDNext is the predicted next UID.
	UIDNext imap.UID

	// UIDValidity is the UID validity value.
	UIDValidity uint32
}

// NewSelectData creates SelectData from a domain Mailbox.
func NewSelectData(mailbox *domain.Mailbox) *SelectData {
	return &SelectData{
		Flags: []imap.Flag{
			imap.FlagSeen,
			imap.FlagAnswered,
			imap.FlagFlagged,
			imap.FlagDeleted,
			imap.FlagDraft,
		},
		PermanentFlags: []imap.Flag{
			imap.FlagSeen,
			imap.FlagAnswered,
			imap.FlagFlagged,
			imap.FlagDeleted,
			imap.FlagDraft,
			imap.FlagWildcard, // Indicates custom flags are allowed
		},
		NumMessages: uint32(mailbox.MessageCount),
		NumRecent:   0, // Recent flag handling is not implemented yet
		FirstUnseen: 1, // Simplified: first message is unseen if there are unseen messages
		UIDNext:     imap.UID(mailbox.MessageCount + 1),
		UIDValidity: generateUIDValidity(mailbox),
	}
}

// ToIMAPSelectData converts SelectData to IMAP SelectData.
func (s *SelectData) ToIMAPSelectData() *imap.SelectData {
	return &imap.SelectData{
		Flags:          s.Flags,
		PermanentFlags: s.PermanentFlags,
		NumMessages:    s.NumMessages,
		UIDNext:        s.UIDNext,
		UIDValidity:    s.UIDValidity,
	}
}

// MailboxPath represents a parsed mailbox path with hierarchy information.
type MailboxPath struct {
	// Full is the full mailbox path.
	Full string

	// Parts are the individual path components.
	Parts []string

	// Parent is the parent mailbox path (empty if no parent).
	Parent string

	// Name is the final component of the path (the mailbox name itself).
	Name string
}

// ParseMailboxPath parses a mailbox path into its components.
func ParseMailboxPath(path string) *MailboxPath {
	normalized := NormalizeMailboxName(path)
	parts := strings.Split(normalized, MailboxHierarchySeparator)

	mp := &MailboxPath{
		Full:  normalized,
		Parts: parts,
		Name:  parts[len(parts)-1],
	}

	if len(parts) > 1 {
		mp.Parent = strings.Join(parts[:len(parts)-1], MailboxHierarchySeparator)
	}

	return mp
}

// IsChildOf returns true if this path is a child of the given parent path.
func (p *MailboxPath) IsChildOf(parent string) bool {
	if parent == "" {
		return true // Root is parent of everything
	}
	parentNorm := NormalizeMailboxName(parent)
	return strings.HasPrefix(p.Full, parentNorm+MailboxHierarchySeparator)
}

// Depth returns the depth of this mailbox in the hierarchy (0 = root level).
func (p *MailboxPath) Depth() int {
	return len(p.Parts) - 1
}

// ValidateMailboxName validates a mailbox name for creation/renaming.
// Returns an error if the name is invalid.
func ValidateMailboxName(name string) error {
	if name == "" {
		return domain.NewValidationError("name", "mailbox name cannot be empty")
	}

	// Check length
	if len(name) > 255 {
		return domain.NewValidationError("name", "mailbox name too long (max 255 characters)")
	}

	// Check for invalid characters
	// Per RFC 5strstrng, the LIST wildcard characters (* and %) should not appear in mailbox names
	if strings.ContainsAny(name, "*%") {
		return domain.NewValidationError("name", "mailbox name cannot contain wildcard characters (* or %)")
	}

	// Check for empty path components
	parts := strings.Split(name, MailboxHierarchySeparator)
	for _, part := range parts {
		if part == "" {
			return domain.NewValidationError("name", "mailbox name cannot have empty path components")
		}
	}

	return nil
}
