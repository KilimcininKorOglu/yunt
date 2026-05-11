package imap

import (
	"context"
	"io"
	"strings"

	"github.com/emersion/go-imap/v2"
	"github.com/google/uuid"

	"yunt/internal/domain"
	"yunt/internal/parser"
)

func (s *Session) appendMessage(mailbox string, r imap.LiteralReader, options *imap.AppendOptions) (*imap.AppendData, error) {
	if s.userSession == nil {
		return nil, &imap.Error{
			Type: imap.StatusResponseTypeNo,
			Text: "Not authenticated",
		}
	}

	ctx := context.Background()
	repo := s.server.backend.repo
	userID := s.userSession.User.ID

	mbox, err := s.findAppendMailbox(ctx, mailbox, userID)
	if err != nil {
		return nil, err
	}

	rawBody, err := io.ReadAll(r)
	if err != nil {
		return nil, &imap.Error{
			Type: imap.StatusResponseTypeNo,
			Text: "Failed to read message data",
		}
	}

	msg := &domain.Message{
		ID:        domain.ID(uuid.New().String()),
		MailboxID: mbox.ID,
		RawBody:   rawBody,
		Size:      int64(len(rawBody)),
		Status:    domain.MessageUnread,
		CreatedAt: domain.Now(),
	}

	if options != nil {
		for _, flag := range options.Flags {
			switch flag {
			case imap.FlagSeen:
				msg.Status = domain.MessageRead
			case imap.FlagFlagged:
				msg.IsStarred = true
			case imap.FlagDeleted:
				msg.IsDeleted = true
			case imap.FlagDraft:
				msg.IsDraft = true
			case imap.FlagAnswered:
				msg.IsAnswered = true
			}
		}
		if !options.Time.IsZero() {
			ts := domain.Timestamp{Time: options.Time}
			msg.SentAt = &ts
		}
	}

	p := parser.NewParser()
	parsed, parseErr := p.Parse(rawBody)
	if parseErr == nil {
		parsed.ApplyTo(msg)
	} else {
		s.logger.Warn().Err(parseErr).Msg("APPEND: failed to parse MIME, storing with raw body only")
	}

	if err := repo.Messages().Create(ctx, msg); err != nil {
		return nil, &imap.Error{
			Type: imap.StatusResponseTypeNo,
			Text: "Failed to store message",
		}
	}

	if err := repo.Messages().StoreRawBody(ctx, msg.ID, rawBody); err != nil {
		s.logger.Warn().Err(err).Msg("APPEND: failed to store raw body")
	}

	if err := repo.Mailboxes().IncrementMessageCount(ctx, mbox.ID, msg.Size); err != nil {
		s.logger.Warn().Err(err).Msg("APPEND: failed to update mailbox stats")
	}

	msgCount, _ := repo.Messages().CountByMailbox(ctx, mbox.ID)
	uid := imap.UID(msgCount)

	return &imap.AppendData{
		UID:         uid,
		UIDValidity: generateUIDValidity(mbox),
	}, nil
}

func (s *Session) findAppendMailbox(ctx context.Context, name string, userID domain.ID) (*domain.Mailbox, error) {
	repo := s.server.backend.repo
	result, err := repo.Mailboxes().ListByUser(ctx, userID, nil)
	if err != nil {
		return nil, &imap.Error{
			Type: imap.StatusResponseTypeNo,
			Text: "Failed to list mailboxes",
		}
	}

	for _, mbox := range result.Items {
		if strings.EqualFold(NormalizeMailboxName(mbox.Name), name) {
			return mbox, nil
		}
	}

	return nil, &imap.Error{
		Type: imap.StatusResponseTypeNo,
		Code: imap.ResponseCodeTryCreate,
		Text: "Mailbox does not exist",
	}
}
