package imap

import (
	"testing"

	"yunt/internal/domain"
)

func TestSortUint32(t *testing.T) {
	tests := []struct {
		name     string
		input    []uint32
		expected []uint32
	}{
		{
			name:     "empty slice",
			input:    []uint32{},
			expected: []uint32{},
		},
		{
			name:     "single element",
			input:    []uint32{5},
			expected: []uint32{5},
		},
		{
			name:     "already sorted",
			input:    []uint32{1, 2, 3, 4, 5},
			expected: []uint32{1, 2, 3, 4, 5},
		},
		{
			name:     "reverse sorted",
			input:    []uint32{5, 4, 3, 2, 1},
			expected: []uint32{1, 2, 3, 4, 5},
		},
		{
			name:     "random order",
			input:    []uint32{3, 1, 4, 1, 5, 9, 2, 6},
			expected: []uint32{1, 1, 2, 3, 4, 5, 6, 9},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sortUint32(tt.input)
			for i := range tt.input {
				if tt.input[i] != tt.expected[i] {
					t.Errorf("At index %d: expected %d, got %d", i, tt.expected[i], tt.input[i])
				}
			}
		})
	}
}

func TestSortUint32Desc(t *testing.T) {
	tests := []struct {
		name     string
		input    []uint32
		expected []uint32
	}{
		{
			name:     "empty slice",
			input:    []uint32{},
			expected: []uint32{},
		},
		{
			name:     "single element",
			input:    []uint32{5},
			expected: []uint32{5},
		},
		{
			name:     "already sorted descending",
			input:    []uint32{5, 4, 3, 2, 1},
			expected: []uint32{5, 4, 3, 2, 1},
		},
		{
			name:     "ascending sorted",
			input:    []uint32{1, 2, 3, 4, 5},
			expected: []uint32{5, 4, 3, 2, 1},
		},
		{
			name:     "random order",
			input:    []uint32{3, 1, 4, 1, 5, 9, 2, 6},
			expected: []uint32{9, 6, 5, 4, 3, 2, 1, 1},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sortUint32Desc(tt.input)
			for i := range tt.input {
				if tt.input[i] != tt.expected[i] {
					t.Errorf("At index %d: expected %d, got %d", i, tt.expected[i], tt.input[i])
				}
			}
		})
	}
}

func TestCopyHandlerCopyMessageData(t *testing.T) {
	handler := &CopyHandler{}

	now := domain.Now()
	srcMsg := &domain.Message{
		ID:          domain.ID("src-msg-1"),
		MailboxID:   domain.ID("src-mailbox"),
		MessageID:   "<test@example.com>",
		From:        domain.EmailAddress{Name: "Test Sender", Address: "sender@example.com"},
		To:          []domain.EmailAddress{{Name: "Test Recipient", Address: "recipient@example.com"}},
		Cc:          []domain.EmailAddress{{Address: "cc@example.com"}},
		Subject:     "Test Subject",
		TextBody:    "Test body content",
		HTMLBody:    "<p>Test body content</p>",
		Headers:     map[string]string{"X-Custom": "value"},
		ContentType: domain.ContentTypePlain,
		Size:        1024,
		Status:      domain.MessageRead,
		IsStarred:   true,
		InReplyTo:   "<original@example.com>",
		References:  []string{"<ref1@example.com>", "<ref2@example.com>"},
		ReceivedAt:  now,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	newID := domain.ID("new-msg-1")
	destMailboxID := domain.ID("dest-mailbox")

	// Copy the message
	newMsg := handler.copyMessageData(srcMsg, newID, destMailboxID)

	// Verify new IDs
	if newMsg.ID != newID {
		t.Errorf("Expected ID %s, got %s", newID, newMsg.ID)
	}
	if newMsg.MailboxID != destMailboxID {
		t.Errorf("Expected MailboxID %s, got %s", destMailboxID, newMsg.MailboxID)
	}

	// Verify copied fields
	if newMsg.MessageID != srcMsg.MessageID {
		t.Errorf("Expected MessageID %s, got %s", srcMsg.MessageID, newMsg.MessageID)
	}
	if newMsg.From != srcMsg.From {
		t.Errorf("Expected From %v, got %v", srcMsg.From, newMsg.From)
	}
	if len(newMsg.To) != len(srcMsg.To) {
		t.Errorf("Expected %d To addresses, got %d", len(srcMsg.To), len(newMsg.To))
	}
	if newMsg.Subject != srcMsg.Subject {
		t.Errorf("Expected Subject %s, got %s", srcMsg.Subject, newMsg.Subject)
	}
	if newMsg.TextBody != srcMsg.TextBody {
		t.Errorf("Expected TextBody %s, got %s", srcMsg.TextBody, newMsg.TextBody)
	}
	if newMsg.HTMLBody != srcMsg.HTMLBody {
		t.Errorf("Expected HTMLBody %s, got %s", srcMsg.HTMLBody, newMsg.HTMLBody)
	}
	if newMsg.Size != srcMsg.Size {
		t.Errorf("Expected Size %d, got %d", srcMsg.Size, newMsg.Size)
	}
	if newMsg.Status != srcMsg.Status {
		t.Errorf("Expected Status %v, got %v", srcMsg.Status, newMsg.Status)
	}
	if newMsg.IsStarred != srcMsg.IsStarred {
		t.Errorf("Expected IsStarred %v, got %v", srcMsg.IsStarred, newMsg.IsStarred)
	}

	// Verify headers are copied (deep copy)
	if newMsg.Headers["X-Custom"] != "value" {
		t.Errorf("Expected header X-Custom to be 'value', got %s", newMsg.Headers["X-Custom"])
	}

	// Modify source to ensure deep copy
	srcMsg.Headers["X-Custom"] = "modified"
	if newMsg.Headers["X-Custom"] == "modified" {
		t.Error("Headers should be deep copied, not shared")
	}

	// Verify References are copied (deep copy)
	if len(newMsg.References) != 2 {
		t.Errorf("Expected 2 references, got %d", len(newMsg.References))
	}
	srcMsg.References[0] = "modified"
	if newMsg.References[0] == "modified" {
		t.Error("References should be deep copied, not shared")
	}
}

func TestCopyResult(t *testing.T) {
	result := &CopyResult{
		UIDValidity: 12345,
	}

	// Initially empty
	if result.CopiedCount != 0 {
		t.Errorf("Expected CopiedCount 0, got %d", result.CopiedCount)
	}

	// Add source and dest UIDs
	result.SourceUIDs.AddNum(1, 2, 3)
	result.DestUIDs.AddNum(100, 101, 102)
	result.CopiedCount = 3

	// Verify UID sets
	if !result.SourceUIDs.Contains(1) {
		t.Error("SourceUIDs should contain UID 1")
	}
	if !result.DestUIDs.Contains(100) {
		t.Error("DestUIDs should contain UID 100")
	}
	if result.CopiedCount != 3 {
		t.Errorf("Expected CopiedCount 3, got %d", result.CopiedCount)
	}
}
