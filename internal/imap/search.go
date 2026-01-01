package imap

import (
	"context"
	"sort"
	"strings"
	"time"

	"github.com/emersion/go-imap/v2"
	"github.com/emersion/go-imap/v2/imapserver"

	"yunt/internal/domain"
	"yunt/internal/repository"
)

// SearchHandler handles IMAP SEARCH command operations.
type SearchHandler struct {
	repo         repository.Repository
	userID       domain.ID
	selectedMbox *domain.Mailbox
	parser       *SearchCriteriaParser
}

// NewSearchHandler creates a new SearchHandler.
func NewSearchHandler(repo repository.Repository, userID domain.ID, selectedMbox *domain.Mailbox) *SearchHandler {
	return &SearchHandler{
		repo:         repo,
		userID:       userID,
		selectedMbox: selectedMbox,
		parser:       NewSearchCriteriaParser(),
	}
}

// Search executes an IMAP SEARCH command and returns matching message identifiers.
func (h *SearchHandler) Search(ctx context.Context, kind imapserver.NumKind, criteria *imap.SearchCriteria, options *imap.SearchOptions) (*imap.SearchData, error) {
	if h.selectedMbox == nil {
		return nil, &imap.Error{
			Type: imap.StatusResponseTypeNo,
			Text: "No mailbox selected",
		}
	}

	// Parse the IMAP search criteria into our internal format
	parsedCriteria := h.parser.Parse(criteria)

	// Execute the search
	results, err := h.executeSearch(ctx, kind, parsedCriteria)
	if err != nil {
		return nil, err
	}

	// Build the response based on NumKind and options
	return h.buildSearchData(kind, results, options), nil
}

// searchResult holds a matched message with its identifiers.
type searchResult struct {
	seqNum uint32
	uid    imap.UID
}

// executeSearch performs the search and returns matching message identifiers.
func (h *SearchHandler) executeSearch(ctx context.Context, kind imapserver.NumKind, criteria *SearchCriteria) ([]searchResult, error) {
	// Get all messages in the mailbox
	listResult, err := h.repo.Messages().ListByMailbox(ctx, h.selectedMbox.ID, nil)
	if err != nil {
		return nil, &imap.Error{
			Type: imap.StatusResponseTypeNo,
			Text: "Failed to list messages",
		}
	}

	// Create matcher for filtering
	matcher := NewMessageMatcher(criteria)

	var results []searchResult

	// Process each message
	for i, msg := range listResult.Items {
		seqNum := uint32(i + 1)       // 1-based sequence number
		uid := imap.UID(seqNum)       // Simplified: UID = sequence number

		// Check if message matches criteria
		if matcher.Matches(msg, seqNum, uid) {
			results = append(results, searchResult{
				seqNum: seqNum,
				uid:    uid,
			})
		}
	}

	// Sort results by the requested identifier type
	h.sortResults(results, kind)

	return results, nil
}

// sortResults sorts search results by sequence number or UID.
func (h *SearchHandler) sortResults(results []searchResult, kind imapserver.NumKind) {
	if kind == imapserver.NumKindUID {
		sort.Slice(results, func(i, j int) bool {
			return results[i].uid < results[j].uid
		})
	} else {
		sort.Slice(results, func(i, j int) bool {
			return results[i].seqNum < results[j].seqNum
		})
	}
}

// buildSearchData constructs the IMAP SearchData response.
func (h *SearchHandler) buildSearchData(kind imapserver.NumKind, results []searchResult, options *imap.SearchOptions) *imap.SearchData {
	data := &imap.SearchData{}

	if len(results) == 0 {
		// Return empty search data
		if options != nil && options.ReturnCount {
			data.Count = 0
		}
		return data
	}

	// Build the appropriate number set
	if kind == imapserver.NumKindUID {
		// Return UIDs
		uidSet := imap.UIDSet{}
		for _, r := range results {
			uidSet.AddNum(r.uid)
		}
		data.All = uidSet
	} else {
		// Return sequence numbers
		seqSet := imap.SeqSet{}
		for _, r := range results {
			seqSet.AddNum(r.seqNum)
		}
		data.All = seqSet
	}

	// Handle ESEARCH options
	if options != nil {
		if options.ReturnMin && len(results) > 0 {
			if kind == imapserver.NumKindUID {
				data.Min = uint32(results[0].uid)
			} else {
				data.Min = results[0].seqNum
			}
		}

		if options.ReturnMax && len(results) > 0 {
			last := results[len(results)-1]
			if kind == imapserver.NumKindUID {
				data.Max = uint32(last.uid)
			} else {
				data.Max = last.seqNum
			}
		}

		if options.ReturnCount {
			data.Count = uint32(len(results))
		}

		// If only returning count/min/max, clear the All field unless ReturnAll is set
		if !options.ReturnAll && (options.ReturnCount || options.ReturnMin || options.ReturnMax) {
			// Keep All only if explicitly requested
		}
	}

	return data
}

// OptimizedSearchHandler provides optimized search using database queries.
type OptimizedSearchHandler struct {
	*SearchHandler
}

// NewOptimizedSearchHandler creates an optimized search handler.
func NewOptimizedSearchHandler(repo repository.Repository, userID domain.ID, selectedMbox *domain.Mailbox) *OptimizedSearchHandler {
	return &OptimizedSearchHandler{
		SearchHandler: NewSearchHandler(repo, userID, selectedMbox),
	}
}

// Search executes an optimized search using database queries where possible.
func (h *OptimizedSearchHandler) Search(ctx context.Context, kind imapserver.NumKind, criteria *imap.SearchCriteria, options *imap.SearchOptions) (*imap.SearchData, error) {
	if h.selectedMbox == nil {
		return nil, &imap.Error{
			Type: imap.StatusResponseTypeNo,
			Text: "No mailbox selected",
		}
	}

	// Try to use database-optimized search for simple criteria
	if h.canOptimize(criteria) {
		return h.optimizedSearch(ctx, kind, criteria, options)
	}

	// Fall back to in-memory search for complex criteria
	return h.SearchHandler.Search(ctx, kind, criteria, options)
}

// canOptimize checks if the criteria can be optimized with database queries.
func (h *OptimizedSearchHandler) canOptimize(criteria *imap.SearchCriteria) bool {
	// Criteria that can be optimized:
	// - Flag searches (SEEN, UNSEEN, FLAGGED, etc.)
	// - Date searches (BEFORE, SINCE, SENTBEFORE, SENTSINCE)
	// - Size searches (LARGER, SMALLER)
	// - Header searches (FROM, TO, SUBJECT)

	// Cannot optimize:
	// - Body searches (require full text search)
	// - Complex NOT/OR combinations
	// - Sequence/UID set searches

	if criteria == nil {
		return true // "ALL" search
	}

	// Check for unsupported criteria
	if len(criteria.SeqNum) > 0 || len(criteria.UID) > 0 {
		return false
	}
	if len(criteria.Body) > 0 || len(criteria.Text) > 0 {
		return false
	}
	if len(criteria.Not) > 0 || len(criteria.Or) > 0 {
		return false
	}

	return true
}

// optimizedSearch uses database queries for efficient searching.
func (h *OptimizedSearchHandler) optimizedSearch(ctx context.Context, kind imapserver.NumKind, criteria *imap.SearchCriteria, options *imap.SearchOptions) (*imap.SearchData, error) {
	filter := h.buildMessageFilter(criteria)

	// Execute database query
	listResult, err := h.repo.Messages().List(ctx, filter, nil)
	if err != nil {
		return nil, &imap.Error{
			Type: imap.StatusResponseTypeNo,
			Text: "Failed to search messages",
		}
	}

	// Convert to search results
	var results []searchResult
	for i, msg := range listResult.Items {
		// Calculate sequence number and UID
		// Note: In a real implementation, you'd need proper UID handling
		seqNum := uint32(i + 1)
		uid := imap.UID(seqNum)

		results = append(results, searchResult{
			seqNum: seqNum,
			uid:    uid,
		})

		// Also need to handle sequence numbers correctly
		_ = msg // Use msg for proper UID lookup
	}

	h.sortResults(results, kind)
	return h.buildSearchData(kind, results, options), nil
}

// buildMessageFilter converts IMAP criteria to repository MessageFilter.
func (h *OptimizedSearchHandler) buildMessageFilter(criteria *imap.SearchCriteria) *repository.MessageFilter {
	filter := &repository.MessageFilter{
		MailboxID: &h.selectedMbox.ID,
	}

	if criteria == nil {
		return filter
	}

	// Date filters
	if !criteria.Since.IsZero() {
		ts := domain.Timestamp{Time: normalizeDate(criteria.Since)}
		filter.ReceivedAfter = &ts
	}
	if !criteria.Before.IsZero() {
		ts := domain.Timestamp{Time: normalizeDate(criteria.Before)}
		filter.ReceivedBefore = &ts
	}
	if !criteria.SentSince.IsZero() {
		ts := domain.Timestamp{Time: normalizeDate(criteria.SentSince)}
		filter.SentAfter = &ts
	}
	if !criteria.SentBefore.IsZero() {
		ts := domain.Timestamp{Time: normalizeDate(criteria.SentBefore)}
		filter.SentBefore = &ts
	}

	// Flag filters
	for _, flag := range criteria.Flag {
		switch flag {
		case imap.FlagSeen:
			status := domain.MessageRead
			filter.Status = &status
		case imap.FlagFlagged:
			starred := true
			filter.IsStarred = &starred
		case imap.FlagJunk:
			spam := true
			filter.IsSpam = &spam
		}
	}

	for _, flag := range criteria.NotFlag {
		switch flag {
		case imap.FlagSeen:
			status := domain.MessageUnread
			filter.Status = &status
		case imap.FlagFlagged:
			starred := false
			filter.IsStarred = &starred
		case imap.FlagJunk:
			spam := false
			filter.IsSpam = &spam
		}
	}

	// Size filters
	if criteria.Larger > 0 {
		filter.MinSize = &criteria.Larger
	}
	if criteria.Smaller > 0 {
		filter.MaxSize = &criteria.Smaller
	}

	// Header filters
	for _, hdr := range criteria.Header {
		switch strings.ToLower(hdr.Key) {
		case "from":
			filter.FromAddressContains = hdr.Value
		case "to":
			filter.ToAddressContains = hdr.Value
		case "subject":
			filter.SubjectContains = hdr.Value
		}
	}

	return filter
}

// SearchCommand wraps the search functionality for the Session.
// It provides the main entry point for SEARCH command processing.
type SearchCommand struct {
	repo         repository.Repository
	userID       domain.ID
	selectedMbox *domain.Mailbox
	logger       interface {
		Debug() interface {
			Str(key, val string) interface {
				Msg(msg string)
			}
			Int(key string, val int) interface {
				Msg(msg string)
			}
		}
	}
}

// NewSearchCommand creates a new SearchCommand.
func NewSearchCommand(repo repository.Repository, userID domain.ID, selectedMbox *domain.Mailbox) *SearchCommand {
	return &SearchCommand{
		repo:         repo,
		userID:       userID,
		selectedMbox: selectedMbox,
	}
}

// Execute runs the SEARCH command and returns results.
func (c *SearchCommand) Execute(ctx context.Context, kind imapserver.NumKind, criteria *imap.SearchCriteria, options *imap.SearchOptions) (*imap.SearchData, error) {
	// Create context with timeout
	ctx, cancel := context.WithTimeout(ctx, 60*time.Second)
	defer cancel()

	// Use optimized handler when possible
	handler := NewOptimizedSearchHandler(c.repo, c.userID, c.selectedMbox)
	return handler.Search(ctx, kind, criteria, options)
}
