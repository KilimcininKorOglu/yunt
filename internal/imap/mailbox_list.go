package imap

import (
	"context"
	"strings"

	"github.com/emersion/go-imap/v2"
	"github.com/emersion/go-imap/v2/imapserver"

	"yunt/internal/domain"
	"yunt/internal/repository"
)

// MailboxLister handles IMAP LIST operations.
type MailboxLister struct {
	repo   repository.Repository
	userID domain.ID
}

// NewMailboxLister creates a new MailboxLister.
func NewMailboxLister(repo repository.Repository, userID domain.ID) *MailboxLister {
	return &MailboxLister{
		repo:   repo,
		userID: userID,
	}
}

// List lists mailboxes matching the given reference and patterns.
// Implements RFC 3501 LIST command semantics.
func (l *MailboxLister) List(ctx context.Context, w *imapserver.ListWriter, ref string, patterns []string, options *imap.ListOptions) error {
	// If patterns is empty, return just the hierarchy delimiter
	if len(patterns) == 0 {
		return l.writeHierarchyDelimiter(w)
	}

	// Check for special pattern "" which requests hierarchy delimiter info
	if len(patterns) == 1 && patterns[0] == "" {
		return l.writeHierarchyDelimiter(w)
	}

	// Get all mailboxes for the user
	mailboxes, err := l.getUserMailboxes(ctx)
	if err != nil {
		return err
	}

	// Filter mailboxes based on reference and patterns
	matches := l.filterMailboxes(mailboxes, ref, patterns, options)

	// Write matching mailboxes
	for _, info := range matches {
		listData := info.ToIMAPListData()

		// Handle special-use only option
		if options != nil && options.ReturnSpecialUse {
			// Only include special-use attributes
			listData.Attrs = filterSpecialUseAttrs(listData.Attrs)
		}

		if err := w.WriteList(listData); err != nil {
			return err
		}
	}

	return nil
}

// writeHierarchyDelimiter writes a response with just the hierarchy delimiter.
func (l *MailboxLister) writeHierarchyDelimiter(w *imapserver.ListWriter) error {
	return w.WriteList(&imap.ListData{
		Mailbox: "",
		Delim:   rune(MailboxHierarchySeparator[0]),
		Attrs:   []imap.MailboxAttr{imap.MailboxAttrNoSelect},
	})
}

// getUserMailboxes retrieves all mailboxes for the current user.
func (l *MailboxLister) getUserMailboxes(ctx context.Context) ([]*domain.Mailbox, error) {
	result, err := l.repo.Mailboxes().ListByUser(ctx, l.userID, nil)
	if err != nil {
		return nil, err
	}
	return result.Items, nil
}

// filterMailboxes filters mailboxes based on reference and patterns.
func (l *MailboxLister) filterMailboxes(mailboxes []*domain.Mailbox, ref string, patterns []string, options *imap.ListOptions) []*MailboxInfo {
	var matches []*MailboxInfo

	for _, mailbox := range mailboxes {
		info := NewMailboxInfo(mailbox)

		// Check if mailbox matches any pattern
		fullPath := info.Name
		if ref != "" {
			fullPath = ref + MailboxHierarchySeparator + info.Name
		}

		for _, pattern := range patterns {
			if matchMailboxPattern(fullPath, ref, pattern) {
				// Check subscribed filter
				if options != nil && options.SelectSubscribed {
					// For now, we treat all mailboxes as subscribed
					// In a full implementation, we'd check subscription status
				}

				// Check special-use filter
				if options != nil && options.SelectSpecialUse {
					if len(filterSpecialUseAttrs(info.Attributes)) == 0 {
						continue // Skip non-special-use mailboxes
					}
				}

				matches = append(matches, info)
				break // Don't add the same mailbox twice
			}
		}
	}

	return matches
}

// matchMailboxPattern checks if a mailbox name matches an IMAP pattern.
// Supports:
//   - * matches zero or more characters including hierarchy delimiter
//   - % matches zero or more characters excluding hierarchy delimiter
func matchMailboxPattern(name, ref, pattern string) bool {
	// Combine reference and pattern
	fullPattern := pattern
	if ref != "" && !strings.HasPrefix(pattern, ref) {
		fullPattern = ref + MailboxHierarchySeparator + pattern
	}

	// Normalize both
	name = NormalizeMailboxName(name)
	fullPattern = NormalizeMailboxName(fullPattern)

	// Check for exact match first (no wildcards)
	if !strings.ContainsAny(fullPattern, "*%") {
		return strings.EqualFold(name, fullPattern)
	}

	// Use recursive matching for wildcards
	return matchWildcard(name, fullPattern, 0, 0)
}

// matchWildcard performs recursive wildcard matching.
func matchWildcard(name, pattern string, ni, pi int) bool {
	for ni < len(name) || pi < len(pattern) {
		if pi >= len(pattern) {
			return false
		}

		switch pattern[pi] {
		case '*':
			// * matches zero or more characters including hierarchy delimiter
			// Try matching zero characters first
			if matchWildcard(name, pattern, ni, pi+1) {
				return true
			}
			// Try matching one or more characters
			if ni < len(name) {
				return matchWildcard(name, pattern, ni+1, pi)
			}
			return false

		case '%':
			// % matches zero or more characters excluding hierarchy delimiter
			// Try matching zero characters first
			if matchWildcard(name, pattern, ni, pi+1) {
				return true
			}
			// Try matching one character (not delimiter)
			if ni < len(name) && name[ni] != MailboxHierarchySeparator[0] {
				return matchWildcard(name, pattern, ni+1, pi)
			}
			return false

		default:
			if ni >= len(name) {
				return false
			}
			// Case-insensitive character comparison
			if !equalFoldChar(name[ni], pattern[pi]) {
				return false
			}
			ni++
			pi++
		}
	}

	return ni == len(name) && pi == len(pattern)
}

// equalFoldChar performs case-insensitive comparison of two ASCII characters.
func equalFoldChar(a, b byte) bool {
	if a == b {
		return true
	}
	// Simple ASCII case folding
	if a >= 'A' && a <= 'Z' {
		a += 'a' - 'A'
	}
	if b >= 'A' && b <= 'Z' {
		b += 'a' - 'A'
	}
	return a == b
}

// filterSpecialUseAttrs returns only special-use attributes from a list.
func filterSpecialUseAttrs(attrs []imap.MailboxAttr) []imap.MailboxAttr {
	specialUseAttrs := map[imap.MailboxAttr]bool{
		imap.MailboxAttrAll:       true,
		imap.MailboxAttrArchive:   true,
		imap.MailboxAttrDrafts:    true,
		imap.MailboxAttrFlagged:   true,
		imap.MailboxAttrJunk:      true,
		imap.MailboxAttrSent:      true,
		imap.MailboxAttrTrash:     true,
		imap.MailboxAttrImportant: true,
	}

	var result []imap.MailboxAttr
	for _, attr := range attrs {
		if specialUseAttrs[attr] {
			result = append(result, attr)
		}
	}
	return result
}

// ListSubscribed lists subscribed mailboxes matching the given criteria.
// For now, all mailboxes are considered subscribed.
func (l *MailboxLister) ListSubscribed(ctx context.Context, w *imapserver.ListWriter, ref string, patterns []string) error {
	// Treat as regular LIST since we don't track subscriptions separately
	return l.List(ctx, w, ref, patterns, &imap.ListOptions{
		SelectSubscribed: true,
	})
}
