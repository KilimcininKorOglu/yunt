package imap

import (
	"context"

	"github.com/emersion/go-imap/v2"
	"github.com/emersion/go-imap/v2/imapserver"

	"yunt/internal/domain"
	"yunt/internal/repository"
)

// StoreHandler handles IMAP STORE command operations.
// STORE modifies message flags and persists changes to the database.
type StoreHandler struct {
	repo         repository.Repository
	userID       domain.ID
	selectedMbox *domain.Mailbox
}

// NewStoreHandler creates a new StoreHandler.
func NewStoreHandler(repo repository.Repository, userID domain.ID, selectedMbox *domain.Mailbox) *StoreHandler {
	return &StoreHandler{
		repo:         repo,
		userID:       userID,
		selectedMbox: selectedMbox,
	}
}

// Store modifies flags for messages specified by the number set.
// It supports three operations: set flags, add flags, and remove flags.
// Returns the updated flags for each message through the FetchWriter.
func (h *StoreHandler) Store(ctx context.Context, w *imapserver.FetchWriter, numSet imap.NumSet, flags *imap.StoreFlags, options *imap.StoreOptions) error {
	if h.selectedMbox == nil {
		return &imap.Error{
			Type: imap.StatusResponseTypeNo,
			Text: "No mailbox selected",
		}
	}

	// Get messages matching the number set
	messages, err := h.getMessagesForNumSet(ctx, numSet)
	if err != nil {
		return err
	}

	// Track flag changes for each message
	changes := make([]*FlagChange, 0, len(messages))

	// Apply flag changes to each message
	for seqNum, msg := range messages {
		uid := imap.UID(seqNum) // Simplified: UID = sequence number

		// Get current flags
		oldFlags := NewFlagSetFromMessage(msg)

		// Calculate new flags based on operation
		newFlags := h.applyFlagOperation(oldFlags, flags)

		// Create flag change record
		change := &FlagChange{
			MessageID: msg.ID,
			SeqNum:    seqNum,
			UID:       uid,
			OldFlags:  oldFlags,
			NewFlags:  newFlags,
		}

		// Only persist and track if there are actual changes
		if change.HasChanges() {
			// Persist flag changes to database
			if err := h.persistFlagChanges(ctx, msg, change); err != nil {
				return err
			}

			changes = append(changes, change)
		}

		// Write response unless silent operation is requested
		if !flags.Silent {
			if err := h.writeFetchResponse(w, seqNum, newFlags, options); err != nil {
				return err
			}
		}
	}

	// Update mailbox unseen count if any \Seen flags changed
	if err := h.updateMailboxUnseenCount(ctx, changes); err != nil {
		return err
	}

	return nil
}

// getMessagesForNumSet retrieves messages matching the given number set.
func (h *StoreHandler) getMessagesForNumSet(ctx context.Context, numSet imap.NumSet) (map[uint32]*domain.Message, error) {
	// Get all messages in the mailbox
	result, err := h.repo.Messages().ListByMailbox(ctx, h.selectedMbox.ID, nil)
	if err != nil {
		return nil, &imap.Error{
			Type: imap.StatusResponseTypeNo,
			Text: "Failed to list messages",
		}
	}

	messages := make(map[uint32]*domain.Message)

	// Match messages against the number set
	for i, msg := range result.Items {
		seqNum := uint32(i + 1)
		uid := imap.UID(i + 1) // Simplified: UID = sequence number

		if numSetContainsMessage(numSet, seqNum, uid) {
			messages[seqNum] = msg
		}
	}

	return messages, nil
}

// numSetContainsMessage checks if a sequence number or UID is in the number set.
func numSetContainsMessage(numSet imap.NumSet, seqNum uint32, uid imap.UID) bool {
	switch ns := numSet.(type) {
	case imap.SeqSet:
		return ns.Contains(seqNum)
	case imap.UIDSet:
		return ns.Contains(uid)
	default:
		return false
	}
}

// applyFlagOperation applies the flag operation and returns the resulting flags.
func (h *StoreHandler) applyFlagOperation(currentFlags *FlagSet, storeFlags *imap.StoreFlags) *FlagSet {
	result := currentFlags.Clone()

	// Filter out non-permanent flags
	flags := FilterPermanentFlags(storeFlags.Flags)

	switch storeFlags.Op {
	case imap.StoreFlagsSet:
		// Replace all flags with the specified flags
		result.Replace(flags)

	case imap.StoreFlagsAdd:
		// Add the specified flags to existing flags
		result.AddAll(flags)

	case imap.StoreFlagsDel:
		// Remove the specified flags from existing flags
		result.RemoveAll(flags)
	}

	return result
}

// persistFlagChanges persists flag changes to the database.
func (h *StoreHandler) persistFlagChanges(ctx context.Context, msg *domain.Message, change *FlagChange) error {
	// Handle \Seen flag changes (maps to read/unread status)
	if change.SeenChanged() {
		if change.IsNowSeen() {
			_, err := h.repo.Messages().MarkAsRead(ctx, msg.ID)
			if err != nil {
				return &imap.Error{
					Type: imap.StatusResponseTypeNo,
					Text: "Failed to mark message as read",
				}
			}
			msg.Status = domain.MessageRead
		} else if change.IsNowUnseen() {
			_, err := h.repo.Messages().MarkAsUnread(ctx, msg.ID)
			if err != nil {
				return &imap.Error{
					Type: imap.StatusResponseTypeNo,
					Text: "Failed to mark message as unread",
				}
			}
			msg.Status = domain.MessageUnread
		}
	}

	// Handle \Flagged flag changes (maps to starred status)
	if change.FlaggedChanged() {
		if change.IsNowFlagged() {
			err := h.repo.Messages().Star(ctx, msg.ID)
			if err != nil {
				return &imap.Error{
					Type: imap.StatusResponseTypeNo,
					Text: "Failed to star message",
				}
			}
			msg.IsStarred = true
		} else if change.IsNowUnflagged() {
			err := h.repo.Messages().Unstar(ctx, msg.ID)
			if err != nil {
				return &imap.Error{
					Type: imap.StatusResponseTypeNo,
					Text: "Failed to unstar message",
				}
			}
			msg.IsStarred = false
		}
	}

	// Handle \Deleted flag changes
	if change.DeletedChanged() {
		if change.IsNowDeleted() {
			if err := h.repo.Messages().MarkAsDeleted(ctx, msg.ID); err != nil {
				return &imap.Error{
					Type: imap.StatusResponseTypeNo,
					Text: "Failed to mark message as deleted",
				}
			}
			msg.IsDeleted = true
		} else if change.IsNowUndeleted() {
			if err := h.repo.Messages().UnmarkAsDeleted(ctx, msg.ID); err != nil {
				return &imap.Error{
					Type: imap.StatusResponseTypeNo,
					Text: "Failed to unmark message as deleted",
				}
			}
			msg.IsDeleted = false
		}
	}

	return nil
}

// writeFetchResponse writes the FETCH response with updated flags.
func (h *StoreHandler) writeFetchResponse(w *imapserver.FetchWriter, seqNum uint32, flags *FlagSet, options *imap.StoreOptions) error {
	respWriter := w.CreateMessage(seqNum)
	defer respWriter.Close()

	respWriter.WriteFlags(flags.ToSlice())

	return nil
}

// updateMailboxUnseenCount updates the mailbox unseen count after flag changes.
func (h *StoreHandler) updateMailboxUnseenCount(ctx context.Context, changes []*FlagChange) error {
	// Count how many messages changed their seen status
	var seenDelta int64 = 0

	for _, change := range changes {
		if change.IsNowSeen() {
			seenDelta-- // Message became read, decrease unseen count
		} else if change.IsNowUnseen() {
			seenDelta++ // Message became unread, increase unseen count
		}
	}

	// If there's no change in unseen count, we're done
	if seenDelta == 0 {
		return nil
	}

	// Update the mailbox unseen count
	// The repository methods already update the mailbox stats internally
	// when MarkAsRead/MarkAsUnread are called, so we may not need this
	// explicit update. However, we keep the structure for extensibility.

	return nil
}

// StoreResult contains the result of a STORE operation.
type StoreResult struct {
	// Changes contains the flag changes for each affected message.
	Changes []*FlagChange

	// AffectedCount is the number of messages that were modified.
	AffectedCount int

	// UnseenCountDelta is the change in unseen message count.
	UnseenCountDelta int64
}

// NewStoreResult creates a new StoreResult.
func NewStoreResult() *StoreResult {
	return &StoreResult{
		Changes: make([]*FlagChange, 0),
	}
}

// AddChange adds a flag change to the result.
func (r *StoreResult) AddChange(change *FlagChange) {
	if change.HasChanges() {
		r.Changes = append(r.Changes, change)
		r.AffectedCount++

		if change.IsNowSeen() {
			r.UnseenCountDelta--
		} else if change.IsNowUnseen() {
			r.UnseenCountDelta++
		}
	}
}
