package blob

import (
	"crypto/sha256"
	"fmt"
	"io"
	"net/http"

	"github.com/labstack/echo/v4"

	"yunt/internal/api/middleware"
	"yunt/internal/repository"
)

// Handler handles JMAP blob upload/download endpoints.
type Handler struct {
	repo repository.Repository
}

// NewHandler creates a new blob handler.
func NewHandler(repo repository.Repository) *Handler {
	return &Handler{repo: repo}
}

// Upload handles POST /jmap/upload/:accountId (RFC 8620 §6.1).
func (h *Handler) Upload(c echo.Context) error {
	accountID := c.Param("accountId")
	userID := middleware.GetUserID(c)

	if string(userID) != accountID {
		return c.JSON(http.StatusForbidden, map[string]string{"error": "account mismatch"})
	}

	body, err := io.ReadAll(c.Request().Body)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "failed to read body"})
	}

	hash := sha256.Sum256(body)
	blobID := fmt.Sprintf("%x", hash[:])

	contentType := c.Request().Header.Get("Content-Type")
	if contentType == "" {
		contentType = "application/octet-stream"
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"accountId": accountID,
		"blobId":    blobID,
		"type":      contentType,
		"size":      len(body),
	})
}

// Download handles GET /jmap/download/:accountId/:blobId/:name (RFC 8620 §6.2).
func (h *Handler) Download(c echo.Context) error {
	accountID := c.Param("accountId")
	blobID := c.Param("blobId")
	name := c.Param("name")
	userID := middleware.GetUserID(c)

	if string(userID) != accountID {
		return c.JSON(http.StatusForbidden, map[string]string{"error": "account mismatch"})
	}

	msg, err := h.repo.Messages().GetByBlobID(c.Request().Context(), blobID)
	if err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "blob not found"})
	}

	acceptType := c.QueryParam("accept")
	if acceptType == "" {
		acceptType = "message/rfc822"
	}

	c.Response().Header().Set("Content-Type", acceptType)
	c.Response().Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, name))
	c.Response().Header().Set("Cache-Control", "private, immutable, max-age=31536000")

	return c.Blob(http.StatusOK, acceptType, msg.RawBody)
}
