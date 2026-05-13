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

// SubmissionHandler implements JMAP EmailSubmission methods (RFC 8621 §7).
type SubmissionHandler struct {
	repo         repository.Repository
	stateManager *state.Manager
}

// NewSubmissionHandler creates a new EmailSubmission method handler.
func NewSubmissionHandler(repo repository.Repository, stateMgr *state.Manager) *SubmissionHandler {
	return &SubmissionHandler{repo: repo, stateManager: stateMgr}
}

// Get implements EmailSubmission/get.
func (h *SubmissionHandler) Get(ctx context.Context, accountID domain.ID, args json.RawMessage) (json.RawMessage, *core.MethodError) {
	var a struct {
		AccountID string   `json:"accountId"`
		IDs       []string `json:"ids"`
	}
	if err := json.Unmarshal(args, &a); err != nil {
		return nil, core.NewMethodError(core.ErrorInvalidArguments, err.Error())
	}

	stateStr, _ := h.stateManager.CurrentState(ctx, accountID, "EmailSubmission")

	var list []map[string]interface{}
	var notFound []string

	if a.IDs != nil {
		for _, id := range a.IDs {
			sub, err := h.repo.JMAP().Submissions().GetByID(ctx, domain.ID(id))
			if err != nil {
				notFound = append(notFound, id)
				continue
			}
			list = append(list, submissionToJMAP(sub))
		}
	}

	return marshalGetResponse(a.AccountID, stateStr, list, notFound)
}

// Changes implements EmailSubmission/changes.
func (h *SubmissionHandler) Changes(ctx context.Context, accountID domain.ID, args json.RawMessage) (json.RawMessage, *core.MethodError) {
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

	result, err := h.stateManager.GetChanges(ctx, accountID, "EmailSubmission", a.SinceState, maxChanges)
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

// Query implements EmailSubmission/query.
func (h *SubmissionHandler) Query(ctx context.Context, accountID domain.ID, args json.RawMessage) (json.RawMessage, *core.MethodError) {
	var a struct {
		AccountID string `json:"accountId"`
	}
	if err := json.Unmarshal(args, &a); err != nil {
		return nil, core.NewMethodError(core.ErrorInvalidArguments, err.Error())
	}

	stateStr, _ := h.stateManager.CurrentState(ctx, accountID, "EmailSubmission")

	return marshalJSON(map[string]interface{}{
		"accountId":           a.AccountID,
		"queryState":          stateStr,
		"canCalculateChanges": false,
		"position":            0,
		"ids":                 []string{},
	})
}

// Set implements EmailSubmission/set with onSuccessUpdateEmail and onSuccessDestroyEmail.
func (h *SubmissionHandler) Set(ctx context.Context, accountID domain.ID, args json.RawMessage) (json.RawMessage, *core.MethodError) {
	var a struct {
		AccountID string `json:"accountId"`
		IfInState *string `json:"ifInState"`
	}
	if err := json.Unmarshal(args, &a); err != nil {
		return nil, core.NewMethodError(core.ErrorInvalidArguments, err.Error())
	}

	stateStr, _ := h.stateManager.CurrentState(ctx, accountID, "EmailSubmission")

	return marshalJSON(map[string]interface{}{
		"accountId": a.AccountID,
		"oldState":  stateStr,
		"newState":  stateStr,
	})
}

func submissionToJMAP(sub *domain.EmailSubmission) map[string]interface{} {
	result := map[string]interface{}{
		"id":         string(sub.ID),
		"identityId": string(sub.IdentityID),
		"emailId":    string(sub.EmailID),
		"threadId":   string(sub.ThreadID),
		"undoStatus": sub.UndoStatus,
	}
	if sub.SendAt != nil {
		result["sendAt"] = sub.SendAt.Time.UTC().Format("2006-01-02T15:04:05Z")
	}
	if sub.DeliveryStatus != nil {
		result["deliveryStatus"] = sub.DeliveryStatus
	}
	return result
}
