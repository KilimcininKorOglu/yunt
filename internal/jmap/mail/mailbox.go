package mail

import (
	"context"
	"encoding/json"
	"fmt"

	"yunt/internal/domain"
	"yunt/internal/jmap/core"
	"yunt/internal/jmap/state"
	"yunt/internal/service"
)

// MailboxHandler implements JMAP Mailbox methods (RFC 8621 §2).
type MailboxHandler struct {
	mailboxService *service.MailboxService
	stateManager   *state.Manager
}

// NewMailboxHandler creates a new Mailbox method handler.
func NewMailboxHandler(mbxSvc *service.MailboxService, stateMgr *state.Manager) *MailboxHandler {
	return &MailboxHandler{
		mailboxService: mbxSvc,
		stateManager:   stateMgr,
	}
}

// Get implements Mailbox/get (RFC 8621 §2.1).
func (h *MailboxHandler) Get(ctx context.Context, accountID domain.ID, args json.RawMessage) (json.RawMessage, *core.MethodError) {
	var a struct {
		AccountID  string   `json:"accountId"`
		IDs        []string `json:"ids"`
		Properties []string `json:"properties"`
	}
	if err := json.Unmarshal(args, &a); err != nil {
		return nil, core.NewMethodError(core.ErrorInvalidArguments, err.Error())
	}

	stateStr, _ := h.stateManager.CurrentState(ctx, accountID, "Mailbox")

	if a.IDs == nil {
	
		result, err := h.mailboxService.ListMailboxes(ctx, accountID, nil)
		if err != nil {
			return nil, core.NewMethodError(core.ErrorServerFail, err.Error())
		}

		list := make([]map[string]interface{}, len(result.Items))
		for i, mbx := range result.Items {
			list[i] = MailboxToJMAP(mbx)
		}

		return marshalGetResponse(a.AccountID, stateStr, list, nil)
	}

	var list []map[string]interface{}
	var notFound []string

	for _, id := range a.IDs {
		mbx, err := h.mailboxService.GetMailbox(ctx, domain.ID(id), accountID)
		if err != nil {
			notFound = append(notFound, id)
			continue
		}
		list = append(list, MailboxToJMAP(mbx))
	}

	return marshalGetResponse(a.AccountID, stateStr, list, notFound)
}

// Changes implements Mailbox/changes (RFC 8621 §2.2).
func (h *MailboxHandler) Changes(ctx context.Context, accountID domain.ID, args json.RawMessage) (json.RawMessage, *core.MethodError) {
	var a struct {
		AccountID  string `json:"accountId"`
		SinceState string `json:"sinceState"`
		MaxChanges *int64 `json:"maxChanges"`
	}
	if err := json.Unmarshal(args, &a); err != nil {
		return nil, core.NewMethodError(core.ErrorInvalidArguments, err.Error())
	}

	maxChanges := int64(1000)
	if a.MaxChanges != nil {
		maxChanges = *a.MaxChanges
	}

	result, err := h.stateManager.GetChanges(ctx, accountID, "Mailbox", a.SinceState, maxChanges)
	if err != nil {
		return nil, core.NewMethodError(core.ErrorServerFail, err.Error())
	}

	return marshalJSON(map[string]interface{}{
		"accountId":      a.AccountID,
		"oldState":       a.SinceState,
		"newState":       fmt.Sprintf("%d", result.NewState),
		"hasMoreChanges": result.HasMore,
		"created":        idSlice(result.Created),
		"updated":        idSlice(result.Updated),
		"destroyed":      idSlice(result.Destroyed),
	})
}

// Query implements Mailbox/query (RFC 8621 §2.3).
func (h *MailboxHandler) Query(ctx context.Context, accountID domain.ID, args json.RawMessage) (json.RawMessage, *core.MethodError) {
	var a struct {
		AccountID string                 `json:"accountId"`
		Filter    map[string]interface{} `json:"filter"`
		Sort      []emailSort            `json:"sort"`
		Position  int                    `json:"position"`
		Limit     *int                   `json:"limit"`
	}
	if err := json.Unmarshal(args, &a); err != nil {
		return nil, core.NewMethodError(core.ErrorInvalidArguments, err.Error())
	}



	if role, ok := a.Filter["role"].(string); ok {
		_ = role // filter by role not in domain.MailboxFilter yet
	}

	result, err := h.mailboxService.ListMailboxes(ctx, accountID, nil)
	if err != nil {
		return nil, core.NewMethodError(core.ErrorServerFail, err.Error())
	}

	stateStr, _ := h.stateManager.CurrentState(ctx, accountID, "Mailbox")

	ids := make([]string, len(result.Items))
	for i, mbx := range result.Items {
		ids[i] = string(mbx.ID)
	}

	return marshalJSON(map[string]interface{}{
		"accountId":          a.AccountID,
		"queryState":         stateStr,
		"canCalculateChanges": false,
		"position":           a.Position,
		"ids":                ids,
		"total":              result.Total,
	})
}
