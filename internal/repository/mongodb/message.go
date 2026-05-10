package mongodb

import (
	"context"
	"fmt"
	"regexp"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"yunt/internal/domain"
	"yunt/internal/repository"
)

// MessageRepository implements the repository.MessageRepository interface for MongoDB.
type MessageRepository struct {
	repo *Repository
}

// emailAddressDocument is the MongoDB document representation of an email address.
type emailAddressDocument struct {
	Name    string `bson:"name,omitempty"`
	Address string `bson:"address"`
}

// messageDocument is the MongoDB document representation of a message.
type messageDocument struct {
	ID              string                 `bson:"_id"`
	MailboxID       string                 `bson:"mailboxId"`
	MessageID       string                 `bson:"messageId,omitempty"`
	From            emailAddressDocument   `bson:"from"`
	To              []emailAddressDocument `bson:"to"`
	Cc              []emailAddressDocument `bson:"cc,omitempty"`
	Bcc             []emailAddressDocument `bson:"bcc,omitempty"`
	ReplyTo         *emailAddressDocument  `bson:"replyTo,omitempty"`
	Subject         string                 `bson:"subject"`
	TextBody        string                 `bson:"textBody,omitempty"`
	HTMLBody        string                 `bson:"htmlBody,omitempty"`
	RawBody         []byte                 `bson:"rawBody,omitempty"`
	Headers         map[string]string      `bson:"headers,omitempty"`
	ContentType     string                 `bson:"contentType"`
	Size            int64                  `bson:"size"`
	AttachmentCount int                    `bson:"attachmentCount"`
	Status          string                 `bson:"status"`
	IsStarred       bool                   `bson:"isStarred"`
	IsSpam          bool                   `bson:"isSpam"`
	IsDeleted       bool                   `bson:"isDeleted"`
	InReplyTo       string                 `bson:"inReplyTo,omitempty"`
	References      []string               `bson:"references,omitempty"`
	ReceivedAt      time.Time              `bson:"receivedAt"`
	SentAt          *time.Time             `bson:"sentAt,omitempty"`
	CreatedAt       time.Time              `bson:"createdAt"`
	UpdatedAt       time.Time              `bson:"updatedAt"`
}

// NewMessageRepository creates a new MongoDB message repository.
func NewMessageRepository(repo *Repository) *MessageRepository {
	return &MessageRepository{repo: repo}
}

// collection returns the messages collection.
func (m *MessageRepository) collection() *mongo.Collection {
	return m.repo.collection(CollectionMessages)
}

// toDocument converts a domain.Message to a MongoDB document.
func (m *MessageRepository) toDocument(msg *domain.Message) *messageDocument {
	doc := &messageDocument{
		ID:              string(msg.ID),
		MailboxID:       string(msg.MailboxID),
		MessageID:       msg.MessageID,
		From:            emailAddressDocument{Name: msg.From.Name, Address: msg.From.Address},
		Subject:         msg.Subject,
		TextBody:        msg.TextBody,
		HTMLBody:        msg.HTMLBody,
		RawBody:         msg.RawBody,
		Headers:         msg.Headers,
		ContentType:     string(msg.ContentType),
		Size:            msg.Size,
		AttachmentCount: msg.AttachmentCount,
		Status:          string(msg.Status),
		IsStarred:       msg.IsStarred,
		IsSpam:          msg.IsSpam,
		IsDeleted:       msg.IsDeleted,
		InReplyTo:       msg.InReplyTo,
		References:      msg.References,
		ReceivedAt:      msg.ReceivedAt.Time,
		CreatedAt:       msg.CreatedAt.Time,
		UpdatedAt:       msg.UpdatedAt.Time,
	}

	// Convert To recipients
	doc.To = make([]emailAddressDocument, len(msg.To))
	for i, addr := range msg.To {
		doc.To[i] = emailAddressDocument{Name: addr.Name, Address: addr.Address}
	}

	// Convert Cc recipients
	if len(msg.Cc) > 0 {
		doc.Cc = make([]emailAddressDocument, len(msg.Cc))
		for i, addr := range msg.Cc {
			doc.Cc[i] = emailAddressDocument{Name: addr.Name, Address: addr.Address}
		}
	}

	// Convert Bcc recipients
	if len(msg.Bcc) > 0 {
		doc.Bcc = make([]emailAddressDocument, len(msg.Bcc))
		for i, addr := range msg.Bcc {
			doc.Bcc[i] = emailAddressDocument{Name: addr.Name, Address: addr.Address}
		}
	}

	// Convert ReplyTo
	if msg.ReplyTo != nil {
		doc.ReplyTo = &emailAddressDocument{Name: msg.ReplyTo.Name, Address: msg.ReplyTo.Address}
	}

	if msg.SentAt != nil {
		t := msg.SentAt.Time
		doc.SentAt = &t
	}

	return doc
}

// toDomain converts a MongoDB document to a domain.Message.
func (m *MessageRepository) toDomain(doc *messageDocument) *domain.Message {
	msg := &domain.Message{
		ID:              domain.ID(doc.ID),
		MailboxID:       domain.ID(doc.MailboxID),
		MessageID:       doc.MessageID,
		From:            domain.EmailAddress{Name: doc.From.Name, Address: doc.From.Address},
		Subject:         doc.Subject,
		TextBody:        doc.TextBody,
		HTMLBody:        doc.HTMLBody,
		RawBody:         doc.RawBody,
		Headers:         doc.Headers,
		ContentType:     domain.ContentType(doc.ContentType),
		Size:            doc.Size,
		AttachmentCount: doc.AttachmentCount,
		Status:          domain.MessageStatus(doc.Status),
		IsStarred:       doc.IsStarred,
		IsSpam:          doc.IsSpam,
		IsDeleted:       doc.IsDeleted,
		InReplyTo:       doc.InReplyTo,
		References:      doc.References,
		ReceivedAt:      domain.Timestamp{Time: doc.ReceivedAt},
		CreatedAt:       domain.Timestamp{Time: doc.CreatedAt},
		UpdatedAt:       domain.Timestamp{Time: doc.UpdatedAt},
	}

	// Convert To recipients
	msg.To = make([]domain.EmailAddress, len(doc.To))
	for i, addr := range doc.To {
		msg.To[i] = domain.EmailAddress{Name: addr.Name, Address: addr.Address}
	}

	// Convert Cc recipients
	if len(doc.Cc) > 0 {
		msg.Cc = make([]domain.EmailAddress, len(doc.Cc))
		for i, addr := range doc.Cc {
			msg.Cc[i] = domain.EmailAddress{Name: addr.Name, Address: addr.Address}
		}
	}

	// Convert Bcc recipients
	if len(doc.Bcc) > 0 {
		msg.Bcc = make([]domain.EmailAddress, len(doc.Bcc))
		for i, addr := range doc.Bcc {
			msg.Bcc[i] = domain.EmailAddress{Name: addr.Name, Address: addr.Address}
		}
	}

	// Convert ReplyTo
	if doc.ReplyTo != nil {
		msg.ReplyTo = &domain.EmailAddress{Name: doc.ReplyTo.Name, Address: doc.ReplyTo.Address}
	}

	if doc.SentAt != nil {
		ts := domain.Timestamp{Time: *doc.SentAt}
		msg.SentAt = &ts
	}

	return msg
}

// toSummary converts a MongoDB document to a domain.MessageSummary.
func (m *MessageRepository) toSummary(doc *messageDocument) *domain.MessageSummary {
	return &domain.MessageSummary{
		ID:             domain.ID(doc.ID),
		MailboxID:      domain.ID(doc.MailboxID),
		From:           domain.EmailAddress{Name: doc.From.Name, Address: doc.From.Address},
		Subject:        doc.Subject,
		Preview:        getPreview(doc.TextBody, doc.HTMLBody, 100),
		Status:         domain.MessageStatus(doc.Status),
		IsStarred:      doc.IsStarred,
		HasAttachments: doc.AttachmentCount > 0,
		ReceivedAt:     domain.Timestamp{Time: doc.ReceivedAt},
	}
}

// getPreview returns a preview of the message body.
func getPreview(textBody, htmlBody string, maxLength int) string {
	body := textBody
	if body == "" {
		// Simple HTML stripping
		body = stripHTMLTags(htmlBody)
	}

	if len(body) <= maxLength {
		return body
	}
	return body[:maxLength] + "..."
}

// stripHTMLTags removes HTML tags from a string.
func stripHTMLTags(s string) string {
	var result []rune
	inTag := false
	for _, r := range s {
		switch {
		case r == '<':
			inTag = true
		case r == '>':
			inTag = false
		case !inTag:
			result = append(result, r)
		}
	}
	return string(result)
}

// GetByID retrieves a message by its unique identifier.
func (m *MessageRepository) GetByID(ctx context.Context, id domain.ID) (*domain.Message, error) {
	ctx = m.repo.getSessionContext(ctx)

	filter := bson.M{"_id": string(id)}

	var doc messageDocument
	if err := m.collection().FindOne(ctx, filter).Decode(&doc); err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, domain.NewNotFoundError("message", string(id))
		}
		return nil, fmt.Errorf("failed to get message by ID: %w", err)
	}

	return m.toDomain(&doc), nil
}

// GetByMessageID retrieves a message by its email Message-ID header.
func (m *MessageRepository) GetByMessageID(ctx context.Context, messageID string) (*domain.Message, error) {
	ctx = m.repo.getSessionContext(ctx)

	filter := bson.M{"messageId": messageID}

	var doc messageDocument
	if err := m.collection().FindOne(ctx, filter).Decode(&doc); err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, domain.NewNotFoundError("message", messageID)
		}
		return nil, fmt.Errorf("failed to get message by Message-ID: %w", err)
	}

	return m.toDomain(&doc), nil
}

// GetWithAttachments retrieves a message with its attachments loaded.
func (m *MessageRepository) GetWithAttachments(ctx context.Context, id domain.ID) (*domain.Message, []*domain.Attachment, error) {
	msg, err := m.GetByID(ctx, id)
	if err != nil {
		return nil, nil, err
	}

	attachments, err := m.repo.attachments.ListByMessage(ctx, id)
	if err != nil {
		return nil, nil, err
	}

	return msg, attachments, nil
}

// List retrieves messages with optional filtering, sorting, and pagination.
func (m *MessageRepository) List(ctx context.Context, filter *repository.MessageFilter, opts *repository.ListOptions) (*repository.ListResult[*domain.Message], error) {
	ctx = m.repo.getSessionContext(ctx)

	mongoFilter := m.buildFilter(filter)
	findOpts := m.buildFindOptions(opts)

	total, err := m.collection().CountDocuments(ctx, mongoFilter)
	if err != nil {
		return nil, fmt.Errorf("failed to count messages: %w", err)
	}

	cursor, err := m.collection().Find(ctx, mongoFilter, findOpts)
	if err != nil {
		return nil, fmt.Errorf("failed to list messages: %w", err)
	}
	defer cursor.Close(ctx)

	var docs []messageDocument
	if err := cursor.All(ctx, &docs); err != nil {
		return nil, fmt.Errorf("failed to decode messages: %w", err)
	}

	messages := make([]*domain.Message, len(docs))
	for i, doc := range docs {
		messages[i] = m.toDomain(&doc)
	}

	result := &repository.ListResult[*domain.Message]{
		Items: messages,
		Total: total,
	}

	if opts != nil && opts.Pagination != nil {
		result.Pagination = &domain.Pagination{
			Page:    opts.Pagination.Page,
			PerPage: opts.Pagination.PerPage,
			Total:   total,
		}
		result.HasMore = opts.Pagination.Page < result.Pagination.TotalPages()
	}

	return result, nil
}

// ListByMailbox retrieves all messages in a specific mailbox.
func (m *MessageRepository) ListByMailbox(ctx context.Context, mailboxID domain.ID, opts *repository.ListOptions) (*repository.ListResult[*domain.Message], error) {
	id := mailboxID
	filter := &repository.MessageFilter{MailboxID: &id}
	return m.List(ctx, filter, opts)
}

// ListSummaries retrieves message summaries for faster list rendering.
func (m *MessageRepository) ListSummaries(ctx context.Context, filter *repository.MessageFilter, opts *repository.ListOptions) (*repository.ListResult[*domain.MessageSummary], error) {
	ctx = m.repo.getSessionContext(ctx)

	mongoFilter := m.buildFilter(filter)
	findOpts := m.buildFindOptions(opts)

	// Only select fields needed for summary
	findOpts.SetProjection(bson.M{
		"_id":             1,
		"mailboxId":       1,
		"from":            1,
		"subject":         1,
		"textBody":        1,
		"htmlBody":        1,
		"status":          1,
		"isStarred":       1,
		"attachmentCount": 1,
		"receivedAt":      1,
	})

	total, err := m.collection().CountDocuments(ctx, mongoFilter)
	if err != nil {
		return nil, fmt.Errorf("failed to count messages: %w", err)
	}

	cursor, err := m.collection().Find(ctx, mongoFilter, findOpts)
	if err != nil {
		return nil, fmt.Errorf("failed to list message summaries: %w", err)
	}
	defer cursor.Close(ctx)

	var docs []messageDocument
	if err := cursor.All(ctx, &docs); err != nil {
		return nil, fmt.Errorf("failed to decode messages: %w", err)
	}

	summaries := make([]*domain.MessageSummary, len(docs))
	for i, doc := range docs {
		summaries[i] = m.toSummary(&doc)
	}

	result := &repository.ListResult[*domain.MessageSummary]{
		Items: summaries,
		Total: total,
	}

	if opts != nil && opts.Pagination != nil {
		result.Pagination = &domain.Pagination{
			Page:    opts.Pagination.Page,
			PerPage: opts.Pagination.PerPage,
			Total:   total,
		}
		result.HasMore = opts.Pagination.Page < result.Pagination.TotalPages()
	}

	return result, nil
}

// buildFilter builds the MongoDB filter from repository.MessageFilter.
func (m *MessageRepository) buildFilter(filter *repository.MessageFilter) bson.M {
	f := bson.M{}

	if filter == nil {
		return f
	}

	if len(filter.IDs) > 0 {
		ids := make([]string, len(filter.IDs))
		for i, id := range filter.IDs {
			ids[i] = string(id)
		}
		f["_id"] = bson.M{"$in": ids}
	}

	if filter.MailboxID != nil {
		f["mailboxId"] = string(*filter.MailboxID)
	}

	if len(filter.MailboxIDs) > 0 {
		mailboxIDs := make([]string, len(filter.MailboxIDs))
		for i, id := range filter.MailboxIDs {
			mailboxIDs[i] = string(id)
		}
		f["mailboxId"] = bson.M{"$in": mailboxIDs}
	}

	if filter.Status != nil {
		f["status"] = string(*filter.Status)
	}

	if filter.IsStarred != nil {
		f["isStarred"] = *filter.IsStarred
	}

	if filter.IsSpam != nil {
		f["isSpam"] = *filter.IsSpam
	}

	if filter.HasAttachments != nil {
		if *filter.HasAttachments {
			f["attachmentCount"] = bson.M{"$gt": 0}
		} else {
			f["attachmentCount"] = 0
		}
	}

	if filter.FromAddress != "" {
		f["from.address"] = bson.M{"$regex": "^" + regexp.QuoteMeta(filter.FromAddress) + "$", "$options": "i"}
	}

	if filter.FromAddressContains != "" {
		f["from.address"] = bson.M{"$regex": regexp.QuoteMeta(filter.FromAddressContains), "$options": "i"}
	}

	if filter.ToAddress != "" {
		f["to.address"] = bson.M{"$regex": "^" + regexp.QuoteMeta(filter.ToAddress) + "$", "$options": "i"}
	}

	if filter.ToAddressContains != "" {
		f["to.address"] = bson.M{"$regex": regexp.QuoteMeta(filter.ToAddressContains), "$options": "i"}
	}

	if filter.Subject != "" {
		f["subject"] = filter.Subject
	}

	if filter.SubjectContains != "" {
		f["subject"] = bson.M{"$regex": regexp.QuoteMeta(filter.SubjectContains), "$options": "i"}
	}

	if filter.BodyContains != "" {
		f["$or"] = bson.A{
			bson.M{"textBody": bson.M{"$regex": regexp.QuoteMeta(filter.BodyContains), "$options": "i"}},
			bson.M{"htmlBody": bson.M{"$regex": regexp.QuoteMeta(filter.BodyContains), "$options": "i"}},
		}
	}

	if filter.Search != "" {
		f["$text"] = bson.M{"$search": filter.Search}
	}

	if filter.MessageID != "" {
		f["messageId"] = filter.MessageID
	}

	if filter.InReplyTo != "" {
		f["inReplyTo"] = filter.InReplyTo
	}

	if filter.ReceivedAfter != nil {
		f["receivedAt"] = bson.M{"$gt": filter.ReceivedAfter.Time}
	}

	if filter.ReceivedBefore != nil {
		if _, exists := f["receivedAt"]; exists {
			f["receivedAt"].(bson.M)["$lt"] = filter.ReceivedBefore.Time
		} else {
			f["receivedAt"] = bson.M{"$lt": filter.ReceivedBefore.Time}
		}
	}

	if filter.SentAfter != nil {
		f["sentAt"] = bson.M{"$gt": filter.SentAfter.Time}
	}

	if filter.SentBefore != nil {
		if _, exists := f["sentAt"]; exists {
			f["sentAt"].(bson.M)["$lt"] = filter.SentBefore.Time
		} else {
			f["sentAt"] = bson.M{"$lt": filter.SentBefore.Time}
		}
	}

	if filter.MinSize != nil {
		f["size"] = bson.M{"$gte": *filter.MinSize}
	}

	if filter.MaxSize != nil {
		if _, exists := f["size"]; exists {
			f["size"].(bson.M)["$lte"] = *filter.MaxSize
		} else {
			f["size"] = bson.M{"$lte": *filter.MaxSize}
		}
	}

	if filter.ContentType != nil {
		f["contentType"] = string(*filter.ContentType)
	}

	if filter.ExcludeSpam {
		f["isSpam"] = false
	}

	return f
}

// buildFindOptions builds MongoDB find options from repository.ListOptions.
func (m *MessageRepository) buildFindOptions(opts *repository.ListOptions) *options.FindOptions {
	findOpts := options.Find()

	if opts == nil {
		findOpts.SetSort(bson.D{{Key: "receivedAt", Value: -1}})
		return findOpts
	}

	if opts.Sort != nil {
		sortOrder := 1
		if opts.Sort.Order == domain.SortDesc {
			sortOrder = -1
		}
		field := m.mapSortField(opts.Sort.Field)
		findOpts.SetSort(bson.D{{Key: field, Value: sortOrder}})
	} else {
		findOpts.SetSort(bson.D{{Key: "receivedAt", Value: -1}})
	}

	if opts.Pagination != nil {
		opts.Pagination.Normalize()
		findOpts.SetSkip(int64(opts.Pagination.Offset()))
		findOpts.SetLimit(int64(opts.Pagination.Limit()))
	}

	return findOpts
}

// mapSortField maps repository sort field to MongoDB field.
func (m *MessageRepository) mapSortField(field string) string {
	switch field {
	case "receivedAt":
		return "receivedAt"
	case "sentAt":
		return "sentAt"
	case "subject":
		return "subject"
	case "from":
		return "from.address"
	case "size":
		return "size"
	case "status":
		return "status"
	case "createdAt":
		return "createdAt"
	default:
		return "receivedAt"
	}
}

// Create creates a new message.
func (m *MessageRepository) Create(ctx context.Context, msg *domain.Message) error {
	ctx = m.repo.getSessionContext(ctx)

	doc := m.toDocument(msg)
	_, err := m.collection().InsertOne(ctx, doc)
	if err != nil {
		if mongo.IsDuplicateKeyError(err) {
			return domain.NewAlreadyExistsError("message", "id", string(msg.ID))
		}
		return fmt.Errorf("failed to create message: %w", err)
	}

	// Update mailbox stats
	if err := m.repo.mailboxes.IncrementMessageCount(ctx, msg.MailboxID, msg.Size); err != nil {
		return fmt.Errorf("failed to update mailbox stats: %w", err)
	}

	return nil
}

// Update updates an existing message.
func (m *MessageRepository) Update(ctx context.Context, msg *domain.Message) error {
	ctx = m.repo.getSessionContext(ctx)

	doc := m.toDocument(msg)
	doc.UpdatedAt = time.Now().UTC()

	filter := bson.M{"_id": string(msg.ID)}
	update := bson.M{"$set": doc}

	result, err := m.collection().UpdateOne(ctx, filter, update)
	if err != nil {
		return fmt.Errorf("failed to update message: %w", err)
	}

	if result.MatchedCount == 0 {
		return domain.NewNotFoundError("message", string(msg.ID))
	}

	return nil
}

// Delete permanently removes a message by its ID.
func (m *MessageRepository) Delete(ctx context.Context, id domain.ID) error {
	ctx = m.repo.getSessionContext(ctx)

	// Get the message first to update mailbox stats
	msg, err := m.GetByID(ctx, id)
	if err != nil {
		return err
	}

	filter := bson.M{"_id": string(id)}
	result, err := m.collection().DeleteOne(ctx, filter)
	if err != nil {
		return fmt.Errorf("failed to delete message: %w", err)
	}

	if result.DeletedCount == 0 {
		return domain.NewNotFoundError("message", string(id))
	}

	// Delete associated attachments
	_, err = m.repo.attachments.DeleteByMessage(ctx, id)
	if err != nil {
		return fmt.Errorf("failed to delete message attachments: %w", err)
	}

	// Update mailbox stats
	wasUnread := msg.Status == domain.MessageUnread
	if err := m.repo.mailboxes.DecrementMessageCount(ctx, msg.MailboxID, msg.Size, wasUnread); err != nil {
		return fmt.Errorf("failed to update mailbox stats: %w", err)
	}

	return nil
}

// DeleteByMailbox removes all messages in a mailbox.
func (m *MessageRepository) DeleteByMailbox(ctx context.Context, mailboxID domain.ID) (int64, error) {
	ctx = m.repo.getSessionContext(ctx)

	// Get message IDs for attachment cleanup
	cursor, err := m.collection().Find(ctx, bson.M{"mailboxId": string(mailboxID)}, options.Find().SetProjection(bson.M{"_id": 1}))
	if err != nil {
		return 0, fmt.Errorf("failed to get message IDs: %w", err)
	}
	defer cursor.Close(ctx)

	var messageIDs []domain.ID
	for cursor.Next(ctx) {
		var doc struct {
			ID string `bson:"_id"`
		}
		if err := cursor.Decode(&doc); err != nil {
			continue
		}
		messageIDs = append(messageIDs, domain.ID(doc.ID))
	}

	// Delete messages
	filter := bson.M{"mailboxId": string(mailboxID)}
	result, err := m.collection().DeleteMany(ctx, filter)
	if err != nil {
		return 0, fmt.Errorf("failed to delete mailbox messages: %w", err)
	}

	// Delete attachments for deleted messages
	if len(messageIDs) > 0 {
		_, err = m.repo.attachments.DeleteByMessages(ctx, messageIDs)
		if err != nil {
			return result.DeletedCount, fmt.Errorf("failed to delete attachments: %w", err)
		}
	}

	return result.DeletedCount, nil
}

// Exists checks if a message with the given ID exists.
func (m *MessageRepository) Exists(ctx context.Context, id domain.ID) (bool, error) {
	ctx = m.repo.getSessionContext(ctx)

	filter := bson.M{"_id": string(id)}
	count, err := m.collection().CountDocuments(ctx, filter, options.Count().SetLimit(1))
	if err != nil {
		return false, fmt.Errorf("failed to check message existence: %w", err)
	}

	return count > 0, nil
}

// ExistsByMessageID checks if a message with the given Message-ID exists.
func (m *MessageRepository) ExistsByMessageID(ctx context.Context, messageID string) (bool, error) {
	ctx = m.repo.getSessionContext(ctx)

	filter := bson.M{"messageId": messageID}
	count, err := m.collection().CountDocuments(ctx, filter, options.Count().SetLimit(1))
	if err != nil {
		return false, fmt.Errorf("failed to check message existence: %w", err)
	}

	return count > 0, nil
}

// Count returns the total number of messages matching the filter.
func (m *MessageRepository) Count(ctx context.Context, filter *repository.MessageFilter) (int64, error) {
	ctx = m.repo.getSessionContext(ctx)

	mongoFilter := m.buildFilter(filter)
	count, err := m.collection().CountDocuments(ctx, mongoFilter)
	if err != nil {
		return 0, fmt.Errorf("failed to count messages: %w", err)
	}

	return count, nil
}

// CountByMailbox returns the number of messages in a mailbox.
func (m *MessageRepository) CountByMailbox(ctx context.Context, mailboxID domain.ID) (int64, error) {
	ctx = m.repo.getSessionContext(ctx)

	filter := bson.M{"mailboxId": string(mailboxID)}
	count, err := m.collection().CountDocuments(ctx, filter)
	if err != nil {
		return 0, fmt.Errorf("failed to count mailbox messages: %w", err)
	}

	return count, nil
}

// CountUnreadByMailbox returns the number of unread messages in a mailbox.
func (m *MessageRepository) CountUnreadByMailbox(ctx context.Context, mailboxID domain.ID) (int64, error) {
	ctx = m.repo.getSessionContext(ctx)

	filter := bson.M{
		"mailboxId": string(mailboxID),
		"status":    string(domain.MessageUnread),
	}
	count, err := m.collection().CountDocuments(ctx, filter)
	if err != nil {
		return 0, fmt.Errorf("failed to count unread messages: %w", err)
	}

	return count, nil
}

// MarkAsRead marks a message as read.
func (m *MessageRepository) MarkAsRead(ctx context.Context, id domain.ID) (bool, error) {
	ctx = m.repo.getSessionContext(ctx)

	filter := bson.M{
		"_id":    string(id),
		"status": string(domain.MessageUnread),
	}
	update := bson.M{
		"$set": bson.M{
			"status":    string(domain.MessageRead),
			"updatedAt": time.Now().UTC(),
		},
	}

	result, err := m.collection().UpdateOne(ctx, filter, update)
	if err != nil {
		return false, fmt.Errorf("failed to mark message as read: %w", err)
	}

	if result.MatchedCount == 0 {
		// Check if message exists
		exists, err := m.Exists(ctx, id)
		if err != nil {
			return false, err
		}
		if !exists {
			return false, domain.NewNotFoundError("message", string(id))
		}
		return false, nil // Already read
	}

	// Update mailbox unread count
	msg, err := m.GetByID(ctx, id)
	if err == nil {
		m.repo.mailboxes.UpdateUnreadCount(ctx, msg.MailboxID, -1)
	}

	return true, nil
}

// MarkAsUnread marks a message as unread.
func (m *MessageRepository) MarkAsUnread(ctx context.Context, id domain.ID) (bool, error) {
	ctx = m.repo.getSessionContext(ctx)

	filter := bson.M{
		"_id":    string(id),
		"status": string(domain.MessageRead),
	}
	update := bson.M{
		"$set": bson.M{
			"status":    string(domain.MessageUnread),
			"updatedAt": time.Now().UTC(),
		},
	}

	result, err := m.collection().UpdateOne(ctx, filter, update)
	if err != nil {
		return false, fmt.Errorf("failed to mark message as unread: %w", err)
	}

	if result.MatchedCount == 0 {
		exists, err := m.Exists(ctx, id)
		if err != nil {
			return false, err
		}
		if !exists {
			return false, domain.NewNotFoundError("message", string(id))
		}
		return false, nil // Already unread
	}

	// Update mailbox unread count
	msg, err := m.GetByID(ctx, id)
	if err == nil {
		m.repo.mailboxes.UpdateUnreadCount(ctx, msg.MailboxID, 1)
	}

	return true, nil
}

// MarkAllAsRead marks all messages in a mailbox as read.
func (m *MessageRepository) MarkAllAsRead(ctx context.Context, mailboxID domain.ID) (int64, error) {
	ctx = m.repo.getSessionContext(ctx)

	filter := bson.M{
		"mailboxId": string(mailboxID),
		"status":    string(domain.MessageUnread),
	}
	update := bson.M{
		"$set": bson.M{
			"status":    string(domain.MessageRead),
			"updatedAt": time.Now().UTC(),
		},
	}

	result, err := m.collection().UpdateMany(ctx, filter, update)
	if err != nil {
		return 0, fmt.Errorf("failed to mark all messages as read: %w", err)
	}

	// Update mailbox unread count
	if result.ModifiedCount > 0 {
		stats := &repository.MailboxStatsUpdate{}
		zero := int64(0)
		stats.UnreadCount = &zero
		m.repo.mailboxes.UpdateStats(ctx, mailboxID, stats)
	}

	return result.ModifiedCount, nil
}

// ToggleStar toggles the starred status of a message.
func (m *MessageRepository) ToggleStar(ctx context.Context, id domain.ID) (bool, error) {
	msg, err := m.GetByID(ctx, id)
	if err != nil {
		return false, err
	}

	newStarred := !msg.IsStarred

	ctx = m.repo.getSessionContext(ctx)
	filter := bson.M{"_id": string(id)}
	update := bson.M{
		"$set": bson.M{
			"isStarred": newStarred,
			"updatedAt": time.Now().UTC(),
		},
	}

	_, err = m.collection().UpdateOne(ctx, filter, update)
	if err != nil {
		return false, fmt.Errorf("failed to toggle star: %w", err)
	}

	return newStarred, nil
}

// Star marks a message as starred.
func (m *MessageRepository) Star(ctx context.Context, id domain.ID) error {
	ctx = m.repo.getSessionContext(ctx)

	filter := bson.M{"_id": string(id)}
	update := bson.M{
		"$set": bson.M{
			"isStarred": true,
			"updatedAt": time.Now().UTC(),
		},
	}

	result, err := m.collection().UpdateOne(ctx, filter, update)
	if err != nil {
		return fmt.Errorf("failed to star message: %w", err)
	}

	if result.MatchedCount == 0 {
		return domain.NewNotFoundError("message", string(id))
	}

	return nil
}

// Unstar removes the star from a message.
func (m *MessageRepository) Unstar(ctx context.Context, id domain.ID) error {
	ctx = m.repo.getSessionContext(ctx)

	filter := bson.M{"_id": string(id)}
	update := bson.M{
		"$set": bson.M{
			"isStarred": false,
			"updatedAt": time.Now().UTC(),
		},
	}

	result, err := m.collection().UpdateOne(ctx, filter, update)
	if err != nil {
		return fmt.Errorf("failed to unstar message: %w", err)
	}

	if result.MatchedCount == 0 {
		return domain.NewNotFoundError("message", string(id))
	}

	return nil
}

// MarkAsSpam marks a message as spam.
func (m *MessageRepository) MarkAsSpam(ctx context.Context, id domain.ID) error {
	ctx = m.repo.getSessionContext(ctx)

	filter := bson.M{"_id": string(id)}
	update := bson.M{
		"$set": bson.M{
			"isSpam":    true,
			"updatedAt": time.Now().UTC(),
		},
	}

	result, err := m.collection().UpdateOne(ctx, filter, update)
	if err != nil {
		return fmt.Errorf("failed to mark message as spam: %w", err)
	}

	if result.MatchedCount == 0 {
		return domain.NewNotFoundError("message", string(id))
	}

	return nil
}

// MarkAsNotSpam removes the spam flag from a message.
func (m *MessageRepository) MarkAsNotSpam(ctx context.Context, id domain.ID) error {
	ctx = m.repo.getSessionContext(ctx)

	filter := bson.M{"_id": string(id)}
	update := bson.M{
		"$set": bson.M{
			"isSpam":    false,
			"updatedAt": time.Now().UTC(),
		},
	}

	result, err := m.collection().UpdateOne(ctx, filter, update)
	if err != nil {
		return fmt.Errorf("failed to mark message as not spam: %w", err)
	}

	if result.MatchedCount == 0 {
		return domain.NewNotFoundError("message", string(id))
	}

	return nil
}

func (m *MessageRepository) MarkAsDeleted(ctx context.Context, id domain.ID) error {
	ctx = m.repo.getSessionContext(ctx)

	filter := bson.M{"_id": string(id)}
	update := bson.M{
		"$set": bson.M{
			"isDeleted": true,
			"updatedAt": time.Now().UTC(),
		},
	}

	result, err := m.collection().UpdateOne(ctx, filter, update)
	if err != nil {
		return fmt.Errorf("failed to mark message as deleted: %w", err)
	}

	if result.MatchedCount == 0 {
		return domain.NewNotFoundError("message", string(id))
	}

	return nil
}

func (m *MessageRepository) UnmarkAsDeleted(ctx context.Context, id domain.ID) error {
	ctx = m.repo.getSessionContext(ctx)

	filter := bson.M{"_id": string(id)}
	update := bson.M{
		"$set": bson.M{
			"isDeleted": false,
			"updatedAt": time.Now().UTC(),
		},
	}

	result, err := m.collection().UpdateOne(ctx, filter, update)
	if err != nil {
		return fmt.Errorf("failed to unmark message as deleted: %w", err)
	}

	if result.MatchedCount == 0 {
		return domain.NewNotFoundError("message", string(id))
	}

	return nil
}

// MoveToMailbox moves a message to a different mailbox.
func (m *MessageRepository) MoveToMailbox(ctx context.Context, id domain.ID, targetMailboxID domain.ID) error {
	ctx = m.repo.getSessionContext(ctx)

	// Get the message first
	msg, err := m.GetByID(ctx, id)
	if err != nil {
		return err
	}

	// Check target mailbox exists
	exists, err := m.repo.mailboxes.Exists(ctx, targetMailboxID)
	if err != nil {
		return err
	}
	if !exists {
		return domain.NewNotFoundError("mailbox", string(targetMailboxID))
	}

	sourceMailboxID := msg.MailboxID
	wasUnread := msg.Status == domain.MessageUnread

	// Update the message
	filter := bson.M{"_id": string(id)}
	update := bson.M{
		"$set": bson.M{
			"mailboxId": string(targetMailboxID),
			"updatedAt": time.Now().UTC(),
		},
	}

	_, err = m.collection().UpdateOne(ctx, filter, update)
	if err != nil {
		return fmt.Errorf("failed to move message: %w", err)
	}

	// Update source mailbox stats
	m.repo.mailboxes.DecrementMessageCount(ctx, sourceMailboxID, msg.Size, wasUnread)

	// Update target mailbox stats
	m.repo.mailboxes.IncrementMessageCount(ctx, targetMailboxID, msg.Size)

	return nil
}

// Search performs a full-text search across message fields.
func (m *MessageRepository) Search(ctx context.Context, searchOpts *repository.SearchOptions, filter *repository.MessageFilter, opts *repository.ListOptions) (*repository.ListResult[*domain.Message], error) {
	if filter == nil {
		filter = &repository.MessageFilter{}
	}

	if searchOpts != nil && searchOpts.Query != "" {
		filter.Search = searchOpts.Query
	}

	return m.List(ctx, filter, opts)
}

// SearchSummaries performs search and returns message summaries.
func (m *MessageRepository) SearchSummaries(ctx context.Context, searchOpts *repository.SearchOptions, filter *repository.MessageFilter, opts *repository.ListOptions) (*repository.ListResult[*domain.MessageSummary], error) {
	if filter == nil {
		filter = &repository.MessageFilter{}
	}

	if searchOpts != nil && searchOpts.Query != "" {
		filter.Search = searchOpts.Query
	}

	return m.ListSummaries(ctx, filter, opts)
}

// GetThread retrieves all messages in a conversation thread.
func (m *MessageRepository) GetThread(ctx context.Context, id domain.ID) ([]*domain.Message, error) {
	msg, err := m.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	ctx = m.repo.getSessionContext(ctx)

	// Find all messages that reference this message or are referenced by it
	var threadIDs []string
	if msg.MessageID != "" {
		threadIDs = append(threadIDs, msg.MessageID)
	}
	threadIDs = append(threadIDs, msg.References...)
	if msg.InReplyTo != "" {
		threadIDs = append(threadIDs, msg.InReplyTo)
	}

	filter := bson.M{
		"$or": bson.A{
			bson.M{"_id": string(id)},
			bson.M{"messageId": bson.M{"$in": threadIDs}},
			bson.M{"inReplyTo": bson.M{"$in": threadIDs}},
			bson.M{"references": bson.M{"$in": threadIDs}},
		},
	}

	cursor, err := m.collection().Find(ctx, filter, options.Find().SetSort(bson.D{{Key: "receivedAt", Value: 1}}))
	if err != nil {
		return nil, fmt.Errorf("failed to get thread: %w", err)
	}
	defer cursor.Close(ctx)

	var docs []messageDocument
	if err := cursor.All(ctx, &docs); err != nil {
		return nil, fmt.Errorf("failed to decode thread: %w", err)
	}

	messages := make([]*domain.Message, len(docs))
	for i, doc := range docs {
		messages[i] = m.toDomain(&doc)
	}

	return messages, nil
}

// GetReplies retrieves all replies to a specific message.
func (m *MessageRepository) GetReplies(ctx context.Context, id domain.ID) ([]*domain.Message, error) {
	msg, err := m.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if msg.MessageID == "" {
		return []*domain.Message{}, nil
	}

	ctx = m.repo.getSessionContext(ctx)

	filter := bson.M{
		"$or": bson.A{
			bson.M{"inReplyTo": msg.MessageID},
			bson.M{"references": msg.MessageID},
		},
	}

	cursor, err := m.collection().Find(ctx, filter, options.Find().SetSort(bson.D{{Key: "receivedAt", Value: 1}}))
	if err != nil {
		return nil, fmt.Errorf("failed to get replies: %w", err)
	}
	defer cursor.Close(ctx)

	var docs []messageDocument
	if err := cursor.All(ctx, &docs); err != nil {
		return nil, fmt.Errorf("failed to decode replies: %w", err)
	}

	messages := make([]*domain.Message, len(docs))
	for i, doc := range docs {
		messages[i] = m.toDomain(&doc)
	}

	return messages, nil
}

// GetStarred retrieves all starred messages.
func (m *MessageRepository) GetStarred(ctx context.Context, opts *repository.ListOptions) (*repository.ListResult[*domain.Message], error) {
	isStarred := true
	filter := &repository.MessageFilter{IsStarred: &isStarred}
	return m.List(ctx, filter, opts)
}

// GetStarredByUser retrieves starred messages from mailboxes owned by a user.
func (m *MessageRepository) GetStarredByUser(ctx context.Context, userID domain.ID, opts *repository.ListOptions) (*repository.ListResult[*domain.Message], error) {
	ctx = m.repo.getSessionContext(ctx)

	// First get user's mailbox IDs
	mailboxResult, err := m.repo.mailboxes.ListByUser(ctx, userID, nil)
	if err != nil {
		return nil, err
	}

	mailboxIDs := make([]domain.ID, len(mailboxResult.Items))
	for i, mb := range mailboxResult.Items {
		mailboxIDs[i] = mb.ID
	}

	isStarred := true
	filter := &repository.MessageFilter{
		MailboxIDs: mailboxIDs,
		IsStarred:  &isStarred,
	}

	return m.List(ctx, filter, opts)
}

// GetSpam retrieves all spam messages.
func (m *MessageRepository) GetSpam(ctx context.Context, opts *repository.ListOptions) (*repository.ListResult[*domain.Message], error) {
	isSpam := true
	filter := &repository.MessageFilter{IsSpam: &isSpam}
	return m.List(ctx, filter, opts)
}

// GetUnread retrieves all unread messages.
func (m *MessageRepository) GetUnread(ctx context.Context, opts *repository.ListOptions) (*repository.ListResult[*domain.Message], error) {
	status := domain.MessageUnread
	filter := &repository.MessageFilter{Status: &status}
	return m.List(ctx, filter, opts)
}

// GetUnreadByMailbox retrieves unread messages in a specific mailbox.
func (m *MessageRepository) GetUnreadByMailbox(ctx context.Context, mailboxID domain.ID, opts *repository.ListOptions) (*repository.ListResult[*domain.Message], error) {
	status := domain.MessageUnread
	id := mailboxID
	filter := &repository.MessageFilter{
		MailboxID: &id,
		Status:    &status,
	}
	return m.List(ctx, filter, opts)
}

// GetMessagesWithAttachments retrieves messages that have attachments.
func (m *MessageRepository) GetMessagesWithAttachments(ctx context.Context, filter *repository.MessageFilter, opts *repository.ListOptions) (*repository.ListResult[*domain.Message], error) {
	if filter == nil {
		filter = &repository.MessageFilter{}
	}
	hasAttachments := true
	filter.HasAttachments = &hasAttachments
	return m.List(ctx, filter, opts)
}

// GetRecent retrieves messages received in the last N hours.
func (m *MessageRepository) GetRecent(ctx context.Context, hours int, opts *repository.ListOptions) (*repository.ListResult[*domain.Message], error) {
	since := time.Now().UTC().Add(-time.Duration(hours) * time.Hour)
	ts := domain.Timestamp{Time: since}
	filter := &repository.MessageFilter{ReceivedAfter: &ts}
	return m.List(ctx, filter, opts)
}

// GetByDateRange retrieves messages within a date range.
func (m *MessageRepository) GetByDateRange(ctx context.Context, dateRange *repository.DateRangeFilter, opts *repository.ListOptions) (*repository.ListResult[*domain.Message], error) {
	filter := &repository.MessageFilter{}
	if dateRange != nil {
		filter.ReceivedAfter = dateRange.From
		filter.ReceivedBefore = dateRange.To
	}
	return m.List(ctx, filter, opts)
}

// GetBySender retrieves all messages from a specific sender.
func (m *MessageRepository) GetBySender(ctx context.Context, senderAddress string, opts *repository.ListOptions) (*repository.ListResult[*domain.Message], error) {
	filter := &repository.MessageFilter{FromAddress: senderAddress}
	return m.List(ctx, filter, opts)
}

// GetByRecipient retrieves all messages sent to a specific recipient.
func (m *MessageRepository) GetByRecipient(ctx context.Context, recipientAddress string, opts *repository.ListOptions) (*repository.ListResult[*domain.Message], error) {
	filter := &repository.MessageFilter{ToAddress: recipientAddress}
	return m.List(ctx, filter, opts)
}

// GetOldMessages retrieves messages older than the specified days.
func (m *MessageRepository) GetOldMessages(ctx context.Context, olderThanDays int, opts *repository.ListOptions) (*repository.ListResult[*domain.Message], error) {
	before := time.Now().UTC().AddDate(0, 0, -olderThanDays)
	ts := domain.Timestamp{Time: before}
	filter := &repository.MessageFilter{ReceivedBefore: &ts}
	return m.List(ctx, filter, opts)
}

// GetLargeMessages retrieves messages larger than the specified size.
func (m *MessageRepository) GetLargeMessages(ctx context.Context, minSize int64, opts *repository.ListOptions) (*repository.ListResult[*domain.Message], error) {
	filter := &repository.MessageFilter{MinSize: &minSize}
	return m.List(ctx, filter, opts)
}

// DeleteOldMessages deletes messages older than the specified days.
func (m *MessageRepository) DeleteOldMessages(ctx context.Context, olderThanDays int) (int64, error) {
	ctx = m.repo.getSessionContext(ctx)

	before := time.Now().UTC().AddDate(0, 0, -olderThanDays)
	filter := bson.M{"receivedAt": bson.M{"$lt": before}}

	result, err := m.collection().DeleteMany(ctx, filter)
	if err != nil {
		return 0, fmt.Errorf("failed to delete old messages: %w", err)
	}

	return result.DeletedCount, nil
}

// DeleteSpam deletes all spam messages.
func (m *MessageRepository) DeleteSpam(ctx context.Context) (int64, error) {
	ctx = m.repo.getSessionContext(ctx)

	filter := bson.M{"isSpam": true}
	result, err := m.collection().DeleteMany(ctx, filter)
	if err != nil {
		return 0, fmt.Errorf("failed to delete spam messages: %w", err)
	}

	return result.DeletedCount, nil
}

// BulkMarkAsRead marks multiple messages as read.
func (m *MessageRepository) BulkMarkAsRead(ctx context.Context, ids []domain.ID) (*repository.BulkOperation, error) {
	result := repository.NewBulkOperation()

	for _, id := range ids {
		if _, err := m.MarkAsRead(ctx, id); err != nil {
			result.AddFailure(string(id), err)
		} else {
			result.AddSuccess()
		}
	}

	return result, nil
}

// BulkMarkAsUnread marks multiple messages as unread.
func (m *MessageRepository) BulkMarkAsUnread(ctx context.Context, ids []domain.ID) (*repository.BulkOperation, error) {
	result := repository.NewBulkOperation()

	for _, id := range ids {
		if _, err := m.MarkAsUnread(ctx, id); err != nil {
			result.AddFailure(string(id), err)
		} else {
			result.AddSuccess()
		}
	}

	return result, nil
}

// BulkDelete permanently removes multiple messages.
func (m *MessageRepository) BulkDelete(ctx context.Context, ids []domain.ID) (*repository.BulkOperation, error) {
	result := repository.NewBulkOperation()

	for _, id := range ids {
		if err := m.Delete(ctx, id); err != nil {
			result.AddFailure(string(id), err)
		} else {
			result.AddSuccess()
		}
	}

	return result, nil
}

// BulkMove moves multiple messages to a different mailbox.
func (m *MessageRepository) BulkMove(ctx context.Context, ids []domain.ID, targetMailboxID domain.ID) (*repository.BulkOperation, error) {
	result := repository.NewBulkOperation()

	for _, id := range ids {
		if err := m.MoveToMailbox(ctx, id, targetMailboxID); err != nil {
			result.AddFailure(string(id), err)
		} else {
			result.AddSuccess()
		}
	}

	return result, nil
}

// BulkStar marks multiple messages as starred.
func (m *MessageRepository) BulkStar(ctx context.Context, ids []domain.ID) (*repository.BulkOperation, error) {
	result := repository.NewBulkOperation()

	for _, id := range ids {
		if err := m.Star(ctx, id); err != nil {
			result.AddFailure(string(id), err)
		} else {
			result.AddSuccess()
		}
	}

	return result, nil
}

// BulkUnstar removes the star from multiple messages.
func (m *MessageRepository) BulkUnstar(ctx context.Context, ids []domain.ID) (*repository.BulkOperation, error) {
	result := repository.NewBulkOperation()

	for _, id := range ids {
		if err := m.Unstar(ctx, id); err != nil {
			result.AddFailure(string(id), err)
		} else {
			result.AddSuccess()
		}
	}

	return result, nil
}

// GetSizeByMailbox calculates the total size of messages in a mailbox.
func (m *MessageRepository) GetSizeByMailbox(ctx context.Context, mailboxID domain.ID) (int64, error) {
	ctx = m.repo.getSessionContext(ctx)

	pipeline := mongo.Pipeline{
		{{Key: "$match", Value: bson.M{"mailboxId": string(mailboxID)}}},
		{{Key: "$group", Value: bson.M{
			"_id":       nil,
			"totalSize": bson.M{"$sum": "$size"},
		}}},
	}

	cursor, err := m.collection().Aggregate(ctx, pipeline)
	if err != nil {
		return 0, fmt.Errorf("failed to get mailbox size: %w", err)
	}
	defer cursor.Close(ctx)

	var result struct {
		TotalSize int64 `bson:"totalSize"`
	}
	if cursor.Next(ctx) {
		if err := cursor.Decode(&result); err != nil {
			return 0, fmt.Errorf("failed to decode size: %w", err)
		}
	}

	return result.TotalSize, nil
}

// GetTotalSize calculates the total size of all messages.
func (m *MessageRepository) GetTotalSize(ctx context.Context) (int64, error) {
	ctx = m.repo.getSessionContext(ctx)

	pipeline := mongo.Pipeline{
		{{Key: "$group", Value: bson.M{
			"_id":       nil,
			"totalSize": bson.M{"$sum": "$size"},
		}}},
	}

	cursor, err := m.collection().Aggregate(ctx, pipeline)
	if err != nil {
		return 0, fmt.Errorf("failed to get total size: %w", err)
	}
	defer cursor.Close(ctx)

	var result struct {
		TotalSize int64 `bson:"totalSize"`
	}
	if cursor.Next(ctx) {
		if err := cursor.Decode(&result); err != nil {
			return 0, fmt.Errorf("failed to decode size: %w", err)
		}
	}

	return result.TotalSize, nil
}

// GetDailyCounts returns message counts grouped by day within a date range.
func (m *MessageRepository) GetDailyCounts(ctx context.Context, dateRange *repository.DateRangeFilter) ([]repository.DateCount, error) {
	ctx = m.repo.getSessionContext(ctx)

	matchStage := bson.M{}
	if dateRange != nil {
		if dateRange.From != nil {
			matchStage["receivedAt"] = bson.M{"$gte": dateRange.From.Time}
		}
		if dateRange.To != nil {
			if _, exists := matchStage["receivedAt"]; exists {
				matchStage["receivedAt"].(bson.M)["$lte"] = dateRange.To.Time
			} else {
				matchStage["receivedAt"] = bson.M{"$lte": dateRange.To.Time}
			}
		}
	}

	pipeline := mongo.Pipeline{
		{{Key: "$match", Value: matchStage}},
		{{Key: "$group", Value: bson.M{
			"_id": bson.M{
				"$dateToString": bson.M{"format": "%Y-%m-%d", "date": "$receivedAt"},
			},
			"count": bson.M{"$sum": 1},
		}}},
		{{Key: "$sort", Value: bson.M{"_id": 1}}},
	}

	cursor, err := m.collection().Aggregate(ctx, pipeline)
	if err != nil {
		return nil, fmt.Errorf("failed to get daily counts: %w", err)
	}
	defer cursor.Close(ctx)

	var counts []repository.DateCount
	for cursor.Next(ctx) {
		var doc struct {
			Date  string `bson:"_id"`
			Count int64  `bson:"count"`
		}
		if err := cursor.Decode(&doc); err != nil {
			continue
		}
		counts = append(counts, repository.DateCount{
			Date:  doc.Date,
			Count: doc.Count,
		})
	}

	return counts, nil
}

// GetSenderCounts returns message counts grouped by sender address.
func (m *MessageRepository) GetSenderCounts(ctx context.Context, limit int) ([]repository.AddressCount, error) {
	ctx = m.repo.getSessionContext(ctx)

	pipeline := mongo.Pipeline{
		{{Key: "$group", Value: bson.M{
			"_id":   "$from.address",
			"name":  bson.M{"$first": "$from.name"},
			"count": bson.M{"$sum": 1},
		}}},
		{{Key: "$sort", Value: bson.M{"count": -1}}},
		{{Key: "$limit", Value: limit}},
	}

	cursor, err := m.collection().Aggregate(ctx, pipeline)
	if err != nil {
		return nil, fmt.Errorf("failed to get sender counts: %w", err)
	}
	defer cursor.Close(ctx)

	var counts []repository.AddressCount
	for cursor.Next(ctx) {
		var doc struct {
			Address string `bson:"_id"`
			Name    string `bson:"name"`
			Count   int64  `bson:"count"`
		}
		if err := cursor.Decode(&doc); err != nil {
			continue
		}
		counts = append(counts, repository.AddressCount{
			Address: doc.Address,
			Name:    doc.Name,
			Count:   doc.Count,
		})
	}

	return counts, nil
}

// GetRecipientCounts returns message counts grouped by recipient address.
func (m *MessageRepository) GetRecipientCounts(ctx context.Context, limit int) ([]repository.AddressCount, error) {
	ctx = m.repo.getSessionContext(ctx)

	pipeline := mongo.Pipeline{
		{{Key: "$unwind", Value: "$to"}},
		{{Key: "$group", Value: bson.M{
			"_id":   "$to.address",
			"name":  bson.M{"$first": "$to.name"},
			"count": bson.M{"$sum": 1},
		}}},
		{{Key: "$sort", Value: bson.M{"count": -1}}},
		{{Key: "$limit", Value: limit}},
	}

	cursor, err := m.collection().Aggregate(ctx, pipeline)
	if err != nil {
		return nil, fmt.Errorf("failed to get recipient counts: %w", err)
	}
	defer cursor.Close(ctx)

	var counts []repository.AddressCount
	for cursor.Next(ctx) {
		var doc struct {
			Address string `bson:"_id"`
			Name    string `bson:"name"`
			Count   int64  `bson:"count"`
		}
		if err := cursor.Decode(&doc); err != nil {
			continue
		}
		counts = append(counts, repository.AddressCount{
			Address: doc.Address,
			Name:    doc.Name,
			Count:   doc.Count,
		})
	}

	return counts, nil
}

// StoreRawBody stores the raw message body for a message.
func (m *MessageRepository) StoreRawBody(ctx context.Context, id domain.ID, rawBody []byte) error {
	ctx = m.repo.getSessionContext(ctx)

	filter := bson.M{"_id": string(id)}
	update := bson.M{
		"$set": bson.M{
			"rawBody":   rawBody,
			"updatedAt": time.Now().UTC(),
		},
	}

	result, err := m.collection().UpdateOne(ctx, filter, update)
	if err != nil {
		return fmt.Errorf("failed to store raw body: %w", err)
	}

	if result.MatchedCount == 0 {
		return domain.NewNotFoundError("message", string(id))
	}

	return nil
}

// GetRawBody retrieves the raw message body.
func (m *MessageRepository) GetRawBody(ctx context.Context, id domain.ID) ([]byte, error) {
	ctx = m.repo.getSessionContext(ctx)

	filter := bson.M{"_id": string(id)}
	opts := options.FindOne().SetProjection(bson.M{"rawBody": 1})

	var doc struct {
		RawBody []byte `bson:"rawBody"`
	}

	if err := m.collection().FindOne(ctx, filter, opts).Decode(&doc); err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, domain.NewNotFoundError("message", string(id))
		}
		return nil, fmt.Errorf("failed to get raw body: %w", err)
	}

	if doc.RawBody == nil {
		return nil, domain.NewNotFoundError("message raw body", string(id))
	}

	return doc.RawBody, nil
}

// Ensure MessageRepository implements repository.MessageRepository
var _ repository.MessageRepository = (*MessageRepository)(nil)
