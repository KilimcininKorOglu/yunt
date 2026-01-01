package imap

import (
	"testing"

	"github.com/emersion/go-imap/v2"
)

func TestStoreHandlerApplyFlagOperation(t *testing.T) {
	handler := &StoreHandler{}

	tests := []struct {
		name          string
		currentFlags  []imap.Flag
		storeFlags    *imap.StoreFlags
		expectedFlags []imap.Flag
	}{
		{
			name:         "set flags on empty",
			currentFlags: []imap.Flag{},
			storeFlags: &imap.StoreFlags{
				Op:    imap.StoreFlagsSet,
				Flags: []imap.Flag{imap.FlagSeen, imap.FlagFlagged},
			},
			expectedFlags: []imap.Flag{imap.FlagSeen, imap.FlagFlagged},
		},
		{
			name:         "set flags replaces existing",
			currentFlags: []imap.Flag{imap.FlagSeen, imap.FlagDeleted},
			storeFlags: &imap.StoreFlags{
				Op:    imap.StoreFlagsSet,
				Flags: []imap.Flag{imap.FlagFlagged},
			},
			expectedFlags: []imap.Flag{imap.FlagFlagged},
		},
		{
			name:         "add flags to empty",
			currentFlags: []imap.Flag{},
			storeFlags: &imap.StoreFlags{
				Op:    imap.StoreFlagsAdd,
				Flags: []imap.Flag{imap.FlagSeen},
			},
			expectedFlags: []imap.Flag{imap.FlagSeen},
		},
		{
			name:         "add flags to existing",
			currentFlags: []imap.Flag{imap.FlagSeen},
			storeFlags: &imap.StoreFlags{
				Op:    imap.StoreFlagsAdd,
				Flags: []imap.Flag{imap.FlagFlagged, imap.FlagDeleted},
			},
			expectedFlags: []imap.Flag{imap.FlagSeen, imap.FlagFlagged, imap.FlagDeleted},
		},
		{
			name:         "add duplicate flags",
			currentFlags: []imap.Flag{imap.FlagSeen},
			storeFlags: &imap.StoreFlags{
				Op:    imap.StoreFlagsAdd,
				Flags: []imap.Flag{imap.FlagSeen, imap.FlagFlagged},
			},
			expectedFlags: []imap.Flag{imap.FlagSeen, imap.FlagFlagged},
		},
		{
			name:         "remove flags",
			currentFlags: []imap.Flag{imap.FlagSeen, imap.FlagFlagged, imap.FlagDeleted},
			storeFlags: &imap.StoreFlags{
				Op:    imap.StoreFlagsDel,
				Flags: []imap.Flag{imap.FlagSeen, imap.FlagDeleted},
			},
			expectedFlags: []imap.Flag{imap.FlagFlagged},
		},
		{
			name:         "remove non-existent flags",
			currentFlags: []imap.Flag{imap.FlagSeen},
			storeFlags: &imap.StoreFlags{
				Op:    imap.StoreFlagsDel,
				Flags: []imap.Flag{imap.FlagFlagged},
			},
			expectedFlags: []imap.Flag{imap.FlagSeen},
		},
		{
			name:         "remove all flags",
			currentFlags: []imap.Flag{imap.FlagSeen, imap.FlagFlagged},
			storeFlags: &imap.StoreFlags{
				Op:    imap.StoreFlagsDel,
				Flags: []imap.Flag{imap.FlagSeen, imap.FlagFlagged},
			},
			expectedFlags: []imap.Flag{},
		},
		{
			name:         "set empty flags clears all",
			currentFlags: []imap.Flag{imap.FlagSeen, imap.FlagFlagged},
			storeFlags: &imap.StoreFlags{
				Op:    imap.StoreFlagsSet,
				Flags: []imap.Flag{},
			},
			expectedFlags: []imap.Flag{},
		},
		{
			name:         "add custom flag with standard flag",
			currentFlags: []imap.Flag{},
			storeFlags: &imap.StoreFlags{
				Op:    imap.StoreFlagsSet,
				Flags: []imap.Flag{imap.FlagSeen, imap.Flag("$Label1")},
			},
			expectedFlags: []imap.Flag{imap.FlagSeen, imap.Flag("$Label1")},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			currentFlags := NewFlagSetFromSlice(tt.currentFlags)
			result := handler.applyFlagOperation(currentFlags, tt.storeFlags)

			expectedFlags := NewFlagSetFromSlice(tt.expectedFlags)

			if !result.Equal(expectedFlags) {
				t.Errorf("Expected flags %v, got %v", expectedFlags.ToSlice(), result.ToSlice())
			}
		})
	}
}

func TestStoreResult(t *testing.T) {
	result := NewStoreResult()

	if result.AffectedCount != 0 {
		t.Errorf("Initial AffectedCount should be 0, got %d", result.AffectedCount)
	}

	// Add a change that marks message as read
	change1 := &FlagChange{
		OldFlags: NewFlagSet(),
		NewFlags: NewFlagSetFromSlice([]imap.Flag{imap.FlagSeen}),
	}
	result.AddChange(change1)

	if result.AffectedCount != 1 {
		t.Errorf("AffectedCount should be 1, got %d", result.AffectedCount)
	}
	if result.UnseenCountDelta != -1 {
		t.Errorf("UnseenCountDelta should be -1, got %d", result.UnseenCountDelta)
	}

	// Add a change that marks message as unread
	change2 := &FlagChange{
		OldFlags: NewFlagSetFromSlice([]imap.Flag{imap.FlagSeen}),
		NewFlags: NewFlagSet(),
	}
	result.AddChange(change2)

	if result.AffectedCount != 2 {
		t.Errorf("AffectedCount should be 2, got %d", result.AffectedCount)
	}
	if result.UnseenCountDelta != 0 { // -1 + 1 = 0
		t.Errorf("UnseenCountDelta should be 0, got %d", result.UnseenCountDelta)
	}

	// Add a change with no actual changes (should not be counted)
	change3 := &FlagChange{
		OldFlags: NewFlagSetFromSlice([]imap.Flag{imap.FlagSeen}),
		NewFlags: NewFlagSetFromSlice([]imap.Flag{imap.FlagSeen}),
	}
	result.AddChange(change3)

	if result.AffectedCount != 2 {
		t.Errorf("AffectedCount should still be 2, got %d", result.AffectedCount)
	}
}

func TestNumSetContainsMessage(t *testing.T) {
	tests := []struct {
		name     string
		numSet   imap.NumSet
		seqNum   uint32
		uid      imap.UID
		expected bool
	}{
		{
			name:     "seq set contains",
			numSet:   imap.SeqSetNum(1, 2, 3, 4, 5),
			seqNum:   3,
			uid:      3,
			expected: true,
		},
		{
			name:     "seq set does not contain",
			numSet:   imap.SeqSetNum(1, 2, 3, 4, 5),
			seqNum:   10,
			uid:      10,
			expected: false,
		},
		{
			name:     "uid set contains",
			numSet:   imap.UIDSetNum(1, 5, 10),
			seqNum:   5,
			uid:      5,
			expected: true,
		},
		{
			name:     "uid set does not contain",
			numSet:   imap.UIDSetNum(1, 5, 10),
			seqNum:   3,
			uid:      3,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := numSetContainsMessage(tt.numSet, tt.seqNum, tt.uid)
			if result != tt.expected {
				t.Errorf("Expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestStoreFlagOperations(t *testing.T) {
	// Test that we can distinguish between different flag operations
	tests := []struct {
		name     string
		op       imap.StoreFlagsOp
		expected string
	}{
		{
			name:     "set operation",
			op:       imap.StoreFlagsSet,
			expected: "set",
		},
		{
			name:     "add operation",
			op:       imap.StoreFlagsAdd,
			expected: "add",
		},
		{
			name:     "delete operation",
			op:       imap.StoreFlagsDel,
			expected: "delete",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var opName string
			switch tt.op {
			case imap.StoreFlagsSet:
				opName = "set"
			case imap.StoreFlagsAdd:
				opName = "add"
			case imap.StoreFlagsDel:
				opName = "delete"
			}
			if opName != tt.expected {
				t.Errorf("Expected operation %s, got %s", tt.expected, opName)
			}
		})
	}
}
