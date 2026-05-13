// Package handlers provides HTTP request handlers for the Yunt API.
package handlers

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"

	"yunt/internal/api"
	"yunt/internal/api/middleware"
	"yunt/internal/config"
	"yunt/internal/domain"
	"yunt/internal/parser"
	"yunt/internal/repository"
	"yunt/internal/service"
)

// MessageHandler handles message-related HTTP requests.
type MessageHandler struct {
	messageService *service.MessageService
	mailboxService *service.MailboxService
	authService    *service.AuthService
	notifyService  *service.NotifyService
	relayService   *service.RelayService
	userService    *service.UserService
	repo           repository.Repository
	serverConfig   *config.ServerConfig
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

// WithRelayService sets the relay service for outbound mail delivery.
func (h *MessageHandler) WithRelayService(rs *service.RelayService) *MessageHandler {
	h.relayService = rs
	return h
}

// WithUserService sets the user service for user lookups.
func (h *MessageHandler) WithUserService(us *service.UserService) *MessageHandler {
	h.userService = us
	return h
}

// WithRepo sets the repository for direct data access operations (e.g., draft management).
func (h *MessageHandler) WithRepo(r repository.Repository) *MessageHandler {
	h.repo = r
	return h
}

// WithServerConfig sets the server config used for local-domain detection during delivery.
func (h *MessageHandler) WithServerConfig(sc *config.ServerConfig) *MessageHandler {
	h.serverConfig = sc
	return h
}

// RegisterRoutes registers the message routes on the given group.
func (h *MessageHandler) RegisterRoutes(g *echo.Group) {
	messages := g.Group("/messages", middleware.Auth(h.authService))
	messages.GET("", h.ListMessages)
	messages.GET("/search", h.SearchMessages)
	messages.POST("/send", h.SendMessage)
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

	// Draft operations
	messages.POST("/draft", h.SaveDraft)
	messages.PUT("/draft/:id", h.UpdateDraft)
	messages.POST("/draft/:id/send", h.SendDraft)
	messages.POST("/draft/:id/attachments", h.UploadDraftAttachment)
	messages.DELETE("/draft/:id/attachments/:attachmentId", h.DeleteDraftAttachment)
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

// DeliveryResult holds the outcome of a deliverMessage call.
type DeliveryResult struct {
	SentMessageID     string   `json:"messageId"`
	InternalDelivered []string `json:"internalDelivered,omitempty"`
	ExternalDelivered []string `json:"externalDelivered,omitempty"`
	FailedRecipients  []string `json:"failedRecipients,omitempty"`
}

// extractDomain returns the domain part of an email address.
func extractDomain(email string) string {
	if idx := strings.LastIndex(email, "@"); idx != -1 {
		return email[idx+1:]
	}
	return ""
}

// deliverMessage stores a message in the Sent mailbox, delivers it to local mailboxes,
// and relays any external recipients. On total failure the Sent copy is rolled back.
func (h *MessageHandler) deliverMessage(
	ctx context.Context,
	fromAddr string,
	allRecipients []string,
	forSent, forRelay []byte,
	userID domain.ID,
	userEmail string,
) (*DeliveryResult, error) {
	// --- a) Store to Sent mailbox ---
	mailboxes, err := h.mailboxService.ListMailboxes(ctx, userID, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to list mailboxes")
	}

	var sentMailboxID domain.ID
	for _, mb := range mailboxes.Items {
		if strings.EqualFold(mb.Name, "Sent") {
			sentMailboxID = mb.ID
			break
		}
	}

	if sentMailboxID.IsEmpty() {
		localPart := fromAddr
		if idx := strings.LastIndex(fromAddr, "@"); idx >= 0 {
			localPart = fromAddr[:idx]
		}
		msgDomain := extractDomain(fromAddr)
		sentAddr := localPart + "+sent@" + msgDomain
		created, createErr := h.mailboxService.CreateMailbox(ctx, &service.CreateMailboxInput{
			UserID:  userID,
			Name:    "Sent",
			Address: sentAddr,
		})
		if createErr != nil {
			return nil, fmt.Errorf("failed to create Sent mailbox")
		}
		sentMailboxID = created.ID
	}

	storeResult, err := h.messageService.StoreMessage(ctx, &service.StoreMessageInput{
		RawData:            forSent,
		TargetMailboxID:    sentMailboxID,
		SkipDuplicateCheck: true,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to store sent message")
	}

	_ = h.messageService.MarkAsRead(ctx, storeResult.Message.ID)

	result := &DeliveryResult{
		SentMessageID:     storeResult.Message.ID.String(),
		InternalDelivered: []string{},
		ExternalDelivered: []string{},
		FailedRecipients:  []string{},
	}

	// --- b) Split recipients by domain ---
	var internalRecipients, externalRecipients []string
	for _, addr := range allRecipients {
		d := extractDomain(addr)
		if h.serverConfig != nil && h.serverConfig.IsLocalDomain(d) {
			internalRecipients = append(internalRecipients, addr)
		} else {
			externalRecipients = append(externalRecipients, addr)
		}
	}

	// --- c) Internal delivery ---
	for _, addr := range internalRecipients {
		mailbox, findErr := h.repo.Mailboxes().FindMatchingMailbox(ctx, addr)
		if findErr != nil || mailbox == nil {
			result.FailedRecipients = append(result.FailedRecipients, addr)
			continue
		}
		_, storeErr := h.messageService.StoreMessage(ctx, &service.StoreMessageInput{
			RawData:            forRelay,
			TargetMailboxID:    mailbox.ID,
			SkipDuplicateCheck: true,
		})
		if storeErr != nil {
			result.FailedRecipients = append(result.FailedRecipients, addr)
		} else {
			result.InternalDelivered = append(result.InternalDelivered, addr)
		}
	}

	// --- d) External delivery ---
	if len(externalRecipients) > 0 {
		if h.relayService != nil && h.relayService.IsEnabled() {
			relayResult := h.relayService.Relay(ctx, fromAddr, externalRecipients, forRelay)
			if relayResult.Success {
				result.ExternalDelivered = append(result.ExternalDelivered, relayResult.Recipients...)
				result.FailedRecipients = append(result.FailedRecipients, relayResult.FailedRecipients...)
			} else {
				result.FailedRecipients = append(result.FailedRecipients, externalRecipients...)
			}
		} else {
			result.FailedRecipients = append(result.FailedRecipients, externalRecipients...)
		}
	}

	// --- e) If ALL recipients failed, roll back the Sent copy ---
	totalDelivered := len(result.InternalDelivered) + len(result.ExternalDelivered)
	if len(allRecipients) > 0 && totalDelivered == 0 {
		_ = h.messageService.DeleteMessageForUser(ctx, storeResult.Message.ID, userID)
		return nil, fmt.Errorf("delivery failed: all recipients failed")
	}

	return result, nil
}

// SendMessageInput represents the input for sending an outbound message via relay.
type SendMessageInput struct {
	FromMailboxID string   `json:"fromMailboxId" validate:"required"`
	To            []string `json:"to"            validate:"required"`
	Cc            []string `json:"cc"`
	Bcc           []string `json:"bcc"`
	Subject       string   `json:"subject"       validate:"required"`
	TextBody      string   `json:"textBody"`
	HTMLBody      string   `json:"htmlBody"`
}

// SendMessageResponse is the success payload returned by SendMessage.
type SendMessageResponse struct {
	MessageID  string   `json:"messageId"`
	Recipients []string `json:"recipients"`
}

// SendMessage handles outbound mail delivery via the relay service.
// @Summary Send message
// @Description Send an outbound email via the configured relay service
// @Tags Messages
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param input body SendMessageInput true "Send message input"
// @Success 200 {object} api.Response{data=SendMessageResponse}
// @Failure 400 {object} api.Response{error=api.ErrorDetail}
// @Failure 403 {object} api.Response{error=api.ErrorDetail}
// @Failure 503 {object} api.Response{error=api.ErrorDetail}
// @Router /messages/send [post]
func (h *MessageHandler) SendMessage(c echo.Context) error {
	userID := middleware.GetUserID(c)
	if userID.IsEmpty() {
		return api.Unauthorized(c, "authentication required")
	}

	// Viewers cannot send mail
	role := middleware.GetUserRole(c)
	if role == domain.RoleViewer {
		return api.Forbidden(c, "viewers cannot send messages")
	}

	var input SendMessageInput
	if err := c.Bind(&input); err != nil {
		return api.BadRequest(c, "invalid request body")
	}

	if len(input.To) == 0 {
		return api.BadRequest(c, "at least one recipient is required")
	}
	if input.Subject == "" {
		return api.BadRequest(c, "subject is required")
	}
	if input.FromMailboxID == "" {
		return api.BadRequest(c, "fromMailboxId is required")
	}

	// Verify the From mailbox belongs to the user
	fromMailboxID := domain.ID(input.FromMailboxID)
	fromMailbox, err := h.mailboxService.GetMailbox(c.Request().Context(), fromMailboxID, userID)
	if err != nil {
		return api.FromError(c, err)
	}

	// Get user info for display name and signature
	var fromName string
	var textSignature, htmlSignature string
	if h.userService != nil {
		user, userErr := h.userService.GetByID(c.Request().Context(), userID)
		if userErr == nil {
			if user.DisplayName != "" {
				fromName = user.DisplayName
			} else {
				fromName = user.Username
			}
			textSignature = user.Signature
			htmlSignature = user.SignatureHTML
		}
	}

	// Extract the message domain from the From mailbox address
	msgDomain := fromMailbox.GetDomain()

	// Build text and HTML bodies with appended signature
	textBody := input.TextBody
	if textSignature != "" {
		textBody += "\r\n\r\n-- \r\n" + textSignature
	}
	htmlBody := input.HTMLBody
	if htmlSignature != "" {
		htmlBody += "<br><br>-- <br>" + htmlSignature
	}

	// Build the raw message
	opts := parser.BuildMessageOpts{
		From:     fromMailbox.Address,
		FromName: fromName,
		To:       input.To,
		Cc:       input.Cc,
		Bcc:      input.Bcc,
		Subject:  input.Subject,
		TextBody: textBody,
		HTMLBody: htmlBody,
		Domain:   msgDomain,
	}
	forSent, forRelay := parser.BuildRawMessage(opts)

	// Collect all recipients (To + Cc + Bcc)
	allRecipients := make([]string, 0, len(input.To)+len(input.Cc)+len(input.Bcc))
	allRecipients = append(allRecipients, input.To...)
	allRecipients = append(allRecipients, input.Cc...)
	allRecipients = append(allRecipients, input.Bcc...)

	result, err := h.deliverMessage(c.Request().Context(), fromMailbox.Address, allRecipients, forSent, forRelay, userID, fromMailbox.Address)
	if err != nil {
		return api.InternalServerError(c, err.Error())
	}

	delivered := append(result.InternalDelivered, result.ExternalDelivered...)
	return api.OK(c, &SendMessageResponse{
		MessageID:  result.SentMessageID,
		Recipients: delivered,
	})
}

// DraftInput represents the input for creating or updating a draft message.
type DraftInput struct {
	FromMailboxID string   `json:"fromMailboxId" validate:"required"`
	To            []string `json:"to"`
	Cc            []string `json:"cc"`
	Bcc           []string `json:"bcc"`
	Subject       string   `json:"subject"`
	TextBody      string   `json:"textBody"`
	HTMLBody      string   `json:"htmlBody"`
}

// SaveDraftResponse is the success payload returned by SaveDraft.
type SaveDraftResponse struct {
	MessageID string `json:"messageId"`
}

// AttachmentSummaryResponse represents a brief summary of an uploaded attachment.
type AttachmentSummaryResponse struct {
	ID          string `json:"id"`
	Filename    string `json:"filename"`
	ContentType string `json:"contentType"`
	Size        int64  `json:"size"`
}

// findOrCreateDraftsMailbox finds or creates the Drafts mailbox for a user.
func (h *MessageHandler) findOrCreateDraftsMailbox(c echo.Context, userID domain.ID, fromMailbox *domain.Mailbox) (domain.ID, error) {
	ctx := c.Request().Context()

	mailboxes, err := h.mailboxService.ListMailboxes(ctx, userID, nil)
	if err != nil {
		return domain.ID(""), err
	}

	for _, mb := range mailboxes.Items {
		if strings.EqualFold(mb.Name, "Drafts") {
			return mb.ID, nil
		}
	}

	localPart := fromMailbox.GetLocalPart()
	draftsAddr := localPart + ".drafts@.internal"
	created, createErr := h.mailboxService.CreateMailbox(ctx, &service.CreateMailboxInput{
		UserID:  userID,
		Name:    "Drafts",
		Address: draftsAddr,
	})
	if createErr != nil {
		return domain.ID(""), createErr
	}
	return created.ID, nil
}

// SaveDraft saves a new draft message.
func (h *MessageHandler) SaveDraft(c echo.Context) error {
	if h.repo == nil {
		return api.InternalServerError(c, "repository not configured")
	}

	userID := middleware.GetUserID(c)
	if userID.IsEmpty() {
		return api.Unauthorized(c, "authentication required")
	}

	var input DraftInput
	if err := c.Bind(&input); err != nil {
		return api.BadRequest(c, "invalid request body")
	}
	if input.FromMailboxID == "" {
		return api.BadRequest(c, "fromMailboxId is required")
	}

	fromMailboxID := domain.ID(input.FromMailboxID)
	fromMailbox, err := h.mailboxService.GetMailbox(c.Request().Context(), fromMailboxID, userID)
	if err != nil {
		return api.FromError(c, err)
	}

	msgDomain := fromMailbox.GetDomain()
	opts := parser.BuildMessageOpts{
		From:     fromMailbox.Address,
		To:       input.To,
		Cc:       input.Cc,
		Bcc:      input.Bcc,
		Subject:  input.Subject,
		TextBody: input.TextBody,
		HTMLBody: input.HTMLBody,
		Domain:   msgDomain,
	}
	forSent, _ := parser.BuildRawMessage(opts)

	draftsMailboxID, err := h.findOrCreateDraftsMailbox(c, userID, fromMailbox)
	if err != nil {
		return api.InternalServerError(c, "failed to find or create Drafts mailbox")
	}

	storeResult, err := h.messageService.StoreMessage(c.Request().Context(), &service.StoreMessageInput{
		RawData:            forSent,
		TargetMailboxID:    draftsMailboxID,
		SkipDuplicateCheck: true,
	})
	if err != nil {
		return api.InternalServerError(c, "failed to store draft")
	}

	_ = h.messageService.MarkAsRead(c.Request().Context(), storeResult.Message.ID)

	if err := h.repo.Messages().MarkAsDraft(c.Request().Context(), storeResult.Message.ID); err != nil {
		return api.InternalServerError(c, "failed to mark message as draft")
	}

	return api.Created(c, &SaveDraftResponse{
		MessageID: storeResult.Message.ID.String(),
	})
}

// UpdateDraft replaces the content of an existing draft message.
func (h *MessageHandler) UpdateDraft(c echo.Context) error {
	if h.repo == nil {
		return api.InternalServerError(c, "repository not configured")
	}

	userID := middleware.GetUserID(c)
	if userID.IsEmpty() {
		return api.Unauthorized(c, "authentication required")
	}

	draftID := domain.ID(c.Param("id"))
	if draftID.IsEmpty() {
		return api.BadRequest(c, "draft ID is required")
	}

	msg, err := h.messageService.GetMessageForUser(c.Request().Context(), draftID, userID)
	if err != nil {
		return api.FromError(c, err)
	}
	if !msg.IsDraft {
		return api.BadRequest(c, "message is not a draft")
	}

	var input DraftInput
	if err := c.Bind(&input); err != nil {
		return api.BadRequest(c, "invalid request body")
	}

	fromAddr := msg.From.Address
	if input.FromMailboxID != "" {
		fromMailboxID := domain.ID(input.FromMailboxID)
		fromMailbox, mbErr := h.mailboxService.GetMailbox(c.Request().Context(), fromMailboxID, userID)
		if mbErr != nil {
			return api.FromError(c, mbErr)
		}
		fromAddr = fromMailbox.Address
		msg.From = domain.EmailAddress{Address: fromAddr}
	}

	msgDomain := msg.MailboxID.String()
	if idx := strings.LastIndex(fromAddr, "@"); idx >= 0 {
		msgDomain = fromAddr[idx+1:]
	}

	toStrs := make([]string, len(input.To))
	copy(toStrs, input.To)
	ccStrs := make([]string, len(input.Cc))
	copy(ccStrs, input.Cc)
	bccStrs := make([]string, len(input.Bcc))
	copy(bccStrs, input.Bcc)

	opts := parser.BuildMessageOpts{
		From:     fromAddr,
		To:       toStrs,
		Cc:       ccStrs,
		Bcc:      bccStrs,
		Subject:  input.Subject,
		TextBody: input.TextBody,
		HTMLBody: input.HTMLBody,
		Domain:   msgDomain,
	}
	newRaw, _ := parser.BuildRawMessage(opts)

	msg.Subject = input.Subject
	msg.TextBody = input.TextBody
	msg.HTMLBody = input.HTMLBody
	msg.RawBody = newRaw

	msg.To = make([]domain.EmailAddress, len(input.To))
	for i, addr := range input.To {
		msg.To[i] = domain.EmailAddress{Address: addr}
	}
	msg.Cc = make([]domain.EmailAddress, len(input.Cc))
	for i, addr := range input.Cc {
		msg.Cc[i] = domain.EmailAddress{Address: addr}
	}
	msg.Bcc = make([]domain.EmailAddress, len(input.Bcc))
	for i, addr := range input.Bcc {
		msg.Bcc[i] = domain.EmailAddress{Address: addr}
	}

	if err := h.repo.Messages().Update(c.Request().Context(), msg); err != nil {
		return api.InternalServerError(c, "failed to update draft")
	}

	return api.OK(c, msg)
}

// SendDraft sends an existing draft message.
func (h *MessageHandler) SendDraft(c echo.Context) error {
	if h.repo == nil {
		return api.InternalServerError(c, "repository not configured")
	}

	userID := middleware.GetUserID(c)
	if userID.IsEmpty() {
		return api.Unauthorized(c, "authentication required")
	}

	role := middleware.GetUserRole(c)
	if role == domain.RoleViewer {
		return api.Forbidden(c, "viewers cannot send messages")
	}

	draftID := domain.ID(c.Param("id"))
	if draftID.IsEmpty() {
		return api.BadRequest(c, "draft ID is required")
	}

	msg, err := h.messageService.GetMessageForUser(c.Request().Context(), draftID, userID)
	if err != nil {
		return api.FromError(c, err)
	}
	if !msg.IsDraft {
		return api.BadRequest(c, "message is not a draft")
	}

	// Load attachments for the draft
	attachments, err := h.repo.Attachments().ListByMessage(c.Request().Context(), draftID)
	if err != nil {
		return api.InternalServerError(c, "failed to load draft attachments")
	}

	attInputs := make([]parser.AttachmentInput, 0, len(attachments))
	for _, att := range attachments {
		rc, contentErr := h.repo.Attachments().GetContent(c.Request().Context(), att.ID)
		if contentErr != nil {
			continue
		}
		data, readErr := io.ReadAll(rc)
		rc.Close()
		if readErr != nil {
			continue
		}
		attInputs = append(attInputs, parser.AttachmentInput{
			Filename:    att.Filename,
			ContentType: att.ContentType,
			Data:        data,
		})
	}

	fromAddr := msg.From.Address
	msgDomain := extractDomain(fromAddr)

	toStrs := make([]string, len(msg.To))
	for i, addr := range msg.To {
		toStrs[i] = addr.Address
	}
	ccStrs := make([]string, len(msg.Cc))
	for i, addr := range msg.Cc {
		ccStrs[i] = addr.Address
	}
	bccStrs := make([]string, len(msg.Bcc))
	for i, addr := range msg.Bcc {
		bccStrs[i] = addr.Address
	}

	opts := parser.BuildMessageOpts{
		From:        fromAddr,
		To:          toStrs,
		Cc:          ccStrs,
		Bcc:         bccStrs,
		Subject:     msg.Subject,
		TextBody:    msg.TextBody,
		HTMLBody:    msg.HTMLBody,
		Attachments: attInputs,
		Domain:      msgDomain,
	}
	forSent, forRelay := parser.BuildRawMessage(opts)

	allRecipients := make([]string, 0, len(toStrs)+len(ccStrs)+len(bccStrs))
	allRecipients = append(allRecipients, toStrs...)
	allRecipients = append(allRecipients, ccStrs...)
	allRecipients = append(allRecipients, bccStrs...)

	result, err := h.deliverMessage(c.Request().Context(), fromAddr, allRecipients, forSent, forRelay, userID, fromAddr)
	if err != nil {
		return api.InternalServerError(c, err.Error())
	}

	// Delete the draft on successful send
	_ = h.messageService.DeleteMessageForUser(c.Request().Context(), draftID, userID)

	delivered := append(result.InternalDelivered, result.ExternalDelivered...)
	return api.OK(c, &SendMessageResponse{
		MessageID:  result.SentMessageID,
		Recipients: delivered,
	})
}

// UploadDraftAttachment uploads a file and attaches it to a draft message.
func (h *MessageHandler) UploadDraftAttachment(c echo.Context) error {
	if h.repo == nil {
		return api.InternalServerError(c, "repository not configured")
	}

	userID := middleware.GetUserID(c)
	if userID.IsEmpty() {
		return api.Unauthorized(c, "authentication required")
	}

	draftID := domain.ID(c.Param("id"))
	if draftID.IsEmpty() {
		return api.BadRequest(c, "draft ID is required")
	}

	msg, err := h.messageService.GetMessageForUser(c.Request().Context(), draftID, userID)
	if err != nil {
		return api.FromError(c, err)
	}
	if !msg.IsDraft {
		return api.BadRequest(c, "message is not a draft")
	}

	file, fileHeader, err := c.Request().FormFile("file")
	if err != nil {
		return api.BadRequest(c, "file is required")
	}
	defer file.Close()

	contentType := fileHeader.Header.Get("Content-Type")
	if contentType == "" {
		contentType = "application/octet-stream"
	}

	attID := domain.ID(uuid.New().String())
	att := domain.NewAttachment(attID, draftID, fileHeader.Filename, contentType, fileHeader.Size)

	if err := h.repo.Attachments().CreateWithContent(c.Request().Context(), att, file); err != nil {
		return api.InternalServerError(c, "failed to store attachment")
	}

	// Update draft's attachment count
	count, countErr := h.repo.Attachments().CountByMessage(c.Request().Context(), draftID)
	if countErr == nil {
		msg.AttachmentCount = int(count)
		_ = h.repo.Messages().Update(c.Request().Context(), msg)
	}

	return api.Created(c, &AttachmentSummaryResponse{
		ID:          att.ID.String(),
		Filename:    att.Filename,
		ContentType: att.ContentType,
		Size:        att.Size,
	})
}

// DeleteDraftAttachment removes an attachment from a draft message.
func (h *MessageHandler) DeleteDraftAttachment(c echo.Context) error {
	if h.repo == nil {
		return api.InternalServerError(c, "repository not configured")
	}

	userID := middleware.GetUserID(c)
	if userID.IsEmpty() {
		return api.Unauthorized(c, "authentication required")
	}

	draftID := domain.ID(c.Param("id"))
	if draftID.IsEmpty() {
		return api.BadRequest(c, "draft ID is required")
	}

	attachmentID := domain.ID(c.Param("attachmentId"))
	if attachmentID.IsEmpty() {
		return api.BadRequest(c, "attachment ID is required")
	}

	msg, err := h.messageService.GetMessageForUser(c.Request().Context(), draftID, userID)
	if err != nil {
		return api.FromError(c, err)
	}
	if !msg.IsDraft {
		return api.BadRequest(c, "message is not a draft")
	}

	if err := h.repo.Attachments().Delete(c.Request().Context(), attachmentID); err != nil {
		return api.FromError(c, err)
	}

	// Update draft's attachment count
	count, countErr := h.repo.Attachments().CountByMessage(c.Request().Context(), draftID)
	if countErr == nil {
		msg.AttachmentCount = int(count)
		_ = h.repo.Messages().Update(c.Request().Context(), msg)
	}

	return api.NoContent(c)
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
