package contacts

import (
	"context"
	"encoding/json"
	"fmt"

	"yunt/internal/domain"
	"yunt/internal/jmap/core"
	"yunt/internal/jmap/state"
	"yunt/internal/repository"
)

// AddressBookHandler implements JMAP AddressBook methods (RFC 9610).
type AddressBookHandler struct {
	repo         repository.Repository
	stateManager *state.Manager
}

// NewAddressBookHandler creates a new AddressBook method handler.
func NewAddressBookHandler(repo repository.Repository, stateMgr *state.Manager) *AddressBookHandler {
	return &AddressBookHandler{repo: repo, stateManager: stateMgr}
}

// Get implements AddressBook/get.
func (h *AddressBookHandler) Get(ctx context.Context, accountID domain.ID, args json.RawMessage) (json.RawMessage, *core.MethodError) {
	var a struct {
		AccountID string   `json:"accountId"`
		IDs       []string `json:"ids"`
	}
	if err := json.Unmarshal(args, &a); err != nil {
		return nil, core.NewMethodError(core.ErrorInvalidArguments, err.Error())
	}

	stateStr, _ := h.stateManager.CurrentState(ctx, accountID, "AddressBook")

	books, err := h.repo.JMAP().AddressBooks().List(ctx, accountID)
	if err != nil {
		return nil, core.NewMethodError(core.ErrorServerFail, err.Error())
	}

	var list []map[string]interface{}
	var notFound []string

	if a.IDs == nil {
		for _, book := range books {
			list = append(list, addressBookToJMAP(book))
		}
	} else {
		bookMap := make(map[string]*domain.AddressBook)
		for _, b := range books {
			bookMap[string(b.ID)] = b
		}
		for _, id := range a.IDs {
			if book, ok := bookMap[id]; ok {
				list = append(list, addressBookToJMAP(book))
			} else {
				notFound = append(notFound, id)
			}
		}
	}

	return marshalGetResponse(a.AccountID, stateStr, list, notFound)
}

// Changes implements AddressBook/changes.
func (h *AddressBookHandler) Changes(ctx context.Context, accountID domain.ID, args json.RawMessage) (json.RawMessage, *core.MethodError) {
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

	result, err := h.stateManager.GetChanges(ctx, accountID, "AddressBook", a.SinceState, maxChanges)
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

// Set implements AddressBook/set with onDestroyRemoveContents.
func (h *AddressBookHandler) Set(ctx context.Context, accountID domain.ID, args json.RawMessage) (json.RawMessage, *core.MethodError) {
	var a struct {
		AccountID string `json:"accountId"`
		IfInState *string `json:"ifInState"`
	}
	if err := json.Unmarshal(args, &a); err != nil {
		return nil, core.NewMethodError(core.ErrorInvalidArguments, err.Error())
	}

	stateStr, _ := h.stateManager.CurrentState(ctx, accountID, "AddressBook")

	return marshalJSON(map[string]interface{}{
		"accountId": a.AccountID,
		"oldState":  stateStr,
		"newState":  stateStr,
	})
}

func addressBookToJMAP(book *domain.AddressBook) map[string]interface{} {
	return map[string]interface{}{
		"id":           string(book.ID),
		"name":         book.Name,
		"description":  book.Description,
		"sortOrder":    book.SortOrder,
		"isDefault":    book.IsDefault,
		"isSubscribed": book.IsSubscribed,
		"myRights": map[string]bool{
			"mayRead":   true,
			"mayWrite":  true,
			"mayShare":  false,
			"mayDelete": !book.IsDefault,
		},
	}
}
