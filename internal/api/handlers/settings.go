package handlers

import (
	"github.com/labstack/echo/v4"

	"yunt/internal/api"
	"yunt/internal/api/middleware"
	"yunt/internal/domain"
	"yunt/internal/repository"
	"yunt/internal/service"
)

type SettingsHandler struct {
	settingsRepo repository.SettingsRepository
	authService  *service.AuthService
}

func NewSettingsHandler(repo repository.SettingsRepository, authService *service.AuthService) *SettingsHandler {
	return &SettingsHandler{
		settingsRepo: repo,
		authService:  authService,
	}
}

func (h *SettingsHandler) RegisterRoutes(g *echo.Group) {
	settings := g.Group("/settings", middleware.Auth(h.authService), middleware.RequireAdmin())
	settings.GET("", h.GetSettings)
	settings.PUT("", h.UpdateSettings)
	settings.POST("/reset", h.ResetSettings)
	settings.GET("/smtp", h.GetSMTP)
	settings.PUT("/smtp", h.UpdateSMTP)
	settings.GET("/imap", h.GetIMAP)
	settings.PUT("/imap", h.UpdateIMAP)
	settings.GET("/storage", h.GetStorage)
	settings.PUT("/storage", h.UpdateStorage)
	settings.GET("/security", h.GetSecurity)
	settings.PUT("/security", h.UpdateSecurity)
}

// @Summary Get Settings
// @Description Get all application settings (admin only)
// @Tags Settings
// @Produce json
// @Security BearerAuth
// @Success 200 {object} api.Response{data=domain.Settings}
// @Failure 401 {object} api.Response{error=api.ErrorDetail}
// @Failure 403 {object} api.Response{error=api.ErrorDetail}
// @Failure 500 {object} api.Response{error=api.ErrorDetail}
// @Router /settings [get]
func (h *SettingsHandler) GetSettings(c echo.Context) error {
	settings, err := h.settingsRepo.Get(c.Request().Context())
	if err != nil {
		return api.FromError(c, err)
	}
	return api.OK(c, settings)
}

// @Summary Update Settings
// @Description Update application settings (admin only, partial update)
// @Tags Settings
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param input body domain.SettingsUpdateInput true "Settings update"
// @Success 200 {object} api.Response{data=domain.Settings}
// @Failure 400 {object} api.Response{error=api.ErrorDetail}
// @Failure 401 {object} api.Response{error=api.ErrorDetail}
// @Failure 403 {object} api.Response{error=api.ErrorDetail}
// @Failure 500 {object} api.Response{error=api.ErrorDetail}
// @Router /settings [put]
func (h *SettingsHandler) UpdateSettings(c echo.Context) error {
	var input domain.SettingsUpdateInput
	if err := c.Bind(&input); err != nil {
		return api.BadRequest(c, "invalid request body")
	}

	current, err := h.settingsRepo.Get(c.Request().Context())
	if err != nil {
		return api.FromError(c, err)
	}

	if err := h.settingsRepo.Update(c.Request().Context(), current.ID, &input); err != nil {
		return api.FromError(c, err)
	}

	updated, err := h.settingsRepo.Get(c.Request().Context())
	if err != nil {
		return api.FromError(c, err)
	}

	return api.OK(c, updated)
}

// @Summary Reset Settings
// @Description Reset all settings to defaults (admin only)
// @Tags Settings
// @Produce json
// @Security BearerAuth
// @Success 204
// @Failure 401 {object} api.Response{error=api.ErrorDetail}
// @Failure 403 {object} api.Response{error=api.ErrorDetail}
// @Failure 500 {object} api.Response{error=api.ErrorDetail}
// @Router /settings/reset [post]
func (h *SettingsHandler) ResetSettings(c echo.Context) error {
	if err := h.settingsRepo.Reset(c.Request().Context()); err != nil {
		return api.FromError(c, err)
	}
	return api.NoContent(c)
}

// @Summary Get SMTP Settings
// @Description Get SMTP server settings (admin only)
// @Tags Settings
// @Produce json
// @Security BearerAuth
// @Success 200 {object} api.Response{data=domain.SMTPSettings}
// @Failure 401 {object} api.Response{error=api.ErrorDetail}
// @Failure 403 {object} api.Response{error=api.ErrorDetail}
// @Failure 500 {object} api.Response{error=api.ErrorDetail}
// @Router /settings/smtp [get]
func (h *SettingsHandler) GetSMTP(c echo.Context) error {
	smtp, err := h.settingsRepo.GetSMTP(c.Request().Context())
	if err != nil {
		return api.FromError(c, err)
	}
	return api.OK(c, smtp)
}

// @Summary Update SMTP Settings
// @Description Update SMTP server settings (admin only)
// @Tags Settings
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param input body domain.SMTPSettingsUpdate true "SMTP settings update"
// @Success 200 {object} api.Response{data=domain.SMTPSettings}
// @Failure 400 {object} api.Response{error=api.ErrorDetail}
// @Failure 401 {object} api.Response{error=api.ErrorDetail}
// @Failure 403 {object} api.Response{error=api.ErrorDetail}
// @Failure 500 {object} api.Response{error=api.ErrorDetail}
// @Router /settings/smtp [put]
func (h *SettingsHandler) UpdateSMTP(c echo.Context) error {
	var input domain.SMTPSettingsUpdate
	if err := c.Bind(&input); err != nil {
		return api.BadRequest(c, "invalid request body")
	}

	if err := h.settingsRepo.UpdateSMTP(c.Request().Context(), &input); err != nil {
		return api.FromError(c, err)
	}

	smtp, err := h.settingsRepo.GetSMTP(c.Request().Context())
	if err != nil {
		return api.FromError(c, err)
	}
	return api.OK(c, smtp)
}

// @Summary Get IMAP Settings
// @Description Get IMAP server settings (admin only)
// @Tags Settings
// @Produce json
// @Security BearerAuth
// @Success 200 {object} api.Response{data=domain.IMAPSettings}
// @Failure 401 {object} api.Response{error=api.ErrorDetail}
// @Failure 403 {object} api.Response{error=api.ErrorDetail}
// @Failure 500 {object} api.Response{error=api.ErrorDetail}
// @Router /settings/imap [get]
func (h *SettingsHandler) GetIMAP(c echo.Context) error {
	imap, err := h.settingsRepo.GetIMAP(c.Request().Context())
	if err != nil {
		return api.FromError(c, err)
	}
	return api.OK(c, imap)
}

// @Summary Update IMAP Settings
// @Description Update IMAP server settings (admin only)
// @Tags Settings
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param input body domain.IMAPSettingsUpdate true "IMAP settings update"
// @Success 200 {object} api.Response{data=domain.IMAPSettings}
// @Failure 400 {object} api.Response{error=api.ErrorDetail}
// @Failure 401 {object} api.Response{error=api.ErrorDetail}
// @Failure 403 {object} api.Response{error=api.ErrorDetail}
// @Failure 500 {object} api.Response{error=api.ErrorDetail}
// @Router /settings/imap [put]
func (h *SettingsHandler) UpdateIMAP(c echo.Context) error {
	var input domain.IMAPSettingsUpdate
	if err := c.Bind(&input); err != nil {
		return api.BadRequest(c, "invalid request body")
	}

	if err := h.settingsRepo.UpdateIMAP(c.Request().Context(), &input); err != nil {
		return api.FromError(c, err)
	}

	imap, err := h.settingsRepo.GetIMAP(c.Request().Context())
	if err != nil {
		return api.FromError(c, err)
	}
	return api.OK(c, imap)
}

// @Summary Get Storage Settings
// @Description Get storage settings (admin only)
// @Tags Settings
// @Produce json
// @Security BearerAuth
// @Success 200 {object} api.Response{data=domain.StorageSettings}
// @Failure 401 {object} api.Response{error=api.ErrorDetail}
// @Failure 403 {object} api.Response{error=api.ErrorDetail}
// @Failure 500 {object} api.Response{error=api.ErrorDetail}
// @Router /settings/storage [get]
func (h *SettingsHandler) GetStorage(c echo.Context) error {
	storage, err := h.settingsRepo.GetStorage(c.Request().Context())
	if err != nil {
		return api.FromError(c, err)
	}
	return api.OK(c, storage)
}

// @Summary Update Storage Settings
// @Description Update storage settings (admin only)
// @Tags Settings
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param input body domain.StorageSettingsUpdate true "Storage settings update"
// @Success 200 {object} api.Response{data=domain.StorageSettings}
// @Failure 400 {object} api.Response{error=api.ErrorDetail}
// @Failure 401 {object} api.Response{error=api.ErrorDetail}
// @Failure 403 {object} api.Response{error=api.ErrorDetail}
// @Failure 500 {object} api.Response{error=api.ErrorDetail}
// @Router /settings/storage [put]
func (h *SettingsHandler) UpdateStorage(c echo.Context) error {
	var input domain.StorageSettingsUpdate
	if err := c.Bind(&input); err != nil {
		return api.BadRequest(c, "invalid request body")
	}

	if err := h.settingsRepo.UpdateStorage(c.Request().Context(), &input); err != nil {
		return api.FromError(c, err)
	}

	storage, err := h.settingsRepo.GetStorage(c.Request().Context())
	if err != nil {
		return api.FromError(c, err)
	}
	return api.OK(c, storage)
}

// @Summary Get Security Settings
// @Description Get security settings (admin only)
// @Tags Settings
// @Produce json
// @Security BearerAuth
// @Success 200 {object} api.Response{data=domain.SecuritySettings}
// @Failure 401 {object} api.Response{error=api.ErrorDetail}
// @Failure 403 {object} api.Response{error=api.ErrorDetail}
// @Failure 500 {object} api.Response{error=api.ErrorDetail}
// @Router /settings/security [get]
func (h *SettingsHandler) GetSecurity(c echo.Context) error {
	security, err := h.settingsRepo.GetSecurity(c.Request().Context())
	if err != nil {
		return api.FromError(c, err)
	}
	return api.OK(c, security)
}

// @Summary Update Security Settings
// @Description Update security settings (admin only)
// @Tags Settings
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param input body domain.SecuritySettingsUpdate true "Security settings update"
// @Success 200 {object} api.Response{data=domain.SecuritySettings}
// @Failure 400 {object} api.Response{error=api.ErrorDetail}
// @Failure 401 {object} api.Response{error=api.ErrorDetail}
// @Failure 403 {object} api.Response{error=api.ErrorDetail}
// @Failure 500 {object} api.Response{error=api.ErrorDetail}
// @Router /settings/security [put]
func (h *SettingsHandler) UpdateSecurity(c echo.Context) error {
	var input domain.SecuritySettingsUpdate
	if err := c.Bind(&input); err != nil {
		return api.BadRequest(c, "invalid request body")
	}

	if err := h.settingsRepo.UpdateSecurity(c.Request().Context(), &input); err != nil {
		return api.FromError(c, err)
	}

	security, err := h.settingsRepo.GetSecurity(c.Request().Context())
	if err != nil {
		return api.FromError(c, err)
	}
	return api.OK(c, security)
}
