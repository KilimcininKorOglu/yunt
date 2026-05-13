package push

import (
	"context"
	"time"

	"github.com/rs/zerolog"

	"yunt/internal/repository"
	"yunt/internal/service"
)

// DelayedSendScheduler periodically checks for pending EmailSubmissions
// with a sendAt time in the past and triggers delivery.
type DelayedSendScheduler struct {
	repo           repository.Repository
	relayService   *service.RelayService
	messageService *service.MessageService
	logger         zerolog.Logger
	interval       time.Duration
	stopCh         chan struct{}
}

// NewDelayedSendScheduler creates a new delayed send scheduler.
func NewDelayedSendScheduler(repo repository.Repository, relaySvc *service.RelayService, msgSvc *service.MessageService, logger zerolog.Logger) *DelayedSendScheduler {
	return &DelayedSendScheduler{
		repo:           repo,
		relayService:   relaySvc,
		messageService: msgSvc,
		logger:         logger,
		interval:       60 * time.Second,
		stopCh:         make(chan struct{}),
	}
}

// Start begins the periodic check loop in a goroutine.
func (s *DelayedSendScheduler) Start() {
	go s.run()
}

// Stop signals the scheduler to stop.
func (s *DelayedSendScheduler) Stop() {
	close(s.stopCh)
}

func (s *DelayedSendScheduler) run() {
	ticker := time.NewTicker(s.interval)
	defer ticker.Stop()

	for {
		select {
		case <-s.stopCh:
			return
		case <-ticker.C:
			s.processPending()
		}
	}
}

func (s *DelayedSendScheduler) processPending() {
	ctx := context.Background()

	pending, err := s.repo.JMAP().Submissions().GetPending(ctx)
	if err != nil {
		s.logger.Error().Err(err).Msg("failed to get pending submissions")
		return
	}

	now := time.Now().UTC()
	for _, sub := range pending {
		if sub.SendAt == nil || sub.SendAt.Time.After(now) {
			continue
		}

		msg, err := s.repo.Messages().GetByID(ctx, sub.EmailID)
		if err != nil {
			s.logger.Error().Err(err).Str("emailId", string(sub.EmailID)).Msg("delayed send: email not found")
			continue
		}

		if s.relayService != nil && s.relayService.IsEnabled() && len(msg.RawBody) > 0 {
			recipients := make([]string, len(msg.To))
			for i, to := range msg.To {
				recipients[i] = to.Address
			}
			result := s.relayService.Relay(ctx, msg.From.Address, recipients, msg.RawBody)
			if result != nil && result.Error != nil {
				s.logger.Error().Err(result.Error).Str("submissionId", string(sub.ID)).Msg("delayed send relay failed")
				continue
			}
		}

		sub.UndoStatus = "final"
		_ = s.repo.JMAP().Submissions().Update(ctx, sub)

		s.logger.Info().Str("submissionId", string(sub.ID)).Msg("delayed send completed")
	}
}
