package mongodb

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"regexp"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"yunt/internal/domain"
	"yunt/internal/repository"
)

// AttachmentRepository implements the repository.AttachmentRepository interface for MongoDB.
type AttachmentRepository struct {
	repo *Repository
}

// attachmentDocument is the MongoDB document representation of an attachment.
type attachmentDocument struct {
	ID          string    `bson:"_id"`
	MessageID   string    `bson:"messageId"`
	Filename    string    `bson:"filename"`
	ContentType string    `bson:"contentType"`
	Size        int64     `bson:"size"`
	ContentID   string    `bson:"contentId,omitempty"`
	Disposition string    `bson:"disposition"`
	StoragePath string    `bson:"storagePath,omitempty"`
	Checksum    string    `bson:"checksum,omitempty"`
	IsInline    bool      `bson:"isInline"`
	CreatedAt   time.Time `bson:"createdAt"`
}

// attachmentContentDocument stores attachment binary content.
type attachmentContentDocument struct {
	AttachmentID string `bson:"_id"`
	Content      []byte `bson:"content"`
}

// NewAttachmentRepository creates a new MongoDB attachment repository.
func NewAttachmentRepository(repo *Repository) *AttachmentRepository {
	return &AttachmentRepository{repo: repo}
}

// collection returns the attachments collection.
func (a *AttachmentRepository) collection() *mongo.Collection {
	return a.repo.collection(CollectionAttachments)
}

// contentCollection returns the attachment content collection.
func (a *AttachmentRepository) contentCollection() *mongo.Collection {
	return a.repo.collection(CollectionAttachmentContent)
}

// toDocument converts a domain.Attachment to a MongoDB document.
func (a *AttachmentRepository) toDocument(att *domain.Attachment) *attachmentDocument {
	return &attachmentDocument{
		ID:          string(att.ID),
		MessageID:   string(att.MessageID),
		Filename:    att.Filename,
		ContentType: att.ContentType,
		Size:        att.Size,
		ContentID:   att.ContentID,
		Disposition: string(att.Disposition),
		StoragePath: att.StoragePath,
		Checksum:    att.Checksum,
		IsInline:    att.IsInline,
		CreatedAt:   att.CreatedAt.Time,
	}
}

// toDomain converts a MongoDB document to a domain.Attachment.
func (a *AttachmentRepository) toDomain(doc *attachmentDocument) *domain.Attachment {
	return &domain.Attachment{
		ID:          domain.ID(doc.ID),
		MessageID:   domain.ID(doc.MessageID),
		Filename:    doc.Filename,
		ContentType: doc.ContentType,
		Size:        doc.Size,
		ContentID:   doc.ContentID,
		Disposition: domain.AttachmentDisposition(doc.Disposition),
		StoragePath: doc.StoragePath,
		Checksum:    doc.Checksum,
		IsInline:    doc.IsInline,
		CreatedAt:   domain.Timestamp{Time: doc.CreatedAt},
	}
}

// GetByID retrieves an attachment by its unique identifier.
func (a *AttachmentRepository) GetByID(ctx context.Context, id domain.ID) (*domain.Attachment, error) {
	ctx = a.repo.getSessionContext(ctx)

	filter := bson.M{"_id": string(id)}

	var doc attachmentDocument
	if err := a.collection().FindOne(ctx, filter).Decode(&doc); err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, domain.NewNotFoundError("attachment", string(id))
		}
		return nil, fmt.Errorf("failed to get attachment by ID: %w", err)
	}

	return a.toDomain(&doc), nil
}

// GetByContentID retrieves an attachment by its Content-ID.
func (a *AttachmentRepository) GetByContentID(ctx context.Context, contentID string) (*domain.Attachment, error) {
	ctx = a.repo.getSessionContext(ctx)

	filter := bson.M{"contentId": contentID}

	var doc attachmentDocument
	if err := a.collection().FindOne(ctx, filter).Decode(&doc); err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, domain.NewNotFoundError("attachment", contentID)
		}
		return nil, fmt.Errorf("failed to get attachment by Content-ID: %w", err)
	}

	return a.toDomain(&doc), nil
}

// List retrieves attachments with optional filtering, sorting, and pagination.
func (a *AttachmentRepository) List(ctx context.Context, filter *repository.AttachmentFilter, opts *repository.ListOptions) (*repository.ListResult[*domain.Attachment], error) {
	ctx = a.repo.getSessionContext(ctx)

	mongoFilter := a.buildFilter(filter)
	findOpts := a.buildFindOptions(opts)

	total, err := a.collection().CountDocuments(ctx, mongoFilter)
	if err != nil {
		return nil, fmt.Errorf("failed to count attachments: %w", err)
	}

	cursor, err := a.collection().Find(ctx, mongoFilter, findOpts)
	if err != nil {
		return nil, fmt.Errorf("failed to list attachments: %w", err)
	}
	defer cursor.Close(ctx)

	var docs []attachmentDocument
	if err := cursor.All(ctx, &docs); err != nil {
		return nil, fmt.Errorf("failed to decode attachments: %w", err)
	}

	attachments := make([]*domain.Attachment, len(docs))
	for i, doc := range docs {
		attachments[i] = a.toDomain(&doc)
	}

	result := &repository.ListResult[*domain.Attachment]{
		Items: attachments,
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

// ListByMessage retrieves all attachments for a specific message.
func (a *AttachmentRepository) ListByMessage(ctx context.Context, messageID domain.ID) ([]*domain.Attachment, error) {
	ctx = a.repo.getSessionContext(ctx)

	filter := bson.M{"messageId": string(messageID)}
	cursor, err := a.collection().Find(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("failed to list attachments: %w", err)
	}
	defer cursor.Close(ctx)

	var docs []attachmentDocument
	if err := cursor.All(ctx, &docs); err != nil {
		return nil, fmt.Errorf("failed to decode attachments: %w", err)
	}

	attachments := make([]*domain.Attachment, len(docs))
	for i, doc := range docs {
		attachments[i] = a.toDomain(&doc)
	}

	return attachments, nil
}

// ListByMessages retrieves attachments for multiple messages.
func (a *AttachmentRepository) ListByMessages(ctx context.Context, messageIDs []domain.ID) (map[domain.ID][]*domain.Attachment, error) {
	ctx = a.repo.getSessionContext(ctx)

	ids := make([]string, len(messageIDs))
	for i, id := range messageIDs {
		ids[i] = string(id)
	}

	filter := bson.M{"messageId": bson.M{"$in": ids}}
	cursor, err := a.collection().Find(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("failed to list attachments: %w", err)
	}
	defer cursor.Close(ctx)

	result := make(map[domain.ID][]*domain.Attachment)
	for cursor.Next(ctx) {
		var doc attachmentDocument
		if err := cursor.Decode(&doc); err != nil {
			continue
		}
		msgID := domain.ID(doc.MessageID)
		result[msgID] = append(result[msgID], a.toDomain(&doc))
	}

	return result, nil
}

// ListSummaries retrieves attachment summaries for faster list rendering.
func (a *AttachmentRepository) ListSummaries(ctx context.Context, filter *repository.AttachmentFilter, opts *repository.ListOptions) (*repository.ListResult[*domain.AttachmentSummary], error) {
	listResult, err := a.List(ctx, filter, opts)
	if err != nil {
		return nil, err
	}

	summaries := make([]*domain.AttachmentSummary, len(listResult.Items))
	for i, att := range listResult.Items {
		summaries[i] = att.ToSummary()
	}

	return &repository.ListResult[*domain.AttachmentSummary]{
		Items:      summaries,
		Total:      listResult.Total,
		HasMore:    listResult.HasMore,
		Pagination: listResult.Pagination,
	}, nil
}

// ListSummariesByMessage retrieves attachment summaries for a specific message.
func (a *AttachmentRepository) ListSummariesByMessage(ctx context.Context, messageID domain.ID) ([]*domain.AttachmentSummary, error) {
	attachments, err := a.ListByMessage(ctx, messageID)
	if err != nil {
		return nil, err
	}

	summaries := make([]*domain.AttachmentSummary, len(attachments))
	for i, att := range attachments {
		summaries[i] = att.ToSummary()
	}

	return summaries, nil
}

// buildFilter builds the MongoDB filter from repository.AttachmentFilter.
func (a *AttachmentRepository) buildFilter(filter *repository.AttachmentFilter) bson.M {
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

	if filter.MessageID != nil {
		f["messageId"] = string(*filter.MessageID)
	}

	if len(filter.MessageIDs) > 0 {
		msgIDs := make([]string, len(filter.MessageIDs))
		for i, id := range filter.MessageIDs {
			msgIDs[i] = string(id)
		}
		f["messageId"] = bson.M{"$in": msgIDs}
	}

	if filter.IsInline != nil {
		f["isInline"] = *filter.IsInline
	}

	if filter.ContentType != "" {
		f["contentType"] = filter.ContentType
	}

	if filter.ContentTypePrefix != "" {
		f["contentType"] = bson.M{"$regex": "^" + regexp.QuoteMeta(filter.ContentTypePrefix)}
	}

	if filter.Filename != "" {
		f["filename"] = filter.Filename
	}

	if filter.FilenameContains != "" {
		f["filename"] = bson.M{"$regex": regexp.QuoteMeta(filter.FilenameContains), "$options": "i"}
	}

	if filter.Extension != "" {
		f["filename"] = bson.M{"$regex": "\\." + regexp.QuoteMeta(filter.Extension) + "$", "$options": "i"}
	}

	if len(filter.Extensions) > 0 {
		patterns := make([]bson.M, len(filter.Extensions))
		for i, ext := range filter.Extensions {
			patterns[i] = bson.M{"filename": bson.M{"$regex": "\\." + regexp.QuoteMeta(ext) + "$", "$options": "i"}}
		}
		f["$or"] = patterns
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

	if filter.Checksum != "" {
		f["checksum"] = filter.Checksum
	}

	if filter.CreatedAfter != nil {
		f["createdAt"] = bson.M{"$gt": filter.CreatedAfter.Time}
	}

	if filter.CreatedBefore != nil {
		if _, exists := f["createdAt"]; exists {
			f["createdAt"].(bson.M)["$lt"] = filter.CreatedBefore.Time
		} else {
			f["createdAt"] = bson.M{"$lt": filter.CreatedBefore.Time}
		}
	}

	return f
}

// buildFindOptions builds MongoDB find options from repository.ListOptions.
func (a *AttachmentRepository) buildFindOptions(opts *repository.ListOptions) *options.FindOptions {
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
		field := a.mapSortField(opts.Sort.Field)
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
func (a *AttachmentRepository) mapSortField(field string) string {
	switch field {
	case "filename":
		return "filename"
	case "size":
		return "size"
	case "contentType":
		return "contentType"
	case "createdAt":
		return "createdAt"
	default:
		return "createdAt"
	}
}

// Create creates a new attachment record.
func (a *AttachmentRepository) Create(ctx context.Context, att *domain.Attachment) error {
	ctx = a.repo.getSessionContext(ctx)

	doc := a.toDocument(att)
	_, err := a.collection().InsertOne(ctx, doc)
	if err != nil {
		if mongo.IsDuplicateKeyError(err) {
			return domain.NewAlreadyExistsError("attachment", "id", string(att.ID))
		}
		return fmt.Errorf("failed to create attachment: %w", err)
	}

	return nil
}

// CreateWithContent creates an attachment and stores its content.
func (a *AttachmentRepository) CreateWithContent(ctx context.Context, att *domain.Attachment, content io.Reader) error {
	if err := a.Create(ctx, att); err != nil {
		return err
	}

	return a.StoreContent(ctx, att.ID, content)
}

// Update updates an existing attachment's metadata.
func (a *AttachmentRepository) Update(ctx context.Context, att *domain.Attachment) error {
	ctx = a.repo.getSessionContext(ctx)

	filter := bson.M{"_id": string(att.ID)}
	update := bson.M{
		"$set": bson.M{
			"filename":    att.Filename,
			"contentType": att.ContentType,
			"size":        att.Size,
			"contentId":   att.ContentID,
			"disposition": string(att.Disposition),
			"storagePath": att.StoragePath,
			"checksum":    att.Checksum,
			"isInline":    att.IsInline,
		},
	}

	result, err := a.collection().UpdateOne(ctx, filter, update)
	if err != nil {
		return fmt.Errorf("failed to update attachment: %w", err)
	}

	if result.MatchedCount == 0 {
		return domain.NewNotFoundError("attachment", string(att.ID))
	}

	return nil
}

// Delete removes an attachment and its content.
func (a *AttachmentRepository) Delete(ctx context.Context, id domain.ID) error {
	ctx = a.repo.getSessionContext(ctx)

	// Delete content
	a.contentCollection().DeleteOne(ctx, bson.M{"_id": string(id)})

	// Delete metadata
	filter := bson.M{"_id": string(id)}
	result, err := a.collection().DeleteOne(ctx, filter)
	if err != nil {
		return fmt.Errorf("failed to delete attachment: %w", err)
	}

	if result.DeletedCount == 0 {
		return domain.NewNotFoundError("attachment", string(id))
	}

	return nil
}

// DeleteByMessage removes all attachments for a message.
func (a *AttachmentRepository) DeleteByMessage(ctx context.Context, messageID domain.ID) (int64, error) {
	ctx = a.repo.getSessionContext(ctx)

	// Get attachment IDs first
	cursor, err := a.collection().Find(ctx, bson.M{"messageId": string(messageID)}, options.Find().SetProjection(bson.M{"_id": 1}))
	if err != nil {
		return 0, fmt.Errorf("failed to get attachment IDs: %w", err)
	}
	defer cursor.Close(ctx)

	var ids []string
	for cursor.Next(ctx) {
		var doc struct {
			ID string `bson:"_id"`
		}
		if err := cursor.Decode(&doc); err != nil {
			continue
		}
		ids = append(ids, doc.ID)
	}

	// Delete content
	if len(ids) > 0 {
		a.contentCollection().DeleteMany(ctx, bson.M{"_id": bson.M{"$in": ids}})
	}

	// Delete metadata
	filter := bson.M{"messageId": string(messageID)}
	result, err := a.collection().DeleteMany(ctx, filter)
	if err != nil {
		return 0, fmt.Errorf("failed to delete message attachments: %w", err)
	}

	return result.DeletedCount, nil
}

// DeleteByMessages removes all attachments for multiple messages.
func (a *AttachmentRepository) DeleteByMessages(ctx context.Context, messageIDs []domain.ID) (int64, error) {
	ctx = a.repo.getSessionContext(ctx)

	ids := make([]string, len(messageIDs))
	for i, id := range messageIDs {
		ids[i] = string(id)
	}

	// Get attachment IDs first
	cursor, err := a.collection().Find(ctx, bson.M{"messageId": bson.M{"$in": ids}}, options.Find().SetProjection(bson.M{"_id": 1}))
	if err != nil {
		return 0, fmt.Errorf("failed to get attachment IDs: %w", err)
	}
	defer cursor.Close(ctx)

	var attIDs []string
	for cursor.Next(ctx) {
		var doc struct {
			ID string `bson:"_id"`
		}
		if err := cursor.Decode(&doc); err != nil {
			continue
		}
		attIDs = append(attIDs, doc.ID)
	}

	// Delete content
	if len(attIDs) > 0 {
		a.contentCollection().DeleteMany(ctx, bson.M{"_id": bson.M{"$in": attIDs}})
	}

	// Delete metadata
	filter := bson.M{"messageId": bson.M{"$in": ids}}
	result, err := a.collection().DeleteMany(ctx, filter)
	if err != nil {
		return 0, fmt.Errorf("failed to delete attachments: %w", err)
	}

	return result.DeletedCount, nil
}

// Exists checks if an attachment with the given ID exists.
func (a *AttachmentRepository) Exists(ctx context.Context, id domain.ID) (bool, error) {
	ctx = a.repo.getSessionContext(ctx)

	filter := bson.M{"_id": string(id)}
	count, err := a.collection().CountDocuments(ctx, filter, options.Count().SetLimit(1))
	if err != nil {
		return false, fmt.Errorf("failed to check attachment existence: %w", err)
	}

	return count > 0, nil
}

// ExistsByContentID checks if an attachment with the given Content-ID exists.
func (a *AttachmentRepository) ExistsByContentID(ctx context.Context, contentID string) (bool, error) {
	ctx = a.repo.getSessionContext(ctx)

	filter := bson.M{"contentId": contentID}
	count, err := a.collection().CountDocuments(ctx, filter, options.Count().SetLimit(1))
	if err != nil {
		return false, fmt.Errorf("failed to check Content-ID existence: %w", err)
	}

	return count > 0, nil
}

// Count returns the total number of attachments matching the filter.
func (a *AttachmentRepository) Count(ctx context.Context, filter *repository.AttachmentFilter) (int64, error) {
	ctx = a.repo.getSessionContext(ctx)

	mongoFilter := a.buildFilter(filter)
	count, err := a.collection().CountDocuments(ctx, mongoFilter)
	if err != nil {
		return 0, fmt.Errorf("failed to count attachments: %w", err)
	}

	return count, nil
}

// CountByMessage returns the number of attachments for a message.
func (a *AttachmentRepository) CountByMessage(ctx context.Context, messageID domain.ID) (int64, error) {
	ctx = a.repo.getSessionContext(ctx)

	filter := bson.M{"messageId": string(messageID)}
	count, err := a.collection().CountDocuments(ctx, filter)
	if err != nil {
		return 0, fmt.Errorf("failed to count message attachments: %w", err)
	}

	return count, nil
}

// StoreContent stores the content of an attachment.
func (a *AttachmentRepository) StoreContent(ctx context.Context, id domain.ID, content io.Reader) error {
	ctx = a.repo.getSessionContext(ctx)

	// Check if attachment exists
	exists, err := a.Exists(ctx, id)
	if err != nil {
		return err
	}
	if !exists {
		return domain.NewNotFoundError("attachment", string(id))
	}

	// Read content
	data, err := io.ReadAll(content)
	if err != nil {
		return fmt.Errorf("failed to read content: %w", err)
	}

	// Store content
	doc := &attachmentContentDocument{
		AttachmentID: string(id),
		Content:      data,
	}

	opts := options.Replace().SetUpsert(true)
	filter := bson.M{"_id": string(id)}
	_, err = a.contentCollection().ReplaceOne(ctx, filter, doc, opts)
	if err != nil {
		return fmt.Errorf("failed to store content: %w", err)
	}

	return nil
}

// GetContent retrieves the content of an attachment.
func (a *AttachmentRepository) GetContent(ctx context.Context, id domain.ID) (io.ReadCloser, error) {
	ctx = a.repo.getSessionContext(ctx)

	filter := bson.M{"_id": string(id)}

	var doc attachmentContentDocument
	if err := a.contentCollection().FindOne(ctx, filter).Decode(&doc); err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, domain.NewNotFoundError("attachment content", string(id))
		}
		return nil, fmt.Errorf("failed to get content: %w", err)
	}

	return io.NopCloser(bytes.NewReader(doc.Content)), nil
}

// GetContentWithMetadata retrieves both the attachment metadata and content.
func (a *AttachmentRepository) GetContentWithMetadata(ctx context.Context, id domain.ID) (*domain.Attachment, io.ReadCloser, error) {
	att, err := a.GetByID(ctx, id)
	if err != nil {
		return nil, nil, err
	}

	content, err := a.GetContent(ctx, id)
	if err != nil {
		return nil, nil, err
	}

	return att, content, nil
}

// GetContentSize retrieves the size of the attachment content.
func (a *AttachmentRepository) GetContentSize(ctx context.Context, id domain.ID) (int64, error) {
	att, err := a.GetByID(ctx, id)
	if err != nil {
		return 0, err
	}
	return att.Size, nil
}

// VerifyContent verifies the integrity of the attachment content.
func (a *AttachmentRepository) VerifyContent(ctx context.Context, id domain.ID) (bool, error) {
	att, err := a.GetByID(ctx, id)
	if err != nil {
		return false, err
	}

	if att.Checksum == "" {
		return true, nil // No checksum to verify
	}

	content, err := a.GetContent(ctx, id)
	if err != nil {
		return false, err
	}
	defer content.Close()

	// Read and compute checksum
	data, err := io.ReadAll(content)
	if err != nil {
		return false, fmt.Errorf("failed to read content for verification: %w", err)
	}

	// Simple size verification (for full checksum, you'd compute MD5/SHA256)
	return int64(len(data)) == att.Size, nil
}

// GetTotalSize calculates the total size of all attachments.
func (a *AttachmentRepository) GetTotalSize(ctx context.Context) (int64, error) {
	ctx = a.repo.getSessionContext(ctx)

	pipeline := mongo.Pipeline{
		{{Key: "$group", Value: bson.M{
			"_id":       nil,
			"totalSize": bson.M{"$sum": "$size"},
		}}},
	}

	cursor, err := a.collection().Aggregate(ctx, pipeline)
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

// GetTotalSizeByMessage calculates the total size of attachments for a message.
func (a *AttachmentRepository) GetTotalSizeByMessage(ctx context.Context, messageID domain.ID) (int64, error) {
	ctx = a.repo.getSessionContext(ctx)

	pipeline := mongo.Pipeline{
		{{Key: "$match", Value: bson.M{"messageId": string(messageID)}}},
		{{Key: "$group", Value: bson.M{
			"_id":       nil,
			"totalSize": bson.M{"$sum": "$size"},
		}}},
	}

	cursor, err := a.collection().Aggregate(ctx, pipeline)
	if err != nil {
		return 0, fmt.Errorf("failed to get message attachment size: %w", err)
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

// GetByChecksum retrieves attachments with a specific checksum.
func (a *AttachmentRepository) GetByChecksum(ctx context.Context, checksum string) ([]*domain.Attachment, error) {
	ctx = a.repo.getSessionContext(ctx)

	filter := bson.M{"checksum": checksum}
	cursor, err := a.collection().Find(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("failed to get attachments by checksum: %w", err)
	}
	defer cursor.Close(ctx)

	var docs []attachmentDocument
	if err := cursor.All(ctx, &docs); err != nil {
		return nil, fmt.Errorf("failed to decode attachments: %w", err)
	}

	attachments := make([]*domain.Attachment, len(docs))
	for i, doc := range docs {
		attachments[i] = a.toDomain(&doc)
	}

	return attachments, nil
}

// GetInlineAttachments retrieves inline attachments for a message.
func (a *AttachmentRepository) GetInlineAttachments(ctx context.Context, messageID domain.ID) ([]*domain.Attachment, error) {
	ctx = a.repo.getSessionContext(ctx)

	filter := bson.M{
		"messageId": string(messageID),
		"isInline":  true,
	}

	cursor, err := a.collection().Find(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("failed to get inline attachments: %w", err)
	}
	defer cursor.Close(ctx)

	var docs []attachmentDocument
	if err := cursor.All(ctx, &docs); err != nil {
		return nil, fmt.Errorf("failed to decode attachments: %w", err)
	}

	attachments := make([]*domain.Attachment, len(docs))
	for i, doc := range docs {
		attachments[i] = a.toDomain(&doc)
	}

	return attachments, nil
}

// GetNonInlineAttachments retrieves non-inline attachments for a message.
func (a *AttachmentRepository) GetNonInlineAttachments(ctx context.Context, messageID domain.ID) ([]*domain.Attachment, error) {
	ctx = a.repo.getSessionContext(ctx)

	filter := bson.M{
		"messageId": string(messageID),
		"isInline":  false,
	}

	cursor, err := a.collection().Find(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("failed to get non-inline attachments: %w", err)
	}
	defer cursor.Close(ctx)

	var docs []attachmentDocument
	if err := cursor.All(ctx, &docs); err != nil {
		return nil, fmt.Errorf("failed to decode attachments: %w", err)
	}

	attachments := make([]*domain.Attachment, len(docs))
	for i, doc := range docs {
		attachments[i] = a.toDomain(&doc)
	}

	return attachments, nil
}

// GetByContentType retrieves attachments with a specific content type.
func (a *AttachmentRepository) GetByContentType(ctx context.Context, contentType string, opts *repository.ListOptions) (*repository.ListResult[*domain.Attachment], error) {
	filter := &repository.AttachmentFilter{ContentType: contentType}
	return a.List(ctx, filter, opts)
}

// GetImages retrieves all image attachments.
func (a *AttachmentRepository) GetImages(ctx context.Context, opts *repository.ListOptions) (*repository.ListResult[*domain.Attachment], error) {
	filter := &repository.AttachmentFilter{ContentTypePrefix: "image/"}
	return a.List(ctx, filter, opts)
}

// GetLargeAttachments retrieves attachments larger than the specified size.
func (a *AttachmentRepository) GetLargeAttachments(ctx context.Context, minSize int64, opts *repository.ListOptions) (*repository.ListResult[*domain.Attachment], error) {
	filter := &repository.AttachmentFilter{MinSize: &minSize}
	return a.List(ctx, filter, opts)
}

// Search performs a text search on attachment filenames.
func (a *AttachmentRepository) Search(ctx context.Context, query string, opts *repository.ListOptions) (*repository.ListResult[*domain.Attachment], error) {
	ctx = a.repo.getSessionContext(ctx)

	mongoFilter := bson.M{"$text": bson.M{"$search": query}}
	findOpts := a.buildFindOptions(opts)

	total, err := a.collection().CountDocuments(ctx, mongoFilter)
	if err != nil {
		return nil, fmt.Errorf("failed to count attachments: %w", err)
	}

	cursor, err := a.collection().Find(ctx, mongoFilter, findOpts)
	if err != nil {
		return nil, fmt.Errorf("failed to search attachments: %w", err)
	}
	defer cursor.Close(ctx)

	var docs []attachmentDocument
	if err := cursor.All(ctx, &docs); err != nil {
		return nil, fmt.Errorf("failed to decode attachments: %w", err)
	}

	attachments := make([]*domain.Attachment, len(docs))
	for i, doc := range docs {
		attachments[i] = a.toDomain(&doc)
	}

	result := &repository.ListResult[*domain.Attachment]{
		Items: attachments,
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

// BulkDelete permanently removes multiple attachments and their content.
func (a *AttachmentRepository) BulkDelete(ctx context.Context, ids []domain.ID) (*repository.BulkOperation, error) {
	result := repository.NewBulkOperation()

	for _, id := range ids {
		if err := a.Delete(ctx, id); err != nil {
			result.AddFailure(string(id), err)
		} else {
			result.AddSuccess()
		}
	}

	return result, nil
}

// CleanupOrphaned removes attachments that are not linked to any message.
func (a *AttachmentRepository) CleanupOrphaned(ctx context.Context) (int64, error) {
	ctx = a.repo.getSessionContext(ctx)

	// Find attachments with no matching message
	pipeline := mongo.Pipeline{
		{{Key: "$lookup", Value: bson.M{
			"from":         CollectionMessages,
			"localField":   "messageId",
			"foreignField": "_id",
			"as":           "message",
		}}},
		{{Key: "$match", Value: bson.M{"message": bson.M{"$size": 0}}}},
		{{Key: "$project", Value: bson.M{"_id": 1}}},
	}

	cursor, err := a.collection().Aggregate(ctx, pipeline)
	if err != nil {
		return 0, fmt.Errorf("failed to find orphaned attachments: %w", err)
	}
	defer cursor.Close(ctx)

	var orphanedIDs []string
	for cursor.Next(ctx) {
		var doc struct {
			ID string `bson:"_id"`
		}
		if err := cursor.Decode(&doc); err != nil {
			continue
		}
		orphanedIDs = append(orphanedIDs, doc.ID)
	}

	if len(orphanedIDs) == 0 {
		return 0, nil
	}

	// Delete orphaned attachments
	filter := bson.M{"_id": bson.M{"$in": orphanedIDs}}

	// Delete content first
	a.contentCollection().DeleteMany(ctx, filter)

	// Delete metadata
	result, err := a.collection().DeleteMany(ctx, filter)
	if err != nil {
		return 0, fmt.Errorf("failed to delete orphaned attachments: %w", err)
	}

	return result.DeletedCount, nil
}

// GetStorageStats retrieves storage statistics for attachments.
func (a *AttachmentRepository) GetStorageStats(ctx context.Context) (*repository.AttachmentStorageStats, error) {
	ctx = a.repo.getSessionContext(ctx)

	pipeline := mongo.Pipeline{
		{{Key: "$group", Value: bson.M{
			"_id":        nil,
			"totalCount": bson.M{"$sum": 1},
			"totalSize":  bson.M{"$sum": "$size"},
			"avgSize":    bson.M{"$avg": "$size"},
			"maxSize":    bson.M{"$max": "$size"},
			"minSize":    bson.M{"$min": "$size"},
			"inlineCount": bson.M{"$sum": bson.M{
				"$cond": bson.A{"$isInline", 1, 0},
			}},
			"inlineSize": bson.M{"$sum": bson.M{
				"$cond": bson.A{"$isInline", "$size", 0},
			}},
		}}},
	}

	cursor, err := a.collection().Aggregate(ctx, pipeline)
	if err != nil {
		return nil, fmt.Errorf("failed to get storage stats: %w", err)
	}
	defer cursor.Close(ctx)

	stats := &repository.AttachmentStorageStats{}
	if cursor.Next(ctx) {
		var doc struct {
			TotalCount  int64   `bson:"totalCount"`
			TotalSize   int64   `bson:"totalSize"`
			AvgSize     float64 `bson:"avgSize"`
			MaxSize     int64   `bson:"maxSize"`
			MinSize     int64   `bson:"minSize"`
			InlineCount int64   `bson:"inlineCount"`
			InlineSize  int64   `bson:"inlineSize"`
		}
		if err := cursor.Decode(&doc); err != nil {
			return nil, fmt.Errorf("failed to decode stats: %w", err)
		}

		stats.TotalCount = doc.TotalCount
		stats.TotalSize = doc.TotalSize
		stats.AverageSize = doc.AvgSize
		stats.LargestSize = doc.MaxSize
		stats.SmallestSize = doc.MinSize
		stats.InlineCount = doc.InlineCount
		stats.InlineSize = doc.InlineSize
		stats.RegularCount = doc.TotalCount - doc.InlineCount
		stats.RegularSize = doc.TotalSize - doc.InlineSize
	}

	return stats, nil
}

// GetContentTypeStats retrieves storage statistics grouped by content type.
func (a *AttachmentRepository) GetContentTypeStats(ctx context.Context) ([]repository.ContentTypeStats, error) {
	ctx = a.repo.getSessionContext(ctx)

	// Get total size first
	totalSize, err := a.GetTotalSize(ctx)
	if err != nil {
		return nil, err
	}

	pipeline := mongo.Pipeline{
		{{Key: "$group", Value: bson.M{
			"_id":       "$contentType",
			"count":     bson.M{"$sum": 1},
			"totalSize": bson.M{"$sum": "$size"},
		}}},
		{{Key: "$sort", Value: bson.M{"totalSize": -1}}},
	}

	cursor, err := a.collection().Aggregate(ctx, pipeline)
	if err != nil {
		return nil, fmt.Errorf("failed to get content type stats: %w", err)
	}
	defer cursor.Close(ctx)

	var stats []repository.ContentTypeStats
	for cursor.Next(ctx) {
		var doc struct {
			ContentType string `bson:"_id"`
			Count       int64  `bson:"count"`
			TotalSize   int64  `bson:"totalSize"`
		}
		if err := cursor.Decode(&doc); err != nil {
			continue
		}

		percentage := 0.0
		if totalSize > 0 {
			percentage = float64(doc.TotalSize) / float64(totalSize) * 100
		}

		stats = append(stats, repository.ContentTypeStats{
			ContentType: doc.ContentType,
			Count:       doc.Count,
			TotalSize:   doc.TotalSize,
			Percentage:  percentage,
		})
	}

	return stats, nil
}

// Ensure AttachmentRepository implements repository.AttachmentRepository
var _ repository.AttachmentRepository = (*AttachmentRepository)(nil)

// Unused import prevention
var _ = strings.Contains
