package mail

import (
	"yunt/internal/domain"
)

// KeywordsFromMessage converts domain message flags to JMAP keywords map.
func KeywordsFromMessage(msg *domain.Message) map[string]bool {
	kw := make(map[string]bool)
	if msg.IsRead() {
		kw["$seen"] = true
	}
	if msg.IsStarred {
		kw["$flagged"] = true
	}
	if msg.IsDraft {
		kw["$draft"] = true
	}
	if msg.IsSpam {
		kw["$junk"] = true
	}
	if msg.IsDeleted {
		kw["$deleted"] = true
	}
	if msg.IsAnswered {
		kw["$answered"] = true
	}
	return kw
}

// KeywordsToMessage applies JMAP keywords to a domain message.
func KeywordsToMessage(keywords map[string]bool, msg *domain.Message) {
	if keywords["$seen"] {
		msg.Status = domain.MessageRead
	} else {
		msg.Status = domain.MessageUnread
	}
	msg.IsStarred = keywords["$flagged"]
	msg.IsDraft = keywords["$draft"]
	msg.IsSpam = keywords["$junk"]
	msg.IsDeleted = keywords["$deleted"]
	msg.IsAnswered = keywords["$answered"]
}

// MailboxIDsFromMessage returns JMAP mailboxIds map (single-valued for Yunt).
func MailboxIDsFromMessage(msg *domain.Message) map[string]bool {
	return map[string]bool{string(msg.MailboxID): true}
}

// EmailAddress converts domain.EmailAddress to JMAP EmailAddress format.
func emailAddrToJMAP(addr domain.EmailAddress) map[string]interface{} {
	result := map[string]interface{}{
		"email": addr.Address,
	}
	if addr.Name != "" {
		result["name"] = addr.Name
	}
	return result
}

// emailAddrsToJMAP converts a slice of EmailAddress to JMAP format.
func emailAddrsToJMAP(addrs []domain.EmailAddress) []map[string]interface{} {
	if len(addrs) == 0 {
		return nil
	}
	result := make([]map[string]interface{}, len(addrs))
	for i, addr := range addrs {
		result[i] = emailAddrToJMAP(addr)
	}
	return result
}

// MessageToJMAPEmail converts a domain.Message to a JMAP Email object.
// If properties is nil, all properties are included.
func MessageToJMAPEmail(msg *domain.Message, properties []string) map[string]interface{} {
	if properties == nil {
		return messageToFullJMAP(msg)
	}

	result := make(map[string]interface{})
	for _, prop := range properties {
		switch prop {
		case "id":
			result["id"] = string(msg.ID)
		case "blobId":
			result["blobId"] = msg.BlobID
		case "threadId":
			result["threadId"] = string(msg.ThreadID)
		case "mailboxIds":
			result["mailboxIds"] = MailboxIDsFromMessage(msg)
		case "keywords":
			result["keywords"] = KeywordsFromMessage(msg)
		case "size":
			result["size"] = msg.Size
		case "receivedAt":
			result["receivedAt"] = msg.ReceivedAt.Time.UTC().Format("2006-01-02T15:04:05Z")
		case "messageId":
			if msg.MessageID != "" {
				result["messageId"] = []string{msg.MessageID}
			}
		case "from":
			result["from"] = []map[string]interface{}{emailAddrToJMAP(msg.From)}
		case "to":
			result["to"] = emailAddrsToJMAP(msg.To)
		case "cc":
			result["cc"] = emailAddrsToJMAP(msg.Cc)
		case "bcc":
			result["bcc"] = emailAddrsToJMAP(msg.Bcc)
		case "replyTo":
			if msg.ReplyTo != nil {
				result["replyTo"] = []map[string]interface{}{emailAddrToJMAP(*msg.ReplyTo)}
			}
		case "subject":
			result["subject"] = msg.Subject
		case "sentAt":
			if msg.SentAt != nil {
				result["sentAt"] = msg.SentAt.Time.UTC().Format("2006-01-02T15:04:05Z")
			}
		case "hasAttachment":
			result["hasAttachment"] = msg.HasAttachments()
		case "preview":
			result["preview"] = msg.GetPreview(256)
		case "inReplyTo":
			if msg.InReplyTo != "" {
				result["inReplyTo"] = []string{msg.InReplyTo}
			}
		case "references":
			if len(msg.References) > 0 {
				result["references"] = msg.References
			}
		case "textBody":
			result["textBody"] = []map[string]interface{}{
				{"partId": "1", "type": "text/plain"},
			}
		case "htmlBody":
			if msg.HTMLBody != "" {
				result["htmlBody"] = []map[string]interface{}{
					{"partId": "2", "type": "text/html"},
				}
			}
		}
	}
	return result
}

func messageToFullJMAP(msg *domain.Message) map[string]interface{} {
	result := map[string]interface{}{
		"id":            string(msg.ID),
		"blobId":        msg.BlobID,
		"threadId":      string(msg.ThreadID),
		"mailboxIds":    MailboxIDsFromMessage(msg),
		"keywords":      KeywordsFromMessage(msg),
		"size":          msg.Size,
		"receivedAt":    msg.ReceivedAt.Time.UTC().Format("2006-01-02T15:04:05Z"),
		"from":          []map[string]interface{}{emailAddrToJMAP(msg.From)},
		"to":            emailAddrsToJMAP(msg.To),
		"subject":       msg.Subject,
		"hasAttachment": msg.HasAttachments(),
		"preview":       msg.GetPreview(256),
	}

	if msg.MessageID != "" {
		result["messageId"] = []string{msg.MessageID}
	}
	if len(msg.Cc) > 0 {
		result["cc"] = emailAddrsToJMAP(msg.Cc)
	}
	if msg.ReplyTo != nil {
		result["replyTo"] = []map[string]interface{}{emailAddrToJMAP(*msg.ReplyTo)}
	}
	if msg.SentAt != nil {
		result["sentAt"] = msg.SentAt.Time.UTC().Format("2006-01-02T15:04:05Z")
	}
	if msg.InReplyTo != "" {
		result["inReplyTo"] = []string{msg.InReplyTo}
	}
	if len(msg.References) > 0 {
		result["references"] = msg.References
	}

	return result
}

// MailboxToJMAP converts a domain.Mailbox to a JMAP Mailbox object.
func MailboxToJMAP(mbx *domain.Mailbox) map[string]interface{} {
	result := map[string]interface{}{
		"id":           string(mbx.ID),
		"name":         mbx.Name,
		"sortOrder":    mbx.SortOrder,
		"totalEmails":  mbx.MessageCount,
		"unreadEmails": mbx.UnreadCount,
		"totalThreads": mbx.MessageCount,
		"unreadThreads": mbx.UnreadCount,
		"isSubscribed": true,
		"myRights": map[string]bool{
			"mayReadItems":   true,
			"mayAddItems":    true,
			"mayRemoveItems": true,
			"maySetSeen":     true,
			"maySetKeywords": true,
			"mayCreateChild": true,
			"mayRename":      mbx.Type != domain.MailboxTypeSystem,
			"mayDelete":      mbx.Type != domain.MailboxTypeSystem,
			"maySubmit":      true,
		},
	}

	if mbx.Role != "" {
		result["role"] = mbx.Role
	} else {
		result["role"] = nil
	}

	return result
}
