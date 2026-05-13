package jmap

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/labstack/echo/v4"

	"yunt/internal/api/middleware"
	"yunt/internal/config"
	"yunt/internal/domain"
	"yunt/internal/jmap/core"
	"yunt/internal/jmap/mail"
	"yunt/internal/jmap/state"
	"yunt/internal/jmap/thread"
	"yunt/internal/repository"
	"yunt/internal/service"
)

// HandlerConfig holds dependencies for the JMAP handler.
type HandlerConfig struct {
	Repo           repository.Repository
	AuthService    *service.AuthService
	MessageService *service.MessageService
	MailboxService *service.MailboxService
	UserService    *service.UserService
	RelayService   *service.RelayService
	NotifyService  *service.NotifyService
	StateManager   *state.Manager
	ThreadResolver *thread.Resolver
	JMAPConfig     config.JMAPConfig
	ServerConfig   *config.ServerConfig
}

// Handler handles JMAP HTTP endpoints.
type Handler struct {
	cfg        HandlerConfig
	dispatcher *Dispatcher
}

// NewHandler creates and initializes a JMAP handler with all method registrations.
func NewHandler(cfg HandlerConfig) *Handler {
	h := &Handler{
		cfg:        cfg,
		dispatcher: NewDispatcher(cfg.JMAPConfig.MaxCallsPerRequest),
	}

	h.dispatcher.Register("Core/echo", h.coreEcho)

	emailHandler := mail.NewEmailHandler(cfg.MessageService, cfg.StateManager, cfg.Repo)
	h.dispatcher.Register("Email/get", emailHandler.Get)
	h.dispatcher.Register("Email/query", emailHandler.Query)
	h.dispatcher.Register("Email/changes", emailHandler.Changes)

	mailboxHandler := mail.NewMailboxHandler(cfg.MailboxService, cfg.StateManager)
	h.dispatcher.Register("Mailbox/get", mailboxHandler.Get)
	h.dispatcher.Register("Mailbox/changes", mailboxHandler.Changes)
	h.dispatcher.Register("Mailbox/query", mailboxHandler.Query)

	threadHandler := mail.NewThreadHandler(cfg.Repo, cfg.StateManager)
	h.dispatcher.Register("Thread/get", threadHandler.Get)
	h.dispatcher.Register("Thread/changes", threadHandler.Changes)

	return h
}

// RegisterRoutes registers JMAP HTTP routes on the Echo instance.
func (h *Handler) RegisterRoutes(e *echo.Echo, authMW echo.MiddlewareFunc) {
	e.GET("/.well-known/jmap", h.Session, authMW)

	jmap := e.Group("/jmap", authMW)
	jmap.POST("/api", h.API)
	jmap.POST("/upload/:accountId", h.Upload)
	jmap.GET("/download/:accountId/:blobId/:name", h.Download)
	jmap.GET("/eventsource", h.EventSource)
}

// Session returns the JMAP Session resource (RFC 8620 §2).
func (h *Handler) Session(c echo.Context) error {
	userID := middleware.GetUserID(c)
	username := middleware.GetUsername(c)

	baseURL := fmt.Sprintf("%s://%s", c.Scheme(), c.Request().Host)

	coreCap, _ := json.Marshal(core.CoreCapability{
		MaxSizeUpload:       h.cfg.JMAPConfig.MaxSizeUpload,
		MaxConcurrentUpload: h.cfg.JMAPConfig.MaxConcurrentUpload,
		MaxSizeRequest:      h.cfg.JMAPConfig.MaxSizeRequest,
		MaxConcurrentReqs:   4,
		MaxCallsInRequest:   h.cfg.JMAPConfig.MaxCallsPerRequest,
		MaxObjectsInGet:     h.cfg.JMAPConfig.MaxObjectsInGet,
		MaxObjectsInSet:     h.cfg.JMAPConfig.MaxObjectsInSet,
		CollationAlgorithms: []string{"i;ascii-casemap", "i;ascii-numeric", "i;unicode-casemap"},
	})

	mailCap, _ := json.Marshal(core.MailCapability{
		MaxMailboxesPerEmail:       intPtr(1),
		MaxSizeMailboxName:         100,
		MaxSizeAttachmentsPerEmail: 25 * 1024 * 1024,
		EmailQuerySortOptions:      []string{"receivedAt", "sentAt", "size", "from", "to", "subject"},
		MayCreateTopLevelMailbox:   true,
	})

	contactsCap, _ := json.Marshal(core.ContactsCapability{
		MayCreateAddressBook: true,
	})

	accountID := string(userID)
	session := core.SessionResource{
		Capabilities: map[string]json.RawMessage{
			"urn:ietf:params:jmap:core":       coreCap,
			"urn:ietf:params:jmap:mail":        json.RawMessage(`{}`),
			"urn:ietf:params:jmap:submission":  json.RawMessage(`{}`),
			"urn:ietf:params:jmap:vacationresponse": json.RawMessage(`{}`),
			"urn:ietf:params:jmap:contacts":    json.RawMessage(`{}`),
		},
		Accounts: map[string]core.Account{
			accountID: {
				Name:       username,
				IsPersonal: true,
				IsReadOnly: false,
				AccountCapabilities: map[string]json.RawMessage{
					"urn:ietf:params:jmap:mail":       mailCap,
					"urn:ietf:params:jmap:submission":  json.RawMessage(`{}`),
					"urn:ietf:params:jmap:vacationresponse": json.RawMessage(`{}`),
					"urn:ietf:params:jmap:contacts":    contactsCap,
				},
			},
		},
		PrimaryAccounts: map[string]string{
			"urn:ietf:params:jmap:mail":       accountID,
			"urn:ietf:params:jmap:submission":  accountID,
			"urn:ietf:params:jmap:vacationresponse": accountID,
			"urn:ietf:params:jmap:contacts":    accountID,
		},
		Username:       username,
		APIUrl:         baseURL + "/jmap/api",
		DownloadUrl:    baseURL + "/jmap/download/{accountId}/{blobId}/{name}?accept={type}",
		UploadUrl:      baseURL + "/jmap/upload/{accountId}/",
		EventSourceUrl: baseURL + "/jmap/eventsource?types={types}&closeafter={closeafter}&ping={ping}",
		State:          "0",
	}

	stateStr, err := h.cfg.StateManager.CurrentState(c.Request().Context(), userID, "Session")
	if err == nil {
		session.State = stateStr
	}

	return c.JSON(http.StatusOK, session)
}

// API handles JMAP method calls (POST /jmap/api).
func (h *Handler) API(c echo.Context) error {
	if c.Request().Header.Get("Content-Type") != "application/json" {
		return c.JSON(http.StatusBadRequest, core.RequestError{
			Type:   core.RequestErrorNotJSON,
			Status: http.StatusBadRequest,
			Detail: "Content-Type must be application/json",
		})
	}

	var req core.Request
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, core.RequestError{
			Type:   core.RequestErrorNotRequest,
			Status: http.StatusBadRequest,
			Detail: "invalid JMAP request",
		})
	}

	for _, cap := range req.Using {
		switch cap {
		case "urn:ietf:params:jmap:core",
			"urn:ietf:params:jmap:mail",
			"urn:ietf:params:jmap:submission",
			"urn:ietf:params:jmap:vacationresponse",
			"urn:ietf:params:jmap:contacts":
		default:
			return c.JSON(http.StatusBadRequest, core.RequestError{
				Type:   core.RequestErrorUnknownCapability,
				Status: http.StatusBadRequest,
				Detail: fmt.Sprintf("unknown capability: %s", cap),
			})
		}
	}

	userID := middleware.GetUserID(c)
	resp := h.dispatcher.Dispatch(c.Request().Context(), userID, &req)

	stateStr, _ := h.cfg.StateManager.CurrentState(c.Request().Context(), userID, "Session")
	resp.SessionState = stateStr

	return c.JSON(http.StatusOK, resp)
}

// coreEcho implements Core/echo — returns args unchanged (RFC 8620 §4.1).
func (h *Handler) coreEcho(_ context.Context, _ domain.ID, args json.RawMessage) (json.RawMessage, *core.MethodError) {
	return args, nil
}

// Upload handles blob upload (POST /jmap/upload/:accountId).
func (h *Handler) Upload(c echo.Context) error {
	return c.JSON(http.StatusNotImplemented, map[string]string{"error": "not yet implemented"})
}

// Download handles blob download (GET /jmap/download/:accountId/:blobId/:name).
func (h *Handler) Download(c echo.Context) error {
	return c.JSON(http.StatusNotImplemented, map[string]string{"error": "not yet implemented"})
}

// EventSource handles JMAP push via SSE (GET /jmap/eventsource).
func (h *Handler) EventSource(c echo.Context) error {
	return c.JSON(http.StatusNotImplemented, map[string]string{"error": "not yet implemented"})
}

func intPtr(v int) *int { return &v }
