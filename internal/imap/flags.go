package imap

import (
	"github.com/emersion/go-imap/v2"

	"yunt/internal/domain"
)

// StandardFlags defines the standard IMAP flags supported by this server.
var StandardFlags = []imap.Flag{
	imap.FlagSeen,
	imap.FlagAnswered,
	imap.FlagFlagged,
	imap.FlagDeleted,
	imap.FlagDraft,
}

// PermanentFlags defines the flags that can be permanently stored.
var PermanentFlags = []imap.Flag{
	imap.FlagSeen,
	imap.FlagAnswered,
	imap.FlagFlagged,
	imap.FlagDeleted,
	imap.FlagDraft,
	imap.FlagWildcard, // Indicates custom flags are allowed
}

// FlagSet represents a set of IMAP flags with efficient operations.
type FlagSet struct {
	flags map[imap.Flag]bool
}

// NewFlagSet creates a new empty FlagSet.
func NewFlagSet() *FlagSet {
	return &FlagSet{
		flags: make(map[imap.Flag]bool),
	}
}

// NewFlagSetFromSlice creates a FlagSet from a slice of flags.
func NewFlagSetFromSlice(flags []imap.Flag) *FlagSet {
	fs := NewFlagSet()
	for _, f := range flags {
		fs.Add(f)
	}
	return fs
}

// NewFlagSetFromMessage creates a FlagSet from a domain.Message.
func NewFlagSetFromMessage(msg *domain.Message) *FlagSet {
	fs := NewFlagSet()

	// \Seen flag - maps to MessageRead status
	if msg.Status == domain.MessageRead {
		fs.Add(imap.FlagSeen)
	}

	// \Flagged - maps to IsStarred
	if msg.IsStarred {
		fs.Add(imap.FlagFlagged)
	}

	// \Deleted - maps to IsDeleted
	if msg.IsDeleted {
		fs.Add(imap.FlagDeleted)
	}

	// \Draft
	if msg.IsDraft {
		fs.Add(imap.FlagDraft)
	}

	// \Answered
	if msg.IsAnswered {
		fs.Add(imap.FlagAnswered)
	}

	return fs
}

// Add adds a flag to the set.
func (fs *FlagSet) Add(flag imap.Flag) {
	fs.flags[flag] = true
}

// Remove removes a flag from the set.
func (fs *FlagSet) Remove(flag imap.Flag) {
	delete(fs.flags, flag)
}

// Has returns true if the flag is in the set.
func (fs *FlagSet) Has(flag imap.Flag) bool {
	return fs.flags[flag]
}

// ToSlice returns the flags as a slice.
func (fs *FlagSet) ToSlice() []imap.Flag {
	result := make([]imap.Flag, 0, len(fs.flags))
	for f := range fs.flags {
		result = append(result, f)
	}
	return result
}

// AddAll adds all flags from the given slice.
func (fs *FlagSet) AddAll(flags []imap.Flag) {
	for _, f := range flags {
		fs.Add(f)
	}
}

// RemoveAll removes all flags from the given slice.
func (fs *FlagSet) RemoveAll(flags []imap.Flag) {
	for _, f := range flags {
		fs.Remove(f)
	}
}

// Replace replaces all flags with the given slice.
func (fs *FlagSet) Replace(flags []imap.Flag) {
	fs.flags = make(map[imap.Flag]bool)
	fs.AddAll(flags)
}

// Equal returns true if both FlagSets contain the same flags.
func (fs *FlagSet) Equal(other *FlagSet) bool {
	if len(fs.flags) != len(other.flags) {
		return false
	}
	for f := range fs.flags {
		if !other.Has(f) {
			return false
		}
	}
	return true
}

// Clone creates a copy of the FlagSet.
func (fs *FlagSet) Clone() *FlagSet {
	clone := NewFlagSet()
	for f := range fs.flags {
		clone.Add(f)
	}
	return clone
}

// Size returns the number of flags in the set.
func (fs *FlagSet) Size() int {
	return len(fs.flags)
}

// IsEmpty returns true if the set has no flags.
func (fs *FlagSet) IsEmpty() bool {
	return len(fs.flags) == 0
}

// Clear removes all flags from the set.
func (fs *FlagSet) Clear() {
	fs.flags = make(map[imap.Flag]bool)
}

// FlagChange represents a change in flags for a message.
type FlagChange struct {
	// MessageID is the domain ID of the message.
	MessageID domain.ID

	// SeqNum is the sequence number of the message in the mailbox.
	SeqNum uint32

	// UID is the UID of the message.
	UID imap.UID

	// OldFlags contains the flags before the change.
	OldFlags *FlagSet

	// NewFlags contains the flags after the change.
	NewFlags *FlagSet
}

// HasChanges returns true if the flags actually changed.
func (fc *FlagChange) HasChanges() bool {
	return !fc.OldFlags.Equal(fc.NewFlags)
}

// SeenChanged returns true if the \Seen flag changed.
func (fc *FlagChange) SeenChanged() bool {
	return fc.OldFlags.Has(imap.FlagSeen) != fc.NewFlags.Has(imap.FlagSeen)
}

// FlaggedChanged returns true if the \Flagged flag changed.
func (fc *FlagChange) FlaggedChanged() bool {
	return fc.OldFlags.Has(imap.FlagFlagged) != fc.NewFlags.Has(imap.FlagFlagged)
}

// DeletedChanged returns true if the \Deleted flag changed.
func (fc *FlagChange) DeletedChanged() bool {
	return fc.OldFlags.Has(imap.FlagDeleted) != fc.NewFlags.Has(imap.FlagDeleted)
}

// IsNowSeen returns true if the message was marked as seen.
func (fc *FlagChange) IsNowSeen() bool {
	return !fc.OldFlags.Has(imap.FlagSeen) && fc.NewFlags.Has(imap.FlagSeen)
}

// IsNowUnseen returns true if the message was marked as unseen.
func (fc *FlagChange) IsNowUnseen() bool {
	return fc.OldFlags.Has(imap.FlagSeen) && !fc.NewFlags.Has(imap.FlagSeen)
}

// IsNowFlagged returns true if the message was flagged (starred).
func (fc *FlagChange) IsNowFlagged() bool {
	return !fc.OldFlags.Has(imap.FlagFlagged) && fc.NewFlags.Has(imap.FlagFlagged)
}

// IsNowUnflagged returns true if the message was unflagged (unstarred).
func (fc *FlagChange) IsNowUnflagged() bool {
	return fc.OldFlags.Has(imap.FlagFlagged) && !fc.NewFlags.Has(imap.FlagFlagged)
}

// IsNowDeleted returns true if the message was marked as deleted.
func (fc *FlagChange) IsNowDeleted() bool {
	return !fc.OldFlags.Has(imap.FlagDeleted) && fc.NewFlags.Has(imap.FlagDeleted)
}

// IsNowUndeleted returns true if the message was unmarked as deleted.
func (fc *FlagChange) IsNowUndeleted() bool {
	return fc.OldFlags.Has(imap.FlagDeleted) && !fc.NewFlags.Has(imap.FlagDeleted)
}

// DraftChanged returns true if the \Draft flag changed.
func (fc *FlagChange) DraftChanged() bool {
	return fc.OldFlags.Has(imap.FlagDraft) != fc.NewFlags.Has(imap.FlagDraft)
}

// IsNowDraft returns true if the message was marked as draft.
func (fc *FlagChange) IsNowDraft() bool {
	return !fc.OldFlags.Has(imap.FlagDraft) && fc.NewFlags.Has(imap.FlagDraft)
}

// IsNowUndraft returns true if the message was unmarked as draft.
func (fc *FlagChange) IsNowUndraft() bool {
	return fc.OldFlags.Has(imap.FlagDraft) && !fc.NewFlags.Has(imap.FlagDraft)
}

// AnsweredChanged returns true if the \Answered flag changed.
func (fc *FlagChange) AnsweredChanged() bool {
	return fc.OldFlags.Has(imap.FlagAnswered) != fc.NewFlags.Has(imap.FlagAnswered)
}

// IsNowAnswered returns true if the message was marked as answered.
func (fc *FlagChange) IsNowAnswered() bool {
	return !fc.OldFlags.Has(imap.FlagAnswered) && fc.NewFlags.Has(imap.FlagAnswered)
}

// IsNowUnanswered returns true if the message was unmarked as answered.
func (fc *FlagChange) IsNowUnanswered() bool {
	return fc.OldFlags.Has(imap.FlagAnswered) && !fc.NewFlags.Has(imap.FlagAnswered)
}

// IsStandardFlag returns true if the flag is one of the standard IMAP system flags.
func IsStandardFlag(flag imap.Flag) bool {
	switch flag {
	case imap.FlagSeen, imap.FlagAnswered, imap.FlagFlagged,
		imap.FlagDeleted, imap.FlagDraft:
		return true
	default:
		return false
	}
}

// IsPermanentFlag returns true if the flag can be permanently stored.
// All flags except the wildcard flag can be stored permanently.
func IsPermanentFlag(_ imap.Flag) bool {
	return true
}

// FilterPermanentFlags returns only the flags that can be permanently stored.
func FilterPermanentFlags(flags []imap.Flag) []imap.Flag {
	result := make([]imap.Flag, 0, len(flags))
	for _, f := range flags {
		if IsPermanentFlag(f) {
			result = append(result, f)
		}
	}
	return result
}
