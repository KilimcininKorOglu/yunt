package imap

import (
	"testing"

	"github.com/emersion/go-imap/v2"

	"yunt/internal/domain"
)

func TestIsSystemMailbox(t *testing.T) {
	tests := []struct {
		name     string
		mailbox  string
		expected bool
	}{
		{"INBOX uppercase", "INBOX", true},
		{"INBOX lowercase", "inbox", true},
		{"INBOX mixed case", "InBoX", true},
		{"Sent", "Sent", true},
		{"Drafts", "Drafts", true},
		{"Trash", "Trash", true},
		{"Spam", "Spam", true},
		{"Custom mailbox", "Work", false},
		{"Custom with similar name", "INBOX2", false},
		{"Nested custom", "INBOX/Archive", false},
		{"Empty", "", false},
		{"sent lowercase", "sent", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsSystemMailbox(tt.mailbox)
			if result != tt.expected {
				t.Errorf("IsSystemMailbox(%q) = %v, expected %v", tt.mailbox, result, tt.expected)
			}
		})
	}
}

func TestNormalizeMailboxName(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"INBOX uppercase", "INBOX", "INBOX"},
		{"inbox lowercase", "inbox", "INBOX"},
		{"InBoX mixed", "InBoX", "INBOX"},
		{"Custom unchanged", "Work", "Work"},
		{"Nested unchanged", "Work/Projects", "Work/Projects"},
		{"Empty", "", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := NormalizeMailboxName(tt.input)
			if result != tt.expected {
				t.Errorf("NormalizeMailboxName(%q) = %q, expected %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestNewMailboxInfo(t *testing.T) {
	t.Run("INBOX mailbox", func(t *testing.T) {
		mailbox := &domain.Mailbox{
			ID:   "test-id",
			Name: "inbox",
		}
		info := NewMailboxInfo(mailbox)

		if info.Name != "INBOX" {
			t.Errorf("Expected normalized name INBOX, got %q", info.Name)
		}
		if info.Delimiter != MailboxHierarchySeparator {
			t.Errorf("Expected delimiter %q, got %q", MailboxHierarchySeparator, info.Delimiter)
		}
		if info.IsSystem != true {
			t.Errorf("Expected IsSystem=true for INBOX")
		}
	})

	t.Run("Sent mailbox", func(t *testing.T) {
		mailbox := &domain.Mailbox{
			ID:   "test-id",
			Name: "Sent",
		}
		info := NewMailboxInfo(mailbox)

		if info.Name != "Sent" {
			t.Errorf("Expected name Sent, got %q", info.Name)
		}
		if info.IsSystem != true {
			t.Errorf("Expected IsSystem=true for Sent")
		}
		// Check for special-use attribute
		hasSentAttr := false
		for _, attr := range info.Attributes {
			if attr == imap.MailboxAttrSent {
				hasSentAttr = true
				break
			}
		}
		if !hasSentAttr {
			t.Errorf("Expected Sent attribute for Sent mailbox")
		}
	})

	t.Run("Custom mailbox", func(t *testing.T) {
		mailbox := &domain.Mailbox{
			ID:   "test-id",
			Name: "Work",
		}
		info := NewMailboxInfo(mailbox)

		if info.IsSystem != false {
			t.Errorf("Expected IsSystem=false for custom mailbox")
		}
	})
}

func TestMailboxInfo_ToIMAPListData(t *testing.T) {
	mailbox := &domain.Mailbox{
		ID:   "test-id",
		Name: "Trash",
	}
	info := NewMailboxInfo(mailbox)
	listData := info.ToIMAPListData()

	if listData.Mailbox != "Trash" {
		t.Errorf("Expected mailbox name Trash, got %q", listData.Mailbox)
	}
	if listData.Delim != '/' {
		t.Errorf("Expected delimiter '/', got %q", listData.Delim)
	}
	// Check for Trash attribute
	hasTrashAttr := false
	for _, attr := range listData.Attrs {
		if attr == imap.MailboxAttrTrash {
			hasTrashAttr = true
			break
		}
	}
	if !hasTrashAttr {
		t.Errorf("Expected Trash attribute in list data")
	}
}

func TestNewMailboxStatus(t *testing.T) {
	now := domain.Now()
	mailbox := &domain.Mailbox{
		ID:           "test-id",
		Name:         "inbox",
		MessageCount: 42,
		UnreadCount:  5,
		TotalSize:    1024000,
		UIDNext:      43,
		CreatedAt:    now,
	}

	status := NewMailboxStatus(mailbox)

	if status.Name != "INBOX" {
		t.Errorf("Expected normalized name INBOX, got %q", status.Name)
	}
	if status.Messages != 42 {
		t.Errorf("Expected 42 messages, got %d", status.Messages)
	}
	if status.Unseen != 5 {
		t.Errorf("Expected 5 unseen, got %d", status.Unseen)
	}
	if status.UIDNext != 43 {
		t.Errorf("Expected UIDNext=43, got %d", status.UIDNext)
	}
	if status.Size != 1024000 {
		t.Errorf("Expected size 1024000, got %d", status.Size)
	}
}

func TestMailboxStatus_ToIMAPStatusData(t *testing.T) {
	status := &MailboxStatus{
		Name:        "INBOX",
		Messages:    100,
		Unseen:      10,
		UIDNext:     101,
		UIDValidity: 12345,
	}

	t.Run("all options", func(t *testing.T) {
		data := status.ToIMAPStatusData(nil)

		if data.Mailbox != "INBOX" {
			t.Errorf("Expected mailbox INBOX, got %q", data.Mailbox)
		}
		if data.NumMessages == nil || *data.NumMessages != 100 {
			t.Errorf("Expected NumMessages=100")
		}
		if data.NumUnseen == nil || *data.NumUnseen != 10 {
			t.Errorf("Expected NumUnseen=10")
		}
		if data.UIDNext != 101 {
			t.Errorf("Expected UIDNext=101, got %d", data.UIDNext)
		}
		if data.UIDValidity != 12345 {
			t.Errorf("Expected UIDValidity=12345, got %d", data.UIDValidity)
		}
	})

	t.Run("specific options", func(t *testing.T) {
		options := &imap.StatusOptions{
			NumMessages: true,
			UIDNext:     true,
		}
		data := status.ToIMAPStatusData(options)

		if data.NumMessages == nil || *data.NumMessages != 100 {
			t.Errorf("Expected NumMessages=100")
		}
		if data.NumUnseen != nil {
			t.Errorf("Expected NumUnseen to be nil when not requested")
		}
		if data.UIDNext != 101 {
			t.Errorf("Expected UIDNext=101")
		}
	})
}

func TestNewSelectData(t *testing.T) {
	now := domain.Now()
	mailbox := &domain.Mailbox{
		ID:           "test-id",
		Name:         "INBOX",
		MessageCount: 50,
		UnreadCount:  3,
		CreatedAt:    now,
	}

	selectData := NewSelectData(mailbox)

	if selectData.NumMessages != 50 {
		t.Errorf("Expected 50 messages, got %d", selectData.NumMessages)
	}
	if selectData.UIDNext != 51 {
		t.Errorf("Expected UIDNext=51, got %d", selectData.UIDNext)
	}
	// Check standard flags are present
	expectedFlags := []imap.Flag{
		imap.FlagSeen,
		imap.FlagAnswered,
		imap.FlagFlagged,
		imap.FlagDeleted,
		imap.FlagDraft,
	}
	for _, expected := range expectedFlags {
		found := false
		for _, flag := range selectData.Flags {
			if flag == expected {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected flag %s in Flags", expected)
		}
	}
}

func TestParseMailboxPath(t *testing.T) {
	tests := []struct {
		name           string
		path           string
		expectedFull   string
		expectedParent string
		expectedName   string
		expectedDepth  int
	}{
		{
			name:           "simple",
			path:           "INBOX",
			expectedFull:   "INBOX",
			expectedParent: "",
			expectedName:   "INBOX",
			expectedDepth:  0,
		},
		{
			name:           "one level",
			path:           "Work/Projects",
			expectedFull:   "Work/Projects",
			expectedParent: "Work",
			expectedName:   "Projects",
			expectedDepth:  1,
		},
		{
			name:           "two levels",
			path:           "Work/Projects/2024",
			expectedFull:   "Work/Projects/2024",
			expectedParent: "Work/Projects",
			expectedName:   "2024",
			expectedDepth:  2,
		},
		{
			name:           "inbox normalized",
			path:           "inbox",
			expectedFull:   "INBOX",
			expectedParent: "",
			expectedName:   "INBOX",
			expectedDepth:  0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mp := ParseMailboxPath(tt.path)

			if mp.Full != tt.expectedFull {
				t.Errorf("Full = %q, expected %q", mp.Full, tt.expectedFull)
			}
			if mp.Parent != tt.expectedParent {
				t.Errorf("Parent = %q, expected %q", mp.Parent, tt.expectedParent)
			}
			if mp.Name != tt.expectedName {
				t.Errorf("Name = %q, expected %q", mp.Name, tt.expectedName)
			}
			if mp.Depth() != tt.expectedDepth {
				t.Errorf("Depth() = %d, expected %d", mp.Depth(), tt.expectedDepth)
			}
		})
	}
}

func TestMailboxPath_IsChildOf(t *testing.T) {
	tests := []struct {
		name     string
		path     string
		parent   string
		expected bool
	}{
		{"root is parent of all", "INBOX", "", true},
		{"child of parent", "Work/Projects", "Work", true},
		{"grandchild of grandparent", "Work/Projects/2024", "Work", true},
		{"not a child", "INBOX", "Work", false},
		{"sibling not child", "Work/Personal", "Work/Projects", false},
		{"same path not child", "Work", "Work", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mp := ParseMailboxPath(tt.path)
			result := mp.IsChildOf(tt.parent)
			if result != tt.expected {
				t.Errorf("IsChildOf(%q) = %v, expected %v", tt.parent, result, tt.expected)
			}
		})
	}
}

func TestValidateMailboxName(t *testing.T) {
	tests := []struct {
		name      string
		mailbox   string
		expectErr bool
	}{
		{"valid simple", "Work", false},
		{"valid with hierarchy", "Work/Projects", false},
		{"valid deep hierarchy", "Work/Projects/2024/Q1", false},
		{"empty name", "", true},
		{"too long", string(make([]byte, 300)), true},
		{"contains asterisk", "Work*", true},
		{"contains percent", "Work%", true},
		{"empty component", "Work//Projects", true},
		{"starts with separator", "/Work", true},
		{"ends with separator", "Work/", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateMailboxName(tt.mailbox)
			if tt.expectErr && err == nil {
				t.Errorf("Expected error for %q, got nil", tt.mailbox)
			}
			if !tt.expectErr && err != nil {
				t.Errorf("Expected no error for %q, got %v", tt.mailbox, err)
			}
		})
	}
}

func TestSystemMailboxNameMap(t *testing.T) {
	// Verify all system mailboxes are in the map
	for _, name := range SystemMailboxNames {
		if !SystemMailboxNameMap[name] {
			t.Errorf("SystemMailboxName %q not found in SystemMailboxNameMap", name)
		}
	}

	// Verify map has same count as slice
	if len(SystemMailboxNameMap) != len(SystemMailboxNames) {
		t.Errorf("SystemMailboxNameMap has %d entries, expected %d",
			len(SystemMailboxNameMap), len(SystemMailboxNames))
	}
}
