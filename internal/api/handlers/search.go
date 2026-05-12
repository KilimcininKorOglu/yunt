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
	search.GET("", h.SimpleSearch)
	search.POST("/advanced", h.AdvancedSearch)
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
	Query          string `json:"q"`
	MailboxID      string `json:"mailboxId"`
	From           string `json:"from"`
	To             string `json:"to"`
	Subject        string `json:"subject"`
	Status         string `json:"status"`
	IsStarred      *bool  `json:"isStarred"`
	IsSpam         *bool  `json:"isSpam"`
	HasAttachments *bool  `json:"hasAttachments"`
	ReceivedAfter  string `json:"receivedAfter"`
	ReceivedBefore string `json:"receivedBefore"`
	MinSize        *int64 `json:"minSize"`
	MaxSize        *int64 `json:"maxSize"`
	Page           int    `json:"page"`
	PerPage        int    `json:"perPage"`
	Sort           string `json:"sort"`
	Order          string `json:"order"`
}

// AdvancedSearch handles requests to search messages with multiple criteria.
// @Summary Advanced message search
// @Description Search messages with multiple structured criteria including sender, recipient, date range, and flags
// @Tags Search
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param input body AdvancedSearchInput true "Advanced search criteria"
// @Success 200 {object} api.Response{data=api.PaginatedData}
// @Failure 400 {object} api.Response{error=api.ErrorDetail}
// @Failure 401 {object} api.Response{error=api.ErrorDetail}
// @Failure 500 {object} api.Response{error=api.ErrorDetail}
// @Router /search/advanced [post]
func (h *SearchHandler) AdvancedSearch(c echo.Context) error {
	userID := middleware.GetUserID(c)
	if userID.IsEmpty() {
		return api.Unauthorized(c, "authentication required")
	}

	var input AdvancedSearchInput
	if err := c.Bind(&input); err != nil {
		return api.BadRequest(c, "invalid request body")
	}

	filter, searchOpts, validationErrors := h.buildAdvancedFilter(&input)
	if len(validationErrors) > 0 {
		return api.ValidationFailed(c, validationErrors)
	}

	opts := h.buildListOptions(&input)

	result, err := h.messageService.SearchMessagesForUser(c.Request().Context(), userID, searchOpts, filter, opts)
	if err != nil {
		return api.FromError(c, err)
	}

	page := 1
	perPage := repository.DefaultPerPage
	if opts != nil && opts.Pagination != nil {
		page = opts.Pagination.Page
		perPage = opts.Pagination.PerPage
	}

	return api.Paginated(c, result.Items, page, perPage, result.Total)
}

// buildAdvancedFilter builds filter and search options from the parsed input.
func (h *SearchHandler) buildAdvancedFilter(input *AdvancedSearchInput) (*repository.MessageFilter, *repository.SearchOptions, []ValidationError) {
	filter := &repository.MessageFilter{}
	searchOpts := &repository.SearchOptions{}
	var errors []ValidationError

	if query := strings.TrimSpace(input.Query); query != "" {
		if len(query) > 500 {
			errors = append(errors, ValidationError{Field: "q", Message: "search query is too long (max 500 characters)"})
		} else {
			searchOpts.Query = query
			searchOpts.Fields = []string{"subject", "textBody", "htmlBody"}
		}
	}

	if input.MailboxID != "" {
		mailboxID := domain.ID(input.MailboxID)
		if mailboxID.IsEmpty() {
			errors = append(errors, ValidationError{Field: "mailboxId", Message: "invalid mailbox ID"})
		} else {
			filter.MailboxID = &mailboxID
		}
	}

	if from := strings.TrimSpace(input.From); from != "" {
		if len(from) > 255 {
			errors = append(errors, ValidationError{Field: "from", Message: "from address filter is too long"})
		} else {
			filter.FromAddressContains = from
		}
	}

	if to := strings.TrimSpace(input.To); to != "" {
		if len(to) > 255 {
			errors = append(errors, ValidationError{Field: "to", Message: "to address filter is too long"})
		} else {
			filter.ToAddressContains = to
		}
	}

	if subject := strings.TrimSpace(input.Subject); subject != "" {
		if len(subject) > 500 {
			errors = append(errors, ValidationError{Field: "subject", Message: "subject filter is too long"})
		} else {
			filter.SubjectContains = subject
		}
	}

	if input.Status != "" {
		var status domain.MessageStatus
		switch strings.ToLower(input.Status) {
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

	if input.IsStarred != nil {
		filter.IsStarred = input.IsStarred
	}
	if input.IsSpam != nil {
		filter.IsSpam = input.IsSpam
	}
	if input.HasAttachments != nil {
		filter.HasAttachments = input.HasAttachments
	}

	if input.ReceivedAfter != "" {
		t, err := time.Parse(time.RFC3339, input.ReceivedAfter)
		if err != nil {
			errors = append(errors, ValidationError{Field: "receivedAfter", Message: "invalid date format, use RFC3339 (e.g., 2024-01-01T00:00:00Z)"})
		} else {
			ts := domain.Timestamp{Time: t}
			filter.ReceivedAfter = &ts
		}
	}

	if input.ReceivedBefore != "" {
		t, err := time.Parse(time.RFC3339, input.ReceivedBefore)
		if err != nil {
			errors = append(errors, ValidationError{Field: "receivedBefore", Message: "invalid date format, use RFC3339 (e.g., 2024-01-01T00:00:00Z)"})
		} else {
			ts := domain.Timestamp{Time: t}
			filter.ReceivedBefore = &ts
		}
	}

	if filter.ReceivedAfter != nil && filter.ReceivedBefore != nil {
		if filter.ReceivedAfter.After(filter.ReceivedBefore.Time) {
			errors = append(errors, ValidationError{Field: "receivedAfter", Message: "receivedAfter cannot be after receivedBefore"})
		}
	}

	if input.MinSize != nil {
		if *input.MinSize < 0 {
			errors = append(errors, ValidationError{Field: "minSize", Message: "minSize must be non-negative"})
		} else {
			filter.MinSize = input.MinSize
		}
	}
	if input.MaxSize != nil {
		if *input.MaxSize < 0 {
			errors = append(errors, ValidationError{Field: "maxSize", Message: "maxSize must be non-negative"})
		} else {
			filter.MaxSize = input.MaxSize
		}
	}
	if filter.MinSize != nil && filter.MaxSize != nil && *filter.MinSize > *filter.MaxSize {
		errors = append(errors, ValidationError{Field: "minSize", Message: "minSize cannot be greater than maxSize"})
	}

	return filter, searchOpts, errors
}

// buildListOptions builds list options from the parsed input.
func (h *SearchHandler) buildListOptions(input *AdvancedSearchInput) *repository.ListOptions {
	page := input.Page
	if page <= 0 {
		page = 1
	}
	perPage := input.PerPage
	if perPage <= 0 {
		perPage = repository.DefaultPerPage
	}
	if perPage > 100 {
		perPage = 100
	}

	opts := &repository.ListOptions{
		Pagination: &repository.PaginationOptions{Page: page, PerPage: perPage},
	}

	if input.Sort != "" {
		opts.Sort = &repository.SortOptions{Field: input.Sort, Order: domain.SortOrder(input.Order)}
	}

	return opts
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
