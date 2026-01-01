// Package repository provides the data access layer interfaces for the Yunt mail server.
// These interfaces define the contract for data persistence operations and are designed
// to be database-agnostic, allowing implementations for SQLite, PostgreSQL, MySQL, and MongoDB.
package repository

import (
	"yunt/internal/domain"
)

// ListOptions provides common options for listing operations.
// It includes pagination, sorting, and cursor-based navigation support.
type ListOptions struct {
	// Pagination contains page-based pagination parameters.
	Pagination *PaginationOptions

	// Cursor contains cursor-based pagination parameters.
	Cursor *CursorOptions

	// Sort specifies the sort field and order.
	Sort *SortOptions
}

// PaginationOptions contains page-based pagination parameters.
type PaginationOptions struct {
	// Page is the current page number (1-indexed).
	// A value of 0 or less defaults to 1.
	Page int

	// PerPage is the number of items per page.
	// A value of 0 or less defaults to the repository's default limit.
	// Values exceeding MaxPerPage will be capped.
	PerPage int
}

// DefaultPerPage is the default number of items per page.
const DefaultPerPage = 20

// MaxPerPage is the maximum allowed items per page.
const MaxPerPage = 100

// Normalize ensures pagination values are within valid bounds.
func (p *PaginationOptions) Normalize() {
	if p.Page < 1 {
		p.Page = 1
	}
	if p.PerPage < 1 {
		p.PerPage = DefaultPerPage
	}
	if p.PerPage > MaxPerPage {
		p.PerPage = MaxPerPage
	}
}

// Offset returns the database offset for the current page.
func (p *PaginationOptions) Offset() int {
	if p.Page <= 0 {
		return 0
	}
	return (p.Page - 1) * p.PerPage
}

// Limit returns the database limit (same as PerPage).
func (p *PaginationOptions) Limit() int {
	return p.PerPage
}

// CursorOptions contains cursor-based pagination parameters.
// Cursor-based pagination is more efficient for large datasets and provides
// consistent results when data changes between requests.
type CursorOptions struct {
	// After is the cursor pointing to the item after which to start fetching.
	// Items will be returned after this cursor position.
	After string

	// Before is the cursor pointing to the item before which to start fetching.
	// Items will be returned before this cursor position.
	Before string

	// First specifies the number of items to return from the beginning.
	// Used with After cursor for forward pagination.
	First int

	// Last specifies the number of items to return from the end.
	// Used with Before cursor for backward pagination.
	Last int
}

// Normalize ensures cursor options are within valid bounds.
func (c *CursorOptions) Normalize() {
	if c.First < 0 {
		c.First = 0
	}
	if c.First > MaxPerPage {
		c.First = MaxPerPage
	}
	if c.Last < 0 {
		c.Last = 0
	}
	if c.Last > MaxPerPage {
		c.Last = MaxPerPage
	}
}

// IsForward returns true if this is forward pagination (using After/First).
func (c *CursorOptions) IsForward() bool {
	return c.After != "" || (c.Before == "" && c.First > 0)
}

// IsBackward returns true if this is backward pagination (using Before/Last).
func (c *CursorOptions) IsBackward() bool {
	return c.Before != "" || (c.After == "" && c.Last > 0)
}

// SortOptions specifies sorting parameters.
type SortOptions struct {
	// Field is the name of the field to sort by.
	Field string

	// Order is the sort direction (asc or desc).
	Order domain.SortOrder
}

// NewSortOptions creates a new SortOptions with the given field and order.
func NewSortOptions(field string, order domain.SortOrder) *SortOptions {
	return &SortOptions{
		Field: field,
		Order: order,
	}
}

// ListResult is a generic result wrapper for paginated list operations.
// It includes the items, pagination info, and cursor information.
type ListResult[T any] struct {
	// Items contains the returned entities.
	Items []T

	// Total is the total count of items matching the query (before pagination).
	Total int64

	// HasMore indicates if there are more items available beyond the current page.
	HasMore bool

	// Pagination contains the pagination state with total pages calculated.
	Pagination *domain.Pagination

	// Cursors contains cursor information for cursor-based pagination.
	Cursors *CursorInfo
}

// CursorInfo contains cursor information for cursor-based pagination responses.
type CursorInfo struct {
	// StartCursor is the cursor of the first item in the current page.
	StartCursor string

	// EndCursor is the cursor of the last item in the current page.
	EndCursor string

	// HasNextPage indicates if there are more items after the current page.
	HasNextPage bool

	// HasPreviousPage indicates if there are items before the current page.
	HasPreviousPage bool
}

// SearchOptions provides text search configuration.
type SearchOptions struct {
	// Query is the search query string.
	Query string

	// Fields specifies which fields to search in.
	// If empty, the repository will use default searchable fields.
	Fields []string

	// Fuzzy enables fuzzy/approximate matching.
	Fuzzy bool

	// MatchAll requires all terms to match when true.
	// When false, any matching term satisfies the search.
	MatchAll bool

	// Highlight enables highlighting of matched terms in results.
	Highlight bool
}

// IsEmpty returns true if the search query is empty.
func (s *SearchOptions) IsEmpty() bool {
	return s.Query == ""
}

// DateRangeFilter provides filtering by date range.
type DateRangeFilter struct {
	// From specifies the start of the date range (inclusive).
	From *domain.Timestamp

	// To specifies the end of the date range (inclusive).
	To *domain.Timestamp
}

// IsEmpty returns true if no date range is specified.
func (d *DateRangeFilter) IsEmpty() bool {
	return d.From == nil && d.To == nil
}

// SizeRangeFilter provides filtering by size range (in bytes).
type SizeRangeFilter struct {
	// MinSize is the minimum size in bytes (inclusive).
	MinSize *int64

	// MaxSize is the maximum size in bytes (inclusive).
	MaxSize *int64
}

// IsEmpty returns true if no size range is specified.
func (s *SizeRangeFilter) IsEmpty() bool {
	return s.MinSize == nil && s.MaxSize == nil
}

// BulkOperation represents the result of a bulk operation.
type BulkOperation struct {
	// Succeeded is the number of items that were successfully processed.
	Succeeded int64

	// Failed is the number of items that failed processing.
	Failed int64

	// Errors contains error details for failed items.
	// Key is the item identifier, value is the error message.
	Errors map[string]string
}

// NewBulkOperation creates a new BulkOperation result.
func NewBulkOperation() *BulkOperation {
	return &BulkOperation{
		Errors: make(map[string]string),
	}
}

// AddSuccess increments the success count.
func (b *BulkOperation) AddSuccess() {
	b.Succeeded++
}

// AddFailure records a failed operation.
func (b *BulkOperation) AddFailure(id string, err error) {
	b.Failed++
	if err != nil {
		b.Errors[id] = err.Error()
	}
}

// HasErrors returns true if any operations failed.
func (b *BulkOperation) HasErrors() bool {
	return b.Failed > 0
}

// Total returns the total number of operations attempted.
func (b *BulkOperation) Total() int64 {
	return b.Succeeded + b.Failed
}

// TransactionOptions configures transaction behavior.
type TransactionOptions struct {
	// ReadOnly indicates if the transaction is read-only.
	ReadOnly bool

	// IsolationLevel specifies the transaction isolation level.
	IsolationLevel IsolationLevel
}

// IsolationLevel represents database transaction isolation levels.
type IsolationLevel int

const (
	// IsolationDefault uses the database's default isolation level.
	IsolationDefault IsolationLevel = iota

	// IsolationReadUncommitted allows reading uncommitted changes from other transactions.
	IsolationReadUncommitted

	// IsolationReadCommitted only reads committed changes from other transactions.
	IsolationReadCommitted

	// IsolationRepeatableRead ensures consistent reads within the same transaction.
	IsolationRepeatableRead

	// IsolationSerializable provides the highest isolation level with full serializability.
	IsolationSerializable
)
