package imap

import (
	"context"

	"github.com/emersion/go-imap/v2"
	"github.com/emersion/go-imap/v2/imapserver"

	"yunt/internal/domain"
	"yunt/internal/repository"
)

// ExpungeHandler handles IMAP EXPUNGE command operations.
// EXPUNGE permanently removes all messages with the \Deleted flag set.
type ExpungeHandler struct {
	repo         repository.Repository
	userID       domain.ID
	selectedMbox *domain.Mailbox
	notifyBridge *NotificationBridge
}

// NewExpungeHandler creates a new ExpungeHandler.
func NewExpungeHandler(repo repository.Repository, userID domain.ID, selectedMbox *domain.Mailbox, bridge *NotificationBridge) *ExpungeHandler {
	return &ExpungeHandler{
		repo:         repo,
		userID:       userID,
		selectedMbox: selectedMbox,
		notifyBridge: bridge,
	}
}

// ExpungeResult contains the result of an EXPUNGE operation.
type ExpungeResult struct {
	// ExpungedSeqNums are the sequence numbers of expunged messages.
	// They are returned in descending order per IMAP protocol.
	ExpungedSeqNums []uint32

	// ExpungedCount is the number of messages expunged.
	ExpungedCount int

	// FreedSpace is the total size of expunged messages in bytes.
	FreedSpace int64
}

// Expunge removes all messages marked with \Deleted flag.
// It sends EXPUNGE responses for each removed message via the ExpungeWriter.
// If uids is provided (UID EXPUNGE), only messages matching those UIDs are considered.
func (h *ExpungeHandler) Expunge(ctx context.Context, w *imapserver.ExpungeWriter, uids *imap.UIDSet) error {
	if h.selectedMbox == nil {
		return &imap.Error{
			Type: imap.StatusResponseTypeNo,
			Text: "No mailbox selected",
		}
	}

	result, err := h.repo.Messages().ListByMailbox(ctx, h.selectedMbox.ID, imapListOptions())
	if err != nil {
		return &imap.Error{
			Type: imap.StatusResponseTypeNo,
			Text: "Failed to list messages",
		}
	}

	// Find messages marked with \Deleted flag
	deletedMessages := h.findDeletedMessages(result.Items, uids)

	if len(deletedMessages.seqNums) == 0 {
		return nil // Nothing to expunge
	}

	// Delete messages in reverse sequence number order
	// This is important because deleting a message changes the sequence numbers
	// of all subsequent messages
	sortUint32Desc(deletedMessages.seqNums)

	for _, seqNum := range deletedMessages.seqNums {
		msg := deletedMessages.messages[seqNum]

		// Delete the message from the repository
		if err := h.repo.Messages().Delete(ctx, msg.ID); err != nil {
			continue
		}

		// Write the EXPUNGE response for this message
		w.WriteExpunge(seqNum)

		if h.notifyBridge != nil {
			h.notifyBridge.NotifyMessageExpunged(h.selectedMbox.ID, domain.ID(""), msg.ID, seqNum)
		}
	}

	// Update mailbox statistics
	if err := h.repo.Mailboxes().RecalculateStats(ctx, h.selectedMbox.ID); err != nil {
		// Log but don't fail - the expunge was successful
	}

	return nil
}

// ExpungeByUIDs removes only messages matching the specified UIDs that have \Deleted flag.
// This implements UID EXPUNGE (RFC 4315).
func (h *ExpungeHandler) ExpungeByUIDs(ctx context.Context, w *imapserver.ExpungeWriter, uids imap.UIDSet) error {
	return h.Expunge(ctx, w, &uids)
}

// deletedMessageSet holds messages to be expunged with their sequence numbers.
type deletedMessageSet struct {
	messages map[uint32]*domain.Message
	seqNums  []uint32
}

// findDeletedMessages identifies messages with \Deleted flag that should be expunged.
func (h *ExpungeHandler) findDeletedMessages(allMessages []*domain.Message, uidFilter *imap.UIDSet) *deletedMessageSet {
	result := &deletedMessageSet{
		messages: make(map[uint32]*domain.Message),
		seqNums:  make([]uint32, 0),
	}

	for i, msg := range allMessages {
		seqNum := uint32(i + 1)
		uid := imap.UID(msg.IMAPUID)

		// If a UID filter is specified, check if this message matches
		if uidFilter != nil && !uidFilter.Contains(uid) {
			continue
		}

		if h.isMessageDeleted(msg) {
			result.messages[seqNum] = msg
			result.seqNums = append(result.seqNums, seqNum)
		}
	}

	return result
}

// isMessageDeleted checks if a message has the \Deleted flag set.
func (h *ExpungeHandler) isMessageDeleted(msg *domain.Message) bool {
	return msg.IsDeleted
}

// WriteExpungeNotifications sends EXPUNGE notifications to other sessions.
// This is used to notify other connected clients about expunged messages.
func WriteExpungeNotifications(w *imapserver.UpdateWriter, expungedSeqNums []uint32) error {
	// Sort in descending order as required by IMAP protocol
	sortUint32Desc(expungedSeqNums)

	for _, seqNum := range expungedSeqNums {
		if err := w.WriteExpunge(seqNum); err != nil {
			return err
		}
	}

	return nil
}

// ExpungeAll removes all messages in the mailbox (used internally for mailbox deletion).
// This is not exposed as an IMAP command.
func (h *ExpungeHandler) ExpungeAll(ctx context.Context) (*ExpungeResult, error) {
	if h.selectedMbox == nil {
		return nil, &imap.Error{
			Type: imap.StatusResponseTypeNo,
			Text: "No mailbox selected",
		}
	}

	// Delete all messages in the mailbox
	deleted, err := h.repo.Messages().DeleteByMailbox(ctx, h.selectedMbox.ID)
	if err != nil {
		return nil, &imap.Error{
			Type: imap.StatusResponseTypeNo,
			Text: "Failed to delete messages",
		}
	}

	// Update mailbox statistics
	if err := h.repo.Mailboxes().RecalculateStats(ctx, h.selectedMbox.ID); err != nil {
		// Log but don't fail
	}

	return &ExpungeResult{
		ExpungedCount: int(deleted),
	}, nil
}
