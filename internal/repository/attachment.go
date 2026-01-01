package repository

import (
	"context"
	"io"

	"yunt/internal/domain"
)

// AttachmentRepository provides data access operations for Attachment entities.
// It supports CRUD operations, content storage, and attachment retrieval.
type AttachmentRepository interface {
	// GetByID retrieves an attachment by its unique identifier.
	// Returns domain.ErrNotFound if the attachment does not exist.
	GetByID(ctx context.Context, id domain.ID) (*domain.Attachment, error)

	// GetByContentID retrieves an attachment by its Content-ID (for inline images).
	// Returns domain.ErrNotFound if no attachment with the Content-ID exists.
	GetByContentID(ctx context.Context, contentID string) (*domain.Attachment, error)

	// List retrieves attachments with optional filtering, sorting, and pagination.
	// Returns an empty slice if no attachments match the criteria.
	List(ctx context.Context, filter *AttachmentFilter, opts *ListOptions) (*ListResult[*domain.Attachment], error)

	// ListByMessage retrieves all attachments for a specific message.
	ListByMessage(ctx context.Context, messageID domain.ID) ([]*domain.Attachment, error)

	// ListByMessages retrieves attachments for multiple messages.
	// Returns a map of message ID to attachments.
	ListByMessages(ctx context.Context, messageIDs []domain.ID) (map[domain.ID][]*domain.Attachment, error)

	// ListSummaries retrieves attachment summaries for faster list rendering.
	ListSummaries(ctx context.Context, filter *AttachmentFilter, opts *ListOptions) (*ListResult[*domain.AttachmentSummary], error)

	// ListSummariesByMessage retrieves attachment summaries for a specific message.
	ListSummariesByMessage(ctx context.Context, messageID domain.ID) ([]*domain.AttachmentSummary, error)

	// Create creates a new attachment record (metadata only).
	// Content should be stored separately using StoreContent.
	Create(ctx context.Context, attachment *domain.Attachment) error

	// CreateWithContent creates an attachment and stores its content.
	// This is a convenience method that combines Create and StoreContent.
	CreateWithContent(ctx context.Context, attachment *domain.Attachment, content io.Reader) error

	// Update updates an existing attachment's metadata.
	// Returns domain.ErrNotFound if the attachment does not exist.
	Update(ctx context.Context, attachment *domain.Attachment) error

	// Delete removes an attachment and its content.
	// Returns domain.ErrNotFound if the attachment does not exist.
	Delete(ctx context.Context, id domain.ID) error

	// DeleteByMessage removes all attachments for a message.
	// Returns the number of deleted attachments.
	DeleteByMessage(ctx context.Context, messageID domain.ID) (int64, error)

	// DeleteByMessages removes all attachments for multiple messages.
	// Returns the total number of deleted attachments.
	DeleteByMessages(ctx context.Context, messageIDs []domain.ID) (int64, error)

	// Exists checks if an attachment with the given ID exists.
	Exists(ctx context.Context, id domain.ID) (bool, error)

	// ExistsByContentID checks if an attachment with the given Content-ID exists.
	ExistsByContentID(ctx context.Context, contentID string) (bool, error)

	// Count returns the total number of attachments matching the filter.
	Count(ctx context.Context, filter *AttachmentFilter) (int64, error)

	// CountByMessage returns the number of attachments for a message.
	CountByMessage(ctx context.Context, messageID domain.ID) (int64, error)

	// StoreContent stores the content of an attachment.
	// Returns domain.ErrNotFound if the attachment does not exist.
	StoreContent(ctx context.Context, id domain.ID, content io.Reader) error

	// GetContent retrieves the content of an attachment.
	// The caller is responsible for closing the returned ReadCloser.
	// Returns domain.ErrNotFound if the attachment or its content does not exist.
	GetContent(ctx context.Context, id domain.ID) (io.ReadCloser, error)

	// GetContentWithMetadata retrieves both the attachment metadata and content.
	// Returns domain.ErrNotFound if the attachment does not exist.
	GetContentWithMetadata(ctx context.Context, id domain.ID) (*domain.Attachment, io.ReadCloser, error)

	// GetContentSize retrieves the size of the attachment content.
	// This may be more efficient than reading the full content.
	// Returns domain.ErrNotFound if the attachment does not exist.
	GetContentSize(ctx context.Context, id domain.ID) (int64, error)

	// VerifyContent verifies the integrity of the attachment content.
	// Compares the stored checksum with a newly calculated one.
	// Returns true if the content is valid, false if corrupted.
	// Returns domain.ErrNotFound if the attachment does not exist.
	VerifyContent(ctx context.Context, id domain.ID) (bool, error)

	// GetTotalSize calculates the total size of all attachments.
	GetTotalSize(ctx context.Context) (int64, error)

	// GetTotalSizeByMessage calculates the total size of attachments for a message.
	GetTotalSizeByMessage(ctx context.Context, messageID domain.ID) (int64, error)

	// GetByChecksum retrieves attachments with a specific checksum.
	// Useful for deduplication scenarios.
	GetByChecksum(ctx context.Context, checksum string) ([]*domain.Attachment, error)

	// GetInlineAttachments retrieves inline attachments for a message.
	GetInlineAttachments(ctx context.Context, messageID domain.ID) ([]*domain.Attachment, error)

	// GetNonInlineAttachments retrieves non-inline (regular) attachments for a message.
	GetNonInlineAttachments(ctx context.Context, messageID domain.ID) ([]*domain.Attachment, error)

	// GetByContentType retrieves attachments with a specific content type.
	GetByContentType(ctx context.Context, contentType string, opts *ListOptions) (*ListResult[*domain.Attachment], error)

	// GetImages retrieves all image attachments.
	GetImages(ctx context.Context, opts *ListOptions) (*ListResult[*domain.Attachment], error)

	// GetLargeAttachments retrieves attachments larger than the specified size.
	GetLargeAttachments(ctx context.Context, minSize int64, opts *ListOptions) (*ListResult[*domain.Attachment], error)

	// Search performs a text search on attachment filenames.
	Search(ctx context.Context, query string, opts *ListOptions) (*ListResult[*domain.Attachment], error)

	// BulkDelete permanently removes multiple attachments and their content.
	BulkDelete(ctx context.Context, ids []domain.ID) (*BulkOperation, error)

	// CleanupOrphaned removes attachments that are not linked to any message.
	// This can happen if a message creation fails after attachments are stored.
	// Returns the number of deleted attachments.
	CleanupOrphaned(ctx context.Context) (int64, error)

	// GetStorageStats retrieves storage statistics for attachments.
	GetStorageStats(ctx context.Context) (*AttachmentStorageStats, error)

	// GetContentTypeStats retrieves storage statistics grouped by content type.
	GetContentTypeStats(ctx context.Context) ([]ContentTypeStats, error)
}

// AttachmentFilter provides filtering options for attachment queries.
type AttachmentFilter struct {
	// IDs filters by specific attachment IDs.
	IDs []domain.ID

	// MessageID filters by message ID.
	MessageID *domain.ID

	// MessageIDs filters by multiple message IDs (OR condition).
	MessageIDs []domain.ID

	// IsInline filters by inline status.
	IsInline *bool

	// ContentType filters by content type (exact match).
	ContentType string

	// ContentTypePrefix filters by content type prefix (e.g., "image/").
	ContentTypePrefix string

	// Filename filters by exact filename match.
	Filename string

	// FilenameContains filters by partial filename match.
	FilenameContains string

	// Extension filters by file extension (without dot).
	Extension string

	// Extensions filters by multiple file extensions (OR condition).
	Extensions []string

	// MinSize filters attachments larger than this size (bytes).
	MinSize *int64

	// MaxSize filters attachments smaller than this size (bytes).
	MaxSize *int64

	// Checksum filters by checksum value.
	Checksum string

	// CreatedAfter filters attachments created after this timestamp.
	CreatedAfter *domain.Timestamp

	// CreatedBefore filters attachments created before this timestamp.
	CreatedBefore *domain.Timestamp
}

// IsEmpty returns true if no filter criteria are set.
func (f *AttachmentFilter) IsEmpty() bool {
	if f == nil {
		return true
	}
	return len(f.IDs) == 0 &&
		f.MessageID == nil &&
		len(f.MessageIDs) == 0 &&
		f.IsInline == nil &&
		f.ContentType == "" &&
		f.ContentTypePrefix == "" &&
		f.Filename == "" &&
		f.FilenameContains == "" &&
		f.Extension == "" &&
		len(f.Extensions) == 0 &&
		f.MinSize == nil &&
		f.MaxSize == nil &&
		f.Checksum == "" &&
		f.CreatedAfter == nil &&
		f.CreatedBefore == nil
}

// AttachmentSortField represents the available fields for sorting attachments.
type AttachmentSortField string

const (
	// AttachmentSortByFilename sorts by filename.
	AttachmentSortByFilename AttachmentSortField = "filename"

	// AttachmentSortBySize sorts by file size.
	AttachmentSortBySize AttachmentSortField = "size"

	// AttachmentSortByContentType sorts by content type.
	AttachmentSortByContentType AttachmentSortField = "contentType"

	// AttachmentSortByCreatedAt sorts by creation timestamp.
	AttachmentSortByCreatedAt AttachmentSortField = "createdAt"
)

// IsValid returns true if the sort field is a recognized value.
func (f AttachmentSortField) IsValid() bool {
	switch f {
	case AttachmentSortByFilename, AttachmentSortBySize,
		AttachmentSortByContentType, AttachmentSortByCreatedAt:
		return true
	default:
		return false
	}
}

// String returns the string representation of the sort field.
func (f AttachmentSortField) String() string {
	return string(f)
}

// AttachmentStorageStats represents storage statistics for attachments.
type AttachmentStorageStats struct {
	// TotalCount is the total number of attachments.
	TotalCount int64

	// TotalSize is the total size of all attachments in bytes.
	TotalSize int64

	// InlineCount is the number of inline attachments.
	InlineCount int64

	// InlineSize is the total size of inline attachments in bytes.
	InlineSize int64

	// RegularCount is the number of regular (non-inline) attachments.
	RegularCount int64

	// RegularSize is the total size of regular attachments in bytes.
	RegularSize int64

	// AverageSize is the average attachment size in bytes.
	AverageSize float64

	// LargestSize is the size of the largest attachment in bytes.
	LargestSize int64

	// SmallestSize is the size of the smallest attachment in bytes.
	SmallestSize int64
}

// ContentTypeStats represents storage statistics for a specific content type.
type ContentTypeStats struct {
	// ContentType is the MIME content type.
	ContentType string

	// Count is the number of attachments with this content type.
	Count int64

	// TotalSize is the total size of attachments with this content type.
	TotalSize int64

	// Percentage is the percentage of total storage used.
	Percentage float64
}
