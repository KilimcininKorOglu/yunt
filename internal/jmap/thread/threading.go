package thread

import (
	"context"
	"sort"
	"strings"

	"github.com/rs/zerolog"

	"yunt/internal/domain"
	"yunt/internal/jmap/state"
	"yunt/internal/repository"
)

// Resolver computes and assigns persistent thread IDs based on InReplyTo/References headers.
type Resolver struct {
	repo    repository.Repository
	state   *state.Manager
	logger  zerolog.Logger
	idGen   func() domain.ID
}

// NewResolver creates a new thread resolver.
func NewResolver(repo repository.Repository, stateManager *state.Manager, logger zerolog.Logger, idGen func() domain.ID) *Resolver {
	return &Resolver{
		repo:   repo,
		state:  stateManager,
		logger: logger,
		idGen:  idGen,
	}
}

// ResolveThreadID determines the thread ID for a message based on its InReplyTo and References headers.
// Returns the thread ID to assign. May trigger a thread merge if the message connects previously separate threads.
func (r *Resolver) ResolveThreadID(ctx context.Context, msg *domain.Message) (domain.ID, error) {
	refIDs := r.collectReferenceIDs(msg)
	if len(refIDs) == 0 {
		return r.idGen(), nil
	}

	related, err := r.repo.Messages().GetByMessageIDs(ctx, refIDs)
	if err != nil {
		return "", err
	}

	threadIDs := make(map[domain.ID]bool)
	for _, m := range related {
		if m.ThreadID != "" {
			threadIDs[m.ThreadID] = true
		}
	}

	switch len(threadIDs) {
	case 0:
		return r.idGen(), nil
	case 1:
		for tid := range threadIDs {
			return tid, nil
		}
	}

	// Multiple threads found — merge: smallest UUID wins
	winner := r.pickWinner(threadIDs)
	for tid := range threadIDs {
		if tid == winner {
			continue
		}
		if err := r.repo.Messages().UpdateThreadID(ctx, tid, winner); err != nil {
			return "", err
		}
		r.logger.Info().
			Str("oldThread", string(tid)).
			Str("newThread", string(winner)).
			Msg("merged threads")
	}

	return winner, nil
}

func (r *Resolver) collectReferenceIDs(msg *domain.Message) []string {
	seen := make(map[string]bool)
	var ids []string

	if msg.InReplyTo != "" {
		clean := strings.Trim(msg.InReplyTo, "<>")
		if !seen[clean] {
			seen[clean] = true
			ids = append(ids, clean)
		}
	}

	for _, ref := range msg.References {
		clean := strings.Trim(ref, "<>")
		if !seen[clean] {
			seen[clean] = true
			ids = append(ids, clean)
		}
	}

	return ids
}

func (r *Resolver) pickWinner(threadIDs map[domain.ID]bool) domain.ID {
	sorted := make([]domain.ID, 0, len(threadIDs))
	for tid := range threadIDs {
		sorted = append(sorted, tid)
	}
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i] < sorted[j]
	})
	return sorted[0]
}

// BackfillThreads assigns thread IDs to all messages that don't have one.
// Processes in receivedAt ASC order in batches.
func (r *Resolver) BackfillThreads(ctx context.Context, batchSize int) (int, error) {
	if batchSize <= 0 {
		batchSize = 1000
	}

	filter := &repository.MessageFilter{}
	opts := &repository.ListOptions{
		Sort: &repository.SortOptions{Field: "received_at", Order: domain.SortAsc},
		Pagination: &repository.PaginationOptions{
			Page:    1,
			PerPage: batchSize,
		},
	}

	total := 0
	for {
		result, err := r.repo.Messages().List(ctx, filter, opts)
		if err != nil {
			return total, err
		}

		assigned := 0
		for _, msg := range result.Items {
			if msg.ThreadID != "" {
				continue
			}

			threadID, err := r.ResolveThreadID(ctx, msg)
			if err != nil {
				r.logger.Error().Err(err).Str("msgID", string(msg.ID)).Msg("failed to resolve thread ID")
				continue
			}

			msg.ThreadID = threadID
			if err := r.repo.Messages().Update(ctx, msg); err != nil {
				r.logger.Error().Err(err).Str("msgID", string(msg.ID)).Msg("failed to update message thread ID")
				continue
			}
			assigned++
		}

		total += assigned
		r.logger.Info().Int("assigned", assigned).Int("total", total).Msg("backfill progress")

		if int64(len(result.Items)) < int64(batchSize) {
			break
		}
		opts.Pagination.Page++
	}

	return total, nil
}
