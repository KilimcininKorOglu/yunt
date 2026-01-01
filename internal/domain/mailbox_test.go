package domain

import (
	"strings"
	"testing"
)

func TestNewMailbox(t *testing.T) {
	mailbox := NewMailbox(ID("mb1"), ID("u1"), "Inbox", "inbox@localhost")

	if mailbox.ID != ID("mb1") {
		t.Errorf("NewMailbox().ID = %v, want %v", mailbox.ID, "mb1")
	}
	if mailbox.UserID != ID("u1") {
		t.Errorf("NewMailbox().UserID = %v, want %v", mailbox.UserID, "u1")
	}
	if mailbox.Name != "Inbox" {
		t.Errorf("NewMailbox().Name = %v, want %v", mailbox.Name, "Inbox")
	}
	if mailbox.Address != "inbox@localhost" {
		t.Errorf("NewMailbox().Address = %v, want %v", mailbox.Address, "inbox@localhost")
	}
	if mailbox.IsCatchAll {
		t.Error("NewMailbox().IsCatchAll should be false")
	}
	if mailbox.IsDefault {
		t.Error("NewMailbox().IsDefault should be false")
	}
	if mailbox.MessageCount != 0 {
		t.Errorf("NewMailbox().MessageCount = %v, want 0", mailbox.MessageCount)
	}
}

func TestMailbox_Validate(t *testing.T) {
	tests := []struct {
		name    string
		mailbox *Mailbox
		wantErr bool
		errMsgs []string
	}{
		{
			name: "valid mailbox",
			mailbox: &Mailbox{
				ID:      ID("mb1"),
				UserID:  ID("u1"),
				Name:    "Inbox",
				Address: "inbox@localhost",
			},
			wantErr: false,
		},
		{
			name: "missing id",
			mailbox: &Mailbox{
				UserID:  ID("u1"),
				Name:    "Inbox",
				Address: "inbox@localhost",
			},
			wantErr: true,
			errMsgs: []string{"id"},
		},
		{
			name: "missing user id",
			mailbox: &Mailbox{
				ID:      ID("mb1"),
				Name:    "Inbox",
				Address: "inbox@localhost",
			},
			wantErr: true,
			errMsgs: []string{"userId"},
		},
		{
			name: "missing name",
			mailbox: &Mailbox{
				ID:      ID("mb1"),
				UserID:  ID("u1"),
				Address: "inbox@localhost",
			},
			wantErr: true,
			errMsgs: []string{"name"},
		},
		{
			name: "name too long",
			mailbox: &Mailbox{
				ID:      ID("mb1"),
				UserID:  ID("u1"),
				Name:    strings.Repeat("a", 101),
				Address: "inbox@localhost",
			},
			wantErr: true,
			errMsgs: []string{"name"},
		},
		{
			name: "missing address",
			mailbox: &Mailbox{
				ID:     ID("mb1"),
				UserID: ID("u1"),
				Name:   "Inbox",
			},
			wantErr: true,
			errMsgs: []string{"address"},
		},
		{
			name: "invalid address format",
			mailbox: &Mailbox{
				ID:      ID("mb1"),
				UserID:  ID("u1"),
				Name:    "Inbox",
				Address: "not-an-email",
			},
			wantErr: true,
			errMsgs: []string{"address"},
		},
		{
			name: "negative retention days",
			mailbox: &Mailbox{
				ID:            ID("mb1"),
				UserID:        ID("u1"),
				Name:          "Inbox",
				Address:       "inbox@localhost",
				RetentionDays: -1,
			},
			wantErr: true,
			errMsgs: []string{"retentionDays"},
		},
		{
			name: "negative message count",
			mailbox: &Mailbox{
				ID:           ID("mb1"),
				UserID:       ID("u1"),
				Name:         "Inbox",
				Address:      "inbox@localhost",
				MessageCount: -1,
			},
			wantErr: true,
			errMsgs: []string{"messageCount"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.mailbox.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Mailbox.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr && err != nil {
				errStr := err.Error()
				for _, msg := range tt.errMsgs {
					if !strings.Contains(errStr, msg) {
						t.Errorf("Mailbox.Validate() error should contain '%s', got %v", msg, errStr)
					}
				}
			}
		})
	}
}

func TestMailbox_IncrementMessageCount(t *testing.T) {
	mailbox := NewMailbox(ID("mb1"), ID("u1"), "Inbox", "inbox@localhost")

	mailbox.IncrementMessageCount(1024)

	if mailbox.MessageCount != 1 {
		t.Errorf("IncrementMessageCount() MessageCount = %v, want 1", mailbox.MessageCount)
	}
	if mailbox.UnreadCount != 1 {
		t.Errorf("IncrementMessageCount() UnreadCount = %v, want 1", mailbox.UnreadCount)
	}
	if mailbox.TotalSize != 1024 {
		t.Errorf("IncrementMessageCount() TotalSize = %v, want 1024", mailbox.TotalSize)
	}
}

func TestMailbox_DecrementMessageCount(t *testing.T) {
	mailbox := NewMailbox(ID("mb1"), ID("u1"), "Inbox", "inbox@localhost")
	mailbox.MessageCount = 5
	mailbox.UnreadCount = 3
	mailbox.TotalSize = 5000

	mailbox.DecrementMessageCount(1000, true)

	if mailbox.MessageCount != 4 {
		t.Errorf("DecrementMessageCount() MessageCount = %v, want 4", mailbox.MessageCount)
	}
	if mailbox.UnreadCount != 2 {
		t.Errorf("DecrementMessageCount() UnreadCount = %v, want 2", mailbox.UnreadCount)
	}
	if mailbox.TotalSize != 4000 {
		t.Errorf("DecrementMessageCount() TotalSize = %v, want 4000", mailbox.TotalSize)
	}

	// Test with wasUnread = false
	mailbox.DecrementMessageCount(1000, false)

	if mailbox.MessageCount != 3 {
		t.Errorf("DecrementMessageCount() MessageCount = %v, want 3", mailbox.MessageCount)
	}
	if mailbox.UnreadCount != 2 {
		t.Errorf("DecrementMessageCount() UnreadCount = %v, want 2 (unchanged)", mailbox.UnreadCount)
	}
}

func TestMailbox_MarkMessageReadUnread(t *testing.T) {
	mailbox := NewMailbox(ID("mb1"), ID("u1"), "Inbox", "inbox@localhost")
	mailbox.MessageCount = 5
	mailbox.UnreadCount = 3

	mailbox.MarkMessageRead()
	if mailbox.UnreadCount != 2 {
		t.Errorf("MarkMessageRead() UnreadCount = %v, want 2", mailbox.UnreadCount)
	}

	mailbox.MarkMessageUnread()
	if mailbox.UnreadCount != 3 {
		t.Errorf("MarkMessageUnread() UnreadCount = %v, want 3", mailbox.UnreadCount)
	}

	// Test boundaries
	mailbox.UnreadCount = 0
	mailbox.MarkMessageRead()
	if mailbox.UnreadCount != 0 {
		t.Error("MarkMessageRead() should not go below 0")
	}

	mailbox.UnreadCount = mailbox.MessageCount
	mailbox.MarkMessageUnread()
	if mailbox.UnreadCount != mailbox.MessageCount {
		t.Error("MarkMessageUnread() should not exceed MessageCount")
	}
}

func TestMailbox_SetAsDefault(t *testing.T) {
	mailbox := NewMailbox(ID("mb1"), ID("u1"), "Inbox", "inbox@localhost")

	mailbox.SetAsDefault()
	if !mailbox.IsDefault {
		t.Error("SetAsDefault() should set IsDefault to true")
	}

	mailbox.UnsetAsDefault()
	if mailbox.IsDefault {
		t.Error("UnsetAsDefault() should set IsDefault to false")
	}
}

func TestMailbox_SetCatchAll(t *testing.T) {
	mailbox := NewMailbox(ID("mb1"), ID("u1"), "Inbox", "inbox@localhost")

	mailbox.SetCatchAll()
	if !mailbox.IsCatchAll {
		t.Error("SetCatchAll() should set IsCatchAll to true")
	}

	mailbox.UnsetCatchAll()
	if mailbox.IsCatchAll {
		t.Error("UnsetCatchAll() should set IsCatchAll to false")
	}
}

func TestMailbox_HasMessages(t *testing.T) {
	mailbox := NewMailbox(ID("mb1"), ID("u1"), "Inbox", "inbox@localhost")

	if mailbox.HasMessages() {
		t.Error("HasMessages() should return false for new mailbox")
	}

	mailbox.MessageCount = 1
	if !mailbox.HasMessages() {
		t.Error("HasMessages() should return true when messages exist")
	}
}

func TestMailbox_HasUnread(t *testing.T) {
	mailbox := NewMailbox(ID("mb1"), ID("u1"), "Inbox", "inbox@localhost")

	if mailbox.HasUnread() {
		t.Error("HasUnread() should return false for new mailbox")
	}

	mailbox.UnreadCount = 1
	if !mailbox.HasUnread() {
		t.Error("HasUnread() should return true when unread messages exist")
	}
}

func TestMailbox_GetLocalPart(t *testing.T) {
	tests := []struct {
		address string
		want    string
	}{
		{"inbox@localhost", "inbox"},
		{"test@example.com", "test"},
		{"invalid", "invalid"},
	}

	for _, tt := range tests {
		t.Run(tt.address, func(t *testing.T) {
			mailbox := &Mailbox{Address: tt.address}
			if got := mailbox.GetLocalPart(); got != tt.want {
				t.Errorf("GetLocalPart() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestMailbox_GetDomain(t *testing.T) {
	tests := []struct {
		address string
		want    string
	}{
		{"inbox@localhost", "localhost"},
		{"test@example.com", "example.com"},
		{"invalid", ""},
	}

	for _, tt := range tests {
		t.Run(tt.address, func(t *testing.T) {
			mailbox := &Mailbox{Address: tt.address}
			if got := mailbox.GetDomain(); got != tt.want {
				t.Errorf("GetDomain() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestMailboxCreateInput_Validate(t *testing.T) {
	tests := []struct {
		name    string
		input   *MailboxCreateInput
		wantErr bool
	}{
		{
			name: "valid input",
			input: &MailboxCreateInput{
				Name:    "Inbox",
				Address: "inbox@localhost",
			},
			wantErr: false,
		},
		{
			name: "missing name",
			input: &MailboxCreateInput{
				Address: "inbox@localhost",
			},
			wantErr: true,
		},
		{
			name: "missing address",
			input: &MailboxCreateInput{
				Name: "Inbox",
			},
			wantErr: true,
		},
		{
			name: "negative retention days",
			input: &MailboxCreateInput{
				Name:          "Inbox",
				Address:       "inbox@localhost",
				RetentionDays: -1,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.input.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("MailboxCreateInput.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestMailboxCreateInput_Normalize(t *testing.T) {
	input := &MailboxCreateInput{
		Name:        "  Inbox  ",
		Address:     "  INBOX@LOCALHOST  ",
		Description: "  Test mailbox  ",
	}

	input.Normalize()

	if input.Name != "Inbox" {
		t.Errorf("Normalize() Name = %v, want %v", input.Name, "Inbox")
	}
	if input.Address != "inbox@localhost" {
		t.Errorf("Normalize() Address = %v, want %v", input.Address, "inbox@localhost")
	}
	if input.Description != "Test mailbox" {
		t.Errorf("Normalize() Description = %v, want %v", input.Description, "Test mailbox")
	}
}

func TestMailboxUpdateInput_Validate(t *testing.T) {
	emptyName := ""
	longName := strings.Repeat("a", 101)
	negativeRetention := -1

	tests := []struct {
		name    string
		input   *MailboxUpdateInput
		wantErr bool
	}{
		{
			name:    "empty update (valid)",
			input:   &MailboxUpdateInput{},
			wantErr: false,
		},
		{
			name:    "empty name",
			input:   &MailboxUpdateInput{Name: &emptyName},
			wantErr: true,
		},
		{
			name:    "name too long",
			input:   &MailboxUpdateInput{Name: &longName},
			wantErr: true,
		},
		{
			name:    "negative retention",
			input:   &MailboxUpdateInput{RetentionDays: &negativeRetention},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.input.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("MailboxUpdateInput.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestMailboxUpdateInput_Apply(t *testing.T) {
	mailbox := NewMailbox(ID("mb1"), ID("u1"), "Inbox", "inbox@localhost")

	newName := "Updated Inbox"
	newDesc := "Updated description"
	isDefault := true
	retention := 30

	input := &MailboxUpdateInput{
		Name:          &newName,
		Description:   &newDesc,
		IsDefault:     &isDefault,
		RetentionDays: &retention,
	}

	input.Apply(mailbox)

	if mailbox.Name != newName {
		t.Errorf("Apply() Name = %v, want %v", mailbox.Name, newName)
	}
	if mailbox.Description != newDesc {
		t.Errorf("Apply() Description = %v, want %v", mailbox.Description, newDesc)
	}
	if mailbox.IsDefault != isDefault {
		t.Errorf("Apply() IsDefault = %v, want %v", mailbox.IsDefault, isDefault)
	}
	if mailbox.RetentionDays != retention {
		t.Errorf("Apply() RetentionDays = %v, want %v", mailbox.RetentionDays, retention)
	}
}
