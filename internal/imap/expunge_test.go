package imap

import (
	"testing"

	"github.com/emersion/go-imap/v2"

	"yunt/internal/domain"
)

func TestExpungeHandler_IsMessageDeleted(t *testing.T) {
	handler := &ExpungeHandler{}

	// Currently, isMessageDeleted always returns false as the domain model
	// doesn't have a dedicated deleted flag. This test documents the current
	// behavior and should be updated when the domain model is extended.
	msg := &domain.Message{
		ID:        domain.ID("test-msg"),
		MailboxID: domain.ID("test-mailbox"),
		Status:    domain.MessageUnread,
	}

	if handler.isMessageDeleted(msg) {
		t.Error("Expected isMessageDeleted to return false (no deleted flag in domain model)")
	}
}

func TestExpungeHandler_FindDeletedMessages(t *testing.T) {
	handler := &ExpungeHandler{
		selectedMbox: &domain.Mailbox{
			ID:   domain.ID("test-mailbox"),
			Name: "INBOX",
		},
	}

	now := domain.Now()
	messages := []*domain.Message{
		{ID: domain.ID("msg-1"), MailboxID: domain.ID("test-mailbox"), ReceivedAt: now},
		{ID: domain.ID("msg-2"), MailboxID: domain.ID("test-mailbox"), ReceivedAt: now},
		{ID: domain.ID("msg-3"), MailboxID: domain.ID("test-mailbox"), ReceivedAt: now},
	}

	// Without UID filter - should find no messages (since isMessageDeleted returns false)
	result := handler.findDeletedMessages(messages, nil)
	if len(result.seqNums) != 0 {
		t.Errorf("Expected 0 deleted messages, got %d", len(result.seqNums))
	}

	// With UID filter - should still find no messages
	uidFilter := imap.UIDSetNum(1, 2)
	result = handler.findDeletedMessages(messages, &uidFilter)
	if len(result.seqNums) != 0 {
		t.Errorf("Expected 0 deleted messages with UID filter, got %d", len(result.seqNums))
	}
}

func TestDeletedMessageSet(t *testing.T) {
	dms := &deletedMessageSet{
		messages: make(map[uint32]*domain.Message),
		seqNums:  make([]uint32, 0),
	}

	// Initially empty
	if len(dms.seqNums) != 0 {
		t.Errorf("Expected 0 seqNums, got %d", len(dms.seqNums))
	}
	if len(dms.messages) != 0 {
		t.Errorf("Expected 0 messages, got %d", len(dms.messages))
	}

	// Add messages
	msg1 := &domain.Message{ID: domain.ID("msg-1")}
	msg2 := &domain.Message{ID: domain.ID("msg-2")}

	dms.messages[1] = msg1
	dms.messages[2] = msg2
	dms.seqNums = append(dms.seqNums, 1, 2)

	if len(dms.seqNums) != 2 {
		t.Errorf("Expected 2 seqNums, got %d", len(dms.seqNums))
	}
	if len(dms.messages) != 2 {
		t.Errorf("Expected 2 messages, got %d", len(dms.messages))
	}
	if dms.messages[1] != msg1 {
		t.Error("Expected msg1 at seqNum 1")
	}
	if dms.messages[2] != msg2 {
		t.Error("Expected msg2 at seqNum 2")
	}
}

func TestExpungeResult(t *testing.T) {
	result := &ExpungeResult{
		ExpungedSeqNums: []uint32{5, 4, 3, 2, 1},
		ExpungedCount:   5,
		FreedSpace:      10240,
	}

	if result.ExpungedCount != 5 {
		t.Errorf("Expected ExpungedCount 5, got %d", result.ExpungedCount)
	}
	if result.FreedSpace != 10240 {
		t.Errorf("Expected FreedSpace 10240, got %d", result.FreedSpace)
	}
	if len(result.ExpungedSeqNums) != 5 {
		t.Errorf("Expected 5 ExpungedSeqNums, got %d", len(result.ExpungedSeqNums))
	}

	// Verify sequence numbers are in descending order
	for i := 0; i < len(result.ExpungedSeqNums)-1; i++ {
		if result.ExpungedSeqNums[i] < result.ExpungedSeqNums[i+1] {
			t.Errorf("Sequence numbers should be in descending order at index %d", i)
		}
	}
}
