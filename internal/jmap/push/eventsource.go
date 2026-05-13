package push

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/labstack/echo/v4"

	"yunt/internal/api/middleware"
	"yunt/internal/jmap/state"
	"yunt/internal/repository"
	"yunt/internal/service"
)

// EventSourceHandler handles GET /jmap/eventsource (RFC 8620 §7.3).
type EventSourceHandler struct {
	notifyService *service.NotifyService
	stateManager  *state.Manager
	mailboxRepo   repository.MailboxRepository
}

// NewEventSourceHandler creates a new JMAP EventSource handler.
func NewEventSourceHandler(notifySvc *service.NotifyService, stateMgr *state.Manager, mbxRepo repository.MailboxRepository) *EventSourceHandler {
	return &EventSourceHandler{
		notifyService: notifySvc,
		stateManager:  stateMgr,
		mailboxRepo:   mbxRepo,
	}
}

// Handle implements the SSE endpoint for JMAP push.
func (h *EventSourceHandler) Handle(c echo.Context) error {
	userID := middleware.GetUserID(c)

	typesParam := c.QueryParam("types")
	closeAfter := c.QueryParam("closeafter")
	pingStr := c.QueryParam("ping")

	pingInterval := 30 * time.Second
	if pingStr != "" {
		if v, err := time.ParseDuration(pingStr + "s"); err == nil && v > 0 {
			pingInterval = v
		}
	}

	var typeFilter map[string]bool
	if typesParam != "" && typesParam != "*" {
		typeFilter = make(map[string]bool)
		for _, t := range strings.Split(typesParam, ",") {
			typeFilter[strings.TrimSpace(t)] = true
		}
	}

	w := c.Response()
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	ctrl := http.NewResponseController(w.Writer)
	_ = ctrl.SetWriteDeadline(time.Time{})

	flusher, ok := w.Writer.(http.Flusher)
	if !ok {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "streaming not supported"})
	}

	mailboxes, _ := h.mailboxRepo.ListByUser(c.Request().Context(), userID, nil)
	subscriptionID := fmt.Sprintf("jmap-sse-%s-%d", userID, time.Now().UnixNano())

	notifyCh := make(chan *service.Notification, 64)
	handler := func(n *service.Notification) {
		select {
		case notifyCh <- n:
		default:
		}
	}

	if mailboxes != nil {
		for _, mbx := range mailboxes.Items {
			h.notifyService.Subscribe(subscriptionID, mbx.ID, userID, handler)
		}
	}
	defer h.notifyService.UnsubscribeByID(subscriptionID)

	eventID := 0
	pingTicker := time.NewTicker(pingInterval)
	defer pingTicker.Stop()

	for {
		select {
		case <-c.Request().Context().Done():
			return nil

		case n := <-notifyCh:
			dataType := notificationToJMAPType(n)
			if typeFilter != nil && !typeFilter[dataType] {
				continue
			}

			stateStr, _ := h.stateManager.CurrentState(c.Request().Context(), userID, dataType)
			eventID++

			stateChange := map[string]interface{}{
				"@type": "StateChange",
				"changed": map[string]interface{}{
					string(userID): map[string]string{
						dataType: stateStr,
					},
				},
			}
			data, _ := json.Marshal(stateChange)

			fmt.Fprintf(w, "id: %d\nevent: state\ndata: %s\n\n", eventID, data)
			flusher.Flush()

			if closeAfter == "state" {
				return nil
			}

		case <-pingTicker.C:
			eventID++
			fmt.Fprintf(w, "id: %d\nevent: ping\ndata: {\"interval\":%d}\n\n", eventID, int(pingInterval.Seconds()))
			flusher.Flush()
		}
	}
}

func notificationToJMAPType(n *service.Notification) string {
	switch n.Type {
	case service.NotificationNewMessage, service.NotificationFlagsChanged, service.NotificationMessageExpunged:
		return "Email"
	case service.NotificationMailboxUpdated:
		return "Mailbox"
	default:
		return "Email"
	}
}
