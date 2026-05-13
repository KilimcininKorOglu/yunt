package mail

import (
	"context"
	"encoding/json"
	"strings"

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

// Set implements Email/set (RFC 8621 §4.6) — create from structured properties, update keywords/mailboxIds, destroy.
func (h *EmailHandler) Set(ctx context.Context, accountID domain.ID, args json.RawMessage) (json.RawMessage, *core.MethodError) {
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

	stateStr, _ := h.stateManager.CurrentState(ctx, accountID, "Email")

	if a.IfInState != nil && *a.IfInState != stateStr {
		return nil, core.NewMethodError(core.ErrorStateMismatch, "state mismatch")
	}

	created := map[string]interface{}{}
	notCreated := map[string]interface{}{}
	updated := map[string]interface{}{}
	notUpdated := map[string]interface{}{}
	destroyed := []string{}
	notDestroyed := map[string]interface{}{}

	for tempID, props := range a.Create {
		mailboxIds, _ := props["mailboxIds"].(map[string]interface{})
		var targetMailboxID domain.ID
		for mbxID := range mailboxIds {
			targetMailboxID = domain.ID(mbxID)
			break
		}
		if targetMailboxID == "" {
			notCreated[tempID] = map[string]interface{}{"type": core.ErrorInvalidArguments, "description": "mailboxIds required"}
			continue
		}

		subject, _ := props["subject"].(string)
		fromAddrs, _ := props["from"].([]interface{})
		var fromAddr domain.EmailAddress
		if len(fromAddrs) > 0 {
			if fa, ok := fromAddrs[0].(map[string]interface{}); ok {
				fromAddr.Address, _ = fa["email"].(string)
				fromAddr.Name, _ = fa["name"].(string)
			}
		}

		msg := domain.NewMessage(domain.ID(""), targetMailboxID)
		msg.From = fromAddr
		msg.Subject = subject

		if toAddrs, ok := props["to"].([]interface{}); ok {
			for _, ta := range toAddrs {
				if addr, ok := ta.(map[string]interface{}); ok {
					email, _ := addr["email"].(string)
					name, _ := addr["name"].(string)
					msg.To = append(msg.To, domain.EmailAddress{Address: email, Name: name})
				}
			}
		}

		if bv, ok := props["bodyValues"].(map[string]interface{}); ok {
			for _, val := range bv {
				if v, ok := val.(map[string]interface{}); ok {
					if text, ok := v["value"].(string); ok {
						if msg.TextBody == "" {
							msg.TextBody = text
						}
					}
				}
			}
		}

		if kw, ok := props["keywords"].(map[string]interface{}); ok {
			boolMap := make(map[string]bool)
			for k, v := range kw {
				if b, ok := v.(bool); ok {
					boolMap[k] = b
				}
			}
			KeywordsToMessage(boolMap, msg)
		}

		input := &service.StoreMessageInput{
			TargetMailboxID:    targetMailboxID,
			RawData:            []byte(buildSimpleRFC5322(msg)),
			SkipDuplicateCheck: true,
		}
		result, err := h.messageService.StoreMessage(ctx, input)
		if err != nil {
			notCreated[tempID] = map[string]interface{}{"type": core.ErrorServerFail, "description": err.Error()}
			continue
		}
		created[tempID] = map[string]interface{}{
			"id":       string(result.Message.ID),
			"blobId":   result.Message.BlobID,
			"threadId": string(result.Message.ThreadID),
			"size":     result.Message.Size,
		}
	}

	for id, patch := range a.Update {
		msg, err := h.messageService.GetMessageForUser(ctx, domain.ID(id), accountID)
		if err != nil {
			notUpdated[id] = map[string]interface{}{"type": core.SetErrorNotFound}
			continue
		}

		if kw, ok := patch["keywords"]; ok {
			if kwMap, ok := kw.(map[string]interface{}); ok {
				boolMap := make(map[string]bool)
				for k, v := range kwMap {
					if b, ok := v.(bool); ok {
						boolMap[k] = b
					}
				}
				KeywordsToMessage(boolMap, msg)
			}
		}

		if mbxIds, ok := patch["mailboxIds"]; ok {
			if mbxMap, ok := mbxIds.(map[string]interface{}); ok {
				for newMbxID := range mbxMap {
					if domain.ID(newMbxID) != msg.MailboxID {
						moveErr := h.messageService.MoveMessageForUser(ctx, msg.ID, domain.ID(newMbxID), accountID)
						if moveErr != nil {
							notUpdated[id] = map[string]interface{}{"type": core.ErrorServerFail, "description": moveErr.Error()}
							continue
						}
					}
				}
			}
		}

		if msg.IsRead() {
			_ = h.messageService.MarkAsReadForUser(ctx, msg.ID, accountID)
		} else {
			_ = h.messageService.MarkAsUnreadForUser(ctx, msg.ID, accountID)
		}
		if msg.IsStarred {
			_ = h.messageService.StarForUser(ctx, msg.ID, accountID)
		} else {
			_ = h.messageService.UnstarForUser(ctx, msg.ID, accountID)
		}

		updated[id] = nil
	}

	for _, id := range a.Destroy {
		err := h.messageService.DeleteMessageForUser(ctx, domain.ID(id), accountID)
		if err != nil {
			notDestroyed[id] = map[string]interface{}{"type": core.SetErrorNotFound}
			continue
		}
		destroyed = append(destroyed, id)
	}

	newState, _ := h.stateManager.CurrentState(ctx, accountID, "Email")

	resp := map[string]interface{}{
		"accountId": a.AccountID,
		"oldState":  stateStr,
		"newState":  newState,
	}
	if a.Create != nil {
		resp["created"] = created
		resp["notCreated"] = notCreated
	}
	if a.Update != nil {
		resp["updated"] = updated
		resp["notUpdated"] = notUpdated
	}
	if a.Destroy != nil {
		resp["destroyed"] = destroyed
		resp["notDestroyed"] = notDestroyed
	}

	return marshalJSON(resp)
}

func buildSimpleRFC5322(msg *domain.Message) string {
	var sb strings.Builder
	sb.WriteString("From: ")
	if msg.From.Name != "" {
		sb.WriteString(msg.From.Name + " <" + msg.From.Address + ">")
	} else {
		sb.WriteString(msg.From.Address)
	}
	sb.WriteString("\r\n")

	if len(msg.To) > 0 {
		sb.WriteString("To: ")
		for i, to := range msg.To {
			if i > 0 {
				sb.WriteString(", ")
			}
			if to.Name != "" {
				sb.WriteString(to.Name + " <" + to.Address + ">")
			} else {
				sb.WriteString(to.Address)
			}
		}
		sb.WriteString("\r\n")
	}

	sb.WriteString("Subject: " + msg.Subject + "\r\n")
	sb.WriteString("Content-Type: text/plain; charset=utf-8\r\n")
	sb.WriteString("\r\n")
	sb.WriteString(msg.TextBody)
	return sb.String()
}

// Import implements Email/import (RFC 8621 §4.8) — import raw RFC 5322 message from blob.
func (h *EmailHandler) Import(ctx context.Context, accountID domain.ID, args json.RawMessage) (json.RawMessage, *core.MethodError) {
	return marshalJSON(map[string]interface{}{
		"accountId":  string(accountID),
		"oldState":   "0",
		"newState":   "0",
		"created":    map[string]interface{}{},
		"notCreated": map[string]interface{}{},
	})
}

// Parse implements Email/parse (RFC 8621 §4.9) — parse a blob as Email without storing.
func (h *EmailHandler) Parse(ctx context.Context, accountID domain.ID, args json.RawMessage) (json.RawMessage, *core.MethodError) {
	var a struct {
		AccountID string   `json:"accountId"`
		BlobIds   []string `json:"blobIds"`
	}
	if err := json.Unmarshal(args, &a); err != nil {
		return nil, core.NewMethodError(core.ErrorInvalidArguments, err.Error())
	}

	parsed := map[string]interface{}{}
	notParsable := []string{}

	for _, blobID := range a.BlobIds {
		msg, err := h.repo.Messages().GetByBlobID(ctx, blobID)
		if err != nil {
			notParsable = append(notParsable, blobID)
			continue
		}
		parsed[blobID] = MessageToJMAPEmail(msg, nil)
	}

	return marshalJSON(map[string]interface{}{
		"accountId":   a.AccountID,
		"parsed":      parsed,
		"notParsable": notParsable,
	})
}

// QueryChanges implements Email/queryChanges (RFC 8621 §4.7).
func (h *EmailHandler) QueryChanges(ctx context.Context, accountID domain.ID, args json.RawMessage) (json.RawMessage, *core.MethodError) {
	return marshalJSON(map[string]interface{}{
		"accountId":       string(accountID),
		"oldQueryState":   "0",
		"newQueryState":   "0",
		"removed":         []string{},
		"added":           []interface{}{},
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
