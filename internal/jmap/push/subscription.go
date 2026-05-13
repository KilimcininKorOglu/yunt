package push

import (
	"context"
	"encoding/json"
	"fmt"

	"yunt/internal/domain"
	"yunt/internal/jmap/core"
	"yunt/internal/repository"
)

// SubscriptionHandler implements JMAP PushSubscription methods (RFC 8620 §7.2).
// PushSubscription does NOT take accountId — it is user-global.
type SubscriptionHandler struct {
	repo repository.Repository
}

// NewSubscriptionHandler creates a new PushSubscription method handler.
func NewSubscriptionHandler(repo repository.Repository) *SubscriptionHandler {
	return &SubscriptionHandler{repo: repo}
}

// Get implements PushSubscription/get (no accountId, no state).
func (h *SubscriptionHandler) Get(ctx context.Context, userID domain.ID, args json.RawMessage) (json.RawMessage, *core.MethodError) {
	var a struct {
		IDs []string `json:"ids"`
	}
	if err := json.Unmarshal(args, &a); err != nil {
		return nil, core.NewMethodError(core.ErrorInvalidArguments, err.Error())
	}

	subs, err := h.repo.JMAP().PushSubscriptions().ListByUser(ctx, userID)
	if err != nil {
		return nil, core.NewMethodError(core.ErrorServerFail, err.Error())
	}

	var list []map[string]interface{}
	var notFound []string

	if a.IDs == nil {
		for _, sub := range subs {
			list = append(list, pushSubToJMAP(sub))
		}
	} else {
		subMap := make(map[string]*domain.PushSubscription)
		for _, s := range subs {
			subMap[string(s.ID)] = s
		}
		for _, id := range a.IDs {
			if sub, ok := subMap[id]; ok {
				list = append(list, pushSubToJMAP(sub))
			} else {
				notFound = append(notFound, id)
			}
		}
	}

	if list == nil {
		list = []map[string]interface{}{}
	}
	if notFound == nil {
		notFound = []string{}
	}

	return marshalJSON(map[string]interface{}{
		"list":     list,
		"notFound": notFound,
	})
}

// Set implements PushSubscription/set (no accountId, no ifInState/oldState/newState).
func (h *SubscriptionHandler) Set(ctx context.Context, userID domain.ID, args json.RawMessage) (json.RawMessage, *core.MethodError) {
	return marshalJSON(map[string]interface{}{})
}

func pushSubToJMAP(sub *domain.PushSubscription) map[string]interface{} {
	result := map[string]interface{}{
		"id":              string(sub.ID),
		"deviceClientId":  sub.DeviceClientID,
		"url":             sub.URL,
		"types":           sub.Types,
	}
	if sub.VerificationCode != "" {
		result["verificationCode"] = sub.VerificationCode
	}
	if sub.Expires != nil {
		result["expires"] = sub.Expires.Time.UTC().Format("2006-01-02T15:04:05Z")
	}
	if sub.KeysP256DH != "" {
		result["keys"] = map[string]string{
			"p256dh": sub.KeysP256DH,
			"auth":   sub.KeysAuth,
		}
	}
	return result
}

func marshalJSON(v interface{}) (json.RawMessage, *core.MethodError) {
	data, err := json.Marshal(v)
	if err != nil {
		return nil, core.NewMethodError(core.ErrorServerFail, fmt.Sprintf("marshal error: %v", err))
	}
	return data, nil
}
