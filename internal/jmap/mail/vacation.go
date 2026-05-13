package mail

import (
	"context"
	"encoding/json"

	"yunt/internal/domain"
	"yunt/internal/jmap/core"
	"yunt/internal/jmap/state"
	"yunt/internal/repository"
)

// VacationHandler implements JMAP VacationResponse methods (RFC 8621 §8).
type VacationHandler struct {
	repo         repository.Repository
	stateManager *state.Manager
}

// NewVacationHandler creates a new VacationResponse method handler.
func NewVacationHandler(repo repository.Repository, stateMgr *state.Manager) *VacationHandler {
	return &VacationHandler{repo: repo, stateManager: stateMgr}
}

// Get implements VacationResponse/get. The only valid id is "singleton".
func (h *VacationHandler) Get(ctx context.Context, accountID domain.ID, args json.RawMessage) (json.RawMessage, *core.MethodError) {
	var a struct {
		AccountID string   `json:"accountId"`
		IDs       []string `json:"ids"`
	}
	if err := json.Unmarshal(args, &a); err != nil {
		return nil, core.NewMethodError(core.ErrorInvalidArguments, err.Error())
	}

	stateStr, _ := h.stateManager.CurrentState(ctx, accountID, "VacationResponse")

	vacation, err := h.repo.JMAP().Vacation().GetByUserID(ctx, accountID)
	if err != nil {
		vacation = &domain.VacationResponse{
			ID:        "singleton",
			UserID:    accountID,
			IsEnabled: false,
		}
	}

	var list []map[string]interface{}
	var notFound []string

	if a.IDs == nil || contains(a.IDs, "singleton") {
		list = append(list, vacationToJMAP(vacation))
	}
	for _, id := range a.IDs {
		if id != "singleton" {
			notFound = append(notFound, id)
		}
	}

	return marshalGetResponse(a.AccountID, stateStr, list, notFound)
}

// Set implements VacationResponse/set. Only "singleton" can be updated.
func (h *VacationHandler) Set(ctx context.Context, accountID domain.ID, args json.RawMessage) (json.RawMessage, *core.MethodError) {
	var a struct {
		AccountID string                            `json:"accountId"`
		Update    map[string]map[string]interface{} `json:"update"`
	}
	if err := json.Unmarshal(args, &a); err != nil {
		return nil, core.NewMethodError(core.ErrorInvalidArguments, err.Error())
	}

	stateStr, _ := h.stateManager.CurrentState(ctx, accountID, "VacationResponse")

	resp := map[string]interface{}{
		"accountId": a.AccountID,
		"oldState":  stateStr,
		"newState":  stateStr,
	}

	if a.Update != nil {
		updated := map[string]interface{}{}
		notUpdated := map[string]interface{}{}

		for id, patch := range a.Update {
			if id != "singleton" {
				notUpdated[id] = map[string]interface{}{
					"type":        core.SetErrorNotFound,
					"description": "only 'singleton' id is valid",
				}
				continue
			}

			vacation, err := h.repo.JMAP().Vacation().GetByUserID(ctx, accountID)
			if err != nil {
				vacation = &domain.VacationResponse{
					ID:     "singleton",
					UserID: accountID,
				}
			}

			if v, ok := patch["isEnabled"].(bool); ok {
				vacation.IsEnabled = v
			}
			if v, ok := patch["subject"].(string); ok {
				vacation.Subject = v
			}
			if v, ok := patch["textBody"].(string); ok {
				vacation.TextBody = v
			}
			if v, ok := patch["htmlBody"].(string); ok {
				vacation.HTMLBody = v
			}

			if err := h.repo.JMAP().Vacation().Set(ctx, vacation); err != nil {
				notUpdated[id] = map[string]interface{}{
					"type":        core.ErrorServerFail,
					"description": err.Error(),
				}
				continue
			}

			updated[id] = nil
		}

		resp["updated"] = updated
		resp["notUpdated"] = notUpdated
	}

	return marshalJSON(resp)
}

func vacationToJMAP(v *domain.VacationResponse) map[string]interface{} {
	result := map[string]interface{}{
		"id":        "singleton",
		"isEnabled": v.IsEnabled,
		"subject":   v.Subject,
		"textBody":  v.TextBody,
		"htmlBody":  v.HTMLBody,
	}
	if v.FromDate != nil {
		result["fromDate"] = v.FromDate.Time.UTC().Format("2006-01-02T15:04:05Z")
	}
	if v.ToDate != nil {
		result["toDate"] = v.ToDate.Time.UTC().Format("2006-01-02T15:04:05Z")
	}
	return result
}

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}
