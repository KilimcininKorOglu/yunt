package mongodb

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"yunt/internal/domain"
	"yunt/internal/repository"
)

// MailboxRepository implements the repository.MailboxRepository interface for MongoDB.
type MailboxRepository struct {
	repo *Repository
}

// mailboxDocument is the MongoDB document representation of a mailbox.
type mailboxDocument struct {
	ID            string    `bson:"_id"`
	UserID        string    `bson:"userId"`
	Name          string    `bson:"name"`
	Address       string    `bson:"address"`
	Description   string    `bson:"description,omitempty"`
	IsCatchAll    bool      `bson:"isCatchAll"`
	IsDefault     bool      `bson:"isDefault"`
	Type          string    `bson:"type"`
	MessageCount  int64     `bson:"messageCount"`
	UnreadCount   int64     `bson:"unreadCount"`
	TotalSize     int64     `bson:"totalSize"`
	RetentionDays int       `bson:"retentionDays"`
	UIDNext       uint32    `bson:"uidNext"`
	CreatedAt     time.Time `bson:"createdAt"`
	UpdatedAt     time.Time `bson:"updatedAt"`
}

// NewMailboxRepository creates a new MongoDB mailbox repository.
func NewMailboxRepository(repo *Repository) *MailboxRepository {
	return &MailboxRepository{repo: repo}
}

// collection returns the mailboxes collection.
func (m *MailboxRepository) collection() *mongo.Collection {
	return m.repo.collection(CollectionMailboxes)
}

// toDocument converts a domain.Mailbox to a MongoDB document.
func (m *MailboxRepository) toDocument(mailbox *domain.Mailbox) *mailboxDocument {
	return &mailboxDocument{
		ID:            string(mailbox.ID),
		UserID:        string(mailbox.UserID),
		Name:          mailbox.Name,
		Address:       mailbox.Address,
		Description:   mailbox.Description,
		IsCatchAll:    mailbox.IsCatchAll,
		IsDefault:     mailbox.IsDefault,
		Type:          string(mailbox.Type),
		MessageCount:  mailbox.MessageCount,
		UnreadCount:   mailbox.UnreadCount,
		TotalSize:     mailbox.TotalSize,
		RetentionDays: mailbox.RetentionDays,
		UIDNext:       mailbox.UIDNext,
		CreatedAt:     mailbox.CreatedAt.Time,
		UpdatedAt:     mailbox.UpdatedAt.Time,
	}
}

// toDomain converts a MongoDB document to a domain.Mailbox.
func (m *MailboxRepository) toDomain(doc *mailboxDocument) *domain.Mailbox {
	return &domain.Mailbox{
		ID:            domain.ID(doc.ID),
		UserID:        domain.ID(doc.UserID),
		Name:          doc.Name,
		Address:       doc.Address,
		Description:   doc.Description,
		IsCatchAll:    doc.IsCatchAll,
		IsDefault:     doc.IsDefault,
		Type:          domain.MailboxType(doc.Type),
		MessageCount:  doc.MessageCount,
		UnreadCount:   doc.UnreadCount,
		TotalSize:     doc.TotalSize,
		RetentionDays: doc.RetentionDays,
		UIDNext:       doc.UIDNext,
		CreatedAt:     domain.Timestamp{Time: doc.CreatedAt},
		UpdatedAt:     domain.Timestamp{Time: doc.UpdatedAt},
	}
}

// GetByID retrieves a mailbox by its unique identifier.
func (m *MailboxRepository) GetByID(ctx context.Context, id domain.ID) (*domain.Mailbox, error) {
	ctx = m.repo.getSessionContext(ctx)

	filter := bson.M{"_id": string(id)}

	var doc mailboxDocument
	if err := m.collection().FindOne(ctx, filter).Decode(&doc); err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, domain.NewNotFoundError("mailbox", string(id))
		}
		return nil, fmt.Errorf("failed to get mailbox by ID: %w", err)
	}

	return m.toDomain(&doc), nil
}

// GetByAddress retrieves a mailbox by its email address.
func (m *MailboxRepository) GetByAddress(ctx context.Context, address string) (*domain.Mailbox, error) {
	ctx = m.repo.getSessionContext(ctx)

	filter := bson.M{
		"address": bson.M{"$regex": "^" + regexp.QuoteMeta(address) + "$", "$options": "i"},
	}

	var doc mailboxDocument
	if err := m.collection().FindOne(ctx, filter).Decode(&doc); err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, domain.NewNotFoundError("mailbox", address)
		}
		return nil, fmt.Errorf("failed to get mailbox by address: %w", err)
	}

	return m.toDomain(&doc), nil
}

// GetCatchAll retrieves the catch-all mailbox for a domain.
func (m *MailboxRepository) GetCatchAll(ctx context.Context, domainName string) (*domain.Mailbox, error) {
	ctx = m.repo.getSessionContext(ctx)

	filter := bson.M{
		"isCatchAll": true,
		"address":    bson.M{"$regex": "@" + regexp.QuoteMeta(domainName) + "$", "$options": "i"},
	}

	var doc mailboxDocument
	if err := m.collection().FindOne(ctx, filter).Decode(&doc); err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, domain.NewNotFoundError("mailbox", "catch-all for "+domainName)
		}
		return nil, fmt.Errorf("failed to get catch-all mailbox: %w", err)
	}

	return m.toDomain(&doc), nil
}

// GetDefault retrieves the default mailbox for a user.
func (m *MailboxRepository) GetDefault(ctx context.Context, userID domain.ID) (*domain.Mailbox, error) {
	ctx = m.repo.getSessionContext(ctx)

	filter := bson.M{
		"userId":    string(userID),
		"isDefault": true,
	}

	var doc mailboxDocument
	if err := m.collection().FindOne(ctx, filter).Decode(&doc); err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, domain.NewNotFoundError("mailbox", "default for user "+string(userID))
		}
		return nil, fmt.Errorf("failed to get default mailbox: %w", err)
	}

	return m.toDomain(&doc), nil
}

// List retrieves mailboxes with optional filtering, sorting, and pagination.
func (m *MailboxRepository) List(ctx context.Context, filter *repository.MailboxFilter, opts *repository.ListOptions) (*repository.ListResult[*domain.Mailbox], error) {
	ctx = m.repo.getSessionContext(ctx)

	mongoFilter := m.buildFilter(filter)
	findOpts := m.buildFindOptions(opts)

	total, err := m.collection().CountDocuments(ctx, mongoFilter)
	if err != nil {
		return nil, fmt.Errorf("failed to count mailboxes: %w", err)
	}

	cursor, err := m.collection().Find(ctx, mongoFilter, findOpts)
	if err != nil {
		return nil, fmt.Errorf("failed to list mailboxes: %w", err)
	}
	defer cursor.Close(ctx)

	var docs []mailboxDocument
	if err := cursor.All(ctx, &docs); err != nil {
		return nil, fmt.Errorf("failed to decode mailboxes: %w", err)
	}

	mailboxes := make([]*domain.Mailbox, len(docs))
	for i, doc := range docs {
		mailboxes[i] = m.toDomain(&doc)
	}

	result := &repository.ListResult[*domain.Mailbox]{
		Items: mailboxes,
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

// ListByUser retrieves all mailboxes owned by a specific user.
func (m *MailboxRepository) ListByUser(ctx context.Context, userID domain.ID, opts *repository.ListOptions) (*repository.ListResult[*domain.Mailbox], error) {
	id := userID
	filter := &repository.MailboxFilter{UserID: &id}
	return m.List(ctx, filter, opts)
}

// buildFilter builds the MongoDB filter from repository.MailboxFilter.
func (m *MailboxRepository) buildFilter(filter *repository.MailboxFilter) bson.M {
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

	if filter.UserID != nil {
		f["userId"] = string(*filter.UserID)
	}

	if len(filter.UserIDs) > 0 {
		userIDs := make([]string, len(filter.UserIDs))
		for i, id := range filter.UserIDs {
			userIDs[i] = string(id)
		}
		f["userId"] = bson.M{"$in": userIDs}
	}

	if filter.Address != "" {
		f["address"] = bson.M{"$regex": "^" + regexp.QuoteMeta(filter.Address) + "$", "$options": "i"}
	}

	if filter.AddressContains != "" {
		f["address"] = bson.M{"$regex": regexp.QuoteMeta(filter.AddressContains), "$options": "i"}
	}

	if filter.Domain != "" {
		f["address"] = bson.M{"$regex": "@" + regexp.QuoteMeta(filter.Domain) + "$", "$options": "i"}
	}

	if filter.IsCatchAll != nil {
		f["isCatchAll"] = *filter.IsCatchAll
	}

	if filter.IsDefault != nil {
		f["isDefault"] = *filter.IsDefault
	}

	if filter.HasMessages != nil {
		if *filter.HasMessages {
			f["messageCount"] = bson.M{"$gt": 0}
		} else {
			f["messageCount"] = 0
		}
	}

	if filter.HasUnread != nil {
		if *filter.HasUnread {
			f["unreadCount"] = bson.M{"$gt": 0}
		} else {
			f["unreadCount"] = 0
		}
	}

	if filter.Search != "" {
		f["$text"] = bson.M{"$search": filter.Search}
	}

	if filter.MinMessageCount != nil {
		f["messageCount"] = bson.M{"$gte": *filter.MinMessageCount}
	}

	if filter.MaxMessageCount != nil {
		if _, exists := f["messageCount"]; exists {
			f["messageCount"].(bson.M)["$lte"] = *filter.MaxMessageCount
		} else {
			f["messageCount"] = bson.M{"$lte": *filter.MaxMessageCount}
		}
	}

	if filter.MinSize != nil {
		f["totalSize"] = bson.M{"$gte": *filter.MinSize}
	}

	if filter.MaxSize != nil {
		if _, exists := f["totalSize"]; exists {
			f["totalSize"].(bson.M)["$lte"] = *filter.MaxSize
		} else {
			f["totalSize"] = bson.M{"$lte": *filter.MaxSize}
		}
	}

	if filter.CreatedBefore != nil {
		f["createdAt"] = bson.M{"$lt": filter.CreatedBefore.Time}
	}

	if filter.CreatedAfter != nil {
		if _, exists := f["createdAt"]; exists {
			f["createdAt"].(bson.M)["$gt"] = filter.CreatedAfter.Time
		} else {
			f["createdAt"] = bson.M{"$gt": filter.CreatedAfter.Time}
		}
	}

	if filter.RetentionDays != nil {
		if *filter.RetentionDays == -1 {
			f["retentionDays"] = 0
		} else {
			f["retentionDays"] = *filter.RetentionDays
		}
	}

	return f
}

// buildFindOptions builds MongoDB find options from repository.ListOptions.
func (m *MailboxRepository) buildFindOptions(opts *repository.ListOptions) *options.FindOptions {
	findOpts := options.Find()

	if opts == nil {
		findOpts.SetSort(bson.D{{Key: "createdAt", Value: -1}})
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
		findOpts.SetSort(bson.D{{Key: "createdAt", Value: -1}})
	}

	if opts.Pagination != nil {
		opts.Pagination.Normalize()
		findOpts.SetSkip(int64(opts.Pagination.Offset()))
		findOpts.SetLimit(int64(opts.Pagination.Limit()))
	}

	return findOpts
}

// mapSortField maps repository sort field to MongoDB field.
func (m *MailboxRepository) mapSortField(field string) string {
	switch field {
	case "name":
		return "name"
	case "address":
		return "address"
	case "messageCount":
		return "messageCount"
	case "unreadCount":
		return "unreadCount"
	case "totalSize":
		return "totalSize"
	case "createdAt":
		return "createdAt"
	case "updatedAt":
		return "updatedAt"
	default:
		return "createdAt"
	}
}

// Create creates a new mailbox.
func (m *MailboxRepository) Create(ctx context.Context, mailbox *domain.Mailbox) error {
	ctx = m.repo.getSessionContext(ctx)

	if mailbox.UIDNext == 0 {
		mailbox.UIDNext = 1
	}

	exists, err := m.ExistsByAddress(ctx, mailbox.Address)
	if err != nil {
		return fmt.Errorf("failed to check address existence: %w", err)
	}
	if exists {
		return domain.NewAlreadyExistsError("mailbox", "address", mailbox.Address)
	}

	doc := m.toDocument(mailbox)
	_, err = m.collection().InsertOne(ctx, doc)
	if err != nil {
		if mongo.IsDuplicateKeyError(err) {
			return domain.NewAlreadyExistsError("mailbox", "id", string(mailbox.ID))
		}
		return fmt.Errorf("failed to create mailbox: %w", err)
	}

	return nil
}

// Update updates an existing mailbox.
func (m *MailboxRepository) Update(ctx context.Context, mailbox *domain.Mailbox) error {
	ctx = m.repo.getSessionContext(ctx)

	filter := bson.M{"_id": string(mailbox.ID)}
	update := bson.M{
		"$set": bson.M{
			"name":          mailbox.Name,
			"address":       mailbox.Address,
			"description":   mailbox.Description,
			"isCatchAll":    mailbox.IsCatchAll,
			"isDefault":     mailbox.IsDefault,
			"messageCount":  mailbox.MessageCount,
			"unreadCount":   mailbox.UnreadCount,
			"totalSize":     mailbox.TotalSize,
			"retentionDays": mailbox.RetentionDays,
			"updatedAt":     time.Now().UTC(),
		},
	}

	result, err := m.collection().UpdateOne(ctx, filter, update)
	if err != nil {
		return fmt.Errorf("failed to update mailbox: %w", err)
	}

	if result.MatchedCount == 0 {
		return domain.NewNotFoundError("mailbox", string(mailbox.ID))
	}

	return nil
}

// Delete permanently removes a mailbox by its ID.
func (m *MailboxRepository) Delete(ctx context.Context, id domain.ID) error {
	ctx = m.repo.getSessionContext(ctx)

	filter := bson.M{"_id": string(id)}
	result, err := m.collection().DeleteOne(ctx, filter)
	if err != nil {
		return fmt.Errorf("failed to delete mailbox: %w", err)
	}

	if result.DeletedCount == 0 {
		return domain.NewNotFoundError("mailbox", string(id))
	}

	return nil
}

// DeleteWithMessages removes a mailbox and all its messages.
func (m *MailboxRepository) DeleteWithMessages(ctx context.Context, id domain.ID) error {
	ctx = m.repo.getSessionContext(ctx)

	// Delete all messages in the mailbox first
	messagesCollection := m.repo.collection(CollectionMessages)
	_, err := messagesCollection.DeleteMany(ctx, bson.M{"mailboxId": string(id)})
	if err != nil {
		return fmt.Errorf("failed to delete mailbox messages: %w", err)
	}

	// Delete the mailbox
	return m.Delete(ctx, id)
}

// DeleteByUser removes all mailboxes owned by a user.
func (m *MailboxRepository) DeleteByUser(ctx context.Context, userID domain.ID) (int64, error) {
	ctx = m.repo.getSessionContext(ctx)

	filter := bson.M{"userId": string(userID)}
	result, err := m.collection().DeleteMany(ctx, filter)
	if err != nil {
		return 0, fmt.Errorf("failed to delete user mailboxes: %w", err)
	}

	return result.DeletedCount, nil
}

// Exists checks if a mailbox with the given ID exists.
func (m *MailboxRepository) Exists(ctx context.Context, id domain.ID) (bool, error) {
	ctx = m.repo.getSessionContext(ctx)

	filter := bson.M{"_id": string(id)}
	count, err := m.collection().CountDocuments(ctx, filter, options.Count().SetLimit(1))
	if err != nil {
		return false, fmt.Errorf("failed to check mailbox existence: %w", err)
	}

	return count > 0, nil
}

// ExistsByAddress checks if a mailbox with the given address exists.
func (m *MailboxRepository) ExistsByAddress(ctx context.Context, address string) (bool, error) {
	ctx = m.repo.getSessionContext(ctx)

	filter := bson.M{
		"address": bson.M{"$regex": "^" + regexp.QuoteMeta(address) + "$", "$options": "i"},
	}

	count, err := m.collection().CountDocuments(ctx, filter, options.Count().SetLimit(1))
	if err != nil {
		return false, fmt.Errorf("failed to check address existence: %w", err)
	}

	return count > 0, nil
}

// Count returns the total number of mailboxes matching the filter.
func (m *MailboxRepository) Count(ctx context.Context, filter *repository.MailboxFilter) (int64, error) {
	ctx = m.repo.getSessionContext(ctx)

	mongoFilter := m.buildFilter(filter)
	count, err := m.collection().CountDocuments(ctx, mongoFilter)
	if err != nil {
		return 0, fmt.Errorf("failed to count mailboxes: %w", err)
	}

	return count, nil
}

// CountByUser returns the number of mailboxes owned by a user.
func (m *MailboxRepository) CountByUser(ctx context.Context, userID domain.ID) (int64, error) {
	ctx = m.repo.getSessionContext(ctx)

	filter := bson.M{"userId": string(userID)}
	count, err := m.collection().CountDocuments(ctx, filter)
	if err != nil {
		return 0, fmt.Errorf("failed to count user mailboxes: %w", err)
	}

	return count, nil
}

// SetDefault sets a mailbox as the default for its owner.
func (m *MailboxRepository) SetDefault(ctx context.Context, id domain.ID) error {
	ctx = m.repo.getSessionContext(ctx)

	// First, get the mailbox to find the owner
	mailbox, err := m.GetByID(ctx, id)
	if err != nil {
		return err
	}

	// Clear default from other mailboxes of the same user
	if err := m.ClearDefault(ctx, mailbox.UserID); err != nil {
		return err
	}

	// Set this mailbox as default
	filter := bson.M{"_id": string(id)}
	update := bson.M{
		"$set": bson.M{
			"isDefault": true,
			"updatedAt": time.Now().UTC(),
		},
	}

	_, err = m.collection().UpdateOne(ctx, filter, update)
	if err != nil {
		return fmt.Errorf("failed to set default mailbox: %w", err)
	}

	return nil
}

// ClearDefault removes the default flag from all mailboxes for a user.
func (m *MailboxRepository) ClearDefault(ctx context.Context, userID domain.ID) error {
	ctx = m.repo.getSessionContext(ctx)

	filter := bson.M{
		"userId":    string(userID),
		"isDefault": true,
	}
	update := bson.M{
		"$set": bson.M{
			"isDefault": false,
			"updatedAt": time.Now().UTC(),
		},
	}

	_, err := m.collection().UpdateMany(ctx, filter, update)
	if err != nil {
		return fmt.Errorf("failed to clear default mailbox: %w", err)
	}

	return nil
}

// SetCatchAll sets a mailbox as the catch-all for its domain.
func (m *MailboxRepository) SetCatchAll(ctx context.Context, id domain.ID) error {
	ctx = m.repo.getSessionContext(ctx)

	// Get the mailbox to find its domain
	mailbox, err := m.GetByID(ctx, id)
	if err != nil {
		return err
	}

	domainName := mailbox.GetDomain()

	// Clear catch-all from other mailboxes in the same domain
	clearFilter := bson.M{
		"isCatchAll": true,
		"address":    bson.M{"$regex": "@" + regexp.QuoteMeta(domainName) + "$", "$options": "i"},
	}
	clearUpdate := bson.M{
		"$set": bson.M{
			"isCatchAll": false,
			"updatedAt":  time.Now().UTC(),
		},
	}
	_, err = m.collection().UpdateMany(ctx, clearFilter, clearUpdate)
	if err != nil {
		return fmt.Errorf("failed to clear catch-all: %w", err)
	}

	// Set this mailbox as catch-all
	filter := bson.M{"_id": string(id)}
	update := bson.M{
		"$set": bson.M{
			"isCatchAll": true,
			"updatedAt":  time.Now().UTC(),
		},
	}

	_, err = m.collection().UpdateOne(ctx, filter, update)
	if err != nil {
		return fmt.Errorf("failed to set catch-all mailbox: %w", err)
	}

	return nil
}

// ClearCatchAll removes the catch-all flag from a mailbox.
func (m *MailboxRepository) ClearCatchAll(ctx context.Context, id domain.ID) error {
	ctx = m.repo.getSessionContext(ctx)

	filter := bson.M{"_id": string(id)}
	update := bson.M{
		"$set": bson.M{
			"isCatchAll": false,
			"updatedAt":  time.Now().UTC(),
		},
	}

	result, err := m.collection().UpdateOne(ctx, filter, update)
	if err != nil {
		return fmt.Errorf("failed to clear catch-all: %w", err)
	}

	if result.MatchedCount == 0 {
		return domain.NewNotFoundError("mailbox", string(id))
	}

	return nil
}

// UpdateStats updates the mailbox statistics.
func (m *MailboxRepository) UpdateStats(ctx context.Context, id domain.ID, stats *repository.MailboxStatsUpdate) error {
	ctx = m.repo.getSessionContext(ctx)

	setFields := bson.M{"updatedAt": time.Now().UTC()}

	if stats.MessageCount != nil {
		setFields["messageCount"] = *stats.MessageCount
	}
	if stats.UnreadCount != nil {
		setFields["unreadCount"] = *stats.UnreadCount
	}
	if stats.TotalSize != nil {
		setFields["totalSize"] = *stats.TotalSize
	}

	filter := bson.M{"_id": string(id)}
	update := bson.M{"$set": setFields}

	result, err := m.collection().UpdateOne(ctx, filter, update)
	if err != nil {
		return fmt.Errorf("failed to update mailbox stats: %w", err)
	}

	if result.MatchedCount == 0 {
		return domain.NewNotFoundError("mailbox", string(id))
	}

	return nil
}

// IncrementMessageCount atomically increments message counters and assigns the next IMAP UID.
func (m *MailboxRepository) IncrementMessageCount(ctx context.Context, id domain.ID, size int64) (uint32, error) {
	ctx = m.repo.getSessionContext(ctx)

	filter := bson.M{"_id": string(id)}
	update := bson.M{
		"$inc": bson.M{
			"messageCount": 1,
			"unreadCount":  1,
			"totalSize":    size,
			"uidNext":      1,
		},
		"$set": bson.M{"updatedAt": time.Now().UTC()},
	}

	// FindOneAndUpdate returns the document BEFORE update, so uidNext is the assigned UID
	opts := options.FindOneAndUpdate().SetReturnDocument(options.Before)
	var doc struct {
		UIDNext uint32 `bson:"uidNext"`
	}
	err := m.collection().FindOneAndUpdate(ctx, filter, update, opts).Decode(&doc)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return 0, domain.NewNotFoundError("mailbox", string(id))
		}
		return 0, fmt.Errorf("failed to increment message count: %w", err)
	}

	return doc.UIDNext, nil
}

// DecrementMessageCount atomically decrements message counters.
func (m *MailboxRepository) DecrementMessageCount(ctx context.Context, id domain.ID, size int64, wasUnread bool) error {
	ctx = m.repo.getSessionContext(ctx)

	incFields := bson.M{
		"messageCount": -1,
		"totalSize":    -size,
	}

	if wasUnread {
		incFields["unreadCount"] = -1
	}

	filter := bson.M{"_id": string(id)}
	update := bson.M{
		"$inc": incFields,
		"$set": bson.M{"updatedAt": time.Now().UTC()},
	}

	result, err := m.collection().UpdateOne(ctx, filter, update)
	if err != nil {
		return fmt.Errorf("failed to decrement message count: %w", err)
	}

	if result.MatchedCount == 0 {
		return domain.NewNotFoundError("mailbox", string(id))
	}

	return nil
}

// UpdateUnreadCount atomically updates the unread count.
func (m *MailboxRepository) UpdateUnreadCount(ctx context.Context, id domain.ID, delta int) error {
	ctx = m.repo.getSessionContext(ctx)

	filter := bson.M{"_id": string(id)}
	update := bson.M{
		"$inc": bson.M{"unreadCount": delta},
		"$set": bson.M{"updatedAt": time.Now().UTC()},
	}

	result, err := m.collection().UpdateOne(ctx, filter, update)
	if err != nil {
		return fmt.Errorf("failed to update unread count: %w", err)
	}

	if result.MatchedCount == 0 {
		return domain.NewNotFoundError("mailbox", string(id))
	}

	return nil
}

// RecalculateStats recalculates mailbox statistics from messages.
func (m *MailboxRepository) RecalculateStats(ctx context.Context, id domain.ID) error {
	ctx = m.repo.getSessionContext(ctx)

	// Check if mailbox exists
	exists, err := m.Exists(ctx, id)
	if err != nil {
		return err
	}
	if !exists {
		return domain.NewNotFoundError("mailbox", string(id))
	}

	messagesCollection := m.repo.collection(CollectionMessages)

	// Aggregate to get stats
	pipeline := mongo.Pipeline{
		{{Key: "$match", Value: bson.M{"mailboxId": string(id)}}},
		{{Key: "$group", Value: bson.M{
			"_id":          nil,
			"messageCount": bson.M{"$sum": 1},
			"unreadCount": bson.M{"$sum": bson.M{
				"$cond": bson.A{bson.M{"$eq": bson.A{"$status", "unread"}}, 1, 0},
			}},
			"totalSize": bson.M{"$sum": "$size"},
		}}},
	}

	cursor, err := messagesCollection.Aggregate(ctx, pipeline)
	if err != nil {
		return fmt.Errorf("failed to recalculate stats: %w", err)
	}
	defer cursor.Close(ctx)

	var stats struct {
		MessageCount int64 `bson:"messageCount"`
		UnreadCount  int64 `bson:"unreadCount"`
		TotalSize    int64 `bson:"totalSize"`
	}

	if cursor.Next(ctx) {
		if err := cursor.Decode(&stats); err != nil {
			return fmt.Errorf("failed to decode stats: %w", err)
		}
	}

	// Update the mailbox with new stats
	filter := bson.M{"_id": string(id)}
	update := bson.M{
		"$set": bson.M{
			"messageCount": stats.MessageCount,
			"unreadCount":  stats.UnreadCount,
			"totalSize":    stats.TotalSize,
			"updatedAt":    time.Now().UTC(),
		},
	}

	_, err = m.collection().UpdateOne(ctx, filter, update)
	if err != nil {
		return fmt.Errorf("failed to update mailbox stats: %w", err)
	}

	return nil
}

// GetStats retrieves detailed statistics for a mailbox.
func (m *MailboxRepository) GetStats(ctx context.Context, id domain.ID) (*domain.MailboxStats, error) {
	mailbox, err := m.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	return &domain.MailboxStats{
		TotalMessages:  mailbox.MessageCount,
		UnreadMessages: mailbox.UnreadCount,
		TotalSize:      mailbox.TotalSize,
	}, nil
}

// GetStatsByUser retrieves aggregated statistics for all mailboxes owned by a user.
func (m *MailboxRepository) GetStatsByUser(ctx context.Context, userID domain.ID) (*domain.MailboxStats, error) {
	ctx = m.repo.getSessionContext(ctx)

	pipeline := mongo.Pipeline{
		{{Key: "$match", Value: bson.M{"userId": string(userID)}}},
		{{Key: "$group", Value: bson.M{
			"_id":            nil,
			"totalMessages":  bson.M{"$sum": "$messageCount"},
			"unreadMessages": bson.M{"$sum": "$unreadCount"},
			"totalSize":      bson.M{"$sum": "$totalSize"},
		}}},
	}

	cursor, err := m.collection().Aggregate(ctx, pipeline)
	if err != nil {
		return nil, fmt.Errorf("failed to get user stats: %w", err)
	}
	defer cursor.Close(ctx)

	stats := &domain.MailboxStats{}
	if cursor.Next(ctx) {
		var doc struct {
			TotalMessages  int64 `bson:"totalMessages"`
			UnreadMessages int64 `bson:"unreadMessages"`
			TotalSize      int64 `bson:"totalSize"`
		}
		if err := cursor.Decode(&doc); err != nil {
			return nil, fmt.Errorf("failed to decode stats: %w", err)
		}
		stats.TotalMessages = doc.TotalMessages
		stats.UnreadMessages = doc.UnreadMessages
		stats.TotalSize = doc.TotalSize
	}

	return stats, nil
}

// GetTotalStats retrieves aggregated statistics for all mailboxes.
func (m *MailboxRepository) GetTotalStats(ctx context.Context) (*domain.MailboxStats, error) {
	ctx = m.repo.getSessionContext(ctx)

	pipeline := mongo.Pipeline{
		{{Key: "$group", Value: bson.M{
			"_id":            nil,
			"totalMessages":  bson.M{"$sum": "$messageCount"},
			"unreadMessages": bson.M{"$sum": "$unreadCount"},
			"totalSize":      bson.M{"$sum": "$totalSize"},
		}}},
	}

	cursor, err := m.collection().Aggregate(ctx, pipeline)
	if err != nil {
		return nil, fmt.Errorf("failed to get total stats: %w", err)
	}
	defer cursor.Close(ctx)

	stats := &domain.MailboxStats{}
	if cursor.Next(ctx) {
		var doc struct {
			TotalMessages  int64 `bson:"totalMessages"`
			UnreadMessages int64 `bson:"unreadMessages"`
			TotalSize      int64 `bson:"totalSize"`
		}
		if err := cursor.Decode(&doc); err != nil {
			return nil, fmt.Errorf("failed to decode stats: %w", err)
		}
		stats.TotalMessages = doc.TotalMessages
		stats.UnreadMessages = doc.UnreadMessages
		stats.TotalSize = doc.TotalSize
	}

	return stats, nil
}

// FindMatchingMailbox finds the mailbox that should receive a message for the given address.
func (m *MailboxRepository) FindMatchingMailbox(ctx context.Context, address string) (*domain.Mailbox, error) {
	// First try exact match
	mailbox, err := m.GetByAddress(ctx, address)
	if err == nil {
		return mailbox, nil
	}

	// If not found, try catch-all for the domain
	parts := strings.Split(address, "@")
	if len(parts) == 2 {
		mailbox, err = m.GetCatchAll(ctx, parts[1])
		if err == nil {
			return mailbox, nil
		}
	}

	return nil, domain.NewNotFoundError("mailbox", address)
}

// Search performs a text search across mailbox fields.
func (m *MailboxRepository) Search(ctx context.Context, query string, opts *repository.ListOptions) (*repository.ListResult[*domain.Mailbox], error) {
	filter := &repository.MailboxFilter{Search: query}
	return m.List(ctx, filter, opts)
}

// GetMailboxesWithMessages retrieves mailboxes that have at least one message.
func (m *MailboxRepository) GetMailboxesWithMessages(ctx context.Context, opts *repository.ListOptions) (*repository.ListResult[*domain.Mailbox], error) {
	hasMessages := true
	filter := &repository.MailboxFilter{HasMessages: &hasMessages}
	return m.List(ctx, filter, opts)
}

// GetMailboxesWithUnread retrieves mailboxes that have unread messages.
func (m *MailboxRepository) GetMailboxesWithUnread(ctx context.Context, opts *repository.ListOptions) (*repository.ListResult[*domain.Mailbox], error) {
	hasUnread := true
	filter := &repository.MailboxFilter{HasUnread: &hasUnread}
	return m.List(ctx, filter, opts)
}

// TransferOwnership transfers all mailboxes from one user to another.
func (m *MailboxRepository) TransferOwnership(ctx context.Context, fromUserID, toUserID domain.ID) (int64, error) {
	ctx = m.repo.getSessionContext(ctx)

	filter := bson.M{"userId": string(fromUserID)}
	update := bson.M{
		"$set": bson.M{
			"userId":    string(toUserID),
			"updatedAt": time.Now().UTC(),
		},
	}

	result, err := m.collection().UpdateMany(ctx, filter, update)
	if err != nil {
		return 0, fmt.Errorf("failed to transfer ownership: %w", err)
	}

	return result.ModifiedCount, nil
}

// BulkDelete permanently removes multiple mailboxes.
func (m *MailboxRepository) BulkDelete(ctx context.Context, ids []domain.ID) (*repository.BulkOperation, error) {
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

// GetDomains retrieves all unique domains from mailbox addresses.
func (m *MailboxRepository) GetDomains(ctx context.Context) ([]string, error) {
	ctx = m.repo.getSessionContext(ctx)

	pipeline := mongo.Pipeline{
		{{Key: "$project", Value: bson.M{
			"domain": bson.M{
				"$arrayElemAt": bson.A{
					bson.M{"$split": bson.A{"$address", "@"}},
					1,
				},
			},
		}}},
		{{Key: "$group", Value: bson.M{"_id": "$domain"}}},
		{{Key: "$sort", Value: bson.M{"_id": 1}}},
	}

	cursor, err := m.collection().Aggregate(ctx, pipeline)
	if err != nil {
		return nil, fmt.Errorf("failed to get domains: %w", err)
	}
	defer cursor.Close(ctx)

	var domains []string
	for cursor.Next(ctx) {
		var doc struct {
			ID string `bson:"_id"`
		}
		if err := cursor.Decode(&doc); err != nil {
			continue
		}
		if doc.ID != "" {
			domains = append(domains, doc.ID)
		}
	}

	return domains, nil
}

// GetMailboxesByDomain retrieves all mailboxes for a specific domain.
func (m *MailboxRepository) GetMailboxesByDomain(ctx context.Context, domainName string, opts *repository.ListOptions) (*repository.ListResult[*domain.Mailbox], error) {
	filter := &repository.MailboxFilter{Domain: domainName}
	return m.List(ctx, filter, opts)
}

// Ensure MailboxRepository implements repository.MailboxRepository
var _ repository.MailboxRepository = (*MailboxRepository)(nil)
