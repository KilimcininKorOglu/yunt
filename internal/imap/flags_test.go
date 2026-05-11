package imap

import (
	"testing"

	"github.com/emersion/go-imap/v2"

	"yunt/internal/domain"
)

func TestNewFlagSet(t *testing.T) {
	fs := NewFlagSet()
	if fs == nil {
		t.Fatal("NewFlagSet returned nil")
	}
	if !fs.IsEmpty() {
		t.Error("New FlagSet should be empty")
	}
}

func TestNewFlagSetFromSlice(t *testing.T) {
	flags := []imap.Flag{imap.FlagSeen, imap.FlagFlagged}
	fs := NewFlagSetFromSlice(flags)

	if fs.Size() != 2 {
		t.Errorf("Expected size 2, got %d", fs.Size())
	}
	if !fs.Has(imap.FlagSeen) {
		t.Error("Expected FlagSeen to be present")
	}
	if !fs.Has(imap.FlagFlagged) {
		t.Error("Expected FlagFlagged to be present")
	}
}

func TestNewFlagSetFromMessage(t *testing.T) {
	tests := []struct {
		name     string
		msg      *domain.Message
		expected []imap.Flag
	}{
		{
			name: "unread message",
			msg: &domain.Message{
				Status:    domain.MessageUnread,
				IsStarred: false,
				InReplyTo: "",
			},
			expected: []imap.Flag{},
		},
		{
			name: "read message",
			msg: &domain.Message{
				Status:    domain.MessageRead,
				IsStarred: false,
				InReplyTo: "",
			},
			expected: []imap.Flag{imap.FlagSeen},
		},
		{
			name: "starred message",
			msg: &domain.Message{
				Status:    domain.MessageUnread,
				IsStarred: true,
				InReplyTo: "",
			},
			expected: []imap.Flag{imap.FlagFlagged},
		},
		{
			name: "replied message",
			msg: &domain.Message{
				Status:     domain.MessageUnread,
				IsStarred:  false,
				IsAnswered: true,
			},
			expected: []imap.Flag{imap.FlagAnswered},
		},
		{
			name: "read and starred message",
			msg: &domain.Message{
				Status:    domain.MessageRead,
				IsStarred: true,
				InReplyTo: "",
			},
			expected: []imap.Flag{imap.FlagSeen, imap.FlagFlagged},
		},
		{
			name: "all flags",
			msg: &domain.Message{
				Status:     domain.MessageRead,
				IsStarred:  true,
				IsAnswered: true,
			},
			expected: []imap.Flag{imap.FlagSeen, imap.FlagFlagged, imap.FlagAnswered},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fs := NewFlagSetFromMessage(tt.msg)
			expectedFs := NewFlagSetFromSlice(tt.expected)

			if fs.Size() != expectedFs.Size() {
				t.Errorf("Expected %d flags, got %d", expectedFs.Size(), fs.Size())
			}

			for _, flag := range tt.expected {
				if !fs.Has(flag) {
					t.Errorf("Expected flag %s to be present", flag)
				}
			}
		})
	}
}

func TestFlagSetAdd(t *testing.T) {
	fs := NewFlagSet()
	fs.Add(imap.FlagSeen)

	if !fs.Has(imap.FlagSeen) {
		t.Error("Flag should be present after Add")
	}
	if fs.Size() != 1 {
		t.Errorf("Expected size 1, got %d", fs.Size())
	}

	// Adding same flag again should not change size
	fs.Add(imap.FlagSeen)
	if fs.Size() != 1 {
		t.Errorf("Adding duplicate flag should not increase size, got %d", fs.Size())
	}
}

func TestFlagSetRemove(t *testing.T) {
	fs := NewFlagSetFromSlice([]imap.Flag{imap.FlagSeen, imap.FlagFlagged})
	fs.Remove(imap.FlagSeen)

	if fs.Has(imap.FlagSeen) {
		t.Error("Flag should not be present after Remove")
	}
	if fs.Size() != 1 {
		t.Errorf("Expected size 1, got %d", fs.Size())
	}

	// Removing non-existent flag should not panic
	fs.Remove(imap.FlagDeleted)
	if fs.Size() != 1 {
		t.Errorf("Removing non-existent flag should not change size, got %d", fs.Size())
	}
}

func TestFlagSetToSlice(t *testing.T) {
	flags := []imap.Flag{imap.FlagSeen, imap.FlagFlagged}
	fs := NewFlagSetFromSlice(flags)

	slice := fs.ToSlice()
	if len(slice) != 2 {
		t.Errorf("Expected 2 flags in slice, got %d", len(slice))
	}
}

func TestFlagSetAddAll(t *testing.T) {
	fs := NewFlagSet()
	fs.AddAll([]imap.Flag{imap.FlagSeen, imap.FlagFlagged, imap.FlagDeleted})

	if fs.Size() != 3 {
		t.Errorf("Expected size 3, got %d", fs.Size())
	}
}

func TestFlagSetRemoveAll(t *testing.T) {
	fs := NewFlagSetFromSlice([]imap.Flag{imap.FlagSeen, imap.FlagFlagged, imap.FlagDeleted})
	fs.RemoveAll([]imap.Flag{imap.FlagSeen, imap.FlagFlagged})

	if fs.Size() != 1 {
		t.Errorf("Expected size 1, got %d", fs.Size())
	}
	if !fs.Has(imap.FlagDeleted) {
		t.Error("FlagDeleted should still be present")
	}
}

func TestFlagSetReplace(t *testing.T) {
	fs := NewFlagSetFromSlice([]imap.Flag{imap.FlagSeen, imap.FlagFlagged})
	fs.Replace([]imap.Flag{imap.FlagDeleted, imap.FlagDraft})

	if fs.Size() != 2 {
		t.Errorf("Expected size 2, got %d", fs.Size())
	}
	if fs.Has(imap.FlagSeen) {
		t.Error("FlagSeen should not be present after Replace")
	}
	if !fs.Has(imap.FlagDeleted) {
		t.Error("FlagDeleted should be present after Replace")
	}
}

func TestFlagSetEqual(t *testing.T) {
	fs1 := NewFlagSetFromSlice([]imap.Flag{imap.FlagSeen, imap.FlagFlagged})
	fs2 := NewFlagSetFromSlice([]imap.Flag{imap.FlagFlagged, imap.FlagSeen})
	fs3 := NewFlagSetFromSlice([]imap.Flag{imap.FlagSeen})

	if !fs1.Equal(fs2) {
		t.Error("fs1 and fs2 should be equal")
	}
	if fs1.Equal(fs3) {
		t.Error("fs1 and fs3 should not be equal")
	}
}

func TestFlagSetClone(t *testing.T) {
	fs := NewFlagSetFromSlice([]imap.Flag{imap.FlagSeen, imap.FlagFlagged})
	clone := fs.Clone()

	if !fs.Equal(clone) {
		t.Error("Clone should be equal to original")
	}

	// Modifying clone should not affect original
	clone.Add(imap.FlagDeleted)
	if fs.Has(imap.FlagDeleted) {
		t.Error("Modifying clone should not affect original")
	}
}

func TestFlagChange(t *testing.T) {
	tests := []struct {
		name           string
		oldFlags       []imap.Flag
		newFlags       []imap.Flag
		hasChanges     bool
		seenChanged    bool
		flaggedChanged bool
		deletedChanged bool
	}{
		{
			name:           "no changes",
			oldFlags:       []imap.Flag{imap.FlagSeen},
			newFlags:       []imap.Flag{imap.FlagSeen},
			hasChanges:     false,
			seenChanged:    false,
			flaggedChanged: false,
			deletedChanged: false,
		},
		{
			name:           "add seen",
			oldFlags:       []imap.Flag{},
			newFlags:       []imap.Flag{imap.FlagSeen},
			hasChanges:     true,
			seenChanged:    true,
			flaggedChanged: false,
			deletedChanged: false,
		},
		{
			name:           "remove seen",
			oldFlags:       []imap.Flag{imap.FlagSeen},
			newFlags:       []imap.Flag{},
			hasChanges:     true,
			seenChanged:    true,
			flaggedChanged: false,
			deletedChanged: false,
		},
		{
			name:           "add flagged",
			oldFlags:       []imap.Flag{},
			newFlags:       []imap.Flag{imap.FlagFlagged},
			hasChanges:     true,
			seenChanged:    false,
			flaggedChanged: true,
			deletedChanged: false,
		},
		{
			name:           "add deleted",
			oldFlags:       []imap.Flag{imap.FlagSeen},
			newFlags:       []imap.Flag{imap.FlagSeen, imap.FlagDeleted},
			hasChanges:     true,
			seenChanged:    false,
			flaggedChanged: false,
			deletedChanged: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fc := &FlagChange{
				OldFlags: NewFlagSetFromSlice(tt.oldFlags),
				NewFlags: NewFlagSetFromSlice(tt.newFlags),
			}

			if fc.HasChanges() != tt.hasChanges {
				t.Errorf("HasChanges: expected %v, got %v", tt.hasChanges, fc.HasChanges())
			}
			if fc.SeenChanged() != tt.seenChanged {
				t.Errorf("SeenChanged: expected %v, got %v", tt.seenChanged, fc.SeenChanged())
			}
			if fc.FlaggedChanged() != tt.flaggedChanged {
				t.Errorf("FlaggedChanged: expected %v, got %v", tt.flaggedChanged, fc.FlaggedChanged())
			}
			if fc.DeletedChanged() != tt.deletedChanged {
				t.Errorf("DeletedChanged: expected %v, got %v", tt.deletedChanged, fc.DeletedChanged())
			}
		})
	}
}

func TestFlagChangeTransitions(t *testing.T) {
	tests := []struct {
		name           string
		oldFlags       []imap.Flag
		newFlags       []imap.Flag
		isNowSeen      bool
		isNowUnseen    bool
		isNowFlagged   bool
		isNowUnflagged bool
		isNowDeleted   bool
		isNowUndeleted bool
	}{
		{
			name:           "mark as seen",
			oldFlags:       []imap.Flag{},
			newFlags:       []imap.Flag{imap.FlagSeen},
			isNowSeen:      true,
			isNowUnseen:    false,
			isNowFlagged:   false,
			isNowUnflagged: false,
			isNowDeleted:   false,
			isNowUndeleted: false,
		},
		{
			name:           "mark as unseen",
			oldFlags:       []imap.Flag{imap.FlagSeen},
			newFlags:       []imap.Flag{},
			isNowSeen:      false,
			isNowUnseen:    true,
			isNowFlagged:   false,
			isNowUnflagged: false,
			isNowDeleted:   false,
			isNowUndeleted: false,
		},
		{
			name:           "star message",
			oldFlags:       []imap.Flag{},
			newFlags:       []imap.Flag{imap.FlagFlagged},
			isNowSeen:      false,
			isNowUnseen:    false,
			isNowFlagged:   true,
			isNowUnflagged: false,
			isNowDeleted:   false,
			isNowUndeleted: false,
		},
		{
			name:           "unstar message",
			oldFlags:       []imap.Flag{imap.FlagFlagged},
			newFlags:       []imap.Flag{},
			isNowSeen:      false,
			isNowUnseen:    false,
			isNowFlagged:   false,
			isNowUnflagged: true,
			isNowDeleted:   false,
			isNowUndeleted: false,
		},
		{
			name:           "mark as deleted",
			oldFlags:       []imap.Flag{},
			newFlags:       []imap.Flag{imap.FlagDeleted},
			isNowSeen:      false,
			isNowUnseen:    false,
			isNowFlagged:   false,
			isNowUnflagged: false,
			isNowDeleted:   true,
			isNowUndeleted: false,
		},
		{
			name:           "undelete message",
			oldFlags:       []imap.Flag{imap.FlagDeleted},
			newFlags:       []imap.Flag{},
			isNowSeen:      false,
			isNowUnseen:    false,
			isNowFlagged:   false,
			isNowUnflagged: false,
			isNowDeleted:   false,
			isNowUndeleted: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fc := &FlagChange{
				OldFlags: NewFlagSetFromSlice(tt.oldFlags),
				NewFlags: NewFlagSetFromSlice(tt.newFlags),
			}

			if fc.IsNowSeen() != tt.isNowSeen {
				t.Errorf("IsNowSeen: expected %v, got %v", tt.isNowSeen, fc.IsNowSeen())
			}
			if fc.IsNowUnseen() != tt.isNowUnseen {
				t.Errorf("IsNowUnseen: expected %v, got %v", tt.isNowUnseen, fc.IsNowUnseen())
			}
			if fc.IsNowFlagged() != tt.isNowFlagged {
				t.Errorf("IsNowFlagged: expected %v, got %v", tt.isNowFlagged, fc.IsNowFlagged())
			}
			if fc.IsNowUnflagged() != tt.isNowUnflagged {
				t.Errorf("IsNowUnflagged: expected %v, got %v", tt.isNowUnflagged, fc.IsNowUnflagged())
			}
			if fc.IsNowDeleted() != tt.isNowDeleted {
				t.Errorf("IsNowDeleted: expected %v, got %v", tt.isNowDeleted, fc.IsNowDeleted())
			}
			if fc.IsNowUndeleted() != tt.isNowUndeleted {
				t.Errorf("IsNowUndeleted: expected %v, got %v", tt.isNowUndeleted, fc.IsNowUndeleted())
			}
		})
	}
}

func TestIsStandardFlag(t *testing.T) {
	tests := []struct {
		flag     imap.Flag
		expected bool
	}{
		{imap.FlagSeen, true},
		{imap.FlagAnswered, true},
		{imap.FlagFlagged, true},
		{imap.FlagDeleted, true},
		{imap.FlagDraft, true},
		{imap.Flag("\\Custom"), false},
		{imap.Flag("$Label1"), false},
	}

	for _, tt := range tests {
		t.Run(string(tt.flag), func(t *testing.T) {
			if IsStandardFlag(tt.flag) != tt.expected {
				t.Errorf("IsStandardFlag(%s): expected %v, got %v", tt.flag, tt.expected, !tt.expected)
			}
		})
	}
}

func TestIsPermanentFlag(t *testing.T) {
	tests := []struct {
		flag     imap.Flag
		expected bool
	}{
		{imap.FlagSeen, true},
		{imap.FlagAnswered, true},
		{imap.FlagFlagged, true},
		{imap.FlagDeleted, true},
		{imap.FlagDraft, true},
		{imap.Flag("$Label1"), true},
	}

	for _, tt := range tests {
		t.Run(string(tt.flag), func(t *testing.T) {
			if IsPermanentFlag(tt.flag) != tt.expected {
				t.Errorf("IsPermanentFlag(%s): expected %v, got %v", tt.flag, tt.expected, !tt.expected)
			}
		})
	}
}

func TestFilterPermanentFlags(t *testing.T) {
	flags := []imap.Flag{imap.FlagSeen, imap.FlagFlagged, imap.Flag("$Custom")}
	filtered := FilterPermanentFlags(flags)

	if len(filtered) != 3 {
		t.Errorf("Expected 3 permanent flags, got %d", len(filtered))
	}

	// All flags should be kept as permanent
	for _, f := range flags {
		found := false
		for _, pf := range filtered {
			if f == pf {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Flag %s should be in filtered result", f)
		}
	}
}
