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

// CopyHandler handles IMAP COPY and MOVE command operations.
// COPY duplicates messages to a target mailbox while preserving the originals.
// MOVE copies messages and then marks the originals with \Deleted flag.
type CopyHandler struct {
	repo         repository.Repository
	userID       domain.ID
	selectedMbox *domain.Mailbox
}

// NewCopyHandler creates a new CopyHandler.
func NewCopyHandler(repo repository.Repository, userID domain.ID, selectedMbox *domain.Mailbox) *CopyHandler {
	return &CopyHandler{
		repo:         repo,
		userID:       userID,
		selectedMbox: selectedMbox,
	}
}

// CopyResult contains the result of a COPY or MOVE operation.
type CopyResult struct {
	// SourceUIDs are the UIDs of the copied/moved messages in the source mailbox.
	SourceUIDs imap.UIDSet

	// DestUIDs are the UIDs of the new messages in the destination mailbox.
	DestUIDs imap.UIDSet

	// UIDValidity is the UID validity of the destination mailbox.
	UIDValidity uint32

	// CopiedCount is the number of messages successfully copied.
	CopiedCount int

	// FailedCount is the number of messages that failed to copy.
	FailedCount int
}

// Copy copies messages to the target mailbox.
// It returns COPYUID response data per RFC 4315.
func (h *CopyHandler) Copy(ctx context.Context, numSet imap.NumSet, destName string) (*imap.CopyData, error) {
	if h.selectedMbox == nil {
		return nil, &imap.Error{
			Type: imap.StatusResponseTypeNo,
			Text: "No mailbox selected",
		}
	}

	// Find the destination mailbox
	destMailbox, err := h.findMailboxByName(ctx, destName)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return nil, &imap.Error{
				Type: imap.StatusResponseTypeNo,
				Code: imap.ResponseCodeTryCreate,
				Text: "Destination mailbox does not exist",
			}
		}
		return nil, &imap.Error{
			Type: imap.StatusResponseTypeNo,
			Text: "Failed to find destination mailbox",
		}
	}

	// Cannot copy to the same mailbox
	if destMailbox.ID == h.selectedMbox.ID {
		return nil, &imap.Error{
			Type: imap.StatusResponseTypeNo,
			Text: "Cannot copy messages to the same mailbox",
		}
	}

	// Get messages matching the number set
	messages, seqToUID, err := h.getMessagesForNumSet(ctx, numSet)
	if err != nil {
		return nil, err
	}

	if len(messages) == 0 {
		// No messages to copy, but this is not an error
		return &imap.CopyData{
			UIDValidity: generateUIDValidity(destMailbox),
		}, nil
	}

	// Copy each message to the destination mailbox
	result, err := h.copyMessages(ctx, messages, seqToUID, destMailbox)
	if err != nil {
		return nil, err
	}

	// Build COPYUID response
	copyData := &imap.CopyData{
		UIDValidity: result.UIDValidity,
		SourceUIDs:  result.SourceUIDs,
		DestUIDs:    result.DestUIDs,
	}

	return copyData, nil
}

// Move copies messages to the target mailbox and marks originals as deleted.
// This implements the MOVE extension (RFC 6851).
func (h *CopyHandler) Move(ctx context.Context, numSet imap.NumSet, destName string) (*imap.CopyData, []uint32, error) {
	if h.selectedMbox == nil {
		return nil, nil, &imap.Error{
			Type: imap.StatusResponseTypeNo,
			Text: "No mailbox selected",
		}
	}

	// Find the destination mailbox
	destMailbox, err := h.findMailboxByName(ctx, destName)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return nil, nil, &imap.Error{
				Type: imap.StatusResponseTypeNo,
				Code: imap.ResponseCodeTryCreate,
				Text: "Destination mailbox does not exist",
			}
		}
		return nil, nil, &imap.Error{
			Type: imap.StatusResponseTypeNo,
			Text: "Failed to find destination mailbox",
		}
	}

	// Cannot move to the same mailbox
	if destMailbox.ID == h.selectedMbox.ID {
		return nil, nil, &imap.Error{
			Type: imap.StatusResponseTypeNo,
			Text: "Cannot move messages to the same mailbox",
		}
	}

	// Get messages matching the number set
	messages, seqToUID, err := h.getMessagesForNumSet(ctx, numSet)
	if err != nil {
		return nil, nil, err
	}

	if len(messages) == 0 {
		return &imap.CopyData{
			UIDValidity: generateUIDValidity(destMailbox),
		}, nil, nil
	}

	// Copy each message to the destination mailbox
	result, err := h.copyMessages(ctx, messages, seqToUID, destMailbox)
	if err != nil {
		return nil, nil, err
	}

	// Delete the original messages from the source mailbox
	expungedSeqNums, err := h.deleteOriginalMessages(ctx, messages)
	if err != nil {
		// Log the error but don't fail the operation
		// Messages were already copied successfully
	}

	// Build response
	copyData := &imap.CopyData{
		UIDValidity: result.UIDValidity,
		SourceUIDs:  result.SourceUIDs,
		DestUIDs:    result.DestUIDs,
	}

	return copyData, expungedSeqNums, nil
}

// getMessagesForNumSet retrieves messages matching the given number set.
func (h *CopyHandler) getMessagesForNumSet(ctx context.Context, numSet imap.NumSet) (map[uint32]*domain.Message, map[uint32]imap.UID, error) {
	result, err := h.repo.Messages().ListByMailbox(ctx, h.selectedMbox.ID, imapListOptions())
	if err != nil {
		return nil, nil, &imap.Error{
			Type: imap.StatusResponseTypeNo,
			Text: "Failed to list messages",
		}
	}

	messages := make(map[uint32]*domain.Message)
	seqToUID := make(map[uint32]imap.UID)

	// Match messages against the number set
	for i, msg := range result.Items {
		seqNum := uint32(i + 1)
		uid := imap.UID(msg.IMAPUID)

		if numSetContainsMessage(numSet, seqNum, uid) {
			messages[seqNum] = msg
			seqToUID[seqNum] = uid
		}
	}

	return messages, seqToUID, nil
}

// copyMessages copies messages to the destination mailbox.
func (h *CopyHandler) copyMessages(ctx context.Context, messages map[uint32]*domain.Message, seqToUID map[uint32]imap.UID, destMailbox *domain.Mailbox) (*CopyResult, error) {
	result := &CopyResult{
		UIDValidity: generateUIDValidity(destMailbox),
	}

	// Sort sequence numbers for consistent ordering
	seqNums := make([]uint32, 0, len(messages))
	for seqNum := range messages {
		seqNums = append(seqNums, seqNum)
	}
	sortUint32(seqNums)

	for _, seqNum := range seqNums {
		msg := messages[seqNum]
		sourceUID := seqToUID[seqNum]

		// Create a copy of the message for the destination mailbox
		newMsgID := domain.ID(uuid.New().String())
		newMsg := h.copyMessageData(msg, newMsgID, destMailbox.ID)

		// Create auto-assigns IMAPUID
		if err := h.repo.Messages().Create(ctx, newMsg); err != nil {
			result.FailedCount++
			continue
		}

		// Copy raw body if available
		rawBody, err := h.repo.Messages().GetRawBody(ctx, msg.ID)
		if err == nil && len(rawBody) > 0 {
			if err := h.repo.Messages().StoreRawBody(ctx, newMsgID, rawBody); err != nil {
				// Log but don't fail the operation
			}
		}

		destUID := imap.UID(newMsg.IMAPUID)

		// Add to result sets
		result.SourceUIDs.AddNum(sourceUID)
		result.DestUIDs.AddNum(destUID)
		result.CopiedCount++
	}

	return result, nil
}

// copyMessageData creates a copy of a message for a new mailbox.
func (h *CopyHandler) copyMessageData(src *domain.Message, newID, destMailboxID domain.ID) *domain.Message {
	now := domain.Now()

	newMsg := &domain.Message{
		ID:              newID,
		MailboxID:       destMailboxID,
		MessageID:       src.MessageID,
		From:            src.From,
		To:              make([]domain.EmailAddress, len(src.To)),
		Cc:              make([]domain.EmailAddress, len(src.Cc)),
		Bcc:             make([]domain.EmailAddress, len(src.Bcc)),
		ReplyTo:         src.ReplyTo,
		Subject:         src.Subject,
		TextBody:        src.TextBody,
		HTMLBody:        src.HTMLBody,
		Headers:         make(map[string]string),
		ContentType:     src.ContentType,
		Size:            src.Size,
		AttachmentCount: src.AttachmentCount,
		Status:          src.Status,
		IsStarred:       src.IsStarred,
		IsSpam:          src.IsSpam,
		InReplyTo:       src.InReplyTo,
		References:      make([]string, len(src.References)),
		ReceivedAt:      src.ReceivedAt,
		SentAt:          src.SentAt,
		CreatedAt:       now,
		UpdatedAt:       now,
	}

	// Copy slices
	copy(newMsg.To, src.To)
	copy(newMsg.Cc, src.Cc)
	copy(newMsg.Bcc, src.Bcc)
	copy(newMsg.References, src.References)

	// Copy headers map
	for k, v := range src.Headers {
		newMsg.Headers[k] = v
	}

	return newMsg
}

// deleteOriginalMessages deletes the original messages after a MOVE operation.
func (h *CopyHandler) deleteOriginalMessages(ctx context.Context, messages map[uint32]*domain.Message) ([]uint32, error) {
	// Sort sequence numbers in descending order for proper expunge
	seqNums := make([]uint32, 0, len(messages))
	for seqNum := range messages {
		seqNums = append(seqNums, seqNum)
	}
	sortUint32Desc(seqNums)

	expungedSeqNums := make([]uint32, 0, len(messages))

	for _, seqNum := range seqNums {
		msg := messages[seqNum]

		// Delete the message
		if err := h.repo.Messages().Delete(ctx, msg.ID); err != nil {
			continue
		}

		expungedSeqNums = append(expungedSeqNums, seqNum)
	}

	// Update source mailbox statistics
	if err := h.repo.Mailboxes().RecalculateStats(ctx, h.selectedMbox.ID); err != nil {
		// Log but don't fail
	}

	return expungedSeqNums, nil
}

// findMailboxByName finds a mailbox by name for the current user.
func (h *CopyHandler) findMailboxByName(ctx context.Context, name string) (*domain.Mailbox, error) {
	normalizedName := NormalizeMailboxName(name)

	// List all mailboxes for the user and find by name
	result, err := h.repo.Mailboxes().ListByUser(ctx, h.userID, nil)
	if err != nil {
		return nil, err
	}

	for _, mailbox := range result.Items {
		if strings.EqualFold(NormalizeMailboxName(mailbox.Name), normalizedName) {
			return mailbox, nil
		}
	}

	return nil, domain.ErrNotFound
}

// sortUint32 sorts a slice of uint32 in ascending order.
func sortUint32(s []uint32) {
	for i := 0; i < len(s)-1; i++ {
		for j := i + 1; j < len(s); j++ {
			if s[i] > s[j] {
				s[i], s[j] = s[j], s[i]
			}
		}
	}
}

// sortUint32Desc sorts a slice of uint32 in descending order.
func sortUint32Desc(s []uint32) {
	for i := 0; i < len(s)-1; i++ {
		for j := i + 1; j < len(s); j++ {
			if s[i] < s[j] {
				s[i], s[j] = s[j], s[i]
			}
		}
	}
}
