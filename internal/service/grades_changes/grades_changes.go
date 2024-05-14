package grades_changes

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/ilyadubrovsky/tracking-bars/internal/config"
	"github.com/ilyadubrovsky/tracking-bars/internal/config/answers"
	"github.com/ilyadubrovsky/tracking-bars/internal/domain"
	ierrors "github.com/ilyadubrovsky/tracking-bars/internal/errors"
	"github.com/ilyadubrovsky/tracking-bars/internal/repository"
	"github.com/ilyadubrovsky/tracking-bars/internal/service"
	"github.com/ilyadubrovsky/tracking-bars/pkg/aes"
	"github.com/ilyadubrovsky/tracking-bars/pkg/bars"
	"github.com/jellydator/ttlcache/v3"
	"github.com/rs/zerolog/log"
	"gopkg.in/telebot.v3"
)

type svc struct {
	telegramSvc         service.Telegram
	barsSvc             service.Bars
	barsCredentialsRepo repository.BarsCredentials
	progressTablesRepo  repository.ProgressTables
	retriesCountCache   *ttlcache.Cache[int64, int]
	cfg                 config.Bars
	stopFunc            func()
}

func NewService(
	telegramSvc service.Telegram,
	barsSvc service.Bars,
	barsCredentialsRepo repository.BarsCredentials,
	progressTablesRepo repository.ProgressTables,
	retriesCountCache *ttlcache.Cache[int64, int],
	cfg config.Bars,
) *svc {
	return &svc{
		telegramSvc:         telegramSvc,
		barsSvc:             barsSvc,
		barsCredentialsRepo: barsCredentialsRepo,
		progressTablesRepo:  progressTablesRepo,
		retriesCountCache:   retriesCountCache,
		cfg:                 cfg,
	}
}

func (s *svc) Start() {
	ctx, cancel := context.WithCancel(context.Background())
	s.stopFunc = cancel

	jobChan := make(chan *domain.BarsCredentials)
	for i := 0; i < s.cfg.CronWorkerPoolSize; i++ {
		log.Info().Msgf("start %d grades changes worker", i+1)
		go s.checkChangesWorker(jobChan)
	}
	func() {
		log.Info().Msg("start actual credentials sender")
		for {
			select {
			case <-time.After(s.cfg.CronDelay):
				log.Info().Msg("sending actual credentials")
				s.sendActualCredentials(ctx, jobChan)
			case <-ctx.Done():
				close(jobChan)
				return
			}
		}
	}()
}

func (s *svc) sendActualCredentials(
	ctx context.Context,
	credentialsChan chan<- *domain.BarsCredentials,
) {
	barsCredentials, err := s.barsCredentialsRepo.GetAll(ctx)
	if err != nil {
		err = fmt.Errorf("barsCredentialsRepo.GetAll: %w", err)
		log.Error().Msgf("sendActualCredentials: %v", err.Error())
		return
	}

	for _, barsCredential := range barsCredentials {
		credentialsChan <- barsCredential
	}
}

func (s *svc) checkChangesWorker(credentialsChan <-chan *domain.BarsCredentials) {
	barsClient := bars.NewClient(config.BARSRegistrationPageURL)
	for credentials := range credentialsChan {
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)

		err := s.checkChanges(ctx, barsClient, credentials)
		if err != nil {
			log.Error().
				Int64("user", credentials.UserID).
				Msgf("checkChangesWorker: checkChanges: %v", err.Error())
		}

		barsClient.Clear()
		cancel()
	}
}

func (s *svc) checkChanges(
	ctx context.Context,
	barsClient bars.Client,
	credentials *domain.BarsCredentials,
) error {
	// TODO в теории могут быть проблемы с тем, что пользователь мог разлогиниться за время обхода
	decryptedPassword, err := aes.Decrypt([]byte(s.cfg.EncryptionKey), credentials.Password)
	if err != nil {
		return fmt.Errorf("aes.Decrypt: %w", err)
	}
	credentials.Password = []byte(decryptedPassword)

	progressTable, err := s.barsSvc.GetProgressTable(ctx, credentials, barsClient)
	if errors.Is(err, bars.ErrAuthorizationFailed) {
		// сервер барса после падений может дропать эту ошибку
		// часто возникает, фикси ретраями
		retriesCount := s.retriesCountCache.Get(credentials.UserID)
		if retriesCount == nil {
			s.retriesCountCache.Set(credentials.UserID, 1, ttlcache.DefaultTTL)
			return nil
		}

		if retriesCount.Value() < s.cfg.AuthorizationFailedRetriesCount && !retriesCount.IsExpired() {
			newRetriesCount := retriesCount.Value() + 1
			s.retriesCountCache.Set(
				credentials.UserID,
				newRetriesCount,
				ttlcache.DefaultTTL,
			)
			log.Info().
				Int64("user", credentials.UserID).
				Msgf("getting err authorization failed, retries %d", newRetriesCount)
			return nil
		}

		sendMsgErr := s.telegramSvc.SendMessageWithOpts(credentials.UserID, answers.CredentialsExpired)
		if sendMsgErr != nil {
			return fmt.Errorf("telegramSvc.SendMessageWithOpts(credentialsExpired): %w", err)
		}

		deleteErr := s.barsSvc.Logout(ctx, credentials.UserID)
		if deleteErr != nil {
			return fmt.Errorf("barsSvc.Logout(authFailed): %w", err)
		}

		log.Info().
			Int64("user", credentials.UserID).
			Msg("deleting user with err authorization failed")
		return nil
	}
	if errors.Is(err, ierrors.ErrWrongGradesPage) {
		sendMsgErr := s.telegramSvc.SendMessageWithOpts(credentials.UserID, answers.GradesPageWrong)
		if sendMsgErr != nil {
			return fmt.Errorf("telegramSvc.SendMessageWithOpts(gradesPageWrong): %w", err)
		}

		deleteErr := s.barsSvc.Logout(ctx, credentials.UserID)
		if deleteErr != nil {
			return fmt.Errorf("barsSvc.Delete(wrongGradesPage): %w", err)
		}

		log.Info().
			Int64("user", credentials.UserID).
			Msg("deleting user with wrong grades page")
		return nil
	}
	if err != nil {
		return fmt.Errorf("barsSvc.GetProgressTable: %w", err)
	}

	oldProgressTable, err := s.progressTablesRepo.GetByUserID(ctx, credentials.UserID)
	if err != nil {
		return fmt.Errorf("progressTableRepo.GetByUserID: %w", err)
	}

	if oldProgressTable != nil {
		changes, err := compareProgressTables(progressTable, oldProgressTable)
		if err != nil && !errors.Is(err, ierrors.ErrProgressTableStructChanged) {
			return fmt.Errorf("compareProgressTables: %w", err)
		}
		if len(changes) == 0 && !errors.Is(err, ierrors.ErrProgressTableStructChanged) {
			return nil
		}

		for _, change := range changes {
			sendMsgErr := s.telegramSvc.SendMessageWithOpts(
				credentials.UserID,
				change.String(),
				// TODO от зависимости телебота нужно избавиться
				telebot.ModeMarkdown,
			)
			if sendMsgErr != nil {
				log.Error().
					Int64("user", change.UserID).
					Msg("sending user's grade change failed")
				// TODO ретраи можно сделать, чтобы не терять изменения
				// или прихранивать их где-то
				continue
			}
		}
	}

	if err = s.progressTablesRepo.Save(ctx, progressTable); err != nil {
		return fmt.Errorf("progressTableRepo.Save: %w", err)
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

func compareProgressTables(
	newProgressTable *domain.ProgressTable,
	oldProgressTable *domain.ProgressTable,
) ([]*domain.GradeChange, error) {
	// TODO можно хэши сравнить сначала?
	// TODO получается, может быть ситуация, когда поменялась структура и мы потеряли какое-то изменение

	changes := make([]*domain.GradeChange, 0, len(newProgressTable.Disciplines))
	if len(newProgressTable.Disciplines) != len(oldProgressTable.Disciplines) {
		return changes, ierrors.ErrProgressTableStructChanged
	}

	for i, discipline := range newProgressTable.Disciplines {
		oldDiscipline := oldProgressTable.Disciplines[i]
		if len(discipline.ControlEvents) != len(oldDiscipline.ControlEvents) {
			return changes, ierrors.ErrProgressTableStructChanged
		}

		for j, controlEvent := range discipline.ControlEvents {
			oldControlEvent := oldDiscipline.ControlEvents[j]
			if controlEvent.Name != oldControlEvent.Name {
				return changes, ierrors.ErrProgressTableStructChanged
			}

			if controlEvent.Grade != oldControlEvent.Grade &&
				!strings.HasPrefix(controlEvent.Name, "Балл текущего контроля") {
				changes = append(changes, &domain.GradeChange{
					UserID:       newProgressTable.UserID,
					Discipline:   discipline.Name,
					ControlEvent: controlEvent.Name,
					OldGrade:     oldControlEvent.Grade,
					NewGrade:     controlEvent.Grade,
				})
			}
		}
	}

	return changes, nil
}
