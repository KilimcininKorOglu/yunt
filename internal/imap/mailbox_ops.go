package imap

import (
	"context"
	"errors"
	"strings"

	"github.com/emersion/go-imap/v2"
	"github.com/google/uuid"

	"yunt/internal/domain"
	"yunt/internal/repository"
)

// MailboxOperator handles IMAP mailbox operations (SELECT, CREATE, DELETE, RENAME, STATUS).
type MailboxOperator struct {
	repo   repository.Repository
	userID domain.ID
}

// NewMailboxOperator creates a new MailboxOperator.
func NewMailboxOperator(repo repository.Repository, userID domain.ID) *MailboxOperator {
	return &MailboxOperator{
		repo:   repo,
		userID: userID,
	}
}

// Select opens a mailbox for reading.
// Returns SELECT data including message counts and UIDs.
func (o *MailboxOperator) Select(ctx context.Context, name string, _ *imap.SelectOptions) (*domain.Mailbox, *imap.SelectData, error) {
	normalizedName := NormalizeMailboxName(name)

	// Find the mailbox
	mailbox, err := o.findMailboxByName(ctx, normalizedName)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return nil, nil, &imap.Error{
				Type: imap.StatusResponseTypeNo,
				Code: imap.ResponseCodeNonExistent,
				Text: "Mailbox does not exist",
			}
		}
		return nil, nil, &imap.Error{
			Type: imap.StatusResponseTypeNo,
			Text: "Internal error",
		}
	}

	// Create SELECT response data
	selectData := NewSelectData(mailbox)

	return mailbox, selectData.ToIMAPSelectData(), nil
}

// Examine opens a mailbox for reading in read-only mode.
// Similar to Select but marks the mailbox as read-only.
func (o *MailboxOperator) Examine(ctx context.Context, name string, options *imap.SelectOptions) (*domain.Mailbox, *imap.SelectData, error) {
	// Examine is the same as Select but with read-only flag
	// The read-only flag is handled at the session level
	return o.Select(ctx, name, options)
}

// Create creates a new mailbox with the given name.
func (o *MailboxOperator) Create(ctx context.Context, name string, _ *imap.CreateOptions) error {
	normalizedName := NormalizeMailboxName(name)

	// Validate the mailbox name
	if err := ValidateMailboxName(normalizedName); err != nil {
		return &imap.Error{
			Type: imap.StatusResponseTypeNo,
			Text: err.Error(),
		}
	}

	// Cannot create system mailboxes
	if IsSystemMailbox(normalizedName) {
		return &imap.Error{
			Type: imap.StatusResponseTypeNo,
			Text: "Cannot create system mailbox",
		}
	}

	// Check if mailbox already exists
	existing, err := o.findMailboxByName(ctx, normalizedName)
	if err == nil && existing != nil {
		return &imap.Error{
			Type: imap.StatusResponseTypeNo,
			Code: imap.ResponseCodeAlreadyExists,
			Text: "Mailbox already exists",
		}
	}

	// For hierarchical mailboxes, ensure parent exists or create it
	path := ParseMailboxPath(normalizedName)
	if path.Parent != "" {
		if err := o.ensureParentExists(ctx, path.Parent); err != nil {
			return err
		}
	}

	// Create the mailbox
	mailboxID := domain.ID(uuid.New().String())
	mailbox := domain.NewMailbox(mailboxID, o.userID, normalizedName, "")

	if err := o.repo.Mailboxes().Create(ctx, mailbox); err != nil {
		if errors.Is(err, domain.ErrAlreadyExists) {
			return &imap.Error{
				Type: imap.StatusResponseTypeNo,
				Code: imap.ResponseCodeAlreadyExists,
				Text: "Mailbox already exists",
			}
		}
		return &imap.Error{
			Type: imap.StatusResponseTypeNo,
			Text: "Failed to create mailbox",
		}
	}

	return nil
}

// Delete removes a mailbox.
// System mailboxes cannot be deleted.
func (o *MailboxOperator) Delete(ctx context.Context, name string) error {
	normalizedName := NormalizeMailboxName(name)

	// Cannot delete system mailboxes
	if IsSystemMailbox(normalizedName) {
		return &imap.Error{
			Type: imap.StatusResponseTypeNo,
			Text: "Cannot delete system mailbox",
		}
	}

	// Find the mailbox
	mailbox, err := o.findMailboxByName(ctx, normalizedName)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return &imap.Error{
				Type: imap.StatusResponseTypeNo,
				Code: imap.ResponseCodeNonExistent,
				Text: "Mailbox does not exist",
			}
		}
		return &imap.Error{
			Type: imap.StatusResponseTypeNo,
			Text: "Internal error",
		}
	}

	// Check if mailbox has child mailboxes
	hasChildren, err := o.hasChildMailboxes(ctx, normalizedName)
	if err != nil {
		return &imap.Error{
			Type: imap.StatusResponseTypeNo,
			Text: "Failed to check child mailboxes",
		}
	}
	if hasChildren {
		return &imap.Error{
			Type: imap.StatusResponseTypeNo,
			Text: "Mailbox has child mailboxes",
		}
	}

	// Delete the mailbox and its messages
	if err := o.repo.Mailboxes().DeleteWithMessages(ctx, mailbox.ID); err != nil {
		return &imap.Error{
			Type: imap.StatusResponseTypeNo,
			Text: "Failed to delete mailbox",
		}
	}

	return nil
}

// Rename renames a mailbox.
// System mailboxes cannot be renamed (except INBOX which is a special case).
func (o *MailboxOperator) Rename(ctx context.Context, oldName, newName string, _ *imap.RenameOptions) error {
	normalizedOld := NormalizeMailboxName(oldName)
	normalizedNew := NormalizeMailboxName(newName)

	// Validate the new mailbox name
	if err := ValidateMailboxName(normalizedNew); err != nil {
		return &imap.Error{
			Type: imap.StatusResponseTypeNo,
			Text: err.Error(),
		}
	}

	// Cannot rename to a system mailbox name
	if IsSystemMailbox(normalizedNew) && !strings.EqualFold(normalizedNew, "INBOX") {
		return &imap.Error{
			Type: imap.StatusResponseTypeNo,
			Text: "Cannot rename to system mailbox name",
		}
	}

	// Cannot rename system mailboxes (except INBOX - special handling)
	if IsSystemMailbox(normalizedOld) && !strings.EqualFold(normalizedOld, "INBOX") {
		return &imap.Error{
			Type: imap.StatusResponseTypeNo,
			Text: "Cannot rename system mailbox",
		}
	}

	// Find the source mailbox
	mailbox, err := o.findMailboxByName(ctx, normalizedOld)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return &imap.Error{
				Type: imap.StatusResponseTypeNo,
				Code: imap.ResponseCodeNonExistent,
				Text: "Mailbox does not exist",
			}
		}
		return &imap.Error{
			Type: imap.StatusResponseTypeNo,
			Text: "Internal error",
		}
	}

	// Check if destination already exists
	existing, err := o.findMailboxByName(ctx, normalizedNew)
	if err == nil && existing != nil {
		return &imap.Error{
			Type: imap.StatusResponseTypeNo,
			Code: imap.ResponseCodeAlreadyExists,
			Text: "Destination mailbox already exists",
		}
	}

	// Special handling for INBOX rename
	// Per RFC 3501, renaming INBOX moves all messages to the new mailbox
	// and leaves INBOX empty (INBOX is automatically recreated)
	if strings.EqualFold(normalizedOld, "INBOX") {
		return o.renameInbox(ctx, mailbox, normalizedNew)
	}

	// For hierarchical mailboxes, ensure parent exists
	path := ParseMailboxPath(normalizedNew)
	if path.Parent != "" {
		if err := o.ensureParentExists(ctx, path.Parent); err != nil {
			return err
		}
	}

	// Rename the mailbox
	mailbox.Name = normalizedNew
	mailbox.UpdatedAt = domain.Now()

	if err := o.repo.Mailboxes().Update(ctx, mailbox); err != nil {
		return &imap.Error{
			Type: imap.StatusResponseTypeNo,
			Text: "Failed to rename mailbox",
		}
	}

	// Rename child mailboxes
	if err := o.renameChildMailboxes(ctx, normalizedOld, normalizedNew); err != nil {
		return &imap.Error{
			Type: imap.StatusResponseTypeNo,
			Text: "Failed to rename child mailboxes",
		}
	}

	return nil
}

// Status returns the status of a mailbox.
func (o *MailboxOperator) Status(ctx context.Context, name string, options *imap.StatusOptions) (*imap.StatusData, error) {
	normalizedName := NormalizeMailboxName(name)

	// Find the mailbox
	mailbox, err := o.findMailboxByName(ctx, normalizedName)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return nil, &imap.Error{
				Type: imap.StatusResponseTypeNo,
				Code: imap.ResponseCodeNonExistent,
				Text: "Mailbox does not exist",
			}
		}
		return nil, &imap.Error{
			Type: imap.StatusResponseTypeNo,
			Text: "Internal error",
		}
	}

	status := NewMailboxStatus(mailbox)
	return status.ToIMAPStatusData(options), nil
}

// Subscribe subscribes to a mailbox.
// For now, this is a no-op as we don't track subscriptions separately.
func (o *MailboxOperator) Subscribe(ctx context.Context, name string) error {
	normalizedName := NormalizeMailboxName(name)

	// Verify the mailbox exists
	_, err := o.findMailboxByName(ctx, normalizedName)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return &imap.Error{
				Type: imap.StatusResponseTypeNo,
				Code: imap.ResponseCodeNonExistent,
				Text: "Mailbox does not exist",
			}
		}
		return &imap.Error{
			Type: imap.StatusResponseTypeNo,
			Text: "Internal error",
		}
	}

	// Subscriptions are not tracked separately; all mailboxes are subscribed
	return nil
}

// Unsubscribe unsubscribes from a mailbox.
// For now, this is a no-op as we don't track subscriptions separately.
func (o *MailboxOperator) Unsubscribe(ctx context.Context, name string) error {
	normalizedName := NormalizeMailboxName(name)

	// Verify the mailbox exists
	_, err := o.findMailboxByName(ctx, normalizedName)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return &imap.Error{
				Type: imap.StatusResponseTypeNo,
				Code: imap.ResponseCodeNonExistent,
				Text: "Mailbox does not exist",
			}
		}
		return &imap.Error{
			Type: imap.StatusResponseTypeNo,
			Text: "Internal error",
		}
	}

	// Subscriptions are not tracked separately
	return nil
}

// findMailboxByName finds a mailbox by name for the current user.
func (o *MailboxOperator) findMailboxByName(ctx context.Context, name string) (*domain.Mailbox, error) {
	// List all mailboxes for the user and find by name
	result, err := o.repo.Mailboxes().ListByUser(ctx, o.userID, nil)
	if err != nil {
		return nil, err
	}

	for _, mailbox := range result.Items {
		if strings.EqualFold(NormalizeMailboxName(mailbox.Name), name) {
			return mailbox, nil
		}
	}

	return nil, domain.ErrNotFound
}

// hasChildMailboxes checks if a mailbox has any child mailboxes.
func (o *MailboxOperator) hasChildMailboxes(ctx context.Context, parentName string) (bool, error) {
	result, err := o.repo.Mailboxes().ListByUser(ctx, o.userID, nil)
	if err != nil {
		return false, err
	}

	prefix := parentName + MailboxHierarchySeparator
	for _, mailbox := range result.Items {
		if strings.HasPrefix(NormalizeMailboxName(mailbox.Name), prefix) {
			return true, nil
		}
	}

	return false, nil
}

// ensureParentExists ensures that all parent mailboxes exist.
func (o *MailboxOperator) ensureParentExists(ctx context.Context, parentPath string) error {
	parts := strings.Split(parentPath, MailboxHierarchySeparator)

	for i := 1; i <= len(parts); i++ {
		path := strings.Join(parts[:i], MailboxHierarchySeparator)

		// Check if this path exists
		_, err := o.findMailboxByName(ctx, path)
		if err == nil {
			continue // Already exists
		}
		if !errors.Is(err, domain.ErrNotFound) {
			return err
		}

		// Create the parent mailbox
		mailboxID := domain.ID(uuid.New().String())
		mailbox := domain.NewMailbox(mailboxID, o.userID, path, "")

		if err := o.repo.Mailboxes().Create(ctx, mailbox); err != nil {
			if !errors.Is(err, domain.ErrAlreadyExists) {
				return err
			}
		}
	}

	return nil
}

// renameInbox handles the special case of renaming INBOX.
// Per RFC 3501, this moves all messages but keeps INBOX.
func (o *MailboxOperator) renameInbox(ctx context.Context, inbox *domain.Mailbox, newName string) error {
	// Create the destination mailbox
	newMailboxID := domain.ID(uuid.New().String())
	newMailbox := domain.NewMailbox(newMailboxID, o.userID, newName, "")
	newMailbox.MessageCount = inbox.MessageCount
	newMailbox.UnreadCount = inbox.UnreadCount
	newMailbox.TotalSize = inbox.TotalSize

	if err := o.repo.Mailboxes().Create(ctx, newMailbox); err != nil {
		return &imap.Error{
			Type: imap.StatusResponseTypeNo,
			Text: "Failed to create destination mailbox",
		}
	}

	// Move all messages from INBOX to the new mailbox
	result, err := o.repo.Messages().ListByMailbox(ctx, inbox.ID, nil)
	if err != nil {
		return &imap.Error{
			Type: imap.StatusResponseTypeNo,
			Text: "Failed to list messages",
		}
	}

	for _, msg := range result.Items {
		if err := o.repo.Messages().MoveToMailbox(ctx, msg.ID, newMailboxID); err != nil {
			return &imap.Error{
				Type: imap.StatusResponseTypeNo,
				Text: "Failed to move messages",
			}
		}
	}

	// Reset INBOX statistics
	inbox.MessageCount = 0
	inbox.UnreadCount = 0
	inbox.TotalSize = 0
	inbox.UpdatedAt = domain.Now()

	if err := o.repo.Mailboxes().Update(ctx, inbox); err != nil {
		return &imap.Error{
			Type: imap.StatusResponseTypeNo,
			Text: "Failed to update INBOX",
		}
	}

	return nil
}

// renameChildMailboxes renames all child mailboxes when a parent is renamed.
func (o *MailboxOperator) renameChildMailboxes(ctx context.Context, oldPrefix, newPrefix string) error {
	result, err := o.repo.Mailboxes().ListByUser(ctx, o.userID, nil)
	if err != nil {
		return err
	}

	oldPrefixWithSep := oldPrefix + MailboxHierarchySeparator
	for _, mailbox := range result.Items {
		normalizedName := NormalizeMailboxName(mailbox.Name)
		if strings.HasPrefix(normalizedName, oldPrefixWithSep) {
			// Calculate new name
			suffix := normalizedName[len(oldPrefixWithSep):]
			newName := newPrefix + MailboxHierarchySeparator + suffix

			mailbox.Name = newName
			mailbox.UpdatedAt = domain.Now()

			if err := o.repo.Mailboxes().Update(ctx, mailbox); err != nil {
				return err
			}
		}
	}

	return nil
}
