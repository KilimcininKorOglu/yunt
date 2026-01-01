package imap

import (
	"testing"
	"time"

	"github.com/emersion/go-imap/v2"

	"yunt/internal/domain"
)

func TestSearchCriteriaParser_Parse(t *testing.T) {
	parser := NewSearchCriteriaParser()

	tests := []struct {
		name     string
		criteria *imap.SearchCriteria
		checkFn  func(t *testing.T, result *SearchCriteria)
	}{
		{
			name:     "nil criteria returns ALL",
			criteria: nil,
			checkFn: func(t *testing.T, result *SearchCriteria) {
				if !result.All {
					t.Error("expected All to be true for nil criteria")
				}
			},
		},
		{
			name:     "empty criteria returns ALL",
			criteria: &imap.SearchCriteria{},
			checkFn: func(t *testing.T, result *SearchCriteria) {
				if !result.All {
					t.Error("expected All to be true for empty criteria")
				}
			},
		},
		{
			name: "SEEN flag",
			criteria: &imap.SearchCriteria{
				Flag: []imap.Flag{imap.FlagSeen},
			},
			checkFn: func(t *testing.T, result *SearchCriteria) {
				if len(result.Flags) != 1 {
					t.Errorf("expected 1 flag, got %d", len(result.Flags))
				}
				if result.Flags[0] != imap.FlagSeen {
					t.Errorf("expected FlagSeen, got %s", result.Flags[0])
				}
			},
		},
		{
			name: "UNSEEN (NOT SEEN)",
			criteria: &imap.SearchCriteria{
				NotFlag: []imap.Flag{imap.FlagSeen},
			},
			checkFn: func(t *testing.T, result *SearchCriteria) {
				if len(result.NotFlags) != 1 {
					t.Errorf("expected 1 not-flag, got %d", len(result.NotFlags))
				}
				if result.NotFlags[0] != imap.FlagSeen {
					t.Errorf("expected FlagSeen, got %s", result.NotFlags[0])
				}
			},
		},
		{
			name: "FROM header",
			criteria: &imap.SearchCriteria{
				Header: []imap.SearchCriteriaHeaderField{
					{Key: "From", Value: "test@example.com"},
				},
			},
			checkFn: func(t *testing.T, result *SearchCriteria) {
				if result.From != "test@example.com" {
					t.Errorf("expected From to be 'test@example.com', got '%s'", result.From)
				}
			},
		},
		{
			name: "TO header",
			criteria: &imap.SearchCriteria{
				Header: []imap.SearchCriteriaHeaderField{
					{Key: "To", Value: "recipient@example.com"},
				},
			},
			checkFn: func(t *testing.T, result *SearchCriteria) {
				if result.To != "recipient@example.com" {
					t.Errorf("expected To to be 'recipient@example.com', got '%s'", result.To)
				}
			},
		},
		{
			name: "SUBJECT header",
			criteria: &imap.SearchCriteria{
				Header: []imap.SearchCriteriaHeaderField{
					{Key: "Subject", Value: "test subject"},
				},
			},
			checkFn: func(t *testing.T, result *SearchCriteria) {
				if result.Subject != "test subject" {
					t.Errorf("expected Subject to be 'test subject', got '%s'", result.Subject)
				}
			},
		},
		{
			name: "BEFORE date",
			criteria: &imap.SearchCriteria{
				Before: time.Date(2024, 1, 15, 12, 0, 0, 0, time.UTC),
			},
			checkFn: func(t *testing.T, result *SearchCriteria) {
				if result.Before == nil {
					t.Error("expected Before to be set")
					return
				}
				expected := time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC)
				if !result.Before.Equal(expected) {
					t.Errorf("expected Before to be %v, got %v", expected, *result.Before)
				}
			},
		},
		{
			name: "SINCE date",
			criteria: &imap.SearchCriteria{
				Since: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
			},
			checkFn: func(t *testing.T, result *SearchCriteria) {
				if result.Since == nil {
					t.Error("expected Since to be set")
				}
			},
		},
		{
			name: "BODY search",
			criteria: &imap.SearchCriteria{
				Body: []string{"hello world"},
			},
			checkFn: func(t *testing.T, result *SearchCriteria) {
				if len(result.Body) != 1 || result.Body[0] != "hello world" {
					t.Errorf("expected Body to be ['hello world'], got %v", result.Body)
				}
			},
		},
		{
			name: "TEXT search",
			criteria: &imap.SearchCriteria{
				Text: []string{"search term"},
			},
			checkFn: func(t *testing.T, result *SearchCriteria) {
				if len(result.Text) != 1 || result.Text[0] != "search term" {
					t.Errorf("expected Text to be ['search term'], got %v", result.Text)
				}
			},
		},
		{
			name: "LARGER size",
			criteria: &imap.SearchCriteria{
				Larger: 1024,
			},
			checkFn: func(t *testing.T, result *SearchCriteria) {
				if result.Larger != 1024 {
					t.Errorf("expected Larger to be 1024, got %d", result.Larger)
				}
			},
		},
		{
			name: "SMALLER size",
			criteria: &imap.SearchCriteria{
				Smaller: 4096,
			},
			checkFn: func(t *testing.T, result *SearchCriteria) {
				if result.Smaller != 4096 {
					t.Errorf("expected Smaller to be 4096, got %d", result.Smaller)
				}
			},
		},
		{
			name: "NOT criteria",
			criteria: &imap.SearchCriteria{
				Not: []imap.SearchCriteria{
					{Body: []string{"spam"}},
				},
			},
			checkFn: func(t *testing.T, result *SearchCriteria) {
				if len(result.Not) != 1 {
					t.Errorf("expected 1 NOT criterion, got %d", len(result.Not))
					return
				}
				if len(result.Not[0].Body) != 1 || result.Not[0].Body[0] != "spam" {
					t.Error("NOT criterion body mismatch")
				}
			},
		},
		{
			name: "OR criteria",
			criteria: &imap.SearchCriteria{
				Or: [][2]imap.SearchCriteria{
					{
						{Flag: []imap.Flag{imap.FlagSeen}},
						{Flag: []imap.Flag{imap.FlagFlagged}},
					},
				},
			},
			checkFn: func(t *testing.T, result *SearchCriteria) {
				if len(result.Or) != 1 {
					t.Errorf("expected 1 OR pair, got %d", len(result.Or))
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parser.Parse(tt.criteria)
			tt.checkFn(t, result)
		})
	}
}

func TestMessageMatcher_Matches(t *testing.T) {
	now := time.Now().UTC()

	baseMsg := &domain.Message{
		ID:        domain.ID("msg1"),
		MailboxID: domain.ID("mbox1"),
		From: domain.EmailAddress{
			Name:    "Sender",
			Address: "sender@example.com",
		},
		To: []domain.EmailAddress{
			{Name: "Recipient", Address: "recipient@example.com"},
		},
		Subject:    "Test Subject",
		TextBody:   "Hello, this is a test message body.",
		HTMLBody:   "<p>Hello, this is a test message body.</p>",
		Size:       1024,
		Status:     domain.MessageUnread,
		IsStarred:  false,
		IsSpam:     false,
		ReceivedAt: domain.Timestamp{Time: now},
		SentAt:     &domain.Timestamp{Time: now.Add(-time.Hour)},
	}

	tests := []struct {
		name     string
		criteria *SearchCriteria
		msg      *domain.Message
		seqNum   uint32
		uid      imap.UID
		want     bool
	}{
		{
			name:     "ALL matches everything",
			criteria: &SearchCriteria{All: true},
			msg:      baseMsg,
			seqNum:   1,
			uid:      1,
			want:     true,
		},
		{
			name:     "UNSEEN matches unread",
			criteria: &SearchCriteria{NotFlags: []imap.Flag{imap.FlagSeen}},
			msg:      baseMsg,
			seqNum:   1,
			uid:      1,
			want:     true,
		},
		{
			name: "SEEN does not match unread",
			criteria: &SearchCriteria{
				Flags: []imap.Flag{imap.FlagSeen},
			},
			msg:    baseMsg,
			seqNum: 1,
			uid:    1,
			want:   false,
		},
		{
			name: "SEEN matches read message",
			criteria: &SearchCriteria{
				Flags: []imap.Flag{imap.FlagSeen},
			},
			msg: func() *domain.Message {
				m := *baseMsg
				m.Status = domain.MessageRead
				return &m
			}(),
			seqNum: 1,
			uid:    1,
			want:   true,
		},
		{
			name:     "FROM matches sender",
			criteria: &SearchCriteria{From: "sender@example.com"},
			msg:      baseMsg,
			seqNum:   1,
			uid:      1,
			want:     true,
		},
		{
			name:     "FROM partial match",
			criteria: &SearchCriteria{From: "sender"},
			msg:      baseMsg,
			seqNum:   1,
			uid:      1,
			want:     true,
		},
		{
			name:     "FROM case insensitive",
			criteria: &SearchCriteria{From: "SENDER@EXAMPLE.COM"},
			msg:      baseMsg,
			seqNum:   1,
			uid:      1,
			want:     true,
		},
		{
			name:     "FROM does not match",
			criteria: &SearchCriteria{From: "other@example.com"},
			msg:      baseMsg,
			seqNum:   1,
			uid:      1,
			want:     false,
		},
		{
			name:     "TO matches recipient",
			criteria: &SearchCriteria{To: "recipient@example.com"},
			msg:      baseMsg,
			seqNum:   1,
			uid:      1,
			want:     true,
		},
		{
			name:     "SUBJECT matches",
			criteria: &SearchCriteria{Subject: "Test Subject"},
			msg:      baseMsg,
			seqNum:   1,
			uid:      1,
			want:     true,
		},
		{
			name:     "SUBJECT partial match",
			criteria: &SearchCriteria{Subject: "Test"},
			msg:      baseMsg,
			seqNum:   1,
			uid:      1,
			want:     true,
		},
		{
			name:     "BODY matches text body",
			criteria: &SearchCriteria{Body: []string{"test message"}},
			msg:      baseMsg,
			seqNum:   1,
			uid:      1,
			want:     true,
		},
		{
			name:     "TEXT matches subject",
			criteria: &SearchCriteria{Text: []string{"Subject"}},
			msg:      baseMsg,
			seqNum:   1,
			uid:      1,
			want:     true,
		},
		{
			name:     "TEXT matches from",
			criteria: &SearchCriteria{Text: []string{"sender"}},
			msg:      baseMsg,
			seqNum:   1,
			uid:      1,
			want:     true,
		},
		{
			name:     "LARGER matches",
			criteria: &SearchCriteria{Larger: 512},
			msg:      baseMsg,
			seqNum:   1,
			uid:      1,
			want:     true,
		},
		{
			name:     "LARGER does not match",
			criteria: &SearchCriteria{Larger: 2048},
			msg:      baseMsg,
			seqNum:   1,
			uid:      1,
			want:     false,
		},
		{
			name:     "SMALLER matches",
			criteria: &SearchCriteria{Smaller: 2048},
			msg:      baseMsg,
			seqNum:   1,
			uid:      1,
			want:     true,
		},
		{
			name:     "SMALLER does not match",
			criteria: &SearchCriteria{Smaller: 512},
			msg:      baseMsg,
			seqNum:   1,
			uid:      1,
			want:     false,
		},
		{
			name: "BEFORE matches",
			criteria: &SearchCriteria{
				Before: func() *time.Time { t := now.Add(time.Hour); return &t }(),
			},
			msg:    baseMsg,
			seqNum: 1,
			uid:    1,
			want:   true,
		},
		{
			name: "SINCE matches",
			criteria: &SearchCriteria{
				Since: func() *time.Time {
					// Use a date before today (normalized to midnight)
					t := normalizeDate(now).Add(-24 * time.Hour)
					return &t
				}(),
			},
			msg:    baseMsg,
			seqNum: 1,
			uid:    1,
			want:   true,
		},
		{
			name: "NOT matches when inner does not match",
			criteria: &SearchCriteria{
				Not: []*SearchCriteria{
					{From: "other@example.com"},
				},
			},
			msg:    baseMsg,
			seqNum: 1,
			uid:    1,
			want:   true,
		},
		{
			name: "NOT does not match when inner matches",
			criteria: &SearchCriteria{
				Not: []*SearchCriteria{
					{From: "sender@example.com"},
				},
			},
			msg:    baseMsg,
			seqNum: 1,
			uid:    1,
			want:   false,
		},
		{
			name: "OR matches when either side matches",
			criteria: &SearchCriteria{
				Or: [][2]*SearchCriteria{
					{
						{From: "other@example.com"},
						{Subject: "Test"},
					},
				},
			},
			msg:    baseMsg,
			seqNum: 1,
			uid:    1,
			want:   true,
		},
		{
			name: "OR does not match when neither side matches",
			criteria: &SearchCriteria{
				Or: [][2]*SearchCriteria{
					{
						{From: "other@example.com"},
						{Subject: "Other Subject"},
					},
				},
			},
			msg:    baseMsg,
			seqNum: 1,
			uid:    1,
			want:   false,
		},
		{
			name: "FLAGGED matches starred message",
			criteria: &SearchCriteria{
				Flags: []imap.Flag{imap.FlagFlagged},
			},
			msg: func() *domain.Message {
				m := *baseMsg
				m.IsStarred = true
				return &m
			}(),
			seqNum: 1,
			uid:    1,
			want:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			matcher := NewMessageMatcher(tt.criteria)
			got := matcher.Matches(tt.msg, tt.seqNum, tt.uid)
			if got != tt.want {
				t.Errorf("Matches() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNormalizeDate(t *testing.T) {
	input := time.Date(2024, 6, 15, 14, 30, 45, 123456789, time.UTC)
	result := normalizeDate(input)

	expected := time.Date(2024, 6, 15, 0, 0, 0, 0, time.UTC)
	if !result.Equal(expected) {
		t.Errorf("normalizeDate() = %v, want %v", result, expected)
	}
}

func TestContainsIgnoreCase(t *testing.T) {
	tests := []struct {
		s      string
		substr string
		want   bool
	}{
		{"Hello World", "world", true},
		{"Hello World", "HELLO", true},
		{"Hello World", "xyz", false},
		{"", "test", false},
		{"test", "", true},
		{"test@example.com", "EXAMPLE", true},
	}

	for _, tt := range tests {
		t.Run(tt.s+"_"+tt.substr, func(t *testing.T) {
			got := containsIgnoreCase(tt.s, tt.substr)
			if got != tt.want {
				t.Errorf("containsIgnoreCase(%q, %q) = %v, want %v", tt.s, tt.substr, got, tt.want)
			}
		})
	}
}

func TestHasFlag(t *testing.T) {
	flags := []imap.Flag{imap.FlagSeen, imap.FlagFlagged}

	tests := []struct {
		name string
		flag imap.Flag
		want bool
	}{
		{"has Seen", imap.FlagSeen, true},
		{"has Flagged", imap.FlagFlagged, true},
		{"missing Deleted", imap.FlagDeleted, false},
		{"missing Answered", imap.FlagAnswered, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := hasFlag(flags, tt.flag)
			if got != tt.want {
				t.Errorf("hasFlag() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSearchCriteria_isEmpty(t *testing.T) {
	tests := []struct {
		name     string
		criteria *SearchCriteria
		want     bool
	}{
		{
			name:     "empty criteria",
			criteria: &SearchCriteria{},
			want:     true,
		},
		{
			name:     "has From",
			criteria: &SearchCriteria{From: "test"},
			want:     false,
		},
		{
			name:     "has Flags",
			criteria: &SearchCriteria{Flags: []imap.Flag{imap.FlagSeen}},
			want:     false,
		},
		{
			name:     "has Body",
			criteria: &SearchCriteria{Body: []string{"test"}},
			want:     false,
		},
		{
			name: "has Since",
			criteria: &SearchCriteria{
				Since: func() *time.Time { t := time.Now(); return &t }(),
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.criteria.isEmpty()
			if got != tt.want {
				t.Errorf("isEmpty() = %v, want %v", got, tt.want)
			}
		})
	}
}
