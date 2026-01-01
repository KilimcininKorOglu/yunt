package mongodb

import (
	"context"
	"fmt"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"yunt/internal/domain"
	"yunt/internal/repository"
)

// SearchRepository provides text search capabilities across collections.
// This file contains helper functions for MongoDB text search operations.

// SearchMessages performs a full-text search across messages.
func (m *MessageRepository) SearchMessages(ctx context.Context, query string, mailboxID *domain.ID, opts *repository.ListOptions) (*repository.ListResult[*domain.Message], error) {
	ctx = m.repo.getSessionContext(ctx)

	filter := bson.M{
		"$text": bson.M{"$search": query},
	}

	if mailboxID != nil {
		filter["mailboxId"] = string(*mailboxID)
	}

	findOpts := m.buildFindOptions(opts)

	// Add text score for relevance sorting
	findOpts.SetProjection(bson.M{
		"score": bson.M{"$meta": "textScore"},
	})
	findOpts.SetSort(bson.D{
		{Key: "score", Value: bson.M{"$meta": "textScore"}},
		{Key: "receivedAt", Value: -1},
	})

	total, err := m.collection().CountDocuments(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("failed to count search results: %w", err)
	}

	cursor, err := m.collection().Find(ctx, filter, findOpts)
	if err != nil {
		return nil, fmt.Errorf("failed to search messages: %w", err)
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

// AdvancedSearch performs an advanced search with multiple criteria.
type AdvancedSearchCriteria struct {
	// General text search query
	Query string

	// Specific field searches
	Subject     string
	FromAddress string
	ToAddress   string
	Body        string

	// Filters
	MailboxID      *domain.ID
	IsStarred      *bool
	IsSpam         *bool
	HasAttachments *bool
	Status         *domain.MessageStatus

	// Date range
	ReceivedAfter  *domain.Timestamp
	ReceivedBefore *domain.Timestamp

	// Size range
	MinSize *int64
	MaxSize *int64
}

// AdvancedSearchMessages performs an advanced search with multiple criteria.
func (m *MessageRepository) AdvancedSearchMessages(ctx context.Context, criteria *AdvancedSearchCriteria, opts *repository.ListOptions) (*repository.ListResult[*domain.Message], error) {
	ctx = m.repo.getSessionContext(ctx)

	filter := bson.M{}

	// Text search
	if criteria.Query != "" {
		filter["$text"] = bson.M{"$search": criteria.Query}
	}

	// Subject search
	if criteria.Subject != "" {
		filter["subject"] = bson.M{"$regex": criteria.Subject, "$options": "i"}
	}

	// From address
	if criteria.FromAddress != "" {
		filter["from.address"] = bson.M{"$regex": criteria.FromAddress, "$options": "i"}
	}

	// To address
	if criteria.ToAddress != "" {
		filter["to.address"] = bson.M{"$regex": criteria.ToAddress, "$options": "i"}
	}

	// Body content
	if criteria.Body != "" {
		filter["$or"] = bson.A{
			bson.M{"textBody": bson.M{"$regex": criteria.Body, "$options": "i"}},
			bson.M{"htmlBody": bson.M{"$regex": criteria.Body, "$options": "i"}},
		}
	}

	// Mailbox filter
	if criteria.MailboxID != nil {
		filter["mailboxId"] = string(*criteria.MailboxID)
	}

	// Boolean filters
	if criteria.IsStarred != nil {
		filter["isStarred"] = *criteria.IsStarred
	}

	if criteria.IsSpam != nil {
		filter["isSpam"] = *criteria.IsSpam
	}

	if criteria.HasAttachments != nil {
		if *criteria.HasAttachments {
			filter["attachmentCount"] = bson.M{"$gt": 0}
		} else {
			filter["attachmentCount"] = 0
		}
	}

	if criteria.Status != nil {
		filter["status"] = string(*criteria.Status)
	}

	// Date range
	if criteria.ReceivedAfter != nil || criteria.ReceivedBefore != nil {
		dateFilter := bson.M{}
		if criteria.ReceivedAfter != nil {
			dateFilter["$gt"] = criteria.ReceivedAfter.Time
		}
		if criteria.ReceivedBefore != nil {
			dateFilter["$lt"] = criteria.ReceivedBefore.Time
		}
		filter["receivedAt"] = dateFilter
	}

	// Size range
	if criteria.MinSize != nil || criteria.MaxSize != nil {
		sizeFilter := bson.M{}
		if criteria.MinSize != nil {
			sizeFilter["$gte"] = *criteria.MinSize
		}
		if criteria.MaxSize != nil {
			sizeFilter["$lte"] = *criteria.MaxSize
		}
		filter["size"] = sizeFilter
	}

	findOpts := m.buildFindOptions(opts)

	// If text search is used, add relevance scoring
	if criteria.Query != "" {
		findOpts.SetProjection(bson.M{
			"score": bson.M{"$meta": "textScore"},
		})
		findOpts.SetSort(bson.D{
			{Key: "score", Value: bson.M{"$meta": "textScore"}},
			{Key: "receivedAt", Value: -1},
		})
	}

	total, err := m.collection().CountDocuments(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("failed to count search results: %w", err)
	}

	cursor, err := m.collection().Find(ctx, filter, findOpts)
	if err != nil {
		return nil, fmt.Errorf("failed to search messages: %w", err)
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

// SearchAcrossCollections performs a search across multiple collections and returns combined results.
type GlobalSearchResult struct {
	Users       []*domain.User
	Mailboxes   []*domain.Mailbox
	Messages    []*domain.MessageSummary
	Attachments []*domain.AttachmentSummary
	Webhooks    []*domain.Webhook
}

// GlobalSearch performs a search across all searchable collections.
func (r *Repository) GlobalSearch(ctx context.Context, query string, limit int) (*GlobalSearchResult, error) {
	ctx = r.getSessionContext(ctx)

	result := &GlobalSearchResult{}
	perTypeLimit := limit / 5 // Divide limit across types
	if perTypeLimit < 1 {
		perTypeLimit = 1
	}

	listOpts := &repository.ListOptions{
		Pagination: &repository.PaginationOptions{
			Page:    1,
			PerPage: perTypeLimit,
		},
	}

	// Search users
	userResults, err := r.users.Search(ctx, query, listOpts)
	if err == nil && userResults != nil {
		result.Users = userResults.Items
	}

	// Search mailboxes
	mailboxResults, err := r.mailboxes.Search(ctx, query, listOpts)
	if err == nil && mailboxResults != nil {
		result.Mailboxes = mailboxResults.Items
	}

	// Search messages (summaries for performance)
	messageFilter := &repository.MessageFilter{Search: query}
	messageResults, err := r.messages.ListSummaries(ctx, messageFilter, listOpts)
	if err == nil && messageResults != nil {
		result.Messages = messageResults.Items
	}

	// Search attachments
	attachmentResults, err := r.attachments.Search(ctx, query, listOpts)
	if err == nil && attachmentResults != nil {
		summaries := make([]*domain.AttachmentSummary, len(attachmentResults.Items))
		for i, att := range attachmentResults.Items {
			summaries[i] = att.ToSummary()
		}
		result.Attachments = summaries
	}

	// Search webhooks
	webhookResults, err := r.webhooks.Search(ctx, query, listOpts)
	if err == nil && webhookResults != nil {
		result.Webhooks = webhookResults.Items
	}

	return result, nil
}

// CreateTextIndexes creates or updates text search indexes for all collections.
// This is called during repository initialization but can also be called to update indexes.
func (p *ConnectionPool) CreateTextIndexes(ctx context.Context) error {
	// Users collection
	if err := p.createTextIndex(ctx, CollectionUsers, bson.D{
		{Key: "username", Value: "text"},
		{Key: "email", Value: "text"},
		{Key: "displayName", Value: "text"},
	}, "users_text_search"); err != nil {
		return fmt.Errorf("failed to create users text index: %w", err)
	}

	// Mailboxes collection
	if err := p.createTextIndex(ctx, CollectionMailboxes, bson.D{
		{Key: "name", Value: "text"},
		{Key: "address", Value: "text"},
		{Key: "description", Value: "text"},
	}, "mailboxes_text_search"); err != nil {
		return fmt.Errorf("failed to create mailboxes text index: %w", err)
	}

	// Messages collection
	if err := p.createTextIndex(ctx, CollectionMessages, bson.D{
		{Key: "subject", Value: "text"},
		{Key: "textBody", Value: "text"},
		{Key: "htmlBody", Value: "text"},
		{Key: "from.address", Value: "text"},
		{Key: "from.name", Value: "text"},
	}, "messages_text_search"); err != nil {
		return fmt.Errorf("failed to create messages text index: %w", err)
	}

	// Attachments collection
	if err := p.createTextIndex(ctx, CollectionAttachments, bson.D{
		{Key: "filename", Value: "text"},
	}, "attachments_text_search"); err != nil {
		return fmt.Errorf("failed to create attachments text index: %w", err)
	}

	// Webhooks collection
	if err := p.createTextIndex(ctx, CollectionWebhooks, bson.D{
		{Key: "name", Value: "text"},
		{Key: "url", Value: "text"},
	}, "webhooks_text_search"); err != nil {
		return fmt.Errorf("failed to create webhooks text index: %w", err)
	}

	return nil
}

// createTextIndex creates a text index on a collection.
func (p *ConnectionPool) createTextIndex(ctx context.Context, collectionName string, keys bson.D, indexName string) error {
	collection := p.Collection(collectionName)
	if collection == nil {
		return fmt.Errorf("collection not found: %s", collectionName)
	}

	indexModel := mongo.IndexModel{
		Keys:    keys,
		Options: options.Index().SetName(indexName),
	}

	_, err := collection.Indexes().CreateOne(ctx, indexModel)
	return err
}

// DropTextIndexes drops all text search indexes.
func (p *ConnectionPool) DropTextIndexes(ctx context.Context) error {
	collections := []string{
		CollectionUsers,
		CollectionMailboxes,
		CollectionMessages,
		CollectionAttachments,
		CollectionWebhooks,
	}

	indexNames := []string{
		"users_text_search",
		"mailboxes_text_search",
		"messages_text_search",
		"attachments_text_search",
		"webhooks_text_search",
	}

	for i, collName := range collections {
		collection := p.Collection(collName)
		if collection == nil {
			continue
		}

		_, err := collection.Indexes().DropOne(ctx, indexNames[i])
		if err != nil {
			// Ignore "index not found" errors
			if !isIndexNotFoundError(err) {
				return fmt.Errorf("failed to drop index %s: %w", indexNames[i], err)
			}
		}
	}

	return nil
}

// isIndexNotFoundError checks if the error is an "index not found" error.
func isIndexNotFoundError(err error) bool {
	if err == nil {
		return false
	}
	// MongoDB returns this error when trying to drop a non-existent index
	return err.Error() == "index not found" ||
		err.Error() == "ns not found" ||
		(len(err.Error()) > 0 && err.Error()[0:5] == "index")
}

// GetSearchStats returns statistics about the text search indexes.
type SearchIndexStats struct {
	CollectionName string
	IndexName      string
	IndexSize      int64
	TotalDocuments int64
}

// GetSearchIndexStats returns statistics for all text search indexes.
func (p *ConnectionPool) GetSearchIndexStats(ctx context.Context) ([]SearchIndexStats, error) {
	collections := []string{
		CollectionUsers,
		CollectionMailboxes,
		CollectionMessages,
		CollectionAttachments,
		CollectionWebhooks,
	}

	var stats []SearchIndexStats

	for _, collName := range collections {
		collection := p.Collection(collName)
		if collection == nil {
			continue
		}

		// Get document count
		count, err := collection.CountDocuments(ctx, bson.M{})
		if err != nil {
			continue
		}

		// Get index info
		cursor, err := collection.Indexes().List(ctx)
		if err != nil {
			continue
		}
		defer cursor.Close(ctx)

		for cursor.Next(ctx) {
			var indexInfo bson.M
			if err := cursor.Decode(&indexInfo); err != nil {
				continue
			}

			name, ok := indexInfo["name"].(string)
			if !ok {
				continue
			}

			// Check if it's a text index
			if keys, ok := indexInfo["key"].(bson.M); ok {
				for _, v := range keys {
					if v == "text" {
						stats = append(stats, SearchIndexStats{
							CollectionName: collName,
							IndexName:      name,
							TotalDocuments: count,
						})
						break
					}
				}
			}
		}
	}

	return stats, nil
}
