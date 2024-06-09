package grades_changes_outbox

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/ilyadubrovsky/tracking-bars/internal/config"
	"github.com/ilyadubrovsky/tracking-bars/internal/repository"
	"github.com/ilyadubrovsky/tracking-bars/internal/service"
	"github.com/rs/zerolog/log"
	"gopkg.in/telebot.v3"
)

const defaultGradesChangesLimit = 50

type svc struct {
	gradesChangesOutboxRepo repository.GradesChangesOutbox
	telegramSvc             service.Telegram
	cfg                     config.Bars
	stopFunc                func()
}

func NewService(
	gradesChangesOutboxRepo repository.GradesChangesOutbox,
	telegramSvc service.Telegram,
	cfg config.Bars,
) *svc {
	return &svc{
		gradesChangesOutboxRepo: gradesChangesOutboxRepo,
		telegramSvc:             telegramSvc,
		cfg:                     cfg,
	}
}

func (s *svc) Start() {
	ctx, cancel := context.WithCancel(context.Background())
	s.stopFunc = cancel

	for {
		select {
		case <-time.After(s.cfg.OutboxCronDelay):
			log.Info().Msg("sending grades changes")
			if err := s.sendGradesChanges(ctx); err != nil {
				log.Error().Msgf("sendGradesChanges: %v", err.Error())
			}
		case <-ctx.Done():
			return
		}
	}
}

// TODO может произойти зацикливание на какой-то ошибке и отправка не выполнится никогда
func (s *svc) sendGradesChanges(ctx context.Context) error {
	gradesChanges, err := s.gradesChangesOutboxRepo.GradesChanges(ctx, defaultGradesChangesLimit)
	if err != nil {
		return fmt.Errorf("gradesChangesOutboxRepo.GradesChanges: %w", err)
	}

	successfulSendingIDs := make([]int64, 0, len(gradesChanges))
	for _, gradeChange := range gradesChanges {
		sendMsgErr := s.telegramSvc.SendMessageWithOpts(
			gradeChange.UserID,
			gradeChange.String(),
			// TODO от зависимости телебота нужно избавиться
			telebot.ModeMarkdown,
		)
		if sendMsgErr != nil {
			log.Error().
				Int64("user", gradeChange.UserID).
				Msgf("sending grade change <id: %d> failed: %v", gradeChange.ID, sendMsgErr)
			continue
		}

		successfulSendingIDs = append(successfulSendingIDs, gradeChange.ID)
	}

	if len(successfulSendingIDs) != 0 {
		err = s.gradesChangesOutboxRepo.Delete(ctx, successfulSendingIDs)
		if err != nil {
			return fmt.Errorf("gradesChangesOutboxRepo.Delete: %w", err)
		}
	}

	return nil
}

func (s *svc) Stop() error {
	if s.stopFunc == nil {
		return errors.New("service is not started")
	}

	s.stopFunc()
	return nil
}
