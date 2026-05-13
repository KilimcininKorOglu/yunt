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
		AccountID                string                            `json:"accountId"`
		Create                   map[string]map[string]interface{} `json:"create"`
		Update                   map[string]map[string]interface{} `json:"update"`
		Destroy                  []string                          `json:"destroy"`
		OnDestroyRemoveContents  bool                              `json:"onDestroyRemoveContents"`
	}
	if err := json.Unmarshal(args, &a); err != nil {
		return nil, core.NewMethodError(core.ErrorInvalidArguments, err.Error())
	}

	stateStr, _ := h.stateManager.CurrentState(ctx, accountID, "AddressBook")

	created := map[string]interface{}{}
	notCreated := map[string]interface{}{}
	updated := map[string]interface{}{}
	notUpdated := map[string]interface{}{}
	destroyed := []string{}
	notDestroyed := map[string]interface{}{}

	for tempID, props := range a.Create {
		name, _ := props["name"].(string)
		if name == "" {
			notCreated[tempID] = map[string]interface{}{"type": "invalidProperties", "description": "name required"}
			continue
		}
		book := &domain.AddressBook{
			ID: domain.ID(fmt.Sprintf("ab-%d", time.Now().UnixNano())),
			UserID: accountID, Name: name, IsSubscribed: true,
			CreatedAt: domain.Now(), UpdatedAt: domain.Now(),
		}
		if desc, ok := props["description"].(string); ok {
			book.Description = desc
		}
		if err := h.repo.JMAP().AddressBooks().Create(ctx, book); err != nil {
			notCreated[tempID] = map[string]interface{}{"type": "serverFail", "description": err.Error()}
			continue
		}
		created[tempID] = map[string]interface{}{"id": string(book.ID)}
	}

	for id, patch := range a.Update {
		book, err := h.repo.JMAP().AddressBooks().GetByID(ctx, domain.ID(id))
		if err != nil {
			notUpdated[id] = map[string]interface{}{"type": "notFound"}
			continue
		}
		if name, ok := patch["name"].(string); ok {
			book.Name = name
		}
		if desc, ok := patch["description"].(string); ok {
			book.Description = desc
		}
		if err := h.repo.JMAP().AddressBooks().Update(ctx, book); err != nil {
			notUpdated[id] = map[string]interface{}{"type": "serverFail", "description": err.Error()}
			continue
		}
		updated[id] = nil
	}

	for _, id := range a.Destroy {
		if a.OnDestroyRemoveContents {
			_, _ = h.repo.JMAP().ContactCards().DeleteByAddressBook(ctx, domain.ID(id))
		}
		if err := h.repo.JMAP().AddressBooks().Delete(ctx, domain.ID(id)); err != nil {
			notDestroyed[id] = map[string]interface{}{"type": "notFound"}
			continue
		}
		destroyed = append(destroyed, id)
	}

	newState, _ := h.stateManager.CurrentState(ctx, accountID, "AddressBook")
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
