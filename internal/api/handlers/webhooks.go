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

// WebhookHandler handles webhook-related HTTP requests.
type WebhookHandler struct {
	webhookService *service.WebhookService
	authService    *service.AuthService
}

// NewWebhookHandler creates a new WebhookHandler.
func NewWebhookHandler(webhookService *service.WebhookService, authService *service.AuthService) *WebhookHandler {
	return &WebhookHandler{
		webhookService: webhookService,
		authService:    authService,
	}
}

// RegisterRoutes registers the webhook routes on the given group.
func (h *WebhookHandler) RegisterRoutes(g *echo.Group) {
	webhooks := g.Group("/webhooks")

	// All webhook routes require authentication
	webhooks.Use(middleware.Auth(h.authService))

	// CRUD operations
	webhooks.POST("", h.CreateWebhook)
	webhooks.GET("", h.ListWebhooks)
	webhooks.GET("/:id", h.GetWebhook)
	webhooks.PUT("/:id", h.UpdateWebhook)
	webhooks.PATCH("/:id", h.UpdateWebhook)
	webhooks.DELETE("/:id", h.DeleteWebhook)

	// Actions
	webhooks.POST("/:id/test", h.TestWebhook)
	webhooks.POST("/:id/activate", h.ActivateWebhook)
	webhooks.POST("/:id/deactivate", h.DeactivateWebhook)

	// Deliveries
	webhooks.GET("/:id/deliveries", h.ListDeliveries)
	webhooks.GET("/:id/stats", h.GetDeliveryStats)
}

// CreateWebhook handles webhook creation requests.
// @Summary Create Webhook
// @Description Create a new webhook for the authenticated user
// @Tags Webhooks
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param input body domain.WebhookCreateInput true "Webhook creation input"
// @Success 201 {object} api.Response{data=domain.Webhook}
// @Failure 400 {object} api.Response{error=api.ErrorDetail}
// @Failure 401 {object} api.Response{error=api.ErrorDetail}
// @Failure 409 {object} api.Response{error=api.ErrorDetail}
// @Failure 422 {object} api.Response{error=api.ErrorDetail}
// @Failure 500 {object} api.Response{error=api.ErrorDetail}
// @Router /webhooks [post]
func (h *WebhookHandler) CreateWebhook(c echo.Context) error {
	userID := middleware.GetUserID(c)
	if userID.IsEmpty() {
		return api.Unauthorized(c, "authentication required")
	}

	var input domain.WebhookCreateInput
	if err := c.Bind(&input); err != nil {
		return api.BadRequest(c, "invalid request body")
	}

	webhook, err := h.webhookService.CreateWebhook(c.Request().Context(), userID, &input)
	if err != nil {
		return api.FromError(c, err)
	}

	return api.Created(c, webhook)
}

// GetWebhook handles requests to get a single webhook.
// @Summary Get Webhook
// @Description Get a webhook by ID
// @Tags Webhooks
// @Produce json
// @Security BearerAuth
// @Param id path string true "Webhook ID"
// @Success 200 {object} api.Response{data=domain.Webhook}
// @Failure 401 {object} api.Response{error=api.ErrorDetail}
// @Failure 404 {object} api.Response{error=api.ErrorDetail}
// @Failure 500 {object} api.Response{error=api.ErrorDetail}
// @Router /webhooks/{id} [get]
func (h *WebhookHandler) GetWebhook(c echo.Context) error {
	userID := middleware.GetUserID(c)
	if userID.IsEmpty() {
		return api.Unauthorized(c, "authentication required")
	}

	webhookID := domain.ID(c.Param("id"))
	if webhookID.IsEmpty() {
		return api.BadRequest(c, "webhook ID is required")
	}

	webhook, err := h.webhookService.GetWebhookForUser(c.Request().Context(), webhookID, userID)
	if err != nil {
		return api.FromError(c, err)
	}

	return api.OK(c, webhook)
}

// ListWebhooks handles requests to list webhooks.
// @Summary List Webhooks
// @Description List all webhooks for the authenticated user
// @Tags Webhooks
// @Produce json
// @Security BearerAuth
// @Param page query int false "Page number (default: 1)"
// @Param pageSize query int false "Page size (default: 20, max: 100)"
// @Param status query string false "Filter by status (active, inactive, failed)"
// @Param event query string false "Filter by subscribed event"
// @Param search query string false "Search in name and URL"
// @Success 200 {object} api.Response{data=api.PaginatedData}
// @Failure 401 {object} api.Response{error=api.ErrorDetail}
// @Failure 500 {object} api.Response{error=api.ErrorDetail}
// @Router /webhooks [get]
func (h *WebhookHandler) ListWebhooks(c echo.Context) error {
	userID := middleware.GetUserID(c)
	if userID.IsEmpty() {
		return api.Unauthorized(c, "authentication required")
	}

	// Parse pagination
	page := parseIntParam(c, "page", 1)
	pageSize := parseIntParam(c, "pageSize", 20)
	if pageSize > 100 {
		pageSize = 100
	}

	// Build filter
	filter := &repository.WebhookFilter{
		UserID: &userID,
	}

	// Status filter
	if status := c.QueryParam("status"); status != "" {
		webhookStatus := domain.WebhookStatus(status)
		if webhookStatus.IsValid() {
			filter.Status = &webhookStatus
		}
	}

	// Event filter
	if event := c.QueryParam("event"); event != "" {
		webhookEvent := domain.WebhookEvent(event)
		if webhookEvent.IsValid() {
			filter.Event = &webhookEvent
		}
	}

	// Search filter
	if search := c.QueryParam("search"); search != "" {
		filter.Search = search
	}

	// Build options
	opts := &repository.ListOptions{
		Pagination: &repository.PaginationOptions{
			Page:    page,
			PerPage: pageSize,
		},
		Sort: &repository.SortOptions{
			Field: "createdAt",
			Order: domain.SortDesc,
		},
	}

	result, err := h.webhookService.ListWebhooks(c.Request().Context(), filter, opts)
	if err != nil {
		return api.FromError(c, err)
	}

	return api.Paginated(c, result.Items, page, pageSize, result.Total)
}

// UpdateWebhook handles webhook update requests.
// @Summary Update Webhook
// @Description Update an existing webhook
// @Tags Webhooks
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "Webhook ID"
// @Param input body domain.WebhookUpdateInput true "Webhook update input"
// @Success 200 {object} api.Response{data=domain.Webhook}
// @Failure 400 {object} api.Response{error=api.ErrorDetail}
// @Failure 401 {object} api.Response{error=api.ErrorDetail}
// @Failure 404 {object} api.Response{error=api.ErrorDetail}
// @Failure 409 {object} api.Response{error=api.ErrorDetail}
// @Failure 422 {object} api.Response{error=api.ErrorDetail}
// @Failure 500 {object} api.Response{error=api.ErrorDetail}
// @Router /webhooks/{id} [put]
func (h *WebhookHandler) UpdateWebhook(c echo.Context) error {
	userID := middleware.GetUserID(c)
	if userID.IsEmpty() {
		return api.Unauthorized(c, "authentication required")
	}

	webhookID := domain.ID(c.Param("id"))
	if webhookID.IsEmpty() {
		return api.BadRequest(c, "webhook ID is required")
	}

	var input domain.WebhookUpdateInput
	if err := c.Bind(&input); err != nil {
		return api.BadRequest(c, "invalid request body")
	}

	webhook, err := h.webhookService.UpdateWebhookForUser(c.Request().Context(), webhookID, userID, &input)
	if err != nil {
		return api.FromError(c, err)
	}

	return api.OK(c, webhook)
}

// DeleteWebhook handles webhook deletion requests.
// @Summary Delete Webhook
// @Description Delete a webhook
// @Tags Webhooks
// @Produce json
// @Security BearerAuth
// @Param id path string true "Webhook ID"
// @Success 204
// @Failure 401 {object} api.Response{error=api.ErrorDetail}
// @Failure 404 {object} api.Response{error=api.ErrorDetail}
// @Failure 500 {object} api.Response{error=api.ErrorDetail}
// @Router /webhooks/{id} [delete]
func (h *WebhookHandler) DeleteWebhook(c echo.Context) error {
	userID := middleware.GetUserID(c)
	if userID.IsEmpty() {
		return api.Unauthorized(c, "authentication required")
	}

	webhookID := domain.ID(c.Param("id"))
	if webhookID.IsEmpty() {
		return api.BadRequest(c, "webhook ID is required")
	}

	if err := h.webhookService.DeleteWebhookForUser(c.Request().Context(), webhookID, userID); err != nil {
		return api.FromError(c, err)
	}

	return api.NoContent(c)
}

// TestWebhook handles requests to test a webhook.
// @Summary Test Webhook
// @Description Send a test payload to the webhook endpoint
// @Tags Webhooks
// @Produce json
// @Security BearerAuth
// @Param id path string true "Webhook ID"
// @Success 200 {object} api.Response{data=domain.WebhookDelivery}
// @Failure 401 {object} api.Response{error=api.ErrorDetail}
// @Failure 404 {object} api.Response{error=api.ErrorDetail}
// @Failure 500 {object} api.Response{error=api.ErrorDetail}
// @Router /webhooks/{id}/test [post]
func (h *WebhookHandler) TestWebhook(c echo.Context) error {
	userID := middleware.GetUserID(c)
	if userID.IsEmpty() {
		return api.Unauthorized(c, "authentication required")
	}

	webhookID := domain.ID(c.Param("id"))
	if webhookID.IsEmpty() {
		return api.BadRequest(c, "webhook ID is required")
	}

	delivery, err := h.webhookService.TestWebhookForUser(c.Request().Context(), webhookID, userID)
	if err != nil {
		return api.FromError(c, err)
	}

	return api.OK(c, delivery)
}

// ActivateWebhook handles requests to activate a webhook.
// @Summary Activate Webhook
// @Description Activate a deactivated webhook
// @Tags Webhooks
// @Produce json
// @Security BearerAuth
// @Param id path string true "Webhook ID"
// @Success 200 {object} api.Response{data=domain.Webhook}
// @Failure 401 {object} api.Response{error=api.ErrorDetail}
// @Failure 404 {object} api.Response{error=api.ErrorDetail}
// @Failure 500 {object} api.Response{error=api.ErrorDetail}
// @Router /webhooks/{id}/activate [post]
func (h *WebhookHandler) ActivateWebhook(c echo.Context) error {
	userID := middleware.GetUserID(c)
	if userID.IsEmpty() {
		return api.Unauthorized(c, "authentication required")
	}

	webhookID := domain.ID(c.Param("id"))
	if webhookID.IsEmpty() {
		return api.BadRequest(c, "webhook ID is required")
	}

	// Verify ownership first
	if _, err := h.webhookService.GetWebhookForUser(c.Request().Context(), webhookID, userID); err != nil {
		return api.FromError(c, err)
	}

	if err := h.webhookService.ActivateWebhook(c.Request().Context(), webhookID); err != nil {
		return api.FromError(c, err)
	}

	// Refresh webhook to return updated state
	webhook, err := h.webhookService.GetWebhook(c.Request().Context(), webhookID)
	if err != nil {
		return api.FromError(c, err)
	}

	return api.OK(c, webhook)
}

// DeactivateWebhook handles requests to deactivate a webhook.
// @Summary Deactivate Webhook
// @Description Deactivate an active webhook
// @Tags Webhooks
// @Produce json
// @Security BearerAuth
// @Param id path string true "Webhook ID"
// @Success 200 {object} api.Response{data=domain.Webhook}
// @Failure 401 {object} api.Response{error=api.ErrorDetail}
// @Failure 404 {object} api.Response{error=api.ErrorDetail}
// @Failure 500 {object} api.Response{error=api.ErrorDetail}
// @Router /webhooks/{id}/deactivate [post]
func (h *WebhookHandler) DeactivateWebhook(c echo.Context) error {
	userID := middleware.GetUserID(c)
	if userID.IsEmpty() {
		return api.Unauthorized(c, "authentication required")
	}

	webhookID := domain.ID(c.Param("id"))
	if webhookID.IsEmpty() {
		return api.BadRequest(c, "webhook ID is required")
	}

	// Verify ownership first
	if _, err := h.webhookService.GetWebhookForUser(c.Request().Context(), webhookID, userID); err != nil {
		return api.FromError(c, err)
	}

	if err := h.webhookService.DeactivateWebhook(c.Request().Context(), webhookID); err != nil {
		return api.FromError(c, err)
	}

	// Refresh webhook to return updated state
	webhook, err := h.webhookService.GetWebhook(c.Request().Context(), webhookID)
	if err != nil {
		return api.FromError(c, err)
	}

	return api.OK(c, webhook)
}

// ListDeliveries handles requests to list webhook deliveries.
// @Summary List Webhook Deliveries
// @Description List delivery history for a webhook
// @Tags Webhooks
// @Produce json
// @Security BearerAuth
// @Param id path string true "Webhook ID"
// @Param page query int false "Page number (default: 1)"
// @Param pageSize query int false "Page size (default: 20, max: 100)"
// @Success 200 {object} api.Response{data=api.PaginatedData}
// @Failure 401 {object} api.Response{error=api.ErrorDetail}
// @Failure 404 {object} api.Response{error=api.ErrorDetail}
// @Failure 500 {object} api.Response{error=api.ErrorDetail}
// @Router /webhooks/{id}/deliveries [get]
func (h *WebhookHandler) ListDeliveries(c echo.Context) error {
	userID := middleware.GetUserID(c)
	if userID.IsEmpty() {
		return api.Unauthorized(c, "authentication required")
	}

	webhookID := domain.ID(c.Param("id"))
	if webhookID.IsEmpty() {
		return api.BadRequest(c, "webhook ID is required")
	}

	// Verify ownership
	_, err := h.webhookService.GetWebhookForUser(c.Request().Context(), webhookID, userID)
	if err != nil {
		return api.FromError(c, err)
	}

	// Parse pagination
	page := parseIntParam(c, "page", 1)
	pageSize := parseIntParam(c, "pageSize", 20)
	if pageSize > 100 {
		pageSize = 100
	}

	opts := &repository.ListOptions{
		Pagination: &repository.PaginationOptions{
			Page:    page,
			PerPage: pageSize,
		},
		Sort: &repository.SortOptions{
			Field: "createdAt",
			Order: domain.SortDesc,
		},
	}

	result, err := h.webhookService.ListDeliveries(c.Request().Context(), webhookID, opts)
	if err != nil {
		return api.FromError(c, err)
	}

	return api.Paginated(c, result.Items, page, pageSize, result.Total)
}

// GetDeliveryStats handles requests to get webhook delivery statistics.
// @Summary Get Webhook Delivery Stats
// @Description Get delivery statistics for a webhook
// @Tags Webhooks
// @Produce json
// @Security BearerAuth
// @Param id path string true "Webhook ID"
// @Success 200 {object} api.Response{data=repository.WebhookDeliveryStats}
// @Failure 401 {object} api.Response{error=api.ErrorDetail}
// @Failure 404 {object} api.Response{error=api.ErrorDetail}
// @Failure 500 {object} api.Response{error=api.ErrorDetail}
// @Router /webhooks/{id}/stats [get]
func (h *WebhookHandler) GetDeliveryStats(c echo.Context) error {
	userID := middleware.GetUserID(c)
	if userID.IsEmpty() {
		return api.Unauthorized(c, "authentication required")
	}

	webhookID := domain.ID(c.Param("id"))
	if webhookID.IsEmpty() {
		return api.BadRequest(c, "webhook ID is required")
	}

	// Verify ownership
	_, err := h.webhookService.GetWebhookForUser(c.Request().Context(), webhookID, userID)
	if err != nil {
		return api.FromError(c, err)
	}

	stats, err := h.webhookService.GetDeliveryStats(c.Request().Context(), webhookID)
	if err != nil {
		return api.FromError(c, err)
	}

	return api.OK(c, stats)
}

// parseIntParam parses an integer query parameter with a default value.
func parseIntParam(c echo.Context, name string, defaultValue int) int {
	valueStr := c.QueryParam(name)
	if valueStr == "" {
		return defaultValue
	}

	value, err := strconv.Atoi(valueStr)
	if err != nil || value < 1 {
		return defaultValue
	}

	return value
}
