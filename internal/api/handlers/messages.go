// Package handlers provides HTTP request handlers for the Yunt API.
package handlers

import (
	"fmt"
	"net/http"
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

// MessageHandler handles message-related HTTP requests.
type MessageHandler struct {
	messageService *service.MessageService
	mailboxService *service.MailboxService
	authService    *service.AuthService
	notifyService  *service.NotifyService
}

// NewMessageHandler creates a new MessageHandler.
func NewMessageHandler(
	messageService *service.MessageService,
	mailboxService *service.MailboxService,
	authService *service.AuthService,
) *MessageHandler {
	return &MessageHandler{
		messageService: messageService,
		mailboxService: mailboxService,
		authService:    authService,
	}
}

// WithNotifyService sets the notification service for real-time events.
func (h *MessageHandler) WithNotifyService(ns *service.NotifyService) *MessageHandler {
	h.notifyService = ns
	return h
}

// RegisterRoutes registers the message routes on the given group.
func (h *MessageHandler) RegisterRoutes(g *echo.Group) {
	messages := g.Group("/messages", middleware.Auth(h.authService))
	messages.GET("", h.ListMessages)
	messages.GET("/search", h.SearchMessages)
	messages.GET("/:id", h.GetMessage)
	messages.GET("/:id/html", h.GetMessageHTML)
	messages.GET("/:id/text", h.GetMessageText)
	messages.GET("/:id/raw", h.GetMessageRaw)
	messages.GET("/:id/attachments", h.ListAttachments)
	messages.GET("/:id/attachments/:attachmentId", h.GetAttachment)
	messages.GET("/:id/attachments/:attachmentId/download", h.DownloadAttachment)
	messages.DELETE("/:id", h.DeleteMessage)
	messages.PUT("/:id/read", h.MarkAsRead)
	messages.PUT("/:id/unread", h.MarkAsUnread)
	messages.PUT("/:id/star", h.Star)
	messages.PUT("/:id/unstar", h.Unstar)
	messages.PUT("/:id/spam", h.MarkAsSpam)
	messages.PUT("/:id/not-spam", h.MarkAsNotSpam)
	messages.PUT("/:id/move", h.MoveMessage)

	// Bulk operations
	messages.DELETE("", h.BulkDelete)
	messages.POST("/bulk/read", h.BulkMarkAsRead)
	messages.POST("/bulk/unread", h.BulkMarkAsUnread)
	messages.POST("/bulk/delete", h.BulkDelete)
	messages.POST("/bulk/move", h.BulkMove)
	messages.POST("/bulk/star", h.BulkStar)
	messages.POST("/bulk/unstar", h.BulkUnstar)
}

// ListMessages handles requests to list messages with filters.
// @Summary List messages
// @Description Get messages with optional filtering, sorting, and pagination
// @Tags Messages
// @Produce json
// @Security BearerAuth
// @Param mailboxId query string false "Filter by mailbox ID"
// @Param status query string false "Filter by status (read, unread)"
// @Param isStarred query bool false "Filter by starred status"
// @Param isSpam query bool false "Filter by spam status"
// @Param hasAttachments query bool false "Filter by attachment presence"
// @Param from query string false "Filter by sender address"
// @Param to query string false "Filter by recipient address"
// @Param subject query string false "Filter by subject (partial match)"
// @Param receivedAfter query string false "Filter messages received after (RFC3339)"
// @Param receivedBefore query string false "Filter messages received before (RFC3339)"
// @Param page query int false "Page number (default: 1)"
// @Param perPage query int false "Items per page (default: 20, max: 100)"
// @Param sort query string false "Sort field (receivedAt, subject, from, size)"
// @Param order query string false "Sort order (asc, desc)"
// @Success 200 {object} api.Response{data=api.PaginatedData}
// @Failure 401 {object} api.Response{error=api.ErrorDetail}
// @Failure 403 {object} api.Response{error=api.ErrorDetail}
// @Failure 500 {object} api.Response{error=api.ErrorDetail}
// @Router /messages [get]
func (h *MessageHandler) ListMessages(c echo.Context) error {
	userID := middleware.GetUserID(c)
	if userID.IsEmpty() {
		return api.Unauthorized(c, "authentication required")
	}

	// Parse filter options
	filter, err := h.parseMessageFilter(c, userID)
	if err != nil {
		return api.BadRequest(c, err.Error())
	}

	// Parse list options
	opts := h.parseListOptions(c)

	result, err := h.messageService.ListMessagesForUser(c.Request().Context(), userID, filter, opts)
	if err != nil {
		return api.FromError(c, err)
	}

	// ETag based on total count + latest message timestamp
	var latestTS int64
	if len(result.Items) > 0 {
		latestTS = result.Items[0].ReceivedAt.Unix()
	}
	etag := fmt.Sprintf(`"%d-%d"`, result.Total, latestTS)

	if match := c.Request().Header.Get("If-None-Match"); match == etag {
		return c.NoContent(http.StatusNotModified)
	}
	c.Response().Header().Set("ETag", etag)

	// Determine page and perPage from options
	page := 1
	perPage := repository.DefaultPerPage
	if opts != nil && opts.Pagination != nil {
		page = opts.Pagination.Page
		perPage = opts.Pagination.PerPage
	}

	return api.Paginated(c, result.Items, page, perPage, result.Total)
}

// SearchMessages handles requests to search messages.
// @Summary Search messages
// @Description Search messages by query string
// @Tags Messages
// @Produce json
// @Security BearerAuth
// @Param q query string true "Search query"
// @Param mailboxId query string false "Filter by mailbox ID"
// @Param page query int false "Page number (default: 1)"
// @Param perPage query int false "Items per page (default: 20, max: 100)"
// @Success 200 {object} api.Response{data=api.PaginatedData}
// @Failure 400 {object} api.Response{error=api.ErrorDetail}
// @Failure 401 {object} api.Response{error=api.ErrorDetail}
// @Failure 500 {object} api.Response{error=api.ErrorDetail}
// @Router /messages/search [get]
func (h *MessageHandler) SearchMessages(c echo.Context) error {
	userID := middleware.GetUserID(c)
	if userID.IsEmpty() {
		return api.Unauthorized(c, "authentication required")
	}

	query := c.QueryParam("q")
	if query == "" {
		return api.BadRequest(c, "search query is required")
	}

	// Parse filter options
	filter, err := h.parseMessageFilter(c, userID)
	if err != nil {
		return api.BadRequest(c, err.Error())
	}

	// Parse list options
	opts := h.parseListOptions(c)

	// Create search options
	searchOpts := &repository.SearchOptions{
		Query: query,
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

// GetMessage handles requests to get a specific message.
// @Summary Get message
// @Description Get a specific message by ID with full details
// @Tags Messages
// @Produce json
// @Security BearerAuth
// @Param id path string true "Message ID"
// @Success 200 {object} api.Response{data=domain.Message}
// @Failure 401 {object} api.Response{error=api.ErrorDetail}
// @Failure 403 {object} api.Response{error=api.ErrorDetail}
// @Failure 404 {object} api.Response{error=api.ErrorDetail}
// @Failure 500 {object} api.Response{error=api.ErrorDetail}
// @Router /messages/{id} [get]
func (h *MessageHandler) GetMessage(c echo.Context) error {
	userID := middleware.GetUserID(c)
	if userID.IsEmpty() {
		return api.Unauthorized(c, "authentication required")
	}

	messageID := domain.ID(c.Param("id"))
	if messageID.IsEmpty() {
		return api.BadRequest(c, "message ID is required")
	}

	message, err := h.messageService.GetMessageForUser(c.Request().Context(), messageID, userID)
	if err != nil {
		return api.FromError(c, err)
	}

	return api.OK(c, message)
}

// GetMessageHTML handles requests to get message HTML content.
// @Summary Get message HTML body
// @Description Get the HTML body of a message
// @Tags Messages
// @Produce html
// @Security BearerAuth
// @Param id path string true "Message ID"
// @Success 200 {string} string "HTML content"
// @Failure 401 {object} api.Response{error=api.ErrorDetail}
// @Failure 403 {object} api.Response{error=api.ErrorDetail}
// @Failure 404 {object} api.Response{error=api.ErrorDetail}
// @Failure 500 {object} api.Response{error=api.ErrorDetail}
// @Router /messages/{id}/html [get]
func (h *MessageHandler) GetMessageHTML(c echo.Context) error {
	userID := middleware.GetUserID(c)
	if userID.IsEmpty() {
		return api.Unauthorized(c, "authentication required")
	}

	messageID := domain.ID(c.Param("id"))
	if messageID.IsEmpty() {
		return api.BadRequest(c, "message ID is required")
	}

	message, err := h.messageService.GetMessageForUser(c.Request().Context(), messageID, userID)
	if err != nil {
		return api.FromError(c, err)
	}

	if message.HTMLBody == "" {
		return api.NotFound(c, "message has no HTML body")
	}

	return c.HTML(http.StatusOK, message.HTMLBody)
}

// GetMessageText handles requests to get message text content.
// @Summary Get message text body
// @Description Get the plain text body of a message
// @Tags Messages
// @Produce plain
// @Security BearerAuth
// @Param id path string true "Message ID"
// @Success 200 {string} string "Text content"
// @Failure 401 {object} api.Response{error=api.ErrorDetail}
// @Failure 403 {object} api.Response{error=api.ErrorDetail}
// @Failure 404 {object} api.Response{error=api.ErrorDetail}
// @Failure 500 {object} api.Response{error=api.ErrorDetail}
// @Router /messages/{id}/text [get]
func (h *MessageHandler) GetMessageText(c echo.Context) error {
	userID := middleware.GetUserID(c)
	if userID.IsEmpty() {
		return api.Unauthorized(c, "authentication required")
	}

	messageID := domain.ID(c.Param("id"))
	if messageID.IsEmpty() {
		return api.BadRequest(c, "message ID is required")
	}

	message, err := h.messageService.GetMessageForUser(c.Request().Context(), messageID, userID)
	if err != nil {
		return api.FromError(c, err)
	}

	if message.TextBody == "" {
		return api.NotFound(c, "message has no text body")
	}

	return c.String(http.StatusOK, message.TextBody)
}

// GetMessageRaw handles requests to get raw message (EML format).
// @Summary Get raw message (EML)
// @Description Get the raw EML data of a message
// @Tags Messages
// @Produce application/octet-stream
// @Security BearerAuth
// @Param id path string true "Message ID"
// @Success 200 {file} binary "EML file"
// @Failure 401 {object} api.Response{error=api.ErrorDetail}
// @Failure 403 {object} api.Response{error=api.ErrorDetail}
// @Failure 404 {object} api.Response{error=api.ErrorDetail}
// @Failure 500 {object} api.Response{error=api.ErrorDetail}
// @Router /messages/{id}/raw [get]
func (h *MessageHandler) GetMessageRaw(c echo.Context) error {
	userID := middleware.GetUserID(c)
	if userID.IsEmpty() {
		return api.Unauthorized(c, "authentication required")
	}

	messageID := domain.ID(c.Param("id"))
	if messageID.IsEmpty() {
		return api.BadRequest(c, "message ID is required")
	}

	rawData, err := h.messageService.GetRawMessageForUser(c.Request().Context(), messageID, userID)
	if err != nil {
		return api.FromError(c, err)
	}

	// Set headers for download
	c.Response().Header().Set("Content-Type", "message/rfc822")
	c.Response().Header().Set("Content-Disposition", "attachment; filename=\"message.eml\"")

	return c.Blob(http.StatusOK, "message/rfc822", rawData)
}

// ListAttachments handles requests to list attachments for a message.
// @Summary List message attachments
// @Description Get all attachments for a specific message
// @Tags Messages
// @Produce json
// @Security BearerAuth
// @Param id path string true "Message ID"
// @Success 200 {object} api.Response{data=[]domain.AttachmentSummary}
// @Failure 401 {object} api.Response{error=api.ErrorDetail}
// @Failure 403 {object} api.Response{error=api.ErrorDetail}
// @Failure 404 {object} api.Response{error=api.ErrorDetail}
// @Failure 500 {object} api.Response{error=api.ErrorDetail}
// @Router /messages/{id}/attachments [get]
func (h *MessageHandler) ListAttachments(c echo.Context) error {
	userID := middleware.GetUserID(c)
	if userID.IsEmpty() {
		return api.Unauthorized(c, "authentication required")
	}

	messageID := domain.ID(c.Param("id"))
	if messageID.IsEmpty() {
		return api.BadRequest(c, "message ID is required")
	}

	attachments, err := h.messageService.GetAttachmentsForUser(c.Request().Context(), messageID, userID)
	if err != nil {
		return api.FromError(c, err)
	}

	// Convert to summaries
	summaries := make([]*domain.AttachmentSummary, len(attachments))
	for i, att := range attachments {
		summaries[i] = att.ToSummary()
	}

	return api.OK(c, summaries)
}

// GetAttachment handles requests to get attachment metadata.
// @Summary Get attachment metadata
// @Description Get metadata for a specific attachment
// @Tags Messages
// @Produce json
// @Security BearerAuth
// @Param id path string true "Message ID"
// @Param attachmentId path string true "Attachment ID"
// @Success 200 {object} api.Response{data=domain.Attachment}
// @Failure 401 {object} api.Response{error=api.ErrorDetail}
// @Failure 403 {object} api.Response{error=api.ErrorDetail}
// @Failure 404 {object} api.Response{error=api.ErrorDetail}
// @Failure 500 {object} api.Response{error=api.ErrorDetail}
// @Router /messages/{id}/attachments/{attachmentId} [get]
func (h *MessageHandler) GetAttachment(c echo.Context) error {
	userID := middleware.GetUserID(c)
	if userID.IsEmpty() {
		return api.Unauthorized(c, "authentication required")
	}

	messageID := domain.ID(c.Param("id"))
	if messageID.IsEmpty() {
		return api.BadRequest(c, "message ID is required")
	}

	attachmentID := domain.ID(c.Param("attachmentId"))
	if attachmentID.IsEmpty() {
		return api.BadRequest(c, "attachment ID is required")
	}

	attachment, err := h.messageService.GetAttachmentForUser(c.Request().Context(), messageID, attachmentID, userID)
	if err != nil {
		return api.FromError(c, err)
	}

	return api.OK(c, attachment)
}

// DownloadAttachment handles requests to download attachment content.
// @Summary Download attachment
// @Description Download the content of an attachment
// @Tags Messages
// @Produce application/octet-stream
// @Security BearerAuth
// @Param id path string true "Message ID"
// @Param attachmentId path string true "Attachment ID"
// @Success 200 {file} binary "Attachment file"
// @Failure 401 {object} api.Response{error=api.ErrorDetail}
// @Failure 403 {object} api.Response{error=api.ErrorDetail}
// @Failure 404 {object} api.Response{error=api.ErrorDetail}
// @Failure 500 {object} api.Response{error=api.ErrorDetail}
// @Router /messages/{id}/attachments/{attachmentId}/download [get]
func (h *MessageHandler) DownloadAttachment(c echo.Context) error {
	userID := middleware.GetUserID(c)
	if userID.IsEmpty() {
		return api.Unauthorized(c, "authentication required")
	}

	messageID := domain.ID(c.Param("id"))
	if messageID.IsEmpty() {
		return api.BadRequest(c, "message ID is required")
	}

	attachmentID := domain.ID(c.Param("attachmentId"))
	if attachmentID.IsEmpty() {
		return api.BadRequest(c, "attachment ID is required")
	}

	attachment, content, err := h.messageService.GetAttachmentContentForUser(c.Request().Context(), messageID, attachmentID, userID)
	if err != nil {
		return api.FromError(c, err)
	}
	defer content.Close()

	// Set headers for download
	c.Response().Header().Set("Content-Type", attachment.ContentType)
	c.Response().Header().Set("Content-Disposition", "attachment; filename=\""+attachment.Filename+"\"")
	c.Response().Header().Set("Content-Length", strconv.FormatInt(attachment.Size, 10))

	return c.Stream(http.StatusOK, attachment.ContentType, content)
}

// DeleteMessage handles requests to delete a message.
// @Summary Delete message
// @Description Delete a message permanently
// @Tags Messages
// @Produce json
// @Security BearerAuth
// @Param id path string true "Message ID"
// @Success 204
// @Failure 401 {object} api.Response{error=api.ErrorDetail}
// @Failure 403 {object} api.Response{error=api.ErrorDetail}
// @Failure 404 {object} api.Response{error=api.ErrorDetail}
// @Failure 500 {object} api.Response{error=api.ErrorDetail}
// @Router /messages/{id} [delete]
func (h *MessageHandler) DeleteMessage(c echo.Context) error {
	userID := middleware.GetUserID(c)
	if userID.IsEmpty() {
		return api.Unauthorized(c, "authentication required")
	}

	messageID := domain.ID(c.Param("id"))
	if messageID.IsEmpty() {
		return api.BadRequest(c, "message ID is required")
	}

	if err := h.messageService.DeleteMessageForUser(c.Request().Context(), messageID, userID); err != nil {
		return api.FromError(c, err)
	}

	return api.NoContent(c)
}

// MarkAsRead handles requests to mark a message as read.
// @Summary Mark message as read
// @Description Mark a message as read
// @Tags Messages
// @Produce json
// @Security BearerAuth
// @Param id path string true "Message ID"
// @Success 204
// @Failure 401 {object} api.Response{error=api.ErrorDetail}
// @Failure 403 {object} api.Response{error=api.ErrorDetail}
// @Failure 404 {object} api.Response{error=api.ErrorDetail}
// @Failure 500 {object} api.Response{error=api.ErrorDetail}
// @Router /messages/{id}/read [post]
func (h *MessageHandler) MarkAsRead(c echo.Context) error {
	userID := middleware.GetUserID(c)
	if userID.IsEmpty() {
		return api.Unauthorized(c, "authentication required")
	}

	messageID := domain.ID(c.Param("id"))
	if messageID.IsEmpty() {
		return api.BadRequest(c, "message ID is required")
	}

	if err := h.messageService.MarkAsReadForUser(c.Request().Context(), messageID, userID); err != nil {
		return api.FromError(c, err)
	}

	return api.NoContent(c)
}

// MarkAsUnread handles requests to mark a message as unread.
// @Summary Mark message as unread
// @Description Mark a message as unread
// @Tags Messages
// @Produce json
// @Security BearerAuth
// @Param id path string true "Message ID"
// @Success 204
// @Failure 401 {object} api.Response{error=api.ErrorDetail}
// @Failure 403 {object} api.Response{error=api.ErrorDetail}
// @Failure 404 {object} api.Response{error=api.ErrorDetail}
// @Failure 500 {object} api.Response{error=api.ErrorDetail}
// @Router /messages/{id}/unread [post]
func (h *MessageHandler) MarkAsUnread(c echo.Context) error {
	userID := middleware.GetUserID(c)
	if userID.IsEmpty() {
		return api.Unauthorized(c, "authentication required")
	}

	messageID := domain.ID(c.Param("id"))
	if messageID.IsEmpty() {
		return api.BadRequest(c, "message ID is required")
	}

	if err := h.messageService.MarkAsUnreadForUser(c.Request().Context(), messageID, userID); err != nil {
		return api.FromError(c, err)
	}

	return api.NoContent(c)
}

// Star handles requests to star a message.
// @Summary Star message
// @Description Mark a message as starred
// @Tags Messages
// @Produce json
// @Security BearerAuth
// @Param id path string true "Message ID"
// @Success 204
// @Failure 401 {object} api.Response{error=api.ErrorDetail}
// @Failure 403 {object} api.Response{error=api.ErrorDetail}
// @Failure 404 {object} api.Response{error=api.ErrorDetail}
// @Failure 500 {object} api.Response{error=api.ErrorDetail}
// @Router /messages/{id}/star [post]
func (h *MessageHandler) Star(c echo.Context) error {
	userID := middleware.GetUserID(c)
	if userID.IsEmpty() {
		return api.Unauthorized(c, "authentication required")
	}

	messageID := domain.ID(c.Param("id"))
	if messageID.IsEmpty() {
		return api.BadRequest(c, "message ID is required")
	}

	if err := h.messageService.StarForUser(c.Request().Context(), messageID, userID); err != nil {
		return api.FromError(c, err)
	}

	return api.NoContent(c)
}

// Unstar handles requests to unstar a message.
// @Summary Unstar message
// @Description Remove star from a message
// @Tags Messages
// @Produce json
// @Security BearerAuth
// @Param id path string true "Message ID"
// @Success 204
// @Failure 401 {object} api.Response{error=api.ErrorDetail}
// @Failure 403 {object} api.Response{error=api.ErrorDetail}
// @Failure 404 {object} api.Response{error=api.ErrorDetail}
// @Failure 500 {object} api.Response{error=api.ErrorDetail}
// @Router /messages/{id}/unstar [post]
func (h *MessageHandler) Unstar(c echo.Context) error {
	userID := middleware.GetUserID(c)
	if userID.IsEmpty() {
		return api.Unauthorized(c, "authentication required")
	}

	messageID := domain.ID(c.Param("id"))
	if messageID.IsEmpty() {
		return api.BadRequest(c, "message ID is required")
	}

	if err := h.messageService.UnstarForUser(c.Request().Context(), messageID, userID); err != nil {
		return api.FromError(c, err)
	}

	return api.NoContent(c)
}

// MarkAsSpam handles requests to mark a message as spam.
// @Summary Mark message as spam
// @Description Mark a message as spam
// @Tags Messages
// @Produce json
// @Security BearerAuth
// @Param id path string true "Message ID"
// @Success 204
// @Failure 401 {object} api.Response{error=api.ErrorDetail}
// @Failure 403 {object} api.Response{error=api.ErrorDetail}
// @Failure 404 {object} api.Response{error=api.ErrorDetail}
// @Failure 500 {object} api.Response{error=api.ErrorDetail}
// @Router /messages/{id}/spam [post]
func (h *MessageHandler) MarkAsSpam(c echo.Context) error {
	userID := middleware.GetUserID(c)
	if userID.IsEmpty() {
		return api.Unauthorized(c, "authentication required")
	}

	messageID := domain.ID(c.Param("id"))
	if messageID.IsEmpty() {
		return api.BadRequest(c, "message ID is required")
	}

	if err := h.messageService.MarkAsSpamForUser(c.Request().Context(), messageID, userID); err != nil {
		return api.FromError(c, err)
	}

	return api.NoContent(c)
}

// MarkAsNotSpam handles requests to mark a message as not spam.
// @Summary Mark message as not spam
// @Description Remove spam flag from a message
// @Tags Messages
// @Produce json
// @Security BearerAuth
// @Param id path string true "Message ID"
// @Success 204
// @Failure 401 {object} api.Response{error=api.ErrorDetail}
// @Failure 403 {object} api.Response{error=api.ErrorDetail}
// @Failure 404 {object} api.Response{error=api.ErrorDetail}
// @Failure 500 {object} api.Response{error=api.ErrorDetail}
// @Router /messages/{id}/not-spam [post]
func (h *MessageHandler) MarkAsNotSpam(c echo.Context) error {
	userID := middleware.GetUserID(c)
	if userID.IsEmpty() {
		return api.Unauthorized(c, "authentication required")
	}

	messageID := domain.ID(c.Param("id"))
	if messageID.IsEmpty() {
		return api.BadRequest(c, "message ID is required")
	}

	if err := h.messageService.MarkAsNotSpamForUser(c.Request().Context(), messageID, userID); err != nil {
		return api.FromError(c, err)
	}

	return api.NoContent(c)
}

// MoveMessageInput represents the input for moving a message.
type MoveMessageInput struct {
	TargetMailboxID string `json:"targetMailboxId"`
}

// MoveMessage handles requests to move a message to another mailbox.
// @Summary Move message
// @Description Move a message to a different mailbox
// @Tags Messages
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "Message ID"
// @Param input body MoveMessageInput true "Move input"
// @Success 204
// @Failure 400 {object} api.Response{error=api.ErrorDetail}
// @Failure 401 {object} api.Response{error=api.ErrorDetail}
// @Failure 403 {object} api.Response{error=api.ErrorDetail}
// @Failure 404 {object} api.Response{error=api.ErrorDetail}
// @Failure 500 {object} api.Response{error=api.ErrorDetail}
// @Router /messages/{id}/move [post]
func (h *MessageHandler) MoveMessage(c echo.Context) error {
	userID := middleware.GetUserID(c)
	if userID.IsEmpty() {
		return api.Unauthorized(c, "authentication required")
	}

	messageID := domain.ID(c.Param("id"))
	if messageID.IsEmpty() {
		return api.BadRequest(c, "message ID is required")
	}

	var input MoveMessageInput
	if err := c.Bind(&input); err != nil {
		return api.BadRequest(c, "invalid request body")
	}

	targetMailboxID := domain.ID(input.TargetMailboxID)
	if targetMailboxID.IsEmpty() {
		return api.BadRequest(c, "target mailbox ID is required")
	}

	if err := h.messageService.MoveMessageForUser(c.Request().Context(), messageID, targetMailboxID, userID); err != nil {
		return api.FromError(c, err)
	}

	return api.NoContent(c)
}

// BulkIDsInput represents the input for bulk operations with IDs.
type BulkIDsInput struct {
	IDs []string `json:"ids"`
}

// BulkMoveInput represents the input for bulk move operation.
type BulkMoveInput struct {
	IDs             []string `json:"ids"`
	TargetMailboxID string   `json:"targetMailboxId"`
}

// BulkOperationResponse represents the result of a bulk operation.
type BulkOperationResponse struct {
	Succeeded int64             `json:"succeeded"`
	Failed    int64             `json:"failed"`
	Errors    map[string]string `json:"errors,omitempty"`
}

// BulkMarkAsRead handles requests to mark multiple messages as read.
// @Summary Bulk mark as read
// @Description Mark multiple messages as read
// @Tags Messages
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param input body BulkIDsInput true "Message IDs"
// @Success 200 {object} api.Response{data=BulkOperationResponse}
// @Failure 400 {object} api.Response{error=api.ErrorDetail}
// @Failure 401 {object} api.Response{error=api.ErrorDetail}
// @Failure 500 {object} api.Response{error=api.ErrorDetail}
// @Router /messages/bulk/read [post]
func (h *MessageHandler) BulkMarkAsRead(c echo.Context) error {
	userID := middleware.GetUserID(c)
	if userID.IsEmpty() {
		return api.Unauthorized(c, "authentication required")
	}

	var input BulkIDsInput
	if err := c.Bind(&input); err != nil {
		return api.BadRequest(c, "invalid request body")
	}

	if len(input.IDs) == 0 {
		return api.BadRequest(c, "at least one message ID is required")
	}

	ids := make([]domain.ID, len(input.IDs))
	for i, id := range input.IDs {
		ids[i] = domain.ID(id)
	}

	result, err := h.messageService.BulkMarkAsReadForUser(c.Request().Context(), ids, userID)
	if err != nil {
		return api.FromError(c, err)
	}

	return api.OK(c, &BulkOperationResponse{
		Succeeded: result.Succeeded,
		Failed:    result.Failed,
		Errors:    result.Errors,
	})
}

// BulkMarkAsUnread handles requests to mark multiple messages as unread.
// @Summary Bulk mark as unread
// @Description Mark multiple messages as unread
// @Tags Messages
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param input body BulkIDsInput true "Message IDs"
// @Success 200 {object} api.Response{data=BulkOperationResponse}
// @Failure 400 {object} api.Response{error=api.ErrorDetail}
// @Failure 401 {object} api.Response{error=api.ErrorDetail}
// @Failure 500 {object} api.Response{error=api.ErrorDetail}
// @Router /messages/bulk/unread [post]
func (h *MessageHandler) BulkMarkAsUnread(c echo.Context) error {
	userID := middleware.GetUserID(c)
	if userID.IsEmpty() {
		return api.Unauthorized(c, "authentication required")
	}

	var input BulkIDsInput
	if err := c.Bind(&input); err != nil {
		return api.BadRequest(c, "invalid request body")
	}

	if len(input.IDs) == 0 {
		return api.BadRequest(c, "at least one message ID is required")
	}

	ids := make([]domain.ID, len(input.IDs))
	for i, id := range input.IDs {
		ids[i] = domain.ID(id)
	}

	result, err := h.messageService.BulkMarkAsUnreadForUser(c.Request().Context(), ids, userID)
	if err != nil {
		return api.FromError(c, err)
	}

	return api.OK(c, &BulkOperationResponse{
		Succeeded: result.Succeeded,
		Failed:    result.Failed,
		Errors:    result.Errors,
	})
}

// BulkDelete handles requests to delete multiple messages.
// @Summary Bulk delete messages
// @Description Delete multiple messages permanently
// @Tags Messages
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param input body BulkIDsInput true "Message IDs"
// @Success 200 {object} api.Response{data=BulkOperationResponse}
// @Failure 400 {object} api.Response{error=api.ErrorDetail}
// @Failure 401 {object} api.Response{error=api.ErrorDetail}
// @Failure 500 {object} api.Response{error=api.ErrorDetail}
// @Router /messages/bulk/delete [post]
func (h *MessageHandler) BulkDelete(c echo.Context) error {
	userID := middleware.GetUserID(c)
	if userID.IsEmpty() {
		return api.Unauthorized(c, "authentication required")
	}

	var input BulkIDsInput
	if err := c.Bind(&input); err != nil {
		return api.BadRequest(c, "invalid request body")
	}

	if len(input.IDs) == 0 {
		return api.BadRequest(c, "at least one message ID is required")
	}

	ids := make([]domain.ID, len(input.IDs))
	for i, id := range input.IDs {
		ids[i] = domain.ID(id)
	}

	result, err := h.messageService.BulkDeleteForUser(c.Request().Context(), ids, userID)
	if err != nil {
		return api.FromError(c, err)
	}

	return api.OK(c, &BulkOperationResponse{
		Succeeded: result.Succeeded,
		Failed:    result.Failed,
		Errors:    result.Errors,
	})
}

// BulkMove handles requests to move multiple messages.
// @Summary Bulk move messages
// @Description Move multiple messages to a different mailbox
// @Tags Messages
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param input body BulkMoveInput true "Message IDs and target mailbox"
// @Success 200 {object} api.Response{data=BulkOperationResponse}
// @Failure 400 {object} api.Response{error=api.ErrorDetail}
// @Failure 401 {object} api.Response{error=api.ErrorDetail}
// @Failure 500 {object} api.Response{error=api.ErrorDetail}
// @Router /messages/bulk/move [post]
func (h *MessageHandler) BulkMove(c echo.Context) error {
	userID := middleware.GetUserID(c)
	if userID.IsEmpty() {
		return api.Unauthorized(c, "authentication required")
	}

	var input BulkMoveInput
	if err := c.Bind(&input); err != nil {
		return api.BadRequest(c, "invalid request body")
	}

	if len(input.IDs) == 0 {
		return api.BadRequest(c, "at least one message ID is required")
	}

	targetMailboxID := domain.ID(input.TargetMailboxID)
	if targetMailboxID.IsEmpty() {
		return api.BadRequest(c, "target mailbox ID is required")
	}

	ids := make([]domain.ID, len(input.IDs))
	for i, id := range input.IDs {
		ids[i] = domain.ID(id)
	}

	result, err := h.messageService.BulkMoveForUser(c.Request().Context(), ids, targetMailboxID, userID)
	if err != nil {
		return api.FromError(c, err)
	}

	return api.OK(c, &BulkOperationResponse{
		Succeeded: result.Succeeded,
		Failed:    result.Failed,
		Errors:    result.Errors,
	})
}

// BulkStar handles requests to star multiple messages.
// @Summary Bulk star messages
// @Description Mark multiple messages as starred
// @Tags Messages
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param input body BulkIDsInput true "Message IDs"
// @Success 200 {object} api.Response{data=BulkOperationResponse}
// @Failure 400 {object} api.Response{error=api.ErrorDetail}
// @Failure 401 {object} api.Response{error=api.ErrorDetail}
// @Failure 500 {object} api.Response{error=api.ErrorDetail}
// @Router /messages/bulk/star [post]
func (h *MessageHandler) BulkStar(c echo.Context) error {
	userID := middleware.GetUserID(c)
	if userID.IsEmpty() {
		return api.Unauthorized(c, "authentication required")
	}

	var input BulkIDsInput
	if err := c.Bind(&input); err != nil {
		return api.BadRequest(c, "invalid request body")
	}

	if len(input.IDs) == 0 {
		return api.BadRequest(c, "at least one message ID is required")
	}

	ids := make([]domain.ID, len(input.IDs))
	for i, id := range input.IDs {
		ids[i] = domain.ID(id)
	}

	result, err := h.messageService.BulkStarForUser(c.Request().Context(), ids, userID)
	if err != nil {
		return api.FromError(c, err)
	}

	return api.OK(c, &BulkOperationResponse{
		Succeeded: result.Succeeded,
		Failed:    result.Failed,
		Errors:    result.Errors,
	})
}

// BulkUnstar handles requests to unstar multiple messages.
// @Summary Bulk unstar messages
// @Description Remove star from multiple messages
// @Tags Messages
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param input body BulkIDsInput true "Message IDs"
// @Success 200 {object} api.Response{data=BulkOperationResponse}
// @Failure 400 {object} api.Response{error=api.ErrorDetail}
// @Failure 401 {object} api.Response{error=api.ErrorDetail}
// @Failure 500 {object} api.Response{error=api.ErrorDetail}
// @Router /messages/bulk/unstar [post]
func (h *MessageHandler) BulkUnstar(c echo.Context) error {
	userID := middleware.GetUserID(c)
	if userID.IsEmpty() {
		return api.Unauthorized(c, "authentication required")
	}

	var input BulkIDsInput
	if err := c.Bind(&input); err != nil {
		return api.BadRequest(c, "invalid request body")
	}

	if len(input.IDs) == 0 {
		return api.BadRequest(c, "at least one message ID is required")
	}

	ids := make([]domain.ID, len(input.IDs))
	for i, id := range input.IDs {
		ids[i] = domain.ID(id)
	}

	result, err := h.messageService.BulkUnstarForUser(c.Request().Context(), ids, userID)
	if err != nil {
		return api.FromError(c, err)
	}

	return api.OK(c, &BulkOperationResponse{
		Succeeded: result.Succeeded,
		Failed:    result.Failed,
		Errors:    result.Errors,
	})
}

// parseMessageFilter extracts message filter options from the request query parameters.
func (h *MessageHandler) parseMessageFilter(c echo.Context, userID domain.ID) (*repository.MessageFilter, error) {
	filter := &repository.MessageFilter{}

	// Parse mailboxId filter
	if mailboxIDStr := c.QueryParam("mailboxId"); mailboxIDStr != "" {
		mailboxID := domain.ID(mailboxIDStr)
		filter.MailboxID = &mailboxID
	}

	// Parse status filter
	if statusStr := c.QueryParam("status"); statusStr != "" {
		var status domain.MessageStatus
		switch strings.ToLower(statusStr) {
		case "read":
			status = domain.MessageRead
		case "unread":
			status = domain.MessageUnread
		default:
			return nil, errorf("invalid status value: %s", statusStr)
		}
		filter.Status = &status
	}

	// Parse isStarred filter
	if starredStr := c.QueryParam("isStarred"); starredStr != "" {
		starred, err := strconv.ParseBool(starredStr)
		if err != nil {
			return nil, errorf("invalid isStarred value: %s", starredStr)
		}
		filter.IsStarred = &starred
	}

	// Parse isSpam filter
	if spamStr := c.QueryParam("isSpam"); spamStr != "" {
		spam, err := strconv.ParseBool(spamStr)
		if err != nil {
			return nil, errorf("invalid isSpam value: %s", spamStr)
		}
		filter.IsSpam = &spam
	}

	// Parse hasAttachments filter
	if attachmentsStr := c.QueryParam("hasAttachments"); attachmentsStr != "" {
		hasAttachments, err := strconv.ParseBool(attachmentsStr)
		if err != nil {
			return nil, errorf("invalid hasAttachments value: %s", attachmentsStr)
		}
		filter.HasAttachments = &hasAttachments
	}

	// Parse from filter
	if from := c.QueryParam("from"); from != "" {
		filter.FromAddressContains = from
	}

	// Parse to filter
	if to := c.QueryParam("to"); to != "" {
		filter.ToAddressContains = to
	}

	// Parse subject filter
	if subject := c.QueryParam("subject"); subject != "" {
		filter.SubjectContains = subject
	}

	// Parse receivedAfter filter
	if afterStr := c.QueryParam("receivedAfter"); afterStr != "" {
		t, err := time.Parse(time.RFC3339, afterStr)
		if err != nil {
			return nil, errorf("invalid receivedAfter format, use RFC3339: %s", afterStr)
		}
		ts := domain.Timestamp{Time: t}
		filter.ReceivedAfter = &ts
	}

	// Parse receivedBefore filter
	if beforeStr := c.QueryParam("receivedBefore"); beforeStr != "" {
		t, err := time.Parse(time.RFC3339, beforeStr)
		if err != nil {
			return nil, errorf("invalid receivedBefore format, use RFC3339: %s", beforeStr)
		}
		ts := domain.Timestamp{Time: t}
		filter.ReceivedBefore = &ts
	}

	// Exclude deleted messages by default (IMAP \Deleted flag)
	if includeDeletedStr := c.QueryParam("includeDeleted"); includeDeletedStr != "" {
		includeDeleted, err := strconv.ParseBool(includeDeletedStr)
		if err != nil {
			return nil, errorf("invalid includeDeleted value: %s", includeDeletedStr)
		}
		filter.ExcludeDeleted = !includeDeleted
	} else {
		filter.ExcludeDeleted = true
	}

	return filter, nil
}

// parseListOptions extracts list options from the request query parameters.
func (h *MessageHandler) parseListOptions(c echo.Context) *repository.ListOptions {
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
		if isValidMessageSortField(sortField) {
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

// isValidMessageSortField checks if a sort field is valid for messages.
func isValidMessageSortField(field string) bool {
	validFields := []string{"receivedAt", "sentAt", "subject", "from", "size", "status", "createdAt"}
	for _, valid := range validFields {
		if field == valid {
			return true
		}
	}
	return false
}

// errorf creates a formatted error.
func errorf(format string, args ...interface{}) error {
	return &formatError{format: format, args: args}
}

type formatError struct {
	format string
	args   []interface{}
}

func (e *formatError) Error() string {
	if len(e.args) == 0 {
		return e.format
	}
	result := e.format
	for _, arg := range e.args {
		result = strings.Replace(result, "%s", toString(arg), 1)
	}
	return result
}

func toString(v interface{}) string {
	switch val := v.(type) {
	case string:
		return val
	default:
		return "?"
	}
}
