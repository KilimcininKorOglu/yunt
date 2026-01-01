// Package handlers provides HTTP request handlers for the Yunt API.
package handlers

import (
	"strconv"

	"github.com/labstack/echo/v4"

	"yunt/internal/api"
	"yunt/internal/api/middleware"
	"yunt/internal/domain"
	"yunt/internal/repository"
	"yunt/internal/service"
)

// MailboxHandler handles mailbox-related HTTP requests.
type MailboxHandler struct {
	mailboxService *service.MailboxService
	authService    *service.AuthService
}

// NewMailboxHandler creates a new MailboxHandler.
func NewMailboxHandler(mailboxService *service.MailboxService, authService *service.AuthService) *MailboxHandler {
	return &MailboxHandler{
		mailboxService: mailboxService,
		authService:    authService,
	}
}

// RegisterRoutes registers the mailbox routes on the given group.
func (h *MailboxHandler) RegisterRoutes(g *echo.Group) {
	mailboxes := g.Group("/mailboxes", middleware.Auth(h.authService))
	mailboxes.GET("", h.ListMailboxes)
	mailboxes.POST("", h.CreateMailbox)
	mailboxes.GET("/stats", h.GetUserStats)
	mailboxes.GET("/:id", h.GetMailbox)
	mailboxes.PUT("/:id", h.UpdateMailbox)
	mailboxes.DELETE("/:id", h.DeleteMailbox)
	mailboxes.GET("/:id/stats", h.GetMailboxStats)
	mailboxes.POST("/:id/default", h.SetDefaultMailbox)
}

// ListMailboxes handles requests to list all mailboxes for the authenticated user.
// @Summary List mailboxes
// @Description Get all mailboxes for the authenticated user
// @Tags Mailboxes
// @Produce json
// @Security BearerAuth
// @Param page query int false "Page number (default: 1)"
// @Param perPage query int false "Items per page (default: 20, max: 100)"
// @Param sort query string false "Sort field (name, address, messageCount, createdAt, updatedAt)"
// @Param order query string false "Sort order (asc, desc)"
// @Success 200 {object} api.Response{data=api.PaginatedData}
// @Failure 401 {object} api.Response{error=api.ErrorDetail}
// @Failure 500 {object} api.Response{error=api.ErrorDetail}
// @Router /mailboxes [get]
func (h *MailboxHandler) ListMailboxes(c echo.Context) error {
	userID := middleware.GetUserID(c)
	if userID.IsEmpty() {
		return api.Unauthorized(c, "authentication required")
	}

	// Parse pagination options
	opts := h.parseListOptions(c)

	result, err := h.mailboxService.ListMailboxes(c.Request().Context(), userID, opts)
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

// CreateMailbox handles requests to create a new mailbox.
// @Summary Create mailbox
// @Description Create a new mailbox for the authenticated user
// @Tags Mailboxes
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param input body domain.MailboxCreateInput true "Mailbox creation input"
// @Success 201 {object} api.Response{data=domain.Mailbox}
// @Failure 400 {object} api.Response{error=api.ErrorDetail}
// @Failure 401 {object} api.Response{error=api.ErrorDetail}
// @Failure 409 {object} api.Response{error=api.ErrorDetail}
// @Failure 422 {object} api.Response{error=api.ErrorDetail}
// @Failure 500 {object} api.Response{error=api.ErrorDetail}
// @Router /mailboxes [post]
func (h *MailboxHandler) CreateMailbox(c echo.Context) error {
	userID := middleware.GetUserID(c)
	if userID.IsEmpty() {
		return api.Unauthorized(c, "authentication required")
	}

	var input domain.MailboxCreateInput
	if err := c.Bind(&input); err != nil {
		return api.BadRequest(c, "invalid request body")
	}

	createInput := &service.CreateMailboxInput{
		UserID:        userID,
		Name:          input.Name,
		Address:       input.Address,
		Description:   input.Description,
		IsCatchAll:    input.IsCatchAll,
		IsDefault:     input.IsDefault,
		RetentionDays: input.RetentionDays,
	}

	mailbox, err := h.mailboxService.CreateMailbox(c.Request().Context(), createInput)
	if err != nil {
		return api.FromError(c, err)
	}

	return api.Created(c, mailbox)
}

// GetMailbox handles requests to get a specific mailbox.
// @Summary Get mailbox
// @Description Get a specific mailbox by ID
// @Tags Mailboxes
// @Produce json
// @Security BearerAuth
// @Param id path string true "Mailbox ID"
// @Success 200 {object} api.Response{data=domain.Mailbox}
// @Failure 401 {object} api.Response{error=api.ErrorDetail}
// @Failure 403 {object} api.Response{error=api.ErrorDetail}
// @Failure 404 {object} api.Response{error=api.ErrorDetail}
// @Failure 500 {object} api.Response{error=api.ErrorDetail}
// @Router /mailboxes/{id} [get]
func (h *MailboxHandler) GetMailbox(c echo.Context) error {
	userID := middleware.GetUserID(c)
	if userID.IsEmpty() {
		return api.Unauthorized(c, "authentication required")
	}

	mailboxID := domain.ID(c.Param("id"))
	if mailboxID.IsEmpty() {
		return api.BadRequest(c, "mailbox ID is required")
	}

	mailbox, err := h.mailboxService.GetMailbox(c.Request().Context(), mailboxID, userID)
	if err != nil {
		return api.FromError(c, err)
	}

	return api.OK(c, mailbox)
}

// UpdateMailbox handles requests to update a mailbox.
// @Summary Update mailbox
// @Description Update a mailbox (rename, change description, etc.)
// @Tags Mailboxes
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "Mailbox ID"
// @Param input body domain.MailboxUpdateInput true "Mailbox update input"
// @Success 200 {object} api.Response{data=domain.Mailbox}
// @Failure 400 {object} api.Response{error=api.ErrorDetail}
// @Failure 401 {object} api.Response{error=api.ErrorDetail}
// @Failure 403 {object} api.Response{error=api.ErrorDetail}
// @Failure 404 {object} api.Response{error=api.ErrorDetail}
// @Failure 422 {object} api.Response{error=api.ErrorDetail}
// @Failure 500 {object} api.Response{error=api.ErrorDetail}
// @Router /mailboxes/{id} [put]
func (h *MailboxHandler) UpdateMailbox(c echo.Context) error {
	userID := middleware.GetUserID(c)
	if userID.IsEmpty() {
		return api.Unauthorized(c, "authentication required")
	}

	mailboxID := domain.ID(c.Param("id"))
	if mailboxID.IsEmpty() {
		return api.BadRequest(c, "mailbox ID is required")
	}

	var input domain.MailboxUpdateInput
	if err := c.Bind(&input); err != nil {
		return api.BadRequest(c, "invalid request body")
	}

	updateInput := &service.UpdateMailboxInput{
		MailboxID:     mailboxID,
		UserID:        userID,
		Name:          input.Name,
		Description:   input.Description,
		IsDefault:     input.IsDefault,
		RetentionDays: input.RetentionDays,
	}

	mailbox, err := h.mailboxService.UpdateMailbox(c.Request().Context(), updateInput)
	if err != nil {
		return api.FromError(c, err)
	}

	return api.OK(c, mailbox)
}

// DeleteMailbox handles requests to delete a mailbox.
// @Summary Delete mailbox
// @Description Delete a mailbox (system mailboxes cannot be deleted)
// @Tags Mailboxes
// @Produce json
// @Security BearerAuth
// @Param id path string true "Mailbox ID"
// @Success 204
// @Failure 401 {object} api.Response{error=api.ErrorDetail}
// @Failure 403 {object} api.Response{error=api.ErrorDetail}
// @Failure 404 {object} api.Response{error=api.ErrorDetail}
// @Failure 409 {object} api.Response{error=api.ErrorDetail}
// @Failure 500 {object} api.Response{error=api.ErrorDetail}
// @Router /mailboxes/{id} [delete]
func (h *MailboxHandler) DeleteMailbox(c echo.Context) error {
	userID := middleware.GetUserID(c)
	if userID.IsEmpty() {
		return api.Unauthorized(c, "authentication required")
	}

	mailboxID := domain.ID(c.Param("id"))
	if mailboxID.IsEmpty() {
		return api.BadRequest(c, "mailbox ID is required")
	}

	if err := h.mailboxService.DeleteMailbox(c.Request().Context(), mailboxID, userID); err != nil {
		return api.FromError(c, err)
	}

	return api.NoContent(c)
}

// GetMailboxStats handles requests to get statistics for a specific mailbox.
// @Summary Get mailbox statistics
// @Description Get detailed statistics for a specific mailbox
// @Tags Mailboxes
// @Produce json
// @Security BearerAuth
// @Param id path string true "Mailbox ID"
// @Success 200 {object} api.Response{data=domain.MailboxStats}
// @Failure 401 {object} api.Response{error=api.ErrorDetail}
// @Failure 403 {object} api.Response{error=api.ErrorDetail}
// @Failure 404 {object} api.Response{error=api.ErrorDetail}
// @Failure 500 {object} api.Response{error=api.ErrorDetail}
// @Router /mailboxes/{id}/stats [get]
func (h *MailboxHandler) GetMailboxStats(c echo.Context) error {
	userID := middleware.GetUserID(c)
	if userID.IsEmpty() {
		return api.Unauthorized(c, "authentication required")
	}

	mailboxID := domain.ID(c.Param("id"))
	if mailboxID.IsEmpty() {
		return api.BadRequest(c, "mailbox ID is required")
	}

	stats, err := h.mailboxService.GetMailboxStats(c.Request().Context(), mailboxID, userID)
	if err != nil {
		return api.FromError(c, err)
	}

	return api.OK(c, stats)
}

// GetUserStats handles requests to get aggregated statistics for all user's mailboxes.
// @Summary Get user mailbox statistics
// @Description Get aggregated statistics for all mailboxes of the authenticated user
// @Tags Mailboxes
// @Produce json
// @Security BearerAuth
// @Success 200 {object} api.Response{data=domain.MailboxStats}
// @Failure 401 {object} api.Response{error=api.ErrorDetail}
// @Failure 500 {object} api.Response{error=api.ErrorDetail}
// @Router /mailboxes/stats [get]
func (h *MailboxHandler) GetUserStats(c echo.Context) error {
	userID := middleware.GetUserID(c)
	if userID.IsEmpty() {
		return api.Unauthorized(c, "authentication required")
	}

	stats, err := h.mailboxService.GetUserMailboxStats(c.Request().Context(), userID)
	if err != nil {
		return api.FromError(c, err)
	}

	return api.OK(c, stats)
}

// SetDefaultMailbox handles requests to set a mailbox as the default.
// @Summary Set default mailbox
// @Description Set a mailbox as the default for the authenticated user
// @Tags Mailboxes
// @Produce json
// @Security BearerAuth
// @Param id path string true "Mailbox ID"
// @Success 204
// @Failure 401 {object} api.Response{error=api.ErrorDetail}
// @Failure 403 {object} api.Response{error=api.ErrorDetail}
// @Failure 404 {object} api.Response{error=api.ErrorDetail}
// @Failure 500 {object} api.Response{error=api.ErrorDetail}
// @Router /mailboxes/{id}/default [post]
func (h *MailboxHandler) SetDefaultMailbox(c echo.Context) error {
	userID := middleware.GetUserID(c)
	if userID.IsEmpty() {
		return api.Unauthorized(c, "authentication required")
	}

	mailboxID := domain.ID(c.Param("id"))
	if mailboxID.IsEmpty() {
		return api.BadRequest(c, "mailbox ID is required")
	}

	if err := h.mailboxService.SetDefaultMailbox(c.Request().Context(), mailboxID, userID); err != nil {
		return api.FromError(c, err)
	}

	return api.NoContent(c)
}

// parseListOptions extracts list options from the request query parameters.
func (h *MailboxHandler) parseListOptions(c echo.Context) *repository.ListOptions {
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
		order := domain.SortAsc
		if orderStr := c.QueryParam("order"); orderStr == "desc" {
			order = domain.SortDesc
		}

		// Validate sort field
		if isValidMailboxSortField(sortField) {
			opts.Sort = &repository.SortOptions{
				Field: sortField,
				Order: order,
			}
		}
	}

	return opts
}

// isValidMailboxSortField checks if a sort field is valid for mailboxes.
func isValidMailboxSortField(field string) bool {
	validFields := []string{"name", "address", "messageCount", "unreadCount", "totalSize", "createdAt", "updatedAt"}
	for _, valid := range validFields {
		if field == valid {
			return true
		}
	}
	return false
}
