package mail

import (
	"context"
	"encoding/json"

	"yunt/internal/domain"
	"yunt/internal/jmap/core"
	"yunt/internal/repository"
	"yunt/internal/service"
)

// SnippetHandler implements JMAP SearchSnippet methods (RFC 8621 §5).
type SnippetHandler struct {
	messageService *service.MessageService
	repo           repository.Repository
}

// NewSnippetHandler creates a new SearchSnippet method handler.
func NewSnippetHandler(msgSvc *service.MessageService, repo repository.Repository) *SnippetHandler {
	return &SnippetHandler{messageService: msgSvc, repo: repo}
}

// Get implements SearchSnippet/get (RFC 8621 §5.1).
func (h *SnippetHandler) Get(ctx context.Context, accountID domain.ID, args json.RawMessage) (json.RawMessage, *core.MethodError) {
	var a struct {
		AccountID string                 `json:"accountId"`
		Filter    map[string]interface{} `json:"filter"`
		EmailIds  []string               `json:"emailIds"`
	}
	if err := json.Unmarshal(args, &a); err != nil {
		return nil, core.NewMethodError(core.ErrorInvalidArguments, err.Error())
	}

	var list []map[string]interface{}
	var notFound []string

	for _, id := range a.EmailIds {
		msg, err := h.messageService.GetMessageForUser(ctx, domain.ID(id), accountID)
		if err != nil {
			notFound = append(notFound, id)
			continue
		}

		snippet := map[string]interface{}{
			"emailId": id,
			"subject": nil,
			"preview": nil,
		}

		searchTerm, _ := a.Filter["text"].(string)
		if searchTerm != "" {
			snippet["preview"] = msg.GetPreview(256)
		}

		list = append(list, snippet)
	}

	if list == nil {
		list = []map[string]interface{}{}
	}
	if notFound == nil {
		notFound = []string{}
	}

	return marshalJSON(map[string]interface{}{
		"accountId": a.AccountID,
		"list":      list,
		"notFound":  notFound,
	})
}
