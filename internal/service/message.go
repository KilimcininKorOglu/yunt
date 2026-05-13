// Package service provides business logic and service layer implementations
// for the Yunt mail server. Services coordinate between repositories, apply
// business rules, and manage transactional operations.
package service

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"

	"yunt/internal/domain"
	"yunt/internal/parser"
	"yunt/internal/repository"
)

// IDGenerator generates unique identifiers for domain entities.
type IDGenerator interface {
	// Generate creates a new unique identifier.
	Generate() domain.ID
}

// MessageService provides business logic for message storage and retrieval.
// It coordinates between message, attachment, and mailbox repositories to ensure
// consistent storage operations and proper statistics updates.
type MessageService struct {
	repo          repository.Repository
	idGenerator   IDGenerator
	parser        *parser.Parser
	notifyService *NotifyService
}

// NewMessageService creates a new MessageService with the given dependencies.
func NewMessageService(repo repository.Repository, idGenerator IDGenerator) *MessageService {
	return &MessageService{
		repo:        repo,
		idGenerator: idGenerator,
		parser:      parser.NewParser(),
	}
}

// WithNotifyService sets the notification service for real-time updates.
func (s *MessageService) WithNotifyService(ns *NotifyService) {
	s.notifyService = ns
}

// WithParser sets a custom parser for the service.
// This is useful for testing or when custom parsing settings are needed.
func (s *MessageService) WithParser(p *parser.Parser) *MessageService {
	s.parser = p
	return s
}

// StoreMessageInput contains the input for storing a new message.
type StoreMessageInput struct {
	// RawData is the raw MIME message data.
	RawData []byte

	// TargetMailboxID is the specific mailbox to store the message in.
	// If empty, the service will attempt to route the message based on recipients.
	TargetMailboxID domain.ID

	// SkipDuplicateCheck allows storing duplicate messages if true.
	SkipDuplicateCheck bool
}

// StoreMessageResult contains the result of storing a message.
type StoreMessageResult struct {
	// Message is the stored message entity.
	Message *domain.Message

	// Attachments contains all stored attachments.
	Attachments []*domain.Attachment

	// IsDuplicate indicates if the message was detected as a duplicate.
	IsDuplicate bool

	// DuplicateID is the ID of the existing message if a duplicate was found.
	DuplicateID domain.ID
}

// StoreMessage parses and stores a raw MIME message along with its attachments.
// The operation is transactional - if any part fails, all changes are rolled back.
// It updates mailbox statistics after successful storage.
func (s *MessageService) StoreMessage(ctx context.Context, input *StoreMessageInput) (*StoreMessageResult, error) {
	if err := s.validateStoreInput(input); err != nil {
		return nil, err
	}

	// Parse the raw message
	parsed, err := s.parser.Parse(input.RawData)
	if err != nil {
		return nil, &MessageServiceError{
			Op:      "parse",
			Message: "failed to parse MIME message",
			Err:     err,
		}
	}

	// Check for duplicates before starting transaction
	if !input.SkipDuplicateCheck && parsed.MessageID != "" {
		existingMsg, err := s.repo.Messages().GetByMessageID(ctx, parsed.MessageID)
		if err == nil && existingMsg != nil {
			return &StoreMessageResult{
				Message:     existingMsg,
				IsDuplicate: true,
				DuplicateID: existingMsg.ID,
			}, nil
		}
		// Ignore ErrNotFound - that's expected for new messages
		if err != nil && !domain.IsNotFound(err) {
			return nil, &MessageServiceError{
				Op:      "duplicate_check",
				Message: "failed to check for duplicate message",
				Err:     err,
			}
		}
	}

	// Determine target mailbox
	mailbox, err := s.resolveMailbox(ctx, input.TargetMailboxID, parsed)
	if err != nil {
		return nil, err
	}

	// Generate message ID
	messageID := s.idGenerator.Generate()

	// Execute storage within a transaction
	var result *StoreMessageResult
	err = s.repo.Transaction(ctx, func(tx repository.Repository) error {
		var txErr error
		result, txErr = s.storeMessageInTransaction(ctx, tx, messageID, mailbox, parsed, input.RawData)
		return txErr
	})

	if err != nil {
		return nil, &MessageServiceError{
			Op:      "store",
			Message: "failed to store message",
			Err:     err,
		}
	}

	if s.notifyService != nil && result != nil && !result.IsDuplicate {
		count, err := s.repo.Messages().CountByMailbox(ctx, result.Message.MailboxID)
		if err == nil {
			s.notifyService.NotifyNewMessage(
				result.Message.MailboxID, domain.ID(""), result.Message.ID,
				uint32(count), uint32(count), uint32(count),
			)
		}
	}

	return result, nil
}

// storeMessageInTransaction handles the actual storage within a transaction.
func (s *MessageService) storeMessageInTransaction(
	ctx context.Context,
	tx repository.Repository,
	messageID domain.ID,
	mailbox *domain.Mailbox,
	parsed *parser.ParsedMessage,
	rawData []byte,
) (*StoreMessageResult, error) {
	// Convert parsed message to domain message
	msg := parsed.ToMessage(messageID, mailbox.ID)
	msg.RawBody = rawData

	if len(rawData) > 0 {
		hash := sha256.Sum256(rawData)
		msg.BlobID = hex.EncodeToString(hash[:])
	}

	// Assign IMAP UID and update mailbox stats
	assignedUID, err := tx.Mailboxes().IncrementMessageCount(ctx, mailbox.ID, msg.Size, msg.Status == domain.MessageUnread)
	if err != nil {
		return nil, err
	}
	msg.IMAPUID = assignedUID

	// Store the message
	if err := tx.Messages().Create(ctx, msg); err != nil {
		return nil, err
	}

	// Store raw body for EML export
	if err := tx.Messages().StoreRawBody(ctx, messageID, rawData); err != nil {
		return nil, err
	}

	// Store attachments
	storedAttachments, err := s.storeAttachments(ctx, tx, messageID, parsed.Attachments)
	if err != nil {
		return nil, err
	}

	return &StoreMessageResult{
		Message:     msg,
		Attachments: storedAttachments,
		IsDuplicate: false,
	}, nil
}

// storeAttachments stores all attachments for a message.
func (s *MessageService) storeAttachments(
	ctx context.Context,
	tx repository.Repository,
	messageID domain.ID,
	attachments []*parser.AttachmentData,
) ([]*domain.Attachment, error) {
	if len(attachments) == 0 {
		return nil, nil
	}

	storedAttachments := make([]*domain.Attachment, 0, len(attachments))

	for _, attData := range attachments {
		att := s.createAttachment(messageID, attData)
		storedAttachments = append(storedAttachments, att)

		// Store attachment with content
		reader := bytes.NewReader(attData.Data)
		if err := tx.Attachments().CreateWithContent(ctx, att, reader); err != nil {
			return nil, err
		}
	}

	return storedAttachments, nil
}

// createAttachment creates a domain.Attachment from parser.AttachmentData.
func (s *MessageService) createAttachment(messageID domain.ID, data *parser.AttachmentData) *domain.Attachment {
	attID := s.idGenerator.Generate()

	var att *domain.Attachment
	if data.IsInline {
		att = domain.NewInlineAttachment(
			attID,
			messageID,
			data.Filename,
			data.ContentType,
			data.ContentID,
			int64(len(data.Data)),
		)
	} else {
		att = domain.NewAttachment(
			attID,
			messageID,
			data.Filename,
			data.ContentType,
			int64(len(data.Data)),
		)
	}

	// Calculate checksum
	hash := sha256.Sum256(data.Data)
	att.Checksum = hex.EncodeToString(hash[:])

	return att
}

// resolveMailbox determines which mailbox should receive the message.
func (s *MessageService) resolveMailbox(
	ctx context.Context,
	targetMailboxID domain.ID,
	parsed *parser.ParsedMessage,
) (*domain.Mailbox, error) {
	// If a specific mailbox is provided, use it
	if !targetMailboxID.IsEmpty() {
		mailbox, err := s.repo.Mailboxes().GetByID(ctx, targetMailboxID)
		if err != nil {
			if domain.IsNotFound(err) {
				return nil, &MessageServiceError{
					Op:      "resolve_mailbox",
					Message: "target mailbox not found",
					Err:     domain.NewNotFoundError("mailbox", targetMailboxID.String()),
				}
			}
			return nil, &MessageServiceError{
				Op:      "resolve_mailbox",
				Message: "failed to get target mailbox",
				Err:     err,
			}
		}
		return mailbox, nil
	}

	// Try to find a mailbox based on recipients
	for _, recipient := range parsed.To {
		mailbox, err := s.repo.Mailboxes().FindMatchingMailbox(ctx, recipient.Address)
		if err == nil && mailbox != nil {
			return mailbox, nil
		}
	}

	// Try CC recipients
	for _, recipient := range parsed.Cc {
		mailbox, err := s.repo.Mailboxes().FindMatchingMailbox(ctx, recipient.Address)
		if err == nil && mailbox != nil {
			return mailbox, nil
		}
	}

	// Try BCC recipients
	for _, recipient := range parsed.Bcc {
		mailbox, err := s.repo.Mailboxes().FindMatchingMailbox(ctx, recipient.Address)
		if err == nil && mailbox != nil {
			return mailbox, nil
		}
	}

	return nil, &MessageServiceError{
		Op:      "resolve_mailbox",
		Message: "no matching mailbox found for message recipients",
		Err:     domain.ErrNotFound,
	}
}

// validateStoreInput validates the input for storing a message.
func (s *MessageService) validateStoreInput(input *StoreMessageInput) error {
	if input == nil {
		return &MessageServiceError{
			Op:      "validate",
			Message: "input is required",
			Err:     domain.ErrInvalidInput,
		}
	}

	if len(input.RawData) == 0 {
		return &MessageServiceError{
			Op:      "validate",
			Message: "raw message data is required",
			Err:     domain.ErrInvalidInput,
		}
	}

	return nil
}

// GetMessage retrieves a message by its ID.
func (s *MessageService) GetMessage(ctx context.Context, id domain.ID) (*domain.Message, error) {
	if id.IsEmpty() {
		return nil, &MessageServiceError{
			Op:      "get",
			Message: "message ID is required",
			Err:     domain.ErrInvalidInput,
		}
	}

	msg, err := s.repo.Messages().GetByID(ctx, id)
	if err != nil {
		return nil, &MessageServiceError{
			Op:      "get",
			Message: "failed to get message",
			Err:     err,
		}
	}

	return msg, nil
}

// GetMessageWithAttachments retrieves a message with its attachments.
func (s *MessageService) GetMessageWithAttachments(ctx context.Context, id domain.ID) (*domain.Message, []*domain.Attachment, error) {
	if id.IsEmpty() {
		return nil, nil, &MessageServiceError{
			Op:      "get_with_attachments",
			Message: "message ID is required",
			Err:     domain.ErrInvalidInput,
		}
	}

	msg, attachments, err := s.repo.Messages().GetWithAttachments(ctx, id)
	if err != nil {
		return nil, nil, &MessageServiceError{
			Op:      "get_with_attachments",
			Message: "failed to get message with attachments",
			Err:     err,
		}
	}

	return msg, attachments, nil
}

// GetRawMessage retrieves the raw message body for EML export.
func (s *MessageService) GetRawMessage(ctx context.Context, id domain.ID) ([]byte, error) {
	if id.IsEmpty() {
		return nil, &MessageServiceError{
			Op:      "get_raw",
			Message: "message ID is required",
			Err:     domain.ErrInvalidInput,
		}
	}

	rawBody, err := s.repo.Messages().GetRawBody(ctx, id)
	if err != nil {
		return nil, &MessageServiceError{
			Op:      "get_raw",
			Message: "failed to get raw message body",
			Err:     err,
		}
	}

	return rawBody, nil
}

// DeleteMessage removes a message and its attachments from storage.
// It updates the mailbox statistics accordingly.
func (s *MessageService) DeleteMessage(ctx context.Context, id domain.ID) error {
	if id.IsEmpty() {
		return &MessageServiceError{
			Op:      "delete",
			Message: "message ID is required",
			Err:     domain.ErrInvalidInput,
		}
	}

	// Get message details for updating mailbox stats
	msg, err := s.repo.Messages().GetByID(ctx, id)
	if err != nil {
		return &MessageServiceError{
			Op:      "delete",
			Message: "failed to get message for deletion",
			Err:     err,
		}
	}

	err = s.repo.Transaction(ctx, func(tx repository.Repository) error {
		// Delete attachments first
		if _, err := tx.Attachments().DeleteByMessage(ctx, id); err != nil {
			return err
		}

		// Delete the message
		if err := tx.Messages().Delete(ctx, id); err != nil {
			return err
		}

		// Update mailbox statistics
		wasUnread := msg.Status == domain.MessageUnread
		if err := tx.Mailboxes().DecrementMessageCount(ctx, msg.MailboxID, msg.Size, wasUnread); err != nil {
			return err
		}

		return nil
	})

	if err != nil {
		return &MessageServiceError{
			Op:      "delete",
			Message: "failed to delete message",
			Err:     err,
		}
	}

	return nil
}

// MarkAsRead marks a message as read and updates mailbox statistics.
func (s *MessageService) MarkAsRead(ctx context.Context, id domain.ID) error {
	if id.IsEmpty() {
		return &MessageServiceError{
			Op:      "mark_read",
			Message: "message ID is required",
			Err:     domain.ErrInvalidInput,
		}
	}

	msg, err := s.repo.Messages().GetByID(ctx, id)
	if err != nil {
		return &MessageServiceError{
			Op:      "mark_read",
			Message: "failed to get message",
			Err:     err,
		}
	}

	err = s.repo.Transaction(ctx, func(tx repository.Repository) error {
		changed, err := tx.Messages().MarkAsRead(ctx, id)
		if err != nil {
			return err
		}

		if changed {
			// Update mailbox unread count
			if err := tx.Mailboxes().UpdateUnreadCount(ctx, msg.MailboxID, -1); err != nil {
				return err
			}
		}

		return nil
	})

	if err != nil {
		return &MessageServiceError{
			Op:      "mark_read",
			Message: "failed to mark message as read",
			Err:     err,
		}
	}

	return nil
}

// MarkAsUnread marks a message as unread and updates mailbox statistics.
func (s *MessageService) MarkAsUnread(ctx context.Context, id domain.ID) error {
	if id.IsEmpty() {
		return &MessageServiceError{
			Op:      "mark_unread",
			Message: "message ID is required",
			Err:     domain.ErrInvalidInput,
		}
	}

	msg, err := s.repo.Messages().GetByID(ctx, id)
	if err != nil {
		return &MessageServiceError{
			Op:      "mark_unread",
			Message: "failed to get message",
			Err:     err,
		}
	}

	err = s.repo.Transaction(ctx, func(tx repository.Repository) error {
		changed, err := tx.Messages().MarkAsUnread(ctx, id)
		if err != nil {
			return err
		}

		if changed {
			// Update mailbox unread count
			if err := tx.Mailboxes().UpdateUnreadCount(ctx, msg.MailboxID, 1); err != nil {
				return err
			}
		}

		return nil
	})

	if err != nil {
		return &MessageServiceError{
			Op:      "mark_unread",
			Message: "failed to mark message as unread",
			Err:     err,
		}
	}

	return nil
}

// MoveMessage moves a message to a different mailbox.
// It updates statistics for both source and target mailboxes.
func (s *MessageService) MoveMessage(ctx context.Context, id, targetMailboxID domain.ID) error {
	if id.IsEmpty() {
		return &MessageServiceError{
			Op:      "move",
			Message: "message ID is required",
			Err:     domain.ErrInvalidInput,
		}
	}

	if targetMailboxID.IsEmpty() {
		return &MessageServiceError{
			Op:      "move",
			Message: "target mailbox ID is required",
			Err:     domain.ErrInvalidInput,
		}
	}

	// Verify target mailbox exists
	_, err := s.repo.Mailboxes().GetByID(ctx, targetMailboxID)
	if err != nil {
		return &MessageServiceError{
			Op:      "move",
			Message: "target mailbox not found",
			Err:     err,
		}
	}

	// Get message details
	msg, err := s.repo.Messages().GetByID(ctx, id)
	if err != nil {
		return &MessageServiceError{
			Op:      "move",
			Message: "failed to get message",
			Err:     err,
		}
	}

	// Check if already in target mailbox
	if msg.MailboxID == targetMailboxID {
		return nil // No-op
	}

	err = s.repo.Transaction(ctx, func(tx repository.Repository) error {
		// MoveToMailbox handles stats (Decrement source, Increment target, UID assignment)
		if err := tx.Messages().MoveToMailbox(ctx, id, targetMailboxID); err != nil {
			return err
		}
		return nil
	})

	if err != nil {
		return &MessageServiceError{
			Op:      "move",
			Message: "failed to move message",
			Err:     err,
		}
	}

	return nil
}

// ListMessages lists messages with optional filtering and pagination.
func (s *MessageService) ListMessages(
	ctx context.Context,
	filter *repository.MessageFilter,
	opts *repository.ListOptions,
) (*repository.ListResult[*domain.Message], error) {
	result, err := s.repo.Messages().List(ctx, filter, opts)
	if err != nil {
		return nil, &MessageServiceError{
			Op:      "list",
			Message: "failed to list messages",
			Err:     err,
		}
	}

	return result, nil
}

// ListMessagesByMailbox lists all messages in a specific mailbox.
func (s *MessageService) ListMessagesByMailbox(
	ctx context.Context,
	mailboxID domain.ID,
	opts *repository.ListOptions,
) (*repository.ListResult[*domain.Message], error) {
	if mailboxID.IsEmpty() {
		return nil, &MessageServiceError{
			Op:      "list_by_mailbox",
			Message: "mailbox ID is required",
			Err:     domain.ErrInvalidInput,
		}
	}

	result, err := s.repo.Messages().ListByMailbox(ctx, mailboxID, opts)
	if err != nil {
		return nil, &MessageServiceError{
			Op:      "list_by_mailbox",
			Message: "failed to list messages by mailbox",
			Err:     err,
		}
	}

	return result, nil
}

// ExistsByMessageID checks if a message with the given Message-ID header exists.
func (s *MessageService) ExistsByMessageID(ctx context.Context, messageID string) (bool, error) {
	if messageID == "" {
		return false, &MessageServiceError{
			Op:      "exists",
			Message: "message ID header is required",
			Err:     domain.ErrInvalidInput,
		}
	}

	exists, err := s.repo.Messages().ExistsByMessageID(ctx, messageID)
	if err != nil {
		return false, &MessageServiceError{
			Op:      "exists",
			Message: "failed to check message existence",
			Err:     err,
		}
	}

	return exists, nil
}

// MessageServiceError represents an error that occurred in the message service.
type MessageServiceError struct {
	// Op is the operation that failed.
	Op string
	// Message is a human-readable error description.
	Message string
	// Err is the underlying error.
	Err error
}

// Error implements the error interface.
func (e *MessageServiceError) Error() string {
	if e.Err != nil {
		return "message service " + e.Op + ": " + e.Message + ": " + e.Err.Error()
	}
	return "message service " + e.Op + ": " + e.Message
}

// Unwrap returns the underlying error.
func (e *MessageServiceError) Unwrap() error {
	return e.Err
}

// Is implements errors.Is for error comparison.
func (e *MessageServiceError) Is(target error) bool {
	if e.Err == nil {
		return false
	}
	return errors.Is(e.Err, target)
}

// User-aware methods that verify mailbox ownership before performing operations.

// verifyMessageAccess checks if the user owns the mailbox containing the message.
func (s *MessageService) verifyMessageAccess(ctx context.Context, message *domain.Message, userID domain.ID) error {
	mailbox, err := s.repo.Mailboxes().GetByID(ctx, message.MailboxID)
	if err != nil {
		return &MessageServiceError{
			Op:      "verify_access",
			Message: "failed to get mailbox",
			Err:     err,
		}
	}

	if mailbox.UserID != userID {
		return &MessageServiceError{
			Op:      "verify_access",
			Message: "access denied to message",
			Err:     domain.ErrForbidden,
		}
	}

	return nil
}

// getUserMailboxIDs returns all mailbox IDs owned by the user.
func (s *MessageService) getUserMailboxIDs(ctx context.Context, userID domain.ID) ([]domain.ID, error) {
	result, err := s.repo.Mailboxes().ListByUser(ctx, userID, nil)
	if err != nil {
		return nil, &MessageServiceError{
			Op:      "get_user_mailboxes",
			Message: "failed to get user mailboxes",
			Err:     err,
		}
	}

	ids := make([]domain.ID, len(result.Items))
	for i, mailbox := range result.Items {
		ids[i] = mailbox.ID
	}

	return ids, nil
}

// ListMessagesForUser lists messages for a specific user with optional filtering.
// It automatically restricts results to mailboxes owned by the user.
func (s *MessageService) ListMessagesForUser(
	ctx context.Context,
	userID domain.ID,
	filter *repository.MessageFilter,
	opts *repository.ListOptions,
) (*repository.ListResult[*domain.Message], error) {
	if userID.IsEmpty() {
		return nil, &MessageServiceError{
			Op:      "list_for_user",
			Message: "user ID is required",
			Err:     domain.ErrInvalidInput,
		}
	}

	// Get user's mailbox IDs
	mailboxIDs, err := s.getUserMailboxIDs(ctx, userID)
	if err != nil {
		return nil, err
	}

	if len(mailboxIDs) == 0 {
		// User has no mailboxes, return empty result
		return &repository.ListResult[*domain.Message]{
			Items: []*domain.Message{},
			Total: 0,
		}, nil
	}

	// Create filter if nil
	if filter == nil {
		filter = &repository.MessageFilter{}
	}

	// If a specific mailbox is requested, verify ownership
	if filter.MailboxID != nil {
		found := false
		for _, id := range mailboxIDs {
			if id == *filter.MailboxID {
				found = true
				break
			}
		}
		if !found {
			return nil, &MessageServiceError{
				Op:      "list_for_user",
				Message: "access denied to mailbox",
				Err:     domain.ErrForbidden,
			}
		}
	} else {
		// Restrict to user's mailboxes
		filter.MailboxIDs = mailboxIDs
	}

	result, err := s.repo.Messages().List(ctx, filter, opts)
	if err != nil {
		return nil, &MessageServiceError{
			Op:      "list_for_user",
			Message: "failed to list messages",
			Err:     err,
		}
	}

	return result, nil
}

// SearchMessagesForUser searches messages for a specific user.
func (s *MessageService) SearchMessagesForUser(
	ctx context.Context,
	userID domain.ID,
	searchOpts *repository.SearchOptions,
	filter *repository.MessageFilter,
	opts *repository.ListOptions,
) (*repository.ListResult[*domain.Message], error) {
	if userID.IsEmpty() {
		return nil, &MessageServiceError{
			Op:      "search_for_user",
			Message: "user ID is required",
			Err:     domain.ErrInvalidInput,
		}
	}

	// Get user's mailbox IDs
	mailboxIDs, err := s.getUserMailboxIDs(ctx, userID)
	if err != nil {
		return nil, err
	}

	if len(mailboxIDs) == 0 {
		return &repository.ListResult[*domain.Message]{
			Items: []*domain.Message{},
			Total: 0,
		}, nil
	}

	// Create filter if nil
	if filter == nil {
		filter = &repository.MessageFilter{}
	}

	// If a specific mailbox is requested, verify ownership
	if filter.MailboxID != nil {
		found := false
		for _, id := range mailboxIDs {
			if id == *filter.MailboxID {
				found = true
				break
			}
		}
		if !found {
			return nil, &MessageServiceError{
				Op:      "search_for_user",
				Message: "access denied to mailbox",
				Err:     domain.ErrForbidden,
			}
		}
	} else {
		// Restrict to user's mailboxes
		filter.MailboxIDs = mailboxIDs
	}

	result, err := s.repo.Messages().Search(ctx, searchOpts, filter, opts)
	if err != nil {
		return nil, &MessageServiceError{
			Op:      "search_for_user",
			Message: "failed to search messages",
			Err:     err,
		}
	}

	return result, nil
}

// GetMessageForUser retrieves a message verifying user ownership.
func (s *MessageService) GetMessageForUser(ctx context.Context, messageID, userID domain.ID) (*domain.Message, error) {
	if messageID.IsEmpty() {
		return nil, &MessageServiceError{
			Op:      "get_for_user",
			Message: "message ID is required",
			Err:     domain.ErrInvalidInput,
		}
	}

	if userID.IsEmpty() {
		return nil, &MessageServiceError{
			Op:      "get_for_user",
			Message: "user ID is required",
			Err:     domain.ErrInvalidInput,
		}
	}

	msg, err := s.repo.Messages().GetByID(ctx, messageID)
	if err != nil {
		return nil, &MessageServiceError{
			Op:      "get_for_user",
			Message: "failed to get message",
			Err:     err,
		}
	}

	if err := s.verifyMessageAccess(ctx, msg, userID); err != nil {
		return nil, err
	}

	return msg, nil
}

// GetRawMessageForUser retrieves the raw message body verifying user ownership.
func (s *MessageService) GetRawMessageForUser(ctx context.Context, messageID, userID domain.ID) ([]byte, error) {
	if messageID.IsEmpty() {
		return nil, &MessageServiceError{
			Op:      "get_raw_for_user",
			Message: "message ID is required",
			Err:     domain.ErrInvalidInput,
		}
	}

	if userID.IsEmpty() {
		return nil, &MessageServiceError{
			Op:      "get_raw_for_user",
			Message: "user ID is required",
			Err:     domain.ErrInvalidInput,
		}
	}

	// First verify access
	msg, err := s.repo.Messages().GetByID(ctx, messageID)
	if err != nil {
		return nil, &MessageServiceError{
			Op:      "get_raw_for_user",
			Message: "failed to get message",
			Err:     err,
		}
	}

	if err := s.verifyMessageAccess(ctx, msg, userID); err != nil {
		return nil, err
	}

	rawBody, err := s.repo.Messages().GetRawBody(ctx, messageID)
	if err != nil {
		return nil, &MessageServiceError{
			Op:      "get_raw_for_user",
			Message: "failed to get raw message body",
			Err:     err,
		}
	}

	return rawBody, nil
}

// GetAttachmentsForUser retrieves attachments for a message verifying user ownership.
func (s *MessageService) GetAttachmentsForUser(ctx context.Context, messageID, userID domain.ID) ([]*domain.Attachment, error) {
	if messageID.IsEmpty() {
		return nil, &MessageServiceError{
			Op:      "get_attachments_for_user",
			Message: "message ID is required",
			Err:     domain.ErrInvalidInput,
		}
	}

	if userID.IsEmpty() {
		return nil, &MessageServiceError{
			Op:      "get_attachments_for_user",
			Message: "user ID is required",
			Err:     domain.ErrInvalidInput,
		}
	}

	// First verify access
	msg, err := s.repo.Messages().GetByID(ctx, messageID)
	if err != nil {
		return nil, &MessageServiceError{
			Op:      "get_attachments_for_user",
			Message: "failed to get message",
			Err:     err,
		}
	}

	if err := s.verifyMessageAccess(ctx, msg, userID); err != nil {
		return nil, err
	}

	attachments, err := s.repo.Attachments().ListByMessage(ctx, messageID)
	if err != nil {
		return nil, &MessageServiceError{
			Op:      "get_attachments_for_user",
			Message: "failed to get attachments",
			Err:     err,
		}
	}

	return attachments, nil
}

// GetAttachmentForUser retrieves a specific attachment verifying user ownership.
func (s *MessageService) GetAttachmentForUser(ctx context.Context, messageID, attachmentID, userID domain.ID) (*domain.Attachment, error) {
	if messageID.IsEmpty() {
		return nil, &MessageServiceError{
			Op:      "get_attachment_for_user",
			Message: "message ID is required",
			Err:     domain.ErrInvalidInput,
		}
	}

	if attachmentID.IsEmpty() {
		return nil, &MessageServiceError{
			Op:      "get_attachment_for_user",
			Message: "attachment ID is required",
			Err:     domain.ErrInvalidInput,
		}
	}

	if userID.IsEmpty() {
		return nil, &MessageServiceError{
			Op:      "get_attachment_for_user",
			Message: "user ID is required",
			Err:     domain.ErrInvalidInput,
		}
	}

	// First verify access to the message
	msg, err := s.repo.Messages().GetByID(ctx, messageID)
	if err != nil {
		return nil, &MessageServiceError{
			Op:      "get_attachment_for_user",
			Message: "failed to get message",
			Err:     err,
		}
	}

	if err := s.verifyMessageAccess(ctx, msg, userID); err != nil {
		return nil, err
	}

	// Get the attachment
	attachment, err := s.repo.Attachments().GetByID(ctx, attachmentID)
	if err != nil {
		return nil, &MessageServiceError{
			Op:      "get_attachment_for_user",
			Message: "failed to get attachment",
			Err:     err,
		}
	}

	// Verify the attachment belongs to the message
	if attachment.MessageID != messageID {
		return nil, &MessageServiceError{
			Op:      "get_attachment_for_user",
			Message: "attachment does not belong to this message",
			Err:     domain.ErrNotFound,
		}
	}

	return attachment, nil
}

// GetAttachmentContentForUser retrieves attachment content verifying user ownership.
func (s *MessageService) GetAttachmentContentForUser(ctx context.Context, messageID, attachmentID, userID domain.ID) (*domain.Attachment, io.ReadCloser, error) {
	// First get and verify the attachment
	attachment, err := s.GetAttachmentForUser(ctx, messageID, attachmentID, userID)
	if err != nil {
		return nil, nil, err
	}

	// Get the content
	content, err := s.repo.Attachments().GetContent(ctx, attachmentID)
	if err != nil {
		return nil, nil, &MessageServiceError{
			Op:      "get_attachment_content_for_user",
			Message: "failed to get attachment content",
			Err:     err,
		}
	}

	return attachment, content, nil
}

// DeleteMessageForUser deletes a message verifying user ownership.
func (s *MessageService) DeleteMessageForUser(ctx context.Context, messageID, userID domain.ID) error {
	if messageID.IsEmpty() {
		return &MessageServiceError{
			Op:      "delete_for_user",
			Message: "message ID is required",
			Err:     domain.ErrInvalidInput,
		}
	}

	if userID.IsEmpty() {
		return &MessageServiceError{
			Op:      "delete_for_user",
			Message: "user ID is required",
			Err:     domain.ErrInvalidInput,
		}
	}

	// First verify access
	msg, err := s.repo.Messages().GetByID(ctx, messageID)
	if err != nil {
		return &MessageServiceError{
			Op:      "delete_for_user",
			Message: "failed to get message",
			Err:     err,
		}
	}

	if err := s.verifyMessageAccess(ctx, msg, userID); err != nil {
		return err
	}

	// Delete the message
	return s.DeleteMessage(ctx, messageID)
}

// MarkAsReadForUser marks a message as read verifying user ownership.
func (s *MessageService) MarkAsReadForUser(ctx context.Context, messageID, userID domain.ID) error {
	if messageID.IsEmpty() {
		return &MessageServiceError{
			Op:      "mark_read_for_user",
			Message: "message ID is required",
			Err:     domain.ErrInvalidInput,
		}
	}

	if userID.IsEmpty() {
		return &MessageServiceError{
			Op:      "mark_read_for_user",
			Message: "user ID is required",
			Err:     domain.ErrInvalidInput,
		}
	}

	// First verify access
	msg, err := s.repo.Messages().GetByID(ctx, messageID)
	if err != nil {
		return &MessageServiceError{
			Op:      "mark_read_for_user",
			Message: "failed to get message",
			Err:     err,
		}
	}

	if err := s.verifyMessageAccess(ctx, msg, userID); err != nil {
		return err
	}

	return s.MarkAsRead(ctx, messageID)
}

// MarkAsUnreadForUser marks a message as unread verifying user ownership.
func (s *MessageService) MarkAsUnreadForUser(ctx context.Context, messageID, userID domain.ID) error {
	if messageID.IsEmpty() {
		return &MessageServiceError{
			Op:      "mark_unread_for_user",
			Message: "message ID is required",
			Err:     domain.ErrInvalidInput,
		}
	}

	if userID.IsEmpty() {
		return &MessageServiceError{
			Op:      "mark_unread_for_user",
			Message: "user ID is required",
			Err:     domain.ErrInvalidInput,
		}
	}

	// First verify access
	msg, err := s.repo.Messages().GetByID(ctx, messageID)
	if err != nil {
		return &MessageServiceError{
			Op:      "mark_unread_for_user",
			Message: "failed to get message",
			Err:     err,
		}
	}

	if err := s.verifyMessageAccess(ctx, msg, userID); err != nil {
		return err
	}

	return s.MarkAsUnread(ctx, messageID)
}

// StarForUser stars a message verifying user ownership.
func (s *MessageService) StarForUser(ctx context.Context, messageID, userID domain.ID) error {
	if messageID.IsEmpty() {
		return &MessageServiceError{
			Op:      "star_for_user",
			Message: "message ID is required",
			Err:     domain.ErrInvalidInput,
		}
	}

	if userID.IsEmpty() {
		return &MessageServiceError{
			Op:      "star_for_user",
			Message: "user ID is required",
			Err:     domain.ErrInvalidInput,
		}
	}

	// First verify access
	msg, err := s.repo.Messages().GetByID(ctx, messageID)
	if err != nil {
		return &MessageServiceError{
			Op:      "star_for_user",
			Message: "failed to get message",
			Err:     err,
		}
	}

	if err := s.verifyMessageAccess(ctx, msg, userID); err != nil {
		return err
	}

	if err := s.repo.Messages().Star(ctx, messageID); err != nil {
		return &MessageServiceError{
			Op:      "star_for_user",
			Message: "failed to star message",
			Err:     err,
		}
	}

	return nil
}

// UnstarForUser unstars a message verifying user ownership.
func (s *MessageService) UnstarForUser(ctx context.Context, messageID, userID domain.ID) error {
	if messageID.IsEmpty() {
		return &MessageServiceError{
			Op:      "unstar_for_user",
			Message: "message ID is required",
			Err:     domain.ErrInvalidInput,
		}
	}

	if userID.IsEmpty() {
		return &MessageServiceError{
			Op:      "unstar_for_user",
			Message: "user ID is required",
			Err:     domain.ErrInvalidInput,
		}
	}

	// First verify access
	msg, err := s.repo.Messages().GetByID(ctx, messageID)
	if err != nil {
		return &MessageServiceError{
			Op:      "unstar_for_user",
			Message: "failed to get message",
			Err:     err,
		}
	}

	if err := s.verifyMessageAccess(ctx, msg, userID); err != nil {
		return err
	}

	if err := s.repo.Messages().Unstar(ctx, messageID); err != nil {
		return &MessageServiceError{
			Op:      "unstar_for_user",
			Message: "failed to unstar message",
			Err:     err,
		}
	}

	return nil
}

// MarkAsSpamForUser marks a message as spam verifying user ownership.
func (s *MessageService) MarkAsSpamForUser(ctx context.Context, messageID, userID domain.ID) error {
	if messageID.IsEmpty() {
		return &MessageServiceError{
			Op:      "mark_spam_for_user",
			Message: "message ID is required",
			Err:     domain.ErrInvalidInput,
		}
	}

	if userID.IsEmpty() {
		return &MessageServiceError{
			Op:      "mark_spam_for_user",
			Message: "user ID is required",
			Err:     domain.ErrInvalidInput,
		}
	}

	// First verify access
	msg, err := s.repo.Messages().GetByID(ctx, messageID)
	if err != nil {
		return &MessageServiceError{
			Op:      "mark_spam_for_user",
			Message: "failed to get message",
			Err:     err,
		}
	}

	if err := s.verifyMessageAccess(ctx, msg, userID); err != nil {
		return err
	}

	if err := s.repo.Messages().MarkAsSpam(ctx, messageID); err != nil {
		return &MessageServiceError{
			Op:      "mark_spam_for_user",
			Message: "failed to mark message as spam",
			Err:     err,
		}
	}

	return nil
}

// MarkAsNotSpamForUser marks a message as not spam verifying user ownership.
func (s *MessageService) MarkAsNotSpamForUser(ctx context.Context, messageID, userID domain.ID) error {
	if messageID.IsEmpty() {
		return &MessageServiceError{
			Op:      "mark_not_spam_for_user",
			Message: "message ID is required",
			Err:     domain.ErrInvalidInput,
		}
	}

	if userID.IsEmpty() {
		return &MessageServiceError{
			Op:      "mark_not_spam_for_user",
			Message: "user ID is required",
			Err:     domain.ErrInvalidInput,
		}
	}

	// First verify access
	msg, err := s.repo.Messages().GetByID(ctx, messageID)
	if err != nil {
		return &MessageServiceError{
			Op:      "mark_not_spam_for_user",
			Message: "failed to get message",
			Err:     err,
		}
	}

	if err := s.verifyMessageAccess(ctx, msg, userID); err != nil {
		return err
	}

	if err := s.repo.Messages().MarkAsNotSpam(ctx, messageID); err != nil {
		return &MessageServiceError{
			Op:      "mark_not_spam_for_user",
			Message: "failed to mark message as not spam",
			Err:     err,
		}
	}

	return nil
}

// MoveMessageForUser moves a message to a different mailbox verifying user ownership of both.
func (s *MessageService) MoveMessageForUser(ctx context.Context, messageID, targetMailboxID, userID domain.ID) error {
	if messageID.IsEmpty() {
		return &MessageServiceError{
			Op:      "move_for_user",
			Message: "message ID is required",
			Err:     domain.ErrInvalidInput,
		}
	}

	if targetMailboxID.IsEmpty() {
		return &MessageServiceError{
			Op:      "move_for_user",
			Message: "target mailbox ID is required",
			Err:     domain.ErrInvalidInput,
		}
	}

	if userID.IsEmpty() {
		return &MessageServiceError{
			Op:      "move_for_user",
			Message: "user ID is required",
			Err:     domain.ErrInvalidInput,
		}
	}

	// First verify access to the message
	msg, err := s.repo.Messages().GetByID(ctx, messageID)
	if err != nil {
		return &MessageServiceError{
			Op:      "move_for_user",
			Message: "failed to get message",
			Err:     err,
		}
	}

	if err := s.verifyMessageAccess(ctx, msg, userID); err != nil {
		return err
	}

	// Verify access to the target mailbox
	targetMailbox, err := s.repo.Mailboxes().GetByID(ctx, targetMailboxID)
	if err != nil {
		return &MessageServiceError{
			Op:      "move_for_user",
			Message: "failed to get target mailbox",
			Err:     err,
		}
	}

	if targetMailbox.UserID != userID {
		return &MessageServiceError{
			Op:      "move_for_user",
			Message: "access denied to target mailbox",
			Err:     domain.ErrForbidden,
		}
	}

	return s.MoveMessage(ctx, messageID, targetMailboxID)
}

// BulkMarkAsReadForUser marks multiple messages as read verifying user ownership.
func (s *MessageService) BulkMarkAsReadForUser(ctx context.Context, messageIDs []domain.ID, userID domain.ID) (*repository.BulkOperation, error) {
	if userID.IsEmpty() {
		return nil, &MessageServiceError{
			Op:      "bulk_mark_read_for_user",
			Message: "user ID is required",
			Err:     domain.ErrInvalidInput,
		}
	}

	result := repository.NewBulkOperation()

	for _, messageID := range messageIDs {
		if err := s.MarkAsReadForUser(ctx, messageID, userID); err != nil {
			result.AddFailure(string(messageID), err)
		} else {
			result.AddSuccess()
		}
	}

	return result, nil
}

// BulkMarkAsUnreadForUser marks multiple messages as unread verifying user ownership.
func (s *MessageService) BulkMarkAsUnreadForUser(ctx context.Context, messageIDs []domain.ID, userID domain.ID) (*repository.BulkOperation, error) {
	if userID.IsEmpty() {
		return nil, &MessageServiceError{
			Op:      "bulk_mark_unread_for_user",
			Message: "user ID is required",
			Err:     domain.ErrInvalidInput,
		}
	}

	result := repository.NewBulkOperation()

	for _, messageID := range messageIDs {
		if err := s.MarkAsUnreadForUser(ctx, messageID, userID); err != nil {
			result.AddFailure(string(messageID), err)
		} else {
			result.AddSuccess()
		}
	}

	return result, nil
}

// BulkDeleteForUser deletes multiple messages verifying user ownership.
func (s *MessageService) BulkDeleteForUser(ctx context.Context, messageIDs []domain.ID, userID domain.ID) (*repository.BulkOperation, error) {
	if userID.IsEmpty() {
		return nil, &MessageServiceError{
			Op:      "bulk_delete_for_user",
			Message: "user ID is required",
			Err:     domain.ErrInvalidInput,
		}
	}

	result := repository.NewBulkOperation()

	for _, messageID := range messageIDs {
		if err := s.DeleteMessageForUser(ctx, messageID, userID); err != nil {
			result.AddFailure(string(messageID), err)
		} else {
			result.AddSuccess()
		}
	}

	return result, nil
}

// BulkMoveForUser moves multiple messages verifying user ownership.
func (s *MessageService) BulkMoveForUser(ctx context.Context, messageIDs []domain.ID, targetMailboxID, userID domain.ID) (*repository.BulkOperation, error) {
	if userID.IsEmpty() {
		return nil, &MessageServiceError{
			Op:      "bulk_move_for_user",
			Message: "user ID is required",
			Err:     domain.ErrInvalidInput,
		}
	}

	if targetMailboxID.IsEmpty() {
		return nil, &MessageServiceError{
			Op:      "bulk_move_for_user",
			Message: "target mailbox ID is required",
			Err:     domain.ErrInvalidInput,
		}
	}

	// Verify access to the target mailbox first
	targetMailbox, err := s.repo.Mailboxes().GetByID(ctx, targetMailboxID)
	if err != nil {
		return nil, &MessageServiceError{
			Op:      "bulk_move_for_user",
			Message: "failed to get target mailbox",
			Err:     err,
		}
	}

	if targetMailbox.UserID != userID {
		return nil, &MessageServiceError{
			Op:      "bulk_move_for_user",
			Message: "access denied to target mailbox",
			Err:     domain.ErrForbidden,
		}
	}

	result := repository.NewBulkOperation()

	for _, messageID := range messageIDs {
		if err := s.MoveMessageForUser(ctx, messageID, targetMailboxID, userID); err != nil {
			result.AddFailure(string(messageID), err)
		} else {
			result.AddSuccess()
		}
	}

	return result, nil
}

// BulkStarForUser stars multiple messages verifying user ownership.
func (s *MessageService) BulkStarForUser(ctx context.Context, messageIDs []domain.ID, userID domain.ID) (*repository.BulkOperation, error) {
	if userID.IsEmpty() {
		return nil, &MessageServiceError{
			Op:      "bulk_star_for_user",
			Message: "user ID is required",
			Err:     domain.ErrInvalidInput,
		}
	}

	result := repository.NewBulkOperation()

	for _, messageID := range messageIDs {
		if err := s.StarForUser(ctx, messageID, userID); err != nil {
			result.AddFailure(string(messageID), err)
		} else {
			result.AddSuccess()
		}
	}

	return result, nil
}

// BulkUnstarForUser unstars multiple messages verifying user ownership.
func (s *MessageService) BulkUnstarForUser(ctx context.Context, messageIDs []domain.ID, userID domain.ID) (*repository.BulkOperation, error) {
	if userID.IsEmpty() {
		return nil, &MessageServiceError{
			Op:      "bulk_unstar_for_user",
			Message: "user ID is required",
			Err:     domain.ErrInvalidInput,
		}
	}

	result := repository.NewBulkOperation()

	for _, messageID := range messageIDs {
		if err := s.UnstarForUser(ctx, messageID, userID); err != nil {
			result.AddFailure(string(messageID), err)
		} else {
			result.AddSuccess()
		}
	}

	return result, nil
}

// ListAttachmentsForUser lists attachments for a user with optional filtering.
// It ensures the user can only see attachments from their own mailboxes.
func (s *MessageService) ListAttachmentsForUser(
	ctx context.Context,
	userID domain.ID,
	filter *repository.AttachmentFilter,
	opts *repository.ListOptions,
) (*repository.ListResult[*domain.Attachment], error) {
	if userID.IsEmpty() {
		return nil, &MessageServiceError{
			Op:      "list_attachments_for_user",
			Message: "user ID is required",
			Err:     domain.ErrInvalidInput,
		}
	}

	// Get user's mailbox IDs
	mailboxIDs, err := s.getUserMailboxIDs(ctx, userID)
	if err != nil {
		return nil, err
	}

	if len(mailboxIDs) == 0 {
		return &repository.ListResult[*domain.Attachment]{
			Items: []*domain.Attachment{},
			Total: 0,
		}, nil
	}

	// If a specific message is requested, verify ownership
	if filter != nil && filter.MessageID != nil {
		msg, err := s.repo.Messages().GetByID(ctx, *filter.MessageID)
		if err != nil {
			return nil, &MessageServiceError{
				Op:      "list_attachments_for_user",
				Message: "failed to get message",
				Err:     err,
			}
		}

		if err := s.verifyMessageAccess(ctx, msg, userID); err != nil {
			return nil, err
		}
	} else {
		// Get all message IDs from user's mailboxes
		msgFilter := &repository.MessageFilter{
			MailboxIDs: mailboxIDs,
		}
		messages, err := s.repo.Messages().List(ctx, msgFilter, nil)
		if err != nil {
			return nil, &MessageServiceError{
				Op:      "list_attachments_for_user",
				Message: "failed to get messages",
				Err:     err,
			}
		}

		if len(messages.Items) == 0 {
			return &repository.ListResult[*domain.Attachment]{
				Items: []*domain.Attachment{},
				Total: 0,
			}, nil
		}

		// Create filter with message IDs
		messageIDs := make([]domain.ID, len(messages.Items))
		for i, msg := range messages.Items {
			messageIDs[i] = msg.ID
		}

		if filter == nil {
			filter = &repository.AttachmentFilter{}
		}
		filter.MessageIDs = messageIDs
	}

	result, err := s.repo.Attachments().List(ctx, filter, opts)
	if err != nil {
		return nil, &MessageServiceError{
			Op:      "list_attachments_for_user",
			Message: "failed to list attachments",
			Err:     err,
		}
	}

	return result, nil
}

// GetAttachmentByIDForUser retrieves an attachment by ID verifying user ownership.
func (s *MessageService) GetAttachmentByIDForUser(ctx context.Context, attachmentID, userID domain.ID) (*domain.Attachment, error) {
	if attachmentID.IsEmpty() {
		return nil, &MessageServiceError{
			Op:      "get_attachment_by_id_for_user",
			Message: "attachment ID is required",
			Err:     domain.ErrInvalidInput,
		}
	}

	if userID.IsEmpty() {
		return nil, &MessageServiceError{
			Op:      "get_attachment_by_id_for_user",
			Message: "user ID is required",
			Err:     domain.ErrInvalidInput,
		}
	}

	// Get the attachment
	attachment, err := s.repo.Attachments().GetByID(ctx, attachmentID)
	if err != nil {
		return nil, &MessageServiceError{
			Op:      "get_attachment_by_id_for_user",
			Message: "failed to get attachment",
			Err:     err,
		}
	}

	// Verify access to the message
	msg, err := s.repo.Messages().GetByID(ctx, attachment.MessageID)
	if err != nil {
		return nil, &MessageServiceError{
			Op:      "get_attachment_by_id_for_user",
			Message: "failed to get message",
			Err:     err,
		}
	}

	if err := s.verifyMessageAccess(ctx, msg, userID); err != nil {
		return nil, err
	}

	return attachment, nil
}

// GetAttachmentContentByIDForUser retrieves attachment content by ID verifying user ownership.
func (s *MessageService) GetAttachmentContentByIDForUser(ctx context.Context, attachmentID, userID domain.ID) (*domain.Attachment, io.ReadCloser, error) {
	// First get and verify the attachment
	attachment, err := s.GetAttachmentByIDForUser(ctx, attachmentID, userID)
	if err != nil {
		return nil, nil, err
	}

	// Get the content
	content, err := s.repo.Attachments().GetContent(ctx, attachmentID)
	if err != nil {
		return nil, nil, &MessageServiceError{
			Op:      "get_attachment_content_by_id_for_user",
			Message: "failed to get attachment content",
			Err:     err,
		}
	}

	return attachment, content, nil
}
