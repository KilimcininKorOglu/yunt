package contacts

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"yunt/internal/domain"
	"yunt/internal/jmap/core"
	"yunt/internal/jmap/state"
	"yunt/internal/repository"
)

// ContactCardHandler implements JMAP ContactCard methods (RFC 9610).
type ContactCardHandler struct {
	repo         repository.Repository
	stateManager *state.Manager
}

// NewContactCardHandler creates a new ContactCard method handler.
func NewContactCardHandler(repo repository.Repository, stateMgr *state.Manager) *ContactCardHandler {
	return &ContactCardHandler{repo: repo, stateManager: stateMgr}
}

// Get implements ContactCard/get.
func (h *ContactCardHandler) Get(ctx context.Context, accountID domain.ID, args json.RawMessage) (json.RawMessage, *core.MethodError) {
	var a struct {
		AccountID string   `json:"accountId"`
		IDs       []string `json:"ids"`
	}
	if err := json.Unmarshal(args, &a); err != nil {
		return nil, core.NewMethodError(core.ErrorInvalidArguments, err.Error())
	}

	stateStr, _ := h.stateManager.CurrentState(ctx, accountID, "ContactCard")

	var list []map[string]interface{}
	var notFound []string

	if a.IDs == nil {
		result, err := h.repo.JMAP().ContactCards().List(ctx, accountID, nil)
		if err != nil {
			return nil, core.NewMethodError(core.ErrorServerFail, err.Error())
		}
		for _, card := range result.Items {
			list = append(list, contactCardToJMAP(card))
		}
	} else {
		for _, id := range a.IDs {
			card, err := h.repo.JMAP().ContactCards().GetByID(ctx, domain.ID(id))
			if err != nil {
				notFound = append(notFound, id)
				continue
			}
			list = append(list, contactCardToJMAP(card))
		}
	}

	return marshalGetResponse(a.AccountID, stateStr, list, notFound)
}

// Changes implements ContactCard/changes.
func (h *ContactCardHandler) Changes(ctx context.Context, accountID domain.ID, args json.RawMessage) (json.RawMessage, *core.MethodError) {
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

	result, err := h.stateManager.GetChanges(ctx, accountID, "ContactCard", a.SinceState, maxChanges)
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

// Query implements ContactCard/query (RFC 9610 §3.3).
func (h *ContactCardHandler) Query(ctx context.Context, accountID domain.ID, args json.RawMessage) (json.RawMessage, *core.MethodError) {
	var a struct {
		AccountID string                 `json:"accountId"`
		Filter    *domain.JMAPContactFilter `json:"filter"`
		Position  int                    `json:"position"`
		Limit     *int                   `json:"limit"`
	}
	if err := json.Unmarshal(args, &a); err != nil {
		return nil, core.NewMethodError(core.ErrorInvalidArguments, err.Error())
	}

	perPage := 50
	if a.Limit != nil && *a.Limit > 0 {
		perPage = *a.Limit
	}

	opts := &repository.ListOptions{
		Pagination: &repository.PaginationOptions{Page: 1, PerPage: perPage},
	}

	result, err := h.repo.JMAP().ContactCards().Query(ctx, accountID, a.Filter, opts)
	if err != nil {
		return nil, core.NewMethodError(core.ErrorServerFail, err.Error())
	}

	stateStr, _ := h.stateManager.CurrentState(ctx, accountID, "ContactCard")

	ids := make([]string, len(result.Items))
	for i, card := range result.Items {
		ids[i] = string(card.ID)
	}

	return marshalJSON(map[string]interface{}{
		"accountId":           a.AccountID,
		"queryState":          stateStr,
		"canCalculateChanges": false,
		"position":            a.Position,
		"ids":                 ids,
		"total":               result.Total,
	})
}

// QueryChanges implements ContactCard/queryChanges (RFC 9610).
func (h *ContactCardHandler) QueryChanges(ctx context.Context, accountID domain.ID, args json.RawMessage) (json.RawMessage, *core.MethodError) {
	return marshalJSON(map[string]interface{}{
		"accountId":     string(accountID),
		"oldQueryState": "0",
		"newQueryState": "0",
		"removed":       []string{},
		"added":         []interface{}{},
	})
}

// Set implements ContactCard/set.
func (h *ContactCardHandler) Set(ctx context.Context, accountID domain.ID, args json.RawMessage) (json.RawMessage, *core.MethodError) {
	var a struct {
		AccountID string                            `json:"accountId"`
		Create    map[string]map[string]interface{} `json:"create"`
		Update    map[string]map[string]interface{} `json:"update"`
		Destroy   []string                          `json:"destroy"`
	}
	if err := json.Unmarshal(args, &a); err != nil {
		return nil, core.NewMethodError(core.ErrorInvalidArguments, err.Error())
	}

	stateStr, _ := h.stateManager.CurrentState(ctx, accountID, "ContactCard")

	created := map[string]interface{}{}
	notCreated := map[string]interface{}{}
	updated := map[string]interface{}{}
	notUpdated := map[string]interface{}{}
	destroyed := []string{}
	notDestroyed := map[string]interface{}{}

	for tempID, props := range a.Create {
		uid, _ := props["uid"].(string)
		if uid == "" {
			uid = fmt.Sprintf("urn:uuid:%s", tempID)
		}
		fullName, _ := props["fullName"].(string)
		kind, _ := props["kind"].(string)
		if kind == "" {
			kind = "individual"
		}

		abIDs := map[domain.ID]bool{}
		if abMap, ok := props["addressBookIds"].(map[string]interface{}); ok {
			for abID := range abMap {
				abIDs[domain.ID(abID)] = true
			}
		}
		if len(abIDs) == 0 {
			defBook, err := h.repo.JMAP().AddressBooks().GetDefault(ctx, accountID)
			if err == nil {
				abIDs[defBook.ID] = true
			}
		}

		card := &domain.ContactCard{
			ID: domain.ID(fmt.Sprintf("cc-%s-%d", accountID, time.Now().UnixNano())),
			UID: uid, UserID: accountID, AddressBookIDs: abIDs,
			Kind: kind, FullName: fullName,
			CreatedAt: domain.Now(), UpdatedAt: domain.Now(),
		}

		if err := h.repo.JMAP().ContactCards().Create(ctx, card); err != nil {
			notCreated[tempID] = map[string]interface{}{"type": "serverFail", "description": err.Error()}
			continue
		}
		created[tempID] = map[string]interface{}{"id": string(card.ID)}
	}

	for id, patch := range a.Update {
		card, err := h.repo.JMAP().ContactCards().GetByID(ctx, domain.ID(id))
		if err != nil {
			notUpdated[id] = map[string]interface{}{"type": "notFound"}
			continue
		}
		if fn, ok := patch["fullName"].(string); ok {
			card.FullName = fn
		}
		if k, ok := patch["kind"].(string); ok {
			card.Kind = k
		}
		if n, ok := patch["notes"].(string); ok {
			card.Notes = n
		}
		if err := h.repo.JMAP().ContactCards().Update(ctx, card); err != nil {
			notUpdated[id] = map[string]interface{}{"type": "serverFail", "description": err.Error()}
			continue
		}
		updated[id] = nil
	}

	for _, id := range a.Destroy {
		if err := h.repo.JMAP().ContactCards().Delete(ctx, domain.ID(id)); err != nil {
			notDestroyed[id] = map[string]interface{}{"type": "notFound"}
			continue
		}
		destroyed = append(destroyed, id)
	}

	newState, _ := h.stateManager.CurrentState(ctx, accountID, "ContactCard")
	resp := map[string]interface{}{
		"accountId": a.AccountID,
		"oldState":  stateStr,
		"newState":  newState,
	}
	if a.Create != nil { resp["created"] = created; resp["notCreated"] = notCreated }
	if a.Update != nil { resp["updated"] = updated; resp["notUpdated"] = notUpdated }
	if a.Destroy != nil { resp["destroyed"] = destroyed; resp["notDestroyed"] = notDestroyed }
	return marshalJSON(resp)
}

func contactCardToJMAP(card *domain.ContactCard) map[string]interface{} {
	result := map[string]interface{}{
		"@type":          "Card",
		"version":        "1.0",
		"id":             string(card.ID),
		"uid":            card.UID,
		"addressBookIds": card.AddressBookIDs,
		"kind":           card.Kind,
		"fullName":       card.FullName,
	}
	if card.Name != nil {
		result["name"] = card.Name
	}
	if len(card.Emails) > 0 {
		emails := make(map[string]interface{})
		for i, e := range card.Emails {
			emails[fmt.Sprintf("e%d", i)] = map[string]interface{}{
				"@type":   "EmailAddress",
				"address": e.Address,
				"label":   e.Label,
			}
		}
		result["emails"] = emails
	}
	if len(card.Phones) > 0 {
		phones := make(map[string]interface{})
		for i, p := range card.Phones {
			phones[fmt.Sprintf("p%d", i)] = map[string]interface{}{
				"@type":  "Phone",
				"number": p.Number,
				"label":  p.Label,
			}
		}
		result["phones"] = phones
	}
	if card.Notes != "" {
		result["notes"] = map[string]interface{}{
			"n0": map[string]interface{}{
				"@type": "Note",
				"note":  card.Notes,
			},
		}
	}
	return result
}
