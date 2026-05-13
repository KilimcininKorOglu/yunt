package mail

import (
	"context"
	"encoding/json"
	"fmt"

	"yunt/internal/domain"
	"yunt/internal/jmap/core"
	"yunt/internal/jmap/state"
	svc "yunt/internal/service"
)

// MailboxHandler implements JMAP Mailbox methods (RFC 8621 §2).
type MailboxHandler struct {
	mailboxService *svc.MailboxService
	stateManager   *state.Manager
}

// NewMailboxHandler creates a new Mailbox method handler.
func NewMailboxHandler(mbxSvc *svc.MailboxService, stateMgr *state.Manager) *MailboxHandler {
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

// Set implements Mailbox/set (RFC 8621 §2.5).
func (h *MailboxHandler) Set(ctx context.Context, accountID domain.ID, args json.RawMessage) (json.RawMessage, *core.MethodError) {
	var a struct {
		AccountID              string                            `json:"accountId"`
		IfInState              *string                           `json:"ifInState"`
		Create                 map[string]map[string]interface{} `json:"create"`
		Update                 map[string]map[string]interface{} `json:"update"`
		Destroy                []string                          `json:"destroy"`
		OnDestroyRemoveEmails  bool                              `json:"onDestroyRemoveEmails"`
	}
	if err := json.Unmarshal(args, &a); err != nil {
		return nil, core.NewMethodError(core.ErrorInvalidArguments, err.Error())
	}

	stateStr, _ := h.stateManager.CurrentState(ctx, accountID, "Mailbox")

	created := map[string]interface{}{}
	notCreated := map[string]interface{}{}
	updated := map[string]interface{}{}
	notUpdated := map[string]interface{}{}
	destroyed := []string{}
	notDestroyed := map[string]interface{}{}

	for tempID, props := range a.Create {
		name, _ := props["name"].(string)
		if name == "" {
			notCreated[tempID] = map[string]interface{}{"type": "invalidProperties", "description": "name is required"}
			continue
		}
		input := &svc.CreateMailboxInput{UserID: accountID, Name: name, Address: name + "@localhost"}
		if desc, ok := props["description"].(string); ok {
			input.Description = desc
		}
		mbx, err := h.mailboxService.CreateMailbox(ctx, input)
		if err != nil {
			notCreated[tempID] = map[string]interface{}{"type": "serverFail", "description": err.Error()}
			continue
		}
		created[tempID] = map[string]interface{}{"id": string(mbx.ID)}
	}

	for id, patch := range a.Update {
		updateInput := &svc.UpdateMailboxInput{MailboxID: domain.ID(id), UserID: accountID}
		if name, ok := patch["name"].(string); ok {
			updateInput.Name = &name
		}
		if desc, ok := patch["description"].(string); ok {
			updateInput.Description = &desc
		}
		_, err := h.mailboxService.UpdateMailbox(ctx, updateInput)
		if err != nil {
			notUpdated[id] = map[string]interface{}{"type": "serverFail", "description": err.Error()}
			continue
		}
		updated[id] = nil
	}

	for _, id := range a.Destroy {
		err := h.mailboxService.DeleteMailbox(ctx, domain.ID(id), accountID)
		if err != nil {
			notDestroyed[id] = map[string]interface{}{"type": "serverFail", "description": err.Error()}
			continue
		}
		destroyed = append(destroyed, id)
	}

	newState, _ := h.stateManager.CurrentState(ctx, accountID, "Mailbox")

	resp := map[string]interface{}{
		"accountId": a.AccountID,
		"oldState":  stateStr,
		"newState":  newState,
	}
	if a.Create != nil {
		resp["created"] = created
		resp["notCreated"] = notCreated
	}
	if a.Update != nil {
		resp["updated"] = updated
		resp["notUpdated"] = notUpdated
	}
	if a.Destroy != nil {
		resp["destroyed"] = destroyed
		resp["notDestroyed"] = notDestroyed
	}

	return marshalJSON(resp)
}
