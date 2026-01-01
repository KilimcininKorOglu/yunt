// Package service provides business logic and service layer implementations
// for the Yunt mail server.
package service

import (
	"context"
	"errors"

	"yunt/internal/domain"
	"yunt/internal/repository"
)

// SystemMailboxNames contains the names of system mailboxes that cannot be deleted.
var SystemMailboxNames = []string{"Inbox", "INBOX", "inbox", "Sent", "SENT", "sent", "Drafts", "DRAFTS", "drafts", "Trash", "TRASH", "trash", "Spam", "SPAM", "spam", "Junk", "JUNK", "junk"}

// MailboxService provides business logic for mailbox management.
// It coordinates between the mailbox repository and applies business rules
// such as ownership validation and system mailbox protection.
type MailboxService struct {
	repo        repository.Repository
	idGenerator IDGenerator
}

// NewMailboxService creates a new MailboxService with the given dependencies.
func NewMailboxService(repo repository.Repository, idGenerator IDGenerator) *MailboxService {
	return &MailboxService{
		repo:        repo,
		idGenerator: idGenerator,
	}
}

// ListMailboxes lists all mailboxes for a specific user.
func (s *MailboxService) ListMailboxes(
	ctx context.Context,
	userID domain.ID,
	opts *repository.ListOptions,
) (*repository.ListResult[*domain.Mailbox], error) {
	if userID.IsEmpty() {
		return nil, &MailboxServiceError{
			Op:      "list",
			Message: "user ID is required",
			Err:     domain.ErrInvalidInput,
		}
	}

	result, err := s.repo.Mailboxes().ListByUser(ctx, userID, opts)
	if err != nil {
		return nil, &MailboxServiceError{
			Op:      "list",
			Message: "failed to list mailboxes",
			Err:     err,
		}
	}

	return result, nil
}

// GetMailbox retrieves a mailbox by ID, verifying ownership.
func (s *MailboxService) GetMailbox(ctx context.Context, mailboxID, userID domain.ID) (*domain.Mailbox, error) {
	if mailboxID.IsEmpty() {
		return nil, &MailboxServiceError{
			Op:      "get",
			Message: "mailbox ID is required",
			Err:     domain.ErrInvalidInput,
		}
	}

	if userID.IsEmpty() {
		return nil, &MailboxServiceError{
			Op:      "get",
			Message: "user ID is required",
			Err:     domain.ErrInvalidInput,
		}
	}

	mailbox, err := s.repo.Mailboxes().GetByID(ctx, mailboxID)
	if err != nil {
		return nil, &MailboxServiceError{
			Op:      "get",
			Message: "failed to get mailbox",
			Err:     err,
		}
	}

	// Verify ownership
	if mailbox.UserID != userID {
		return nil, &MailboxServiceError{
			Op:      "get",
			Message: "access denied to mailbox",
			Err:     domain.ErrForbidden,
		}
	}

	return mailbox, nil
}

// CreateMailboxInput contains the input for creating a new mailbox.
type CreateMailboxInput struct {
	// UserID is the ID of the user creating the mailbox.
	UserID domain.ID
	// Name is the display name of the mailbox.
	Name string
	// Address is the email address for this mailbox.
	Address string
	// Description is an optional description.
	Description string
	// IsCatchAll indicates if this mailbox should catch all unmatched emails.
	IsCatchAll bool
	// IsDefault indicates if this should be the default mailbox.
	IsDefault bool
	// RetentionDays is the number of days to retain messages (0 = forever).
	RetentionDays int
}

// CreateMailbox creates a new mailbox for the user.
func (s *MailboxService) CreateMailbox(ctx context.Context, input *CreateMailboxInput) (*domain.Mailbox, error) {
	if input == nil {
		return nil, &MailboxServiceError{
			Op:      "create",
			Message: "input is required",
			Err:     domain.ErrInvalidInput,
		}
	}

	if input.UserID.IsEmpty() {
		return nil, &MailboxServiceError{
			Op:      "create",
			Message: "user ID is required",
			Err:     domain.ErrInvalidInput,
		}
	}

	// Create domain input for validation
	domainInput := &domain.MailboxCreateInput{
		Name:          input.Name,
		Address:       input.Address,
		Description:   input.Description,
		IsCatchAll:    input.IsCatchAll,
		IsDefault:     input.IsDefault,
		RetentionDays: input.RetentionDays,
	}

	// Normalize and validate
	domainInput.Normalize()
	if err := domainInput.Validate(); err != nil {
		return nil, &MailboxServiceError{
			Op:      "create",
			Message: "validation failed",
			Err:     err,
		}
	}

	// Check if address already exists
	exists, err := s.repo.Mailboxes().ExistsByAddress(ctx, domainInput.Address)
	if err != nil {
		return nil, &MailboxServiceError{
			Op:      "create",
			Message: "failed to check address uniqueness",
			Err:     err,
		}
	}
	if exists {
		return nil, &MailboxServiceError{
			Op:      "create",
			Message: "mailbox with this address already exists",
			Err:     domain.NewAlreadyExistsError("mailbox", "address", domainInput.Address),
		}
	}

	// Generate ID and create mailbox
	mailboxID := s.idGenerator.Generate()
	mailbox := domain.NewMailbox(mailboxID, input.UserID, domainInput.Name, domainInput.Address)
	mailbox.Description = domainInput.Description
	mailbox.IsCatchAll = domainInput.IsCatchAll
	mailbox.IsDefault = domainInput.IsDefault
	mailbox.RetentionDays = domainInput.RetentionDays

	// If this is the default mailbox, clear any existing default
	if mailbox.IsDefault {
		err = s.repo.Transaction(ctx, func(tx repository.Repository) error {
			if err := tx.Mailboxes().ClearDefault(ctx, input.UserID); err != nil {
				return err
			}
			return tx.Mailboxes().Create(ctx, mailbox)
		})
	} else {
		err = s.repo.Mailboxes().Create(ctx, mailbox)
	}

	if err != nil {
		return nil, &MailboxServiceError{
			Op:      "create",
			Message: "failed to create mailbox",
			Err:     err,
		}
	}

	return mailbox, nil
}

// UpdateMailboxInput contains the input for updating a mailbox.
type UpdateMailboxInput struct {
	// MailboxID is the ID of the mailbox to update.
	MailboxID domain.ID
	// UserID is the ID of the user making the update.
	UserID domain.ID
	// Name is the new display name (optional).
	Name *string
	// Description is the new description (optional).
	Description *string
	// IsDefault indicates if this should be the default mailbox (optional).
	IsDefault *bool
	// RetentionDays is the new retention period (optional).
	RetentionDays *int
}

// UpdateMailbox updates an existing mailbox (rename, etc.).
func (s *MailboxService) UpdateMailbox(ctx context.Context, input *UpdateMailboxInput) (*domain.Mailbox, error) {
	if input == nil {
		return nil, &MailboxServiceError{
			Op:      "update",
			Message: "input is required",
			Err:     domain.ErrInvalidInput,
		}
	}

	if input.MailboxID.IsEmpty() {
		return nil, &MailboxServiceError{
			Op:      "update",
			Message: "mailbox ID is required",
			Err:     domain.ErrInvalidInput,
		}
	}

	if input.UserID.IsEmpty() {
		return nil, &MailboxServiceError{
			Op:      "update",
			Message: "user ID is required",
			Err:     domain.ErrInvalidInput,
		}
	}

	// Create domain input for validation
	domainInput := &domain.MailboxUpdateInput{
		Name:          input.Name,
		Description:   input.Description,
		IsDefault:     input.IsDefault,
		RetentionDays: input.RetentionDays,
	}

	if err := domainInput.Validate(); err != nil {
		return nil, &MailboxServiceError{
			Op:      "update",
			Message: "validation failed",
			Err:     err,
		}
	}

	// Get existing mailbox
	mailbox, err := s.repo.Mailboxes().GetByID(ctx, input.MailboxID)
	if err != nil {
		return nil, &MailboxServiceError{
			Op:      "update",
			Message: "failed to get mailbox",
			Err:     err,
		}
	}

	// Verify ownership
	if mailbox.UserID != input.UserID {
		return nil, &MailboxServiceError{
			Op:      "update",
			Message: "access denied to mailbox",
			Err:     domain.ErrForbidden,
		}
	}

	// Apply updates
	domainInput.Apply(mailbox)

	// If setting as default, clear existing default first
	if input.IsDefault != nil && *input.IsDefault {
		err = s.repo.Transaction(ctx, func(tx repository.Repository) error {
			if err := tx.Mailboxes().ClearDefault(ctx, input.UserID); err != nil {
				return err
			}
			return tx.Mailboxes().Update(ctx, mailbox)
		})
	} else {
		err = s.repo.Mailboxes().Update(ctx, mailbox)
	}

	if err != nil {
		return nil, &MailboxServiceError{
			Op:      "update",
			Message: "failed to update mailbox",
			Err:     err,
		}
	}

	return mailbox, nil
}

// DeleteMailbox deletes a mailbox if it's not a system mailbox.
func (s *MailboxService) DeleteMailbox(ctx context.Context, mailboxID, userID domain.ID) error {
	if mailboxID.IsEmpty() {
		return &MailboxServiceError{
			Op:      "delete",
			Message: "mailbox ID is required",
			Err:     domain.ErrInvalidInput,
		}
	}

	if userID.IsEmpty() {
		return &MailboxServiceError{
			Op:      "delete",
			Message: "user ID is required",
			Err:     domain.ErrInvalidInput,
		}
	}

	// Get existing mailbox
	mailbox, err := s.repo.Mailboxes().GetByID(ctx, mailboxID)
	if err != nil {
		return &MailboxServiceError{
			Op:      "delete",
			Message: "failed to get mailbox",
			Err:     err,
		}
	}

	// Verify ownership
	if mailbox.UserID != userID {
		return &MailboxServiceError{
			Op:      "delete",
			Message: "access denied to mailbox",
			Err:     domain.ErrForbidden,
		}
	}

	// Check if it's a system mailbox
	if s.isSystemMailbox(mailbox.Name) {
		return &MailboxServiceError{
			Op:      "delete",
			Message: "cannot delete system mailbox",
			Err:     domain.NewConflictError("mailbox", "system mailboxes cannot be deleted"),
		}
	}

	// Delete mailbox and its messages
	if err := s.repo.Mailboxes().DeleteWithMessages(ctx, mailboxID); err != nil {
		return &MailboxServiceError{
			Op:      "delete",
			Message: "failed to delete mailbox",
			Err:     err,
		}
	}

	return nil
}

// GetMailboxStats retrieves statistics for a specific mailbox.
func (s *MailboxService) GetMailboxStats(ctx context.Context, mailboxID, userID domain.ID) (*domain.MailboxStats, error) {
	if mailboxID.IsEmpty() {
		return nil, &MailboxServiceError{
			Op:      "get_stats",
			Message: "mailbox ID is required",
			Err:     domain.ErrInvalidInput,
		}
	}

	if userID.IsEmpty() {
		return nil, &MailboxServiceError{
			Op:      "get_stats",
			Message: "user ID is required",
			Err:     domain.ErrInvalidInput,
		}
	}

	// Get mailbox to verify ownership
	mailbox, err := s.repo.Mailboxes().GetByID(ctx, mailboxID)
	if err != nil {
		return nil, &MailboxServiceError{
			Op:      "get_stats",
			Message: "failed to get mailbox",
			Err:     err,
		}
	}

	// Verify ownership
	if mailbox.UserID != userID {
		return nil, &MailboxServiceError{
			Op:      "get_stats",
			Message: "access denied to mailbox",
			Err:     domain.ErrForbidden,
		}
	}

	stats, err := s.repo.Mailboxes().GetStats(ctx, mailboxID)
	if err != nil {
		return nil, &MailboxServiceError{
			Op:      "get_stats",
			Message: "failed to get mailbox stats",
			Err:     err,
		}
	}

	return stats, nil
}

// GetUserMailboxStats retrieves aggregated statistics for all user's mailboxes.
func (s *MailboxService) GetUserMailboxStats(ctx context.Context, userID domain.ID) (*domain.MailboxStats, error) {
	if userID.IsEmpty() {
		return nil, &MailboxServiceError{
			Op:      "get_user_stats",
			Message: "user ID is required",
			Err:     domain.ErrInvalidInput,
		}
	}

	stats, err := s.repo.Mailboxes().GetStatsByUser(ctx, userID)
	if err != nil {
		return nil, &MailboxServiceError{
			Op:      "get_user_stats",
			Message: "failed to get user mailbox stats",
			Err:     err,
		}
	}

	return stats, nil
}

// SetDefaultMailbox sets a mailbox as the default for the user.
func (s *MailboxService) SetDefaultMailbox(ctx context.Context, mailboxID, userID domain.ID) error {
	if mailboxID.IsEmpty() {
		return &MailboxServiceError{
			Op:      "set_default",
			Message: "mailbox ID is required",
			Err:     domain.ErrInvalidInput,
		}
	}

	if userID.IsEmpty() {
		return &MailboxServiceError{
			Op:      "set_default",
			Message: "user ID is required",
			Err:     domain.ErrInvalidInput,
		}
	}

	// Get mailbox to verify ownership
	mailbox, err := s.repo.Mailboxes().GetByID(ctx, mailboxID)
	if err != nil {
		return &MailboxServiceError{
			Op:      "set_default",
			Message: "failed to get mailbox",
			Err:     err,
		}
	}

	// Verify ownership
	if mailbox.UserID != userID {
		return &MailboxServiceError{
			Op:      "set_default",
			Message: "access denied to mailbox",
			Err:     domain.ErrForbidden,
		}
	}

	err = s.repo.Transaction(ctx, func(tx repository.Repository) error {
		if err := tx.Mailboxes().ClearDefault(ctx, userID); err != nil {
			return err
		}
		return tx.Mailboxes().SetDefault(ctx, mailboxID)
	})

	if err != nil {
		return &MailboxServiceError{
			Op:      "set_default",
			Message: "failed to set default mailbox",
			Err:     err,
		}
	}

	return nil
}

// isSystemMailbox checks if a mailbox name is a system mailbox.
func (s *MailboxService) isSystemMailbox(name string) bool {
	for _, sysName := range SystemMailboxNames {
		if name == sysName {
			return true
		}
	}
	return false
}

// MailboxServiceError represents an error that occurred in the mailbox service.
type MailboxServiceError struct {
	// Op is the operation that failed.
	Op string
	// Message is a human-readable error description.
	Message string
	// Err is the underlying error.
	Err error
}

// Error implements the error interface.
func (e *MailboxServiceError) Error() string {
	if e.Err != nil {
		return "mailbox service " + e.Op + ": " + e.Message + ": " + e.Err.Error()
	}
	return "mailbox service " + e.Op + ": " + e.Message
}

// Unwrap returns the underlying error.
func (e *MailboxServiceError) Unwrap() error {
	return e.Err
}

// Is implements errors.Is for error comparison.
func (e *MailboxServiceError) Is(target error) bool {
	if e.Err == nil {
		return false
	}
	return errors.Is(e.Err, target)
}
