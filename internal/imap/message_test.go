package imap

import (
	"strings"
	"testing"
	"time"

	"github.com/emersion/go-imap/v2"

	"yunt/internal/domain"
)

func TestNewIMAPMessage(t *testing.T) {
	msg := &domain.Message{
		ID:        domain.ID("msg-1"),
		MailboxID: domain.ID("inbox-1"),
		Subject:   "Test",
	}

	imapMsg := NewIMAPMessage(msg, 5, 10)

	if imapMsg.Message() != msg {
		t.Error("Message() should return the underlying message")
	}

	if imapMsg.SeqNum() != 5 {
		t.Errorf("SeqNum() = %d, want 5", imapMsg.SeqNum())
	}

	if imapMsg.UID() != 10 {
		t.Errorf("UID() = %d, want 10", imapMsg.UID())
	}
}

func TestIMAPMessage_Flags(t *testing.T) {
	tests := []struct {
		name     string
		msg      *domain.Message
		wantSeen bool
		wantFlag bool
		wantAns  bool
	}{
		{
			name: "read and starred message",
			msg: &domain.Message{
				Status:    domain.MessageRead,
				IsStarred: true,
			},
			wantSeen: true,
			wantFlag: true,
			wantAns:  false,
		},
		{
			name: "unread message",
			msg: &domain.Message{
				Status: domain.MessageUnread,
			},
			wantSeen: false,
			wantFlag: false,
			wantAns:  false,
		},
		{
			name: "replied message",
			msg: &domain.Message{
				InReplyTo: "original@example.com",
			},
			wantAns: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			imapMsg := NewIMAPMessage(tt.msg, 1, 1)
			flags := imapMsg.Flags()

			hasSeen := false
			hasFlag := false
			hasAns := false

			for _, f := range flags {
				switch f {
				case imap.FlagSeen:
					hasSeen = true
				case imap.FlagFlagged:
					hasFlag = true
				case imap.FlagAnswered:
					hasAns = true
				}
			}

			if hasSeen != tt.wantSeen {
				t.Errorf("\\Seen flag = %v, want %v", hasSeen, tt.wantSeen)
			}
			if hasFlag != tt.wantFlag {
				t.Errorf("\\Flagged flag = %v, want %v", hasFlag, tt.wantFlag)
			}
			if hasAns != tt.wantAns {
				t.Errorf("\\Answered flag = %v, want %v", hasAns, tt.wantAns)
			}
		})
	}
}

func TestIMAPMessage_HasFlag(t *testing.T) {
	msg := &domain.Message{
		Status:    domain.MessageRead,
		IsStarred: true,
	}

	imapMsg := NewIMAPMessage(msg, 1, 1)

	if !imapMsg.HasFlag(imap.FlagSeen) {
		t.Error("Expected HasFlag(\\Seen) = true")
	}

	if !imapMsg.HasFlag(imap.FlagFlagged) {
		t.Error("Expected HasFlag(\\Flagged) = true")
	}

	if imapMsg.HasFlag(imap.FlagDeleted) {
		t.Error("Expected HasFlag(\\Deleted) = false")
	}
}

func TestIMAPMessage_Envelope(t *testing.T) {
	sentTime := time.Date(2024, 1, 15, 10, 0, 0, 0, time.UTC)
	msg := &domain.Message{
		Subject:   "Test Subject",
		MessageID: "msg123@example.com",
		From:      domain.EmailAddress{Name: "Sender", Address: "sender@example.com"},
		To: []domain.EmailAddress{
			{Address: "to@example.com"},
		},
		Cc: []domain.EmailAddress{
			{Address: "cc@example.com"},
		},
		SentAt:    &domain.Timestamp{Time: sentTime},
		InReplyTo: "original@example.com",
	}

	imapMsg := NewIMAPMessage(msg, 1, 1)
	env := imapMsg.Envelope()

	if env.Subject != "Test Subject" {
		t.Errorf("Subject = %q, want %q", env.Subject, "Test Subject")
	}

	if env.MessageID != "msg123@example.com" {
		t.Errorf("MessageID = %q, want %q", env.MessageID, "msg123@example.com")
	}

	if !env.Date.Equal(sentTime) {
		t.Errorf("Date = %v, want %v", env.Date, sentTime)
	}

	if len(env.From) != 1 || env.From[0].Name != "Sender" {
		t.Error("From not set correctly")
	}

	if len(env.To) != 1 {
		t.Error("To not set correctly")
	}

	if len(env.Cc) != 1 {
		t.Error("Cc not set correctly")
	}

	if len(env.InReplyTo) != 1 || env.InReplyTo[0] != "original@example.com" {
		t.Error("InReplyTo not set correctly")
	}
}

func TestIMAPMessage_InternalDate(t *testing.T) {
	receivedAt := time.Date(2024, 1, 20, 15, 30, 0, 0, time.UTC)
	msg := &domain.Message{
		ReceivedAt: domain.Timestamp{Time: receivedAt},
	}

	imapMsg := NewIMAPMessage(msg, 1, 1)

	if !imapMsg.InternalDate().Equal(receivedAt) {
		t.Errorf("InternalDate() = %v, want %v", imapMsg.InternalDate(), receivedAt)
	}
}

func TestIMAPMessage_Size(t *testing.T) {
	msg := &domain.Message{
		Size: 12345,
	}

	imapMsg := NewIMAPMessage(msg, 1, 1)

	if imapMsg.Size() != 12345 {
		t.Errorf("Size() = %d, want 12345", imapMsg.Size())
	}

	// Test with raw body set
	imapMsg.SetRawBody([]byte("Short body"))
	if imapMsg.Size() != 10 {
		t.Errorf("Size() with raw body = %d, want 10", imapMsg.Size())
	}
}

func TestMessageSequence(t *testing.T) {
	seq := NewMessageSequence(12345)

	if seq.UIDValidity() != 12345 {
		t.Errorf("UIDValidity() = %d, want 12345", seq.UIDValidity())
	}

	if seq.Count() != 0 {
		t.Errorf("Initial Count() = %d, want 0", seq.Count())
	}

	// Add some messages
	msg1 := &domain.Message{ID: domain.ID("msg-1")}
	msg2 := &domain.Message{ID: domain.ID("msg-2")}

	imapMsg1 := seq.Add(msg1)
	imapMsg2 := seq.Add(msg2)

	if seq.Count() != 2 {
		t.Errorf("Count() after adds = %d, want 2", seq.Count())
	}

	if imapMsg1.SeqNum() != 1 {
		t.Errorf("First message SeqNum = %d, want 1", imapMsg1.SeqNum())
	}

	if imapMsg2.SeqNum() != 2 {
		t.Errorf("Second message SeqNum = %d, want 2", imapMsg2.SeqNum())
	}

	// Test lookup
	if seq.GetBySeqNum(1) != imapMsg1 {
		t.Error("GetBySeqNum(1) returned wrong message")
	}

	if seq.GetByUID(2) != imapMsg2 {
		t.Error("GetByUID(2) returned wrong message")
	}

	if seq.GetBySeqNum(999) != nil {
		t.Error("GetBySeqNum(999) should return nil")
	}
}

func TestMessageSequence_GetByNumSet(t *testing.T) {
	seq := NewMessageSequence(1)

	for i := 0; i < 5; i++ {
		seq.Add(&domain.Message{ID: domain.ID("msg-" + string(rune('1'+i)))})
	}

	// Test with SeqSet
	seqSet := imap.SeqSetNum(1, 2, 3)
	msgs := seq.GetByNumSet(seqSet)

	if len(msgs) != 3 {
		t.Errorf("SeqSet 1,2,3 returned %d messages, want 3", len(msgs))
	}

	// Test with UIDSet
	uidSet := imap.UIDSetNum(2, 4)
	msgs = seq.GetByNumSet(uidSet)

	if len(msgs) != 2 {
		t.Errorf("UIDSet 2,4 returned %d messages, want 2", len(msgs))
	}
}

func TestFlagStore(t *testing.T) {
	store := NewFlagStore()
	msgID := domain.ID("msg-1")

	// Initially no flags
	if store.HasFlag(msgID, imap.FlagSeen) {
		t.Error("HasFlag should return false initially")
	}

	// Set flags
	store.SetFlags(msgID, []imap.Flag{imap.FlagSeen, imap.FlagFlagged})

	if !store.HasFlag(msgID, imap.FlagSeen) {
		t.Error("HasFlag(\\Seen) should return true after SetFlags")
	}

	if !store.HasFlag(msgID, imap.FlagFlagged) {
		t.Error("HasFlag(\\Flagged) should return true after SetFlags")
	}

	// Add flags
	store.AddFlags(msgID, []imap.Flag{imap.FlagAnswered})

	if !store.HasFlag(msgID, imap.FlagAnswered) {
		t.Error("HasFlag(\\Answered) should return true after AddFlags")
	}

	// Remove flags
	store.RemoveFlags(msgID, []imap.Flag{imap.FlagSeen})

	if store.HasFlag(msgID, imap.FlagSeen) {
		t.Error("HasFlag(\\Seen) should return false after RemoveFlags")
	}

	// GetFlags
	flags := store.GetFlags(msgID)
	if len(flags) != 2 {
		t.Errorf("GetFlags returned %d flags, want 2", len(flags))
	}
}

func TestEmailToIMAPAddress(t *testing.T) {
	tests := []struct {
		addr        domain.EmailAddress
		wantMailbox string
		wantHost    string
		wantName    string
	}{
		{
			addr:        domain.EmailAddress{Name: "John Doe", Address: "john@example.com"},
			wantMailbox: "john",
			wantHost:    "example.com",
			wantName:    "John Doe",
		},
		{
			addr:        domain.EmailAddress{Address: "user@sub.domain.org"},
			wantMailbox: "user",
			wantHost:    "sub.domain.org",
			wantName:    "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.addr.Address, func(t *testing.T) {
			result := emailToIMAPAddress(tt.addr)

			if result.Mailbox != tt.wantMailbox {
				t.Errorf("Mailbox = %q, want %q", result.Mailbox, tt.wantMailbox)
			}
			if result.Host != tt.wantHost {
				t.Errorf("Host = %q, want %q", result.Host, tt.wantHost)
			}
			if result.Name != tt.wantName {
				t.Errorf("Name = %q, want %q", result.Name, tt.wantName)
			}
		})
	}
}

func TestReconstructRFC822Message(t *testing.T) {
	now := domain.Now()
	msg := &domain.Message{
		ID:        domain.ID("test-1"),
		From:      domain.EmailAddress{Name: "Sender", Address: "sender@example.com"},
		To:        []domain.EmailAddress{{Address: "recipient@example.com"}},
		Subject:   "Test Subject",
		MessageID: "unique@example.com",
		TextBody:  "This is the body.",
		ReceivedAt: now,
	}

	result := reconstructRFC822Message(msg)
	resultStr := string(result)

	// Check headers
	if !strings.Contains(resultStr, "From: Sender <sender@example.com>") {
		t.Error("Missing From header")
	}
	if !strings.Contains(resultStr, "To: recipient@example.com") {
		t.Error("Missing To header")
	}
	if !strings.Contains(resultStr, "Subject: Test Subject") {
		t.Error("Missing Subject header")
	}
	if !strings.Contains(resultStr, "Message-ID: <unique@example.com>") {
		t.Error("Missing Message-ID header")
	}
	if !strings.Contains(resultStr, "This is the body.") {
		t.Error("Missing body")
	}
}

func TestMessageIndex(t *testing.T) {
	idx := NewMessageIndex()

	msg1 := NewIMAPMessage(&domain.Message{ID: domain.ID("msg-1")}, 1, 100)
	msg2 := NewIMAPMessage(&domain.Message{ID: domain.ID("msg-2")}, 2, 101)

	idx.Add(msg1)
	idx.Add(msg2)

	if idx.Count() != 2 {
		t.Errorf("Count() = %d, want 2", idx.Count())
	}

	if idx.GetBySeqNum(1) != msg1 {
		t.Error("GetBySeqNum(1) returned wrong message")
	}

	if idx.GetByUID(101) != msg2 {
		t.Error("GetByUID(101) returned wrong message")
	}

	if idx.GetByID(domain.ID("msg-1")) != msg1 {
		t.Error("GetByID(msg-1) returned wrong message")
	}

	all := idx.All()
	if len(all) != 2 {
		t.Errorf("All() returned %d messages, want 2", len(all))
	}

	// Test remove
	idx.Remove(msg1)

	if idx.Count() != 1 {
		t.Errorf("Count() after remove = %d, want 1", idx.Count())
	}

	if idx.GetBySeqNum(1) != msg2 {
		t.Error("After remove, GetBySeqNum(1) should return msg2")
	}
}
