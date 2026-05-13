package mail

import (
	"context"
	"encoding/json"
	"fmt"

	"yunt/internal/domain"
	"yunt/internal/jmap/core"
	"yunt/internal/jmap/state"
	"yunt/internal/repository"
)

// IdentityHandler implements JMAP Identity methods (RFC 8621 §6).
type IdentityHandler struct {
	repo         repository.Repository
	stateManager *state.Manager
}

// NewIdentityHandler creates a new Identity method handler.
func NewIdentityHandler(repo repository.Repository, stateMgr *state.Manager) *IdentityHandler {
	return &IdentityHandler{repo: repo, stateManager: stateMgr}
}

// Get implements Identity/get.
func (h *IdentityHandler) Get(ctx context.Context, accountID domain.ID, args json.RawMessage) (json.RawMessage, *core.MethodError) {
	var a struct {
		AccountID string   `json:"accountId"`
		IDs       []string `json:"ids"`
	}
	if err := json.Unmarshal(args, &a); err != nil {
		return nil, core.NewMethodError(core.ErrorInvalidArguments, err.Error())
	}

	stateStr, _ := h.stateManager.CurrentState(ctx, accountID, "Identity")

	identities, err := h.repo.JMAP().Identities().List(ctx, accountID)
	if err != nil {
		return nil, core.NewMethodError(core.ErrorServerFail, err.Error())
	}

	if identities == nil {
		user, err := h.repo.Users().GetByID(ctx, accountID)
		if err == nil {
			identities = []*domain.Identity{{
				ID:        accountID,
				UserID:    accountID,
				Name:      user.DisplayName,
				Email:     user.Email,
				MayDelete: false,
			}}
		}
	}

	var list []map[string]interface{}
	var notFound []string

	if a.IDs == nil {
		for _, id := range identities {
			list = append(list, identityToJMAP(id))
		}
	} else {
		idMap := make(map[string]*domain.Identity)
		for _, id := range identities {
			idMap[string(id.ID)] = id
		}
		for _, id := range a.IDs {
			if identity, ok := idMap[id]; ok {
				list = append(list, identityToJMAP(identity))
			} else {
				notFound = append(notFound, id)
			}
		}
	}

	return marshalGetResponse(a.AccountID, stateStr, list, notFound)
}

// Changes implements Identity/changes.
func (h *IdentityHandler) Changes(ctx context.Context, accountID domain.ID, args json.RawMessage) (json.RawMessage, *core.MethodError) {
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

	result, err := h.stateManager.GetChanges(ctx, accountID, "Identity", a.SinceState, maxChanges)
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

// Set implements Identity/set.
func (h *IdentityHandler) Set(ctx context.Context, accountID domain.ID, args json.RawMessage) (json.RawMessage, *core.MethodError) {
	var a struct {
		AccountID string                            `json:"accountId"`
		IfInState *string                           `json:"ifInState"`
		Create    map[string]map[string]interface{} `json:"create"`
		Update    map[string]map[string]interface{} `json:"update"`
		Destroy   []string                          `json:"destroy"`
	}
	if err := json.Unmarshal(args, &a); err != nil {
		return nil, core.NewMethodError(core.ErrorInvalidArguments, err.Error())
	}

	stateStr, _ := h.stateManager.CurrentState(ctx, accountID, "Identity")

	resp := map[string]interface{}{
		"accountId": a.AccountID,
		"oldState":  stateStr,
		"newState":  stateStr,
	}

	if a.Create != nil {
		resp["created"] = map[string]interface{}{}
		resp["notCreated"] = map[string]interface{}{}
	}
	if a.Update != nil {
		resp["updated"] = map[string]interface{}{}
		resp["notUpdated"] = map[string]interface{}{}
	}
	if a.Destroy != nil {
		resp["destroyed"] = []string{}
		resp["notDestroyed"] = map[string]interface{}{}
	}

	return marshalJSON(resp)
}

func identityToJMAP(id *domain.Identity) map[string]interface{} {
	result := map[string]interface{}{
		"id":            string(id.ID),
		"name":          id.Name,
		"email":         id.Email,
		"textSignature": id.TextSignature,
		"htmlSignature": id.HTMLSignature,
		"mayDelete":     id.MayDelete,
	}
	if len(id.ReplyTo) > 0 {
		result["replyTo"] = emailAddrsToJMAP(id.ReplyTo)
	}
	if len(id.Bcc) > 0 {
		result["bcc"] = emailAddrsToJMAP(id.Bcc)
	}
	return result
}
