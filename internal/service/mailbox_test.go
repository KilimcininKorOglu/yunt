package service

import (
	"context"
	"testing"

	"yunt/internal/domain"
)



// mailboxServiceMockIDGen generates predictable IDs for testing
type mailboxServiceMockIDGen struct {
	counter int
}

func newMailboxServiceMockIDGen() *mailboxServiceMockIDGen {
	return &mailboxServiceMockIDGen{counter: 0}
}

func (g *mailboxServiceMockIDGen) Generate() domain.ID {
	g.counter++
	return domain.ID("generated-id-" + string(rune('0'+g.counter)))
}

func newTestMailboxServiceWithMocks() (*MailboxService, *mockRepository, *mailboxServiceMockIDGen) {
	repo := newMockRepository()
	idGen := newMailboxServiceMockIDGen()
	svc := NewMailboxService(repo, idGen)
	return svc, repo, idGen
}

func TestMailboxService_ListMailboxes(t *testing.T) {
	svc, repo, _ := newTestMailboxServiceWithMocks()
	ctx := context.Background()
	userID := domain.ID("user-1")

	// Add some test mailboxes
	mailbox1 := domain.NewMailbox("mb-1", userID, "Inbox", "test@example.com")
	mailbox2 := domain.NewMailbox("mb-2", userID, "Work", "work@example.com")
	mailbox3 := domain.NewMailbox("mb-3", domain.ID("user-2"), "Other", "other@example.com")
	repo.mailboxes.AddMailbox(mailbox1)
	repo.mailboxes.AddMailbox(mailbox2)
	repo.mailboxes.AddMailbox(mailbox3)

	result, err := svc.ListMailboxes(ctx, userID, nil)
	if err != nil {
		t.Fatalf("ListMailboxes() error = %v", err)
	}

	if len(result.Items) != 2 {
		t.Errorf("ListMailboxes() returned %d items, want 2", len(result.Items))
	}
}

func TestMailboxService_ListMailboxes_EmptyUserID(t *testing.T) {
	svc, _, _ := newTestMailboxServiceWithMocks()
	ctx := context.Background()

	_, err := svc.ListMailboxes(ctx, domain.ID(""), nil)
	if err == nil {
		t.Error("ListMailboxes() should fail with empty user ID")
	}
}

func TestMailboxService_GetMailbox(t *testing.T) {
	svc, repo, _ := newTestMailboxServiceWithMocks()
	ctx := context.Background()
	userID := domain.ID("user-1")
	mailboxID := domain.ID("mb-1")

	mailbox := domain.NewMailbox(mailboxID, userID, "Inbox", "test@example.com")
	repo.mailboxes.AddMailbox(mailbox)

	result, err := svc.GetMailbox(ctx, mailboxID, userID)
	if err != nil {
		t.Fatalf("GetMailbox() error = %v", err)
	}

	if result.ID != mailboxID {
		t.Errorf("GetMailbox() returned mailbox with ID = %v, want %v", result.ID, mailboxID)
	}
}

func TestMailboxService_GetMailbox_Forbidden(t *testing.T) {
	svc, repo, _ := newTestMailboxServiceWithMocks()
	ctx := context.Background()
	ownerID := domain.ID("user-1")
	otherUserID := domain.ID("user-2")
	mailboxID := domain.ID("mb-1")

	mailbox := domain.NewMailbox(mailboxID, ownerID, "Inbox", "test@example.com")
	repo.mailboxes.AddMailbox(mailbox)

	_, err := svc.GetMailbox(ctx, mailboxID, otherUserID)
	if err == nil {
		t.Error("GetMailbox() should fail when accessing another user's mailbox")
	}
	if !domain.IsForbidden(err) {
		t.Errorf("GetMailbox() error should be forbidden, got %v", err)
	}
}

func TestMailboxService_CreateMailbox(t *testing.T) {
	svc, _, _ := newTestMailboxServiceWithMocks()
	ctx := context.Background()
	userID := domain.ID("user-1")

	input := &CreateMailboxInput{
		UserID:      userID,
		Name:        "Test Mailbox",
		Address:     "test@example.com",
		Description: "A test mailbox",
	}

	mailbox, err := svc.CreateMailbox(ctx, input)
	if err != nil {
		t.Fatalf("CreateMailbox() error = %v", err)
	}

	if mailbox == nil {
		t.Fatal("CreateMailbox() returned nil mailbox")
	}
	if mailbox.Name != input.Name {
		t.Errorf("CreateMailbox() mailbox.Name = %v, want %v", mailbox.Name, input.Name)
	}
	if mailbox.Address != input.Address {
		t.Errorf("CreateMailbox() mailbox.Address = %v, want %v", mailbox.Address, input.Address)
	}
	if mailbox.UserID != userID {
		t.Errorf("CreateMailbox() mailbox.UserID = %v, want %v", mailbox.UserID, userID)
	}
}

func TestMailboxService_CreateMailbox_DuplicateAddress(t *testing.T) {
	svc, repo, _ := newTestMailboxServiceWithMocks()
	ctx := context.Background()
	userID := domain.ID("user-1")

	// Add existing mailbox
	existing := domain.NewMailbox("mb-1", userID, "Existing", "test@example.com")
	repo.mailboxes.AddMailbox(existing)

	input := &CreateMailboxInput{
		UserID:  userID,
		Name:    "New Mailbox",
		Address: "test@example.com", // Same address
	}

	_, err := svc.CreateMailbox(ctx, input)
	if err == nil {
		t.Error("CreateMailbox() should fail with duplicate address")
	}
}

func TestMailboxService_CreateMailbox_Validation(t *testing.T) {
	svc, _, _ := newTestMailboxServiceWithMocks()
	ctx := context.Background()
	userID := domain.ID("user-1")

	tests := []struct {
		name    string
		input   *CreateMailboxInput
		wantErr bool
	}{
		{
			name:    "nil input",
			input:   nil,
			wantErr: true,
		},
		{
			name: "empty user ID",
			input: &CreateMailboxInput{
				Name:    "Test",
				Address: "test@example.com",
			},
			wantErr: true,
		},
		{
			name: "empty name",
			input: &CreateMailboxInput{
				UserID:  userID,
				Address: "test@example.com",
			},
			wantErr: true,
		},
		{
			name: "empty address",
			input: &CreateMailboxInput{
				UserID: userID,
				Name:   "Test",
			},
			wantErr: true,
		},
		{
			name: "invalid email",
			input: &CreateMailboxInput{
				UserID:  userID,
				Name:    "Test",
				Address: "not-an-email",
			},
			wantErr: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, err := svc.CreateMailbox(ctx, tc.input)
			if (err != nil) != tc.wantErr {
				t.Errorf("CreateMailbox() error = %v, wantErr %v", err, tc.wantErr)
			}
		})
	}
}

func TestMailboxService_UpdateMailbox(t *testing.T) {
	svc, repo, _ := newTestMailboxServiceWithMocks()
	ctx := context.Background()
	userID := domain.ID("user-1")
	mailboxID := domain.ID("mb-1")

	mailbox := domain.NewMailbox(mailboxID, userID, "Original", "test@example.com")
	repo.mailboxes.AddMailbox(mailbox)

	newName := "Updated"
	input := &UpdateMailboxInput{
		MailboxID: mailboxID,
		UserID:    userID,
		Name:      &newName,
	}

	updated, err := svc.UpdateMailbox(ctx, input)
	if err != nil {
		t.Fatalf("UpdateMailbox() error = %v", err)
	}

	if updated.Name != newName {
		t.Errorf("UpdateMailbox() mailbox.Name = %v, want %v", updated.Name, newName)
	}
}

func TestMailboxService_UpdateMailbox_Forbidden(t *testing.T) {
	svc, repo, _ := newTestMailboxServiceWithMocks()
	ctx := context.Background()
	ownerID := domain.ID("user-1")
	otherUserID := domain.ID("user-2")
	mailboxID := domain.ID("mb-1")

	mailbox := domain.NewMailbox(mailboxID, ownerID, "Original", "test@example.com")
	repo.mailboxes.AddMailbox(mailbox)

	newName := "Hacked"
	input := &UpdateMailboxInput{
		MailboxID: mailboxID,
		UserID:    otherUserID, // Wrong user
		Name:      &newName,
	}

	_, err := svc.UpdateMailbox(ctx, input)
	if err == nil {
		t.Error("UpdateMailbox() should fail when updating another user's mailbox")
	}
	if !domain.IsForbidden(err) {
		t.Errorf("UpdateMailbox() error should be forbidden, got %v", err)
	}
}

func TestMailboxService_DeleteMailbox(t *testing.T) {
	svc, repo, _ := newTestMailboxServiceWithMocks()
	ctx := context.Background()
	userID := domain.ID("user-1")
	mailboxID := domain.ID("mb-1")

	// Use a custom name that's not a system mailbox
	mailbox := domain.NewMailbox(mailboxID, userID, "Custom Folder", "custom@example.com")
	repo.mailboxes.AddMailbox(mailbox)

	err := svc.DeleteMailbox(ctx, mailboxID, userID)
	if err != nil {
		t.Fatalf("DeleteMailbox() error = %v", err)
	}

	// Verify mailbox is deleted
	exists, _ := repo.mailboxes.Exists(ctx, mailboxID)
	if exists {
		t.Error("DeleteMailbox() should delete the mailbox")
	}
}

func TestMailboxService_DeleteMailbox_SystemMailbox(t *testing.T) {
	svc, repo, _ := newTestMailboxServiceWithMocks()
	ctx := context.Background()
	userID := domain.ID("user-1")

	// Test deleting system mailboxes
	systemNames := []string{"Inbox", "Sent", "Trash", "Drafts", "Spam", "Junk"}
	for _, name := range systemNames {
		mailboxID := domain.ID("mb-" + name)
		mailbox := domain.NewMailbox(mailboxID, userID, name, name+"@example.com")
		repo.mailboxes.AddMailbox(mailbox)

		err := svc.DeleteMailbox(ctx, mailboxID, userID)
		if err == nil {
			t.Errorf("DeleteMailbox() should fail for system mailbox %s", name)
		}
	}
}

func TestMailboxService_DeleteMailbox_Forbidden(t *testing.T) {
	svc, repo, _ := newTestMailboxServiceWithMocks()
	ctx := context.Background()
	ownerID := domain.ID("user-1")
	otherUserID := domain.ID("user-2")
	mailboxID := domain.ID("mb-1")

	mailbox := domain.NewMailbox(mailboxID, ownerID, "Custom", "custom@example.com")
	repo.mailboxes.AddMailbox(mailbox)

	err := svc.DeleteMailbox(ctx, mailboxID, otherUserID)
	if err == nil {
		t.Error("DeleteMailbox() should fail when deleting another user's mailbox")
	}
	if !domain.IsForbidden(err) {
		t.Errorf("DeleteMailbox() error should be forbidden, got %v", err)
	}
}

func TestMailboxService_GetMailboxStats(t *testing.T) {
	svc, repo, _ := newTestMailboxServiceWithMocks()
	ctx := context.Background()
	userID := domain.ID("user-1")
	mailboxID := domain.ID("mb-1")

	mailbox := domain.NewMailbox(mailboxID, userID, "Inbox", "test@example.com")
	repo.mailboxes.AddMailbox(mailbox)

	stats, err := svc.GetMailboxStats(ctx, mailboxID, userID)
	if err != nil {
		t.Fatalf("GetMailboxStats() error = %v", err)
	}

	if stats == nil {
		t.Error("GetMailboxStats() returned nil stats")
	}
}

func TestMailboxService_GetMailboxStats_Forbidden(t *testing.T) {
	svc, repo, _ := newTestMailboxServiceWithMocks()
	ctx := context.Background()
	ownerID := domain.ID("user-1")
	otherUserID := domain.ID("user-2")
	mailboxID := domain.ID("mb-1")

	mailbox := domain.NewMailbox(mailboxID, ownerID, "Inbox", "test@example.com")
	repo.mailboxes.AddMailbox(mailbox)

	_, err := svc.GetMailboxStats(ctx, mailboxID, otherUserID)
	if err == nil {
		t.Error("GetMailboxStats() should fail when accessing another user's mailbox")
	}
}

func TestMailboxService_SetDefaultMailbox(t *testing.T) {
	svc, repo, _ := newTestMailboxServiceWithMocks()
	ctx := context.Background()
	userID := domain.ID("user-1")
	mailboxID := domain.ID("mb-1")

	mailbox := domain.NewMailbox(mailboxID, userID, "Inbox", "test@example.com")
	repo.mailboxes.AddMailbox(mailbox)

	err := svc.SetDefaultMailbox(ctx, mailboxID, userID)
	if err != nil {
		t.Fatalf("SetDefaultMailbox() error = %v", err)
	}

	// Verify it's now default
	updated, _ := repo.mailboxes.GetByID(ctx, mailboxID)
	if !updated.IsDefault {
		t.Error("SetDefaultMailbox() should set mailbox as default")
	}
}

func TestMailboxService_SetDefaultMailbox_Forbidden(t *testing.T) {
	svc, repo, _ := newTestMailboxServiceWithMocks()
	ctx := context.Background()
	ownerID := domain.ID("user-1")
	otherUserID := domain.ID("user-2")
	mailboxID := domain.ID("mb-1")

	mailbox := domain.NewMailbox(mailboxID, ownerID, "Inbox", "test@example.com")
	repo.mailboxes.AddMailbox(mailbox)

	err := svc.SetDefaultMailbox(ctx, mailboxID, otherUserID)
	if err == nil {
		t.Error("SetDefaultMailbox() should fail for another user's mailbox")
	}
}

func TestMailboxService_isSystemMailbox(t *testing.T) {
	svc, _, _ := newTestMailboxServiceWithMocks()

	tests := []struct {
		name     string
		mailbox  string
		isSystem bool
	}{
		{"Inbox lowercase", "inbox", true},
		{"Inbox uppercase", "INBOX", true},
		{"Inbox mixed case", "Inbox", true},
		{"Sent", "Sent", true},
		{"Trash", "Trash", true},
		{"Drafts", "Drafts", true},
		{"Spam", "Spam", true},
		{"Junk", "Junk", true},
		{"Custom folder", "Work", false},
		{"Custom folder 2", "Projects", false},
		{"Random name", "myFolder123", false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := svc.isSystemMailbox(tc.mailbox)
			if result != tc.isSystem {
				t.Errorf("isSystemMailbox(%q) = %v, want %v", tc.mailbox, result, tc.isSystem)
			}
		})
	}
}
