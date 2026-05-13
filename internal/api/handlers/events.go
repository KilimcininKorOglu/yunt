package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/labstack/echo/v4"

	"yunt/internal/domain"
	"yunt/internal/repository"
	"yunt/internal/service"
)

type EventHandler struct {
	notifyService *service.NotifyService
	authService   *service.AuthService
	mailboxRepo   repository.MailboxRepository
}

func NewEventHandler(
	notifyService *service.NotifyService,
	authService *service.AuthService,
	mailboxRepo repository.MailboxRepository,
) *EventHandler {
	return &EventHandler{
		notifyService: notifyService,
		authService:   authService,
		mailboxRepo:   mailboxRepo,
	}
}

func (h *EventHandler) RegisterRoutes(g *echo.Group) {
	events := g.Group("/events")
	events.GET("/stream", h.StreamEvents)
}

type sseEvent struct {
	Event     string    `json:"event"`
	MailboxID string    `json:"mailboxId"`
	MessageID string    `json:"messageId,omitempty"`
	Count     uint32    `json:"messageCount,omitempty"`
	Flags     []string  `json:"flags,omitempty"`
	Timestamp time.Time `json:"timestamp"`
}

func (h *EventHandler) StreamEvents(c echo.Context) error {
	tokenStr := extractToken(c)
	if tokenStr == "" {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "token required"})
	}

	claims, err := h.authService.ValidateAccessToken(c.Request().Context(), tokenStr)
	if err != nil {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "invalid token"})
	}

	userID := claims.UserID

	mailboxes, err := h.mailboxRepo.ListByUser(c.Request().Context(), userID, nil)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to load mailboxes"})
	}

	w := c.Response()
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")

	// Disable write deadline for SSE long-lived connections
	if rc := http.NewResponseController(w); rc != nil {
		_ = rc.SetWriteDeadline(time.Time{})
	}

	w.WriteHeader(http.StatusOK)

	notifCh := make(chan *service.Notification, 64)

	subscriptionID := fmt.Sprintf("sse-%s-%d", userID, time.Now().UnixNano())

	handler := func(n *service.Notification) {
		select {
		case notifCh <- n:
		default:
		}
	}

	for _, mbx := range mailboxes.Items {
		h.notifyService.Subscribe(subscriptionID, mbx.ID, userID, handler)
	}

	defer func() {
		h.notifyService.UnsubscribeByID(subscriptionID)
		close(notifCh)
	}()

	writeSSE(w, "connected", fmt.Sprintf(`{"userId":"%s","mailboxCount":%d}`, userID, len(mailboxes.Items)))
	w.Flush()

	ctx := c.Request().Context()
	keepalive := time.NewTicker(30 * time.Second)
	defer keepalive.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil
		case n := <-notifCh:
			evt := notificationToSSE(n)
			data, _ := json.Marshal(evt)
			writeSSE(w, evt.Event, string(data))
			w.Flush()
		case <-keepalive.C:
			fmt.Fprintf(w, ": keepalive\n\n")
			w.Flush()
		}
	}
}

func extractToken(c echo.Context) string {
	auth := c.Request().Header.Get("Authorization")
	if strings.HasPrefix(auth, "Bearer ") {
		return auth[7:]
	}
	if token := c.QueryParam("token"); token != "" {
		return token
	}
	return ""
}

func notificationToSSE(n *service.Notification) sseEvent {
	evt := sseEvent{
		MailboxID: n.MailboxID.String(),
		MessageID: n.MessageID.String(),
		Timestamp: n.Timestamp,
	}

	switch n.Type {
	case service.NotificationNewMessage:
		evt.Event = "message.new"
		evt.Count = n.MessageCount
	case service.NotificationFlagsChanged:
		evt.Event = "message.flags"
		evt.Flags = n.Flags
	case service.NotificationMessageExpunged:
		evt.Event = "message.deleted"
	case service.NotificationMailboxUpdated:
		evt.Event = "mailbox.updated"
		evt.Count = n.MessageCount
	default:
		evt.Event = "unknown"
	}

	return evt
}

func writeSSE(w http.ResponseWriter, event, data string) {
	fmt.Fprintf(w, "event: %s\ndata: %s\n\n", event, data)
}

// Ensure interface is correct.
var _ domain.ID = domain.ID("")
