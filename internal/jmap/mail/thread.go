package mail

import (
	"context"
	"encoding/json"

	"yunt/internal/domain"
	"yunt/internal/jmap/core"
	"yunt/internal/jmap/state"
	"yunt/internal/repository"
)

// ThreadHandler implements JMAP Thread methods (RFC 8621 §3).
type ThreadHandler struct {
	repo         repository.Repository
	stateManager *state.Manager
}

// NewThreadHandler creates a new Thread method handler.
func NewThreadHandler(repo repository.Repository, stateMgr *state.Manager) *ThreadHandler {
	return &ThreadHandler{repo: repo, stateManager: stateMgr}
}

// Get implements Thread/get (RFC 8621 §3.1).
// Returns emailIds sorted by receivedAt oldest-first (MUST per RFC).
func (h *ThreadHandler) Get(ctx context.Context, accountID domain.ID, args json.RawMessage) (json.RawMessage, *core.MethodError) {
	var a struct {
		AccountID string   `json:"accountId"`
		IDs       []string `json:"ids"`
	}
	if err := json.Unmarshal(args, &a); err != nil {
		return nil, core.NewMethodError(core.ErrorInvalidArguments, err.Error())
	}

	stateStr, _ := h.stateManager.CurrentState(ctx, accountID, "Thread")

	var list []map[string]interface{}
	var notFound []string

	for _, id := range a.IDs {
		result, err := h.repo.Messages().GetByThreadID(ctx, domain.ID(id), nil)
		if err != nil || len(result.Items) == 0 {
			notFound = append(notFound, id)
			continue
		}

		emailIDs := make([]string, len(result.Items))
		for i, msg := range result.Items {
			emailIDs[i] = string(msg.ID)
		}

		list = append(list, map[string]interface{}{
			"id":       id,
			"emailIds": emailIDs,
		})
	}

	return marshalGetResponse(a.AccountID, stateStr, list, notFound)
}

// Changes implements Thread/changes (RFC 8621 §3.2).
func (h *ThreadHandler) Changes(ctx context.Context, accountID domain.ID, args json.RawMessage) (json.RawMessage, *core.MethodError) {
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

	result, err := h.stateManager.GetChanges(ctx, accountID, "Thread", a.SinceState, maxChanges)
	if err != nil {
		return nil, core.NewMethodError(core.ErrorServerFail, err.Error())
	}

	return marshalJSON(map[string]interface{}{
		"accountId":      a.AccountID,
		"oldState":       a.SinceState,
		"newState":       formatState(result.NewState),
		"hasMoreChanges": result.HasMore,
		"created":        idSlice(result.Created),
		"updated":        idSlice(result.Updated),
		"destroyed":      idSlice(result.Destroyed),
	})
}
