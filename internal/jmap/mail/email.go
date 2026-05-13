package mail

import (
	"context"
	"encoding/json"

	"yunt/internal/domain"
	"yunt/internal/jmap/core"
	"yunt/internal/jmap/state"
	"yunt/internal/repository"
	"yunt/internal/service"
)

// EmailHandler implements JMAP Email methods (RFC 8621 §4).
type EmailHandler struct {
	messageService *service.MessageService
	stateManager   *state.Manager
	repo           repository.Repository
}

// NewEmailHandler creates a new Email method handler.
func NewEmailHandler(msgSvc *service.MessageService, stateMgr *state.Manager, repo repository.Repository) *EmailHandler {
	return &EmailHandler{
		messageService: msgSvc,
		stateManager:   stateMgr,
		repo:           repo,
	}
}

type emailGetArgs struct {
	AccountID            string   `json:"accountId"`
	IDs                  []string `json:"ids"`
	Properties           []string `json:"properties"`
	FetchTextBodyValues  bool     `json:"fetchTextBodyValues"`
	FetchHTMLBodyValues  bool     `json:"fetchHTMLBodyValues"`
	FetchAllBodyValues   bool     `json:"fetchAllBodyValues"`
	MaxBodyValueBytes    int      `json:"maxBodyValueBytes"`
}

// Get implements Email/get (RFC 8621 §4.1).
func (h *EmailHandler) Get(ctx context.Context, accountID domain.ID, args json.RawMessage) (json.RawMessage, *core.MethodError) {
	var a emailGetArgs
	if err := json.Unmarshal(args, &a); err != nil {
		return nil, core.NewMethodError(core.ErrorInvalidArguments, err.Error())
	}

	stateStr, _ := h.stateManager.CurrentState(ctx, accountID, "Email")

	if a.IDs == nil {
		filter := &repository.MessageFilter{}
		opts := &repository.ListOptions{
			Pagination: &repository.PaginationOptions{Page: 1, PerPage: 100},
		}
		result, err := h.messageService.ListMessagesForUser(ctx, accountID, filter, opts)
		if err != nil {
			return nil, core.NewMethodError(core.ErrorServerFail, err.Error())
		}

		list := make([]map[string]interface{}, len(result.Items))
		for i, msg := range result.Items {
			list[i] = MessageToJMAPEmail(msg, a.Properties)
		}

		return marshalGetResponse(a.AccountID, stateStr, list, nil)
	}

	var list []map[string]interface{}
	var notFound []string

	for _, id := range a.IDs {
		msg, err := h.messageService.GetMessageForUser(ctx, domain.ID(id), accountID)
		if err != nil {
			notFound = append(notFound, id)
			continue
		}
		list = append(list, MessageToJMAPEmail(msg, a.Properties))
	}

	return marshalGetResponse(a.AccountID, stateStr, list, notFound)
}

type emailQueryArgs struct {
	AccountID       string                 `json:"accountId"`
	Filter          map[string]interface{} `json:"filter"`
	Sort            []emailSort            `json:"sort"`
	Position        int                    `json:"position"`
	Limit           *int                   `json:"limit"`
	CalculateTotal  bool                   `json:"calculateTotal"`
	CollapseThreads bool                   `json:"collapseThreads"`
}

type emailSort struct {
	Property    string `json:"property"`
	IsAscending *bool  `json:"isAscending"`
}

// Query implements Email/query (RFC 8621 §4.5).
func (h *EmailHandler) Query(ctx context.Context, accountID domain.ID, args json.RawMessage) (json.RawMessage, *core.MethodError) {
	var a emailQueryArgs
	if err := json.Unmarshal(args, &a); err != nil {
		return nil, core.NewMethodError(core.ErrorInvalidArguments, err.Error())
	}

	filter := &repository.MessageFilter{}

	if mbxID, ok := a.Filter["inMailbox"].(string); ok {
		id := domain.ID(mbxID)
		filter.MailboxID = &id
	}
	if hasAtt, ok := a.Filter["hasAttachment"].(bool); ok {
		filter.HasAttachments = &hasAtt
	}
	if text, ok := a.Filter["text"].(string); ok {
		filter.Search = text
	}
	if from, ok := a.Filter["from"].(string); ok {
		filter.FromAddressContains = from
	}
	if to, ok := a.Filter["to"].(string); ok {
		filter.ToAddressContains = to
	}
	if subj, ok := a.Filter["subject"].(string); ok {
		filter.SubjectContains = subj
	}

	if kw, ok := a.Filter["hasKeyword"].(string); ok {
		switch kw {
		case "$seen":
			read := domain.MessageRead
			filter.Status = &read
		case "$flagged":
			t := true
			filter.IsStarred = &t
		case "$junk":
			t := true
			filter.IsSpam = &t
		case "$draft":
			t := true
			filter.IsDraft = &t
		}
	}

	perPage := 50
	if a.Limit != nil && *a.Limit > 0 {
		perPage = *a.Limit
	}

	page := 1
	if a.Position > 0 && perPage > 0 {
		page = (a.Position / perPage) + 1
	}

	sortOpts := &repository.SortOptions{Field: "received_at", Order: domain.SortDesc}
	if len(a.Sort) > 0 {
		field := "received_at"
		switch a.Sort[0].Property {
		case "receivedAt":
			field = "received_at"
		case "sentAt":
			field = "sent_at"
		case "size":
			field = "size"
		case "from":
			field = "from_address"
		case "subject":
			field = "subject"
		}
		order := domain.SortDesc
		if a.Sort[0].IsAscending != nil && *a.Sort[0].IsAscending {
			order = domain.SortAsc
		}
		sortOpts = &repository.SortOptions{Field: field, Order: order}
	}

	opts := &repository.ListOptions{
		Sort:       sortOpts,
		Pagination: &repository.PaginationOptions{Page: page, PerPage: perPage},
	}

	result, err := h.messageService.ListMessagesForUser(ctx, accountID, filter, opts)
	if err != nil {
		return nil, core.NewMethodError(core.ErrorServerFail, err.Error())
	}

	stateStr, _ := h.stateManager.CurrentState(ctx, accountID, "Email")

	ids := make([]string, len(result.Items))
	for i, msg := range result.Items {
		ids[i] = string(msg.ID)
	}

	resp := map[string]interface{}{
		"accountId":          a.AccountID,
		"queryState":         stateStr,
		"canCalculateChanges": false,
		"position":           a.Position,
		"ids":                ids,
	}
	if a.CalculateTotal {
		resp["total"] = result.Total
	}

	return marshalJSON(resp)
}

// Changes implements Email/changes (RFC 8621 §4.2).
func (h *EmailHandler) Changes(ctx context.Context, accountID domain.ID, args json.RawMessage) (json.RawMessage, *core.MethodError) {
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

	result, err := h.stateManager.GetChanges(ctx, accountID, "Email", a.SinceState, maxChanges)
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

func marshalJSON(v interface{}) (json.RawMessage, *core.MethodError) {
	data, err := json.Marshal(v)
	if err != nil {
		return nil, core.NewMethodError(core.ErrorServerFail, err.Error())
	}
	return data, nil
}

func marshalGetResponse(accountID, stateStr string, list []map[string]interface{}, notFound []string) (json.RawMessage, *core.MethodError) {
	if list == nil {
		list = []map[string]interface{}{}
	}
	if notFound == nil {
		notFound = []string{}
	}
	resp := map[string]interface{}{
		"accountId": accountID,
		"state":     stateStr,
		"list":      list,
		"notFound":  notFound,
	}
	data, err := json.Marshal(resp)
	if err != nil {
		return nil, core.NewMethodError(core.ErrorServerFail, err.Error())
	}
	return data, nil
}

func formatState(n int64) string {
	return json.Number(json.Number(string(rune(n + '0')))).String()
}

func idSlice(ids []domain.ID) []string {
	if ids == nil {
		return []string{}
	}
	result := make([]string, len(ids))
	for i, id := range ids {
		result[i] = string(id)
	}
	return result
}
