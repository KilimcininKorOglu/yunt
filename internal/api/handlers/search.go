// Package handlers provides HTTP request handlers for the Yunt API.
package handlers

import (
	"strconv"
	"strings"
	"time"

	"github.com/labstack/echo/v4"

	"yunt/internal/api"
	"yunt/internal/api/middleware"
	"yunt/internal/domain"
	"yunt/internal/repository"
	"yunt/internal/service"
)

// SearchHandler handles search-related HTTP requests.
type SearchHandler struct {
	messageService *service.MessageService
	authService    *service.AuthService
}

// NewSearchHandler creates a new SearchHandler.
func NewSearchHandler(
	messageService *service.MessageService,
	authService *service.AuthService,
) *SearchHandler {
	return &SearchHandler{
		messageService: messageService,
		authService:    authService,
	}
}

// RegisterRoutes registers the search routes on the given group.
func (h *SearchHandler) RegisterRoutes(g *echo.Group) {
	search := g.Group("/search", middleware.Auth(h.authService))
	search.GET("/simple", h.SimpleSearch)
	search.GET("/advanced", h.AdvancedSearch)
}

// SimpleSearch handles requests to search messages by text.
// @Summary Simple message search
// @Description Search messages by text query across subject and body
// @Tags Search
// @Produce json
// @Security BearerAuth
// @Param q query string true "Search query text"
// @Param mailboxId query string false "Filter by mailbox ID"
// @Param page query int false "Page number (default: 1)"
// @Param perPage query int false "Items per page (default: 20, max: 100)"
// @Param sort query string false "Sort field (receivedAt, subject, from, size)"
// @Param order query string false "Sort order (asc, desc)"
// @Success 200 {object} api.Response{data=api.PaginatedData}
// @Failure 400 {object} api.Response{error=api.ErrorDetail}
// @Failure 401 {object} api.Response{error=api.ErrorDetail}
// @Failure 500 {object} api.Response{error=api.ErrorDetail}
// @Router /search/simple [get]
func (h *SearchHandler) SimpleSearch(c echo.Context) error {
	userID := middleware.GetUserID(c)
	if userID.IsEmpty() {
		return api.Unauthorized(c, "authentication required")
	}

	// Get and validate search query
	query := strings.TrimSpace(c.QueryParam("q"))
	if query == "" {
		return api.BadRequest(c, "search query is required")
	}

	// Validate query length
	if len(query) < 2 {
		return api.BadRequest(c, "search query must be at least 2 characters")
	}

	if len(query) > 500 {
		return api.BadRequest(c, "search query is too long (max 500 characters)")
	}

	// Parse filter options
	filter, err := h.parseMailboxFilter(c, userID)
	if err != nil {
		return api.BadRequest(c, err.Error())
	}

	// Parse list options
	opts := h.parseListOptions(c)

	// Create search options for full-text search
	searchOpts := &repository.SearchOptions{
		Query:    query,
		Fields:   []string{"subject", "textBody", "htmlBody"},
		MatchAll: false,
	}

	result, err := h.messageService.SearchMessagesForUser(c.Request().Context(), userID, searchOpts, filter, opts)
	if err != nil {
		return api.FromError(c, err)
	}

	// Determine page and perPage from options
	page := 1
	perPage := repository.DefaultPerPage
	if opts != nil && opts.Pagination != nil {
		page = opts.Pagination.Page
		perPage = opts.Pagination.PerPage
	}

	return api.Paginated(c, result.Items, page, perPage, result.Total)
}

// AdvancedSearchInput represents the input parameters for advanced search.
type AdvancedSearchInput struct {
	// Query is the optional text search query.
	Query string `query:"q"`
	// MailboxID filters by specific mailbox.
	MailboxID string `query:"mailboxId"`
	// From filters by sender address (partial match).
	From string `query:"from"`
	// To filters by recipient address (partial match).
	To string `query:"to"`
	// Subject filters by subject (partial match).
	Subject string `query:"subject"`
	// Status filters by message status (read/unread).
	Status string `query:"status"`
	// IsStarred filters by starred status.
	IsStarred *bool `query:"isStarred"`
	// IsSpam filters by spam status.
	IsSpam *bool `query:"isSpam"`
	// HasAttachments filters by attachment presence.
	HasAttachments *bool `query:"hasAttachments"`
	// ReceivedAfter filters messages received after this date (RFC3339).
	ReceivedAfter string `query:"receivedAfter"`
	// ReceivedBefore filters messages received before this date (RFC3339).
	ReceivedBefore string `query:"receivedBefore"`
	// MinSize filters messages larger than this size in bytes.
	MinSize *int64 `query:"minSize"`
	// MaxSize filters messages smaller than this size in bytes.
	MaxSize *int64 `query:"maxSize"`
	// Page is the page number for pagination.
	Page int `query:"page"`
	// PerPage is the number of items per page.
	PerPage int `query:"perPage"`
	// Sort is the field to sort by.
	Sort string `query:"sort"`
	// Order is the sort order (asc/desc).
	Order string `query:"order"`
}

// AdvancedSearch handles requests to search messages with multiple criteria.
// @Summary Advanced message search
// @Description Search messages with multiple structured criteria including sender, recipient, date range, and flags
// @Tags Search
// @Produce json
// @Security BearerAuth
// @Param q query string false "Text search query (subject and body)"
// @Param mailboxId query string false "Filter by mailbox ID"
// @Param from query string false "Filter by sender address (partial match)"
// @Param to query string false "Filter by recipient address (partial match)"
// @Param subject query string false "Filter by subject (partial match)"
// @Param status query string false "Filter by status (read, unread)"
// @Param isStarred query bool false "Filter by starred status"
// @Param isSpam query bool false "Filter by spam status"
// @Param hasAttachments query bool false "Filter by attachment presence"
// @Param receivedAfter query string false "Filter messages received after (RFC3339)"
// @Param receivedBefore query string false "Filter messages received before (RFC3339)"
// @Param minSize query int false "Filter messages larger than this size (bytes)"
// @Param maxSize query int false "Filter messages smaller than this size (bytes)"
// @Param page query int false "Page number (default: 1)"
// @Param perPage query int false "Items per page (default: 20, max: 100)"
// @Param sort query string false "Sort field (receivedAt, subject, from, size)"
// @Param order query string false "Sort order (asc, desc)"
// @Success 200 {object} api.Response{data=api.PaginatedData}
// @Failure 400 {object} api.Response{error=api.ErrorDetail}
// @Failure 401 {object} api.Response{error=api.ErrorDetail}
// @Failure 500 {object} api.Response{error=api.ErrorDetail}
// @Router /search/advanced [get]
func (h *SearchHandler) AdvancedSearch(c echo.Context) error {
	userID := middleware.GetUserID(c)
	if userID.IsEmpty() {
		return api.Unauthorized(c, "authentication required")
	}

	// Parse and validate all search parameters
	filter, searchOpts, validationErrors := h.parseAdvancedSearchParams(c)
	if len(validationErrors) > 0 {
		return api.ValidationFailed(c, validationErrors)
	}

	// Verify mailbox access if specified
	if filter.MailboxID != nil {
		// Ownership verification happens in the service layer
	}

	// Parse list options
	opts := h.parseListOptions(c)

	// Execute search
	result, err := h.messageService.SearchMessagesForUser(c.Request().Context(), userID, searchOpts, filter, opts)
	if err != nil {
		return api.FromError(c, err)
	}

	// Determine page and perPage from options
	page := 1
	perPage := repository.DefaultPerPage
	if opts != nil && opts.Pagination != nil {
		page = opts.Pagination.Page
		perPage = opts.Pagination.PerPage
	}

	return api.Paginated(c, result.Items, page, perPage, result.Total)
}

// parseAdvancedSearchParams parses and validates all advanced search parameters.
func (h *SearchHandler) parseAdvancedSearchParams(c echo.Context) (*repository.MessageFilter, *repository.SearchOptions, []ValidationError) {
	filter := &repository.MessageFilter{}
	searchOpts := &repository.SearchOptions{}
	var errors []ValidationError

	// Parse text query
	if query := strings.TrimSpace(c.QueryParam("q")); query != "" {
		if len(query) > 500 {
			errors = append(errors, ValidationError{Field: "q", Message: "search query is too long (max 500 characters)"})
		} else {
			searchOpts.Query = query
			searchOpts.Fields = []string{"subject", "textBody", "htmlBody"}
		}
	}

	// Parse mailboxId filter
	if mailboxIDStr := c.QueryParam("mailboxId"); mailboxIDStr != "" {
		mailboxID := domain.ID(mailboxIDStr)
		if mailboxID.IsEmpty() {
			errors = append(errors, ValidationError{Field: "mailboxId", Message: "invalid mailbox ID"})
		} else {
			filter.MailboxID = &mailboxID
		}
	}

	// Parse from filter (sender)
	if from := strings.TrimSpace(c.QueryParam("from")); from != "" {
		if len(from) > 255 {
			errors = append(errors, ValidationError{Field: "from", Message: "from address filter is too long"})
		} else {
			filter.FromAddressContains = from
		}
	}

	// Parse to filter (recipient)
	if to := strings.TrimSpace(c.QueryParam("to")); to != "" {
		if len(to) > 255 {
			errors = append(errors, ValidationError{Field: "to", Message: "to address filter is too long"})
		} else {
			filter.ToAddressContains = to
		}
	}

	// Parse subject filter
	if subject := strings.TrimSpace(c.QueryParam("subject")); subject != "" {
		if len(subject) > 500 {
			errors = append(errors, ValidationError{Field: "subject", Message: "subject filter is too long"})
		} else {
			filter.SubjectContains = subject
		}
	}

	// Parse status filter
	if statusStr := c.QueryParam("status"); statusStr != "" {
		var status domain.MessageStatus
		switch strings.ToLower(statusStr) {
		case "read":
			status = domain.MessageRead
			filter.Status = &status
		case "unread":
			status = domain.MessageUnread
			filter.Status = &status
		default:
			errors = append(errors, ValidationError{Field: "status", Message: "invalid status value, must be 'read' or 'unread'"})
		}
	}

	// Parse isStarred filter
	if starredStr := c.QueryParam("isStarred"); starredStr != "" {
		starred, err := strconv.ParseBool(starredStr)
		if err != nil {
			errors = append(errors, ValidationError{Field: "isStarred", Message: "invalid boolean value for isStarred"})
		} else {
			filter.IsStarred = &starred
		}
	}

	// Parse isSpam filter
	if spamStr := c.QueryParam("isSpam"); spamStr != "" {
		spam, err := strconv.ParseBool(spamStr)
		if err != nil {
			errors = append(errors, ValidationError{Field: "isSpam", Message: "invalid boolean value for isSpam"})
		} else {
			filter.IsSpam = &spam
		}
	}

	// Parse hasAttachments filter
	if attachmentsStr := c.QueryParam("hasAttachments"); attachmentsStr != "" {
		hasAttachments, err := strconv.ParseBool(attachmentsStr)
		if err != nil {
			errors = append(errors, ValidationError{Field: "hasAttachments", Message: "invalid boolean value for hasAttachments"})
		} else {
			filter.HasAttachments = &hasAttachments
		}
	}

	// Parse receivedAfter filter
	if afterStr := c.QueryParam("receivedAfter"); afterStr != "" {
		t, err := time.Parse(time.RFC3339, afterStr)
		if err != nil {
			errors = append(errors, ValidationError{Field: "receivedAfter", Message: "invalid date format, use RFC3339 (e.g., 2024-01-01T00:00:00Z)"})
		} else {
			ts := domain.Timestamp{Time: t}
			filter.ReceivedAfter = &ts
		}
	}

	// Parse receivedBefore filter
	if beforeStr := c.QueryParam("receivedBefore"); beforeStr != "" {
		t, err := time.Parse(time.RFC3339, beforeStr)
		if err != nil {
			errors = append(errors, ValidationError{Field: "receivedBefore", Message: "invalid date format, use RFC3339 (e.g., 2024-01-01T00:00:00Z)"})
		} else {
			ts := domain.Timestamp{Time: t}
			filter.ReceivedBefore = &ts
		}
	}

	// Validate date range if both are specified
	if filter.ReceivedAfter != nil && filter.ReceivedBefore != nil {
		if filter.ReceivedAfter.After(filter.ReceivedBefore.Time) {
			errors = append(errors, ValidationError{Field: "receivedAfter", Message: "receivedAfter cannot be after receivedBefore"})
		}
	}

	// Parse minSize filter
	if minSizeStr := c.QueryParam("minSize"); minSizeStr != "" {
		minSize, err := strconv.ParseInt(minSizeStr, 10, 64)
		if err != nil || minSize < 0 {
			errors = append(errors, ValidationError{Field: "minSize", Message: "invalid minSize value, must be a non-negative integer"})
		} else {
			filter.MinSize = &minSize
		}
	}

	// Parse maxSize filter
	if maxSizeStr := c.QueryParam("maxSize"); maxSizeStr != "" {
		maxSize, err := strconv.ParseInt(maxSizeStr, 10, 64)
		if err != nil || maxSize < 0 {
			errors = append(errors, ValidationError{Field: "maxSize", Message: "invalid maxSize value, must be a non-negative integer"})
		} else {
			filter.MaxSize = &maxSize
		}
	}

	// Validate size range if both are specified
	if filter.MinSize != nil && filter.MaxSize != nil {
		if *filter.MinSize > *filter.MaxSize {
			errors = append(errors, ValidationError{Field: "minSize", Message: "minSize cannot be greater than maxSize"})
		}
	}

	return filter, searchOpts, errors
}

// parseMailboxFilter parses the mailbox filter from query parameters.
func (h *SearchHandler) parseMailboxFilter(c echo.Context, _ domain.ID) (*repository.MessageFilter, error) {
	filter := &repository.MessageFilter{}

	// Parse mailboxId filter
	if mailboxIDStr := c.QueryParam("mailboxId"); mailboxIDStr != "" {
		mailboxID := domain.ID(mailboxIDStr)
		filter.MailboxID = &mailboxID
	}

	return filter, nil
}

// parseListOptions extracts list options from the request query parameters.
func (h *SearchHandler) parseListOptions(c echo.Context) *repository.ListOptions {
	opts := &repository.ListOptions{}

	// Parse pagination
	page := 1
	perPage := repository.DefaultPerPage

	if pageStr := c.QueryParam("page"); pageStr != "" {
		if p, err := strconv.Atoi(pageStr); err == nil && p > 0 {
			page = p
		}
	}

	if perPageStr := c.QueryParam("perPage"); perPageStr != "" {
		if pp, err := strconv.Atoi(perPageStr); err == nil && pp > 0 {
			perPage = pp
			if perPage > repository.MaxPerPage {
				perPage = repository.MaxPerPage
			}
		}
	}

	opts.Pagination = &repository.PaginationOptions{
		Page:    page,
		PerPage: perPage,
	}

	// Parse sort options
	if sortField := c.QueryParam("sort"); sortField != "" {
		order := domain.SortDesc // Default to descending for messages
		if orderStr := c.QueryParam("order"); orderStr == "asc" {
			order = domain.SortAsc
		}

		// Validate sort field
		if isValidSearchSortField(sortField) {
			opts.Sort = &repository.SortOptions{
				Field: sortField,
				Order: order,
			}
		}
	} else {
		// Default sort by receivedAt descending
		opts.Sort = &repository.SortOptions{
			Field: "receivedAt",
			Order: domain.SortDesc,
		}
	}

	return opts
}

// isValidSearchSortField checks if a sort field is valid for search results.
func isValidSearchSortField(field string) bool {
	validFields := []string{"receivedAt", "sentAt", "subject", "from", "size", "status", "createdAt"}
	for _, valid := range validFields {
		if field == valid {
			return true
		}
	}
	return false
}

// ValidationError represents a field-level validation error.
type ValidationError struct {
	// Field is the name of the field that failed validation.
	Field string `json:"field"`
	// Message is the validation error message.
	Message string `json:"message"`
}
