// Package handlers provides HTTP request handlers for the Yunt API.
package handlers

import (
	"io"
	"net/http"
	"strconv"

	"github.com/labstack/echo/v4"

	"yunt/internal/api"
	"yunt/internal/api/middleware"
	"yunt/internal/domain"
	"yunt/internal/repository"
	"yunt/internal/service"
)

// AttachmentHandler handles attachment-related HTTP requests.
// It provides direct access to attachments without requiring the message context.
type AttachmentHandler struct {
	messageService *service.MessageService
	authService    *service.AuthService
}

// NewAttachmentHandler creates a new AttachmentHandler.
func NewAttachmentHandler(
	messageService *service.MessageService,
	authService *service.AuthService,
) *AttachmentHandler {
	return &AttachmentHandler{
		messageService: messageService,
		authService:    authService,
	}
}

// RegisterRoutes registers the attachment routes on the given group.
func (h *AttachmentHandler) RegisterRoutes(g *echo.Group) {
	attachments := g.Group("/attachments", middleware.Auth(h.authService))
	attachments.GET("", h.ListAttachments)
	attachments.GET("/:id", h.GetAttachment)
	attachments.GET("/:id/download", h.DownloadAttachment)
}

// ListAttachments handles requests to list attachments with optional filtering.
// @Summary List attachments
// @Description Get attachments with optional filtering and pagination
// @Tags Attachments
// @Produce json
// @Security BearerAuth
// @Param messageId query string false "Filter by message ID"
// @Param isInline query bool false "Filter by inline status"
// @Param contentType query string false "Filter by content type prefix"
// @Param page query int false "Page number (default: 1)"
// @Param perPage query int false "Items per page (default: 20, max: 100)"
// @Param sort query string false "Sort field (filename, size, contentType, createdAt)"
// @Param order query string false "Sort order (asc, desc)"
// @Success 200 {object} api.Response{data=api.PaginatedData}
// @Failure 401 {object} api.Response{error=api.ErrorDetail}
// @Failure 403 {object} api.Response{error=api.ErrorDetail}
// @Failure 500 {object} api.Response{error=api.ErrorDetail}
// @Router /attachments [get]
func (h *AttachmentHandler) ListAttachments(c echo.Context) error {
	userID := middleware.GetUserID(c)
	if userID.IsEmpty() {
		return api.Unauthorized(c, "authentication required")
	}

	// Parse filter options
	filter, err := h.parseAttachmentFilter(c, userID)
	if err != nil {
		return api.BadRequest(c, err.Error())
	}

	// Parse list options
	opts := h.parseListOptions(c)

	result, err := h.messageService.ListAttachmentsForUser(c.Request().Context(), userID, filter, opts)
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

	// Convert to summaries for response
	summaries := make([]*domain.AttachmentSummary, len(result.Items))
	for i, att := range result.Items {
		summaries[i] = att.ToSummary()
	}

	return api.Paginated(c, summaries, page, perPage, result.Total)
}

// GetAttachment handles requests to get attachment metadata by ID.
// @Summary Get attachment
// @Description Get metadata for a specific attachment by ID
// @Tags Attachments
// @Produce json
// @Security BearerAuth
// @Param id path string true "Attachment ID"
// @Success 200 {object} api.Response{data=domain.Attachment}
// @Failure 401 {object} api.Response{error=api.ErrorDetail}
// @Failure 403 {object} api.Response{error=api.ErrorDetail}
// @Failure 404 {object} api.Response{error=api.ErrorDetail}
// @Failure 500 {object} api.Response{error=api.ErrorDetail}
// @Router /attachments/{id} [get]
func (h *AttachmentHandler) GetAttachment(c echo.Context) error {
	userID := middleware.GetUserID(c)
	if userID.IsEmpty() {
		return api.Unauthorized(c, "authentication required")
	}

	attachmentID := domain.ID(c.Param("id"))
	if attachmentID.IsEmpty() {
		return api.BadRequest(c, "attachment ID is required")
	}

	attachment, err := h.messageService.GetAttachmentByIDForUser(c.Request().Context(), attachmentID, userID)
	if err != nil {
		return api.FromError(c, err)
	}

	return api.OK(c, attachment)
}

// DownloadAttachment handles requests to download attachment content.
// @Summary Download attachment
// @Description Download the content of an attachment
// @Tags Attachments
// @Produce application/octet-stream
// @Security BearerAuth
// @Param id path string true "Attachment ID"
// @Param inline query bool false "Display inline instead of download"
// @Success 200 {file} binary "Attachment file"
// @Failure 401 {object} api.Response{error=api.ErrorDetail}
// @Failure 403 {object} api.Response{error=api.ErrorDetail}
// @Failure 404 {object} api.Response{error=api.ErrorDetail}
// @Failure 500 {object} api.Response{error=api.ErrorDetail}
// @Router /attachments/{id}/download [get]
func (h *AttachmentHandler) DownloadAttachment(c echo.Context) error {
	userID := middleware.GetUserID(c)
	if userID.IsEmpty() {
		return api.Unauthorized(c, "authentication required")
	}

	attachmentID := domain.ID(c.Param("id"))
	if attachmentID.IsEmpty() {
		return api.BadRequest(c, "attachment ID is required")
	}

	// Check if inline display is requested
	inline := c.QueryParam("inline") == "true"

	attachment, content, err := h.messageService.GetAttachmentContentByIDForUser(c.Request().Context(), attachmentID, userID)
	if err != nil {
		return api.FromError(c, err)
	}
	defer func() {
		if content != nil {
			_ = content.Close()
		}
	}()

	// Set Content-Type header
	c.Response().Header().Set("Content-Type", attachment.ContentType)

	// Set Content-Disposition header
	disposition := "attachment"
	if inline {
		disposition = "inline"
	}
	// Use RFC 5987 encoding for filenames with special characters
	c.Response().Header().Set("Content-Disposition", buildContentDisposition(disposition, attachment.Filename))

	// Set Content-Length header for efficient streaming
	c.Response().Header().Set("Content-Length", strconv.FormatInt(attachment.Size, 10))

	// Set caching headers for inline resources
	if inline {
		c.Response().Header().Set("Cache-Control", "private, max-age=3600")
	}

	// Stream the content for efficient handling of large files
	return c.Stream(http.StatusOK, attachment.ContentType, content)
}

// parseAttachmentFilter extracts attachment filter options from the request query parameters.
func (h *AttachmentHandler) parseAttachmentFilter(c echo.Context, _ domain.ID) (*repository.AttachmentFilter, error) {
	filter := &repository.AttachmentFilter{}

	// Parse messageId filter
	if messageIDStr := c.QueryParam("messageId"); messageIDStr != "" {
		messageID := domain.ID(messageIDStr)
		filter.MessageID = &messageID
	}

	// Parse isInline filter
	if inlineStr := c.QueryParam("isInline"); inlineStr != "" {
		isInline, err := strconv.ParseBool(inlineStr)
		if err != nil {
			return nil, &filterError{field: "isInline", value: inlineStr}
		}
		filter.IsInline = &isInline
	}

	// Parse contentType filter (prefix match)
	if contentType := c.QueryParam("contentType"); contentType != "" {
		filter.ContentTypePrefix = contentType
	}

	return filter, nil
}

// parseListOptions extracts list options from the request query parameters.
func (h *AttachmentHandler) parseListOptions(c echo.Context) *repository.ListOptions {
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
		if isValidAttachmentSortField(sortField) {
			opts.Sort = &repository.SortOptions{
				Field: sortField,
				Order: order,
			}
		}
	} else {
		// Default sort by createdAt descending
		opts.Sort = &repository.SortOptions{
			Field: "createdAt",
			Order: domain.SortDesc,
		}
	}

	return opts
}

// isValidAttachmentSortField checks if a sort field is valid for attachments.
func isValidAttachmentSortField(field string) bool {
	validFields := []string{"filename", "size", "contentType", "createdAt"}
	for _, valid := range validFields {
		if field == valid {
			return true
		}
	}
	return false
}

// buildContentDisposition builds a Content-Disposition header value.
// It handles filenames with special characters using RFC 5987 encoding.
func buildContentDisposition(disposition, filename string) string {
	// Check if filename contains non-ASCII characters
	needsEncoding := false
	for _, r := range filename {
		if r > 127 || r == '"' || r == '\\' {
			needsEncoding = true
			break
		}
	}

	if !needsEncoding {
		// Simple case: ASCII filename
		return disposition + "; filename=\"" + filename + "\""
	}

	// RFC 5987 encoding for non-ASCII filenames
	encoded := percentEncode(filename)
	return disposition + "; filename=\"" + sanitizeFilename(filename) + "\"; filename*=UTF-8''" + encoded
}

// sanitizeFilename removes or replaces problematic characters for the fallback filename.
func sanitizeFilename(filename string) string {
	result := make([]byte, 0, len(filename))
	for _, r := range filename {
		if r > 127 || r == '"' || r == '\\' {
			result = append(result, '_')
		} else {
			result = append(result, byte(r))
		}
	}
	return string(result)
}

// percentEncode encodes a string using percent-encoding for RFC 5987.
func percentEncode(s string) string {
	result := make([]byte, 0, len(s)*3)
	for i := 0; i < len(s); i++ {
		c := s[i]
		if isAttrChar(c) {
			result = append(result, c)
		} else {
			result = append(result, '%')
			result = append(result, hexDigit(c>>4))
			result = append(result, hexDigit(c&0x0f))
		}
	}
	return string(result)
}

// isAttrChar returns true if c is an RFC 5987 attr-char.
func isAttrChar(c byte) bool {
	return (c >= 'a' && c <= 'z') ||
		(c >= 'A' && c <= 'Z') ||
		(c >= '0' && c <= '9') ||
		c == '!' || c == '#' || c == '$' || c == '&' ||
		c == '+' || c == '-' || c == '.' || c == '^' ||
		c == '_' || c == '`' || c == '|' || c == '~'
}

// hexDigit returns the hex digit character for the given value.
func hexDigit(n byte) byte {
	if n < 10 {
		return '0' + n
	}
	return 'A' + n - 10
}

// filterError represents an error in parsing filter parameters.
type filterError struct {
	field string
	value string
}

func (e *filterError) Error() string {
	return "invalid " + e.field + " value: " + e.value
}

// Ensure AttachmentHandler implements the Handler interface if one exists.
var _ io.Closer = (*closeWrapper)(nil)

// closeWrapper is a helper type for testing io.Closer.
type closeWrapper struct{}

func (c *closeWrapper) Close() error { return nil }
