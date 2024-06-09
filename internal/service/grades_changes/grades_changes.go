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
	"github.com/ilyadubrovsky/tracking-bars/internal/service"
	"github.com/ilyadubrovsky/tracking-bars/pkg/aes"
	"github.com/ilyadubrovsky/tracking-bars/pkg/bars"
	"github.com/jellydator/ttlcache/v3"
	"github.com/rs/zerolog/log"
)

type svc struct {
	telegramSvc       service.Telegram
	barsSvc           service.Bars
	userSvc           service.User
	retriesCountCache *ttlcache.Cache[int64, int]
	cfg               config.Bars
	stopFunc          func()
}

func NewService(
	telegramSvc service.Telegram,
	barsSvc service.Bars,
	userSvc service.User,
	retriesCountCache *ttlcache.Cache[int64, int],
	cfg config.Bars,
) *svc {
	return &svc{
		telegramSvc:       telegramSvc,
		barsSvc:           barsSvc,
		userSvc:           userSvc,
		retriesCountCache: retriesCountCache,
		cfg:               cfg,
	}
}

func (s *svc) Start() {
	ctx, cancel := context.WithCancel(context.Background())
	s.stopFunc = cancel

	usersChan := make(chan *domain.User)
	for i := 0; i < s.cfg.CronWorkerPoolSize; i++ {
		log.Info().Msgf("start %d grades changes worker", i+1)
		go s.checkChangesWorker(usersChan)
	}
	func() {
		log.Info().Msg("start actual credentials sender")
		for {
			select {
			case <-time.After(s.cfg.CronDelay):
				log.Info().Msg("sending actual credentials")
				s.sendActualCredentials(ctx, usersChan)
			case <-ctx.Done():
				close(usersChan)
				return
			}
		}
	}()
}

func (s *svc) sendActualCredentials(
	ctx context.Context,
	usersChan chan<- *domain.User,
) {
	users, err := s.userSvc.Users(ctx)
	if err != nil {
		err = fmt.Errorf("barsCredentialsRepo.Users: %w", err)
		log.Error().Msgf("sendActualCredentials: %v", err.Error())
		return
	}

	for _, user := range users {
		usersChan <- user
	}
}

func (s *svc) checkChangesWorker(usersChan <-chan *domain.User) {
	barsClient := bars.NewClient(config.BARSRegistrationPageURL)
	for user := range usersChan {
		func() {
			defer barsClient.Clear()

			if user.BarsCredentials == nil {
				log.Error().
					Int64("user", user.ID).
					Msg("checkChangesWorker: received user with empty bars credentials")
				return
			}

			ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
			defer cancel()

			err := s.checkChanges(ctx, barsClient, user)
			if err != nil {
				log.Error().
					Int64("user", user.ID).
					Msgf("checkChangesWorker: checkChanges: %v", err.Error())
				return
			}
		}()
		// попытка делать запросы реже, чтобы не долбить БАРС
		time.Sleep(s.cfg.CronWorkerDelay)
	}
}

func (s *svc) checkChanges(
	ctx context.Context,
	barsClient bars.Client,
	user *domain.User,
) error {
	// TODO в теории могут быть проблемы с тем, что пользователь мог разлогиниться за время обхода
	decryptedPassword, err := aes.Decrypt([]byte(s.cfg.EncryptionKey), user.BarsCredentials.Password)
	if err != nil {
		return fmt.Errorf("aes.Decrypt: %w", err)
	}

	progressTable, err := s.barsSvc.GetProgressTable(
		ctx,
		user.BarsCredentials.Username,
		[]byte(decryptedPassword),
		barsClient,
	)
	if errors.Is(err, bars.ErrAuthorizationFailed) {
		retriesCount := s.nextRetriesCount(user.ID)
		if retriesCount < s.cfg.AuthorizationFailedRetriesCount {
			log.Info().
				Int64("user", user.ID).
				Str("reason", bars.ErrAuthorizationFailed.Error()).
				Msgf("new retries count value <%d>", retriesCount)
			return nil
		}

		sendMsgErr := s.telegramSvc.SendMessageWithOpts(user.ID, answers.CredentialsExpired)
		if sendMsgErr != nil {
			return fmt.Errorf("telegramSvc.SendMessageWithOpts(credentialsExpired): %w", err)
		}

		deleteErr := s.barsSvc.Logout(ctx, user.ID)
		if deleteErr != nil {
			return fmt.Errorf("barsSvc.Logout(authFailed): %w", err)
		}

		log.Info().
			Int64("user", user.ID).
			Msg("deleting user with err authorization failed")
		return nil
	}
	if errors.Is(err, ierrors.ErrWrongGradesPage) {
		retriesCount := s.nextRetriesCount(user.ID)
		if retriesCount < s.cfg.AuthorizationFailedRetriesCount {
			log.Info().
				Int64("user", user.ID).
				Str("reason", ierrors.ErrWrongGradesPage.Error()).
				Msgf("new retries count value <%d>", retriesCount)
			return nil
		}

		sendMsgErr := s.telegramSvc.SendMessageWithOpts(user.ID, answers.GradesPageWrong)
		if sendMsgErr != nil {
			return fmt.Errorf("telegramSvc.SendMessageWithOpts(gradesPageWrong): %w", err)
		}

		deleteErr := s.barsSvc.Logout(ctx, user.ID)
		if deleteErr != nil {
			return fmt.Errorf("barsSvc.Delete(wrongGradesPage): %w", err)
		}

		log.Info().
			Int64("user", user.ID).
			Msg("deleting user with wrong grades page")
		return nil
	}
	if err != nil {
		return fmt.Errorf("barsSvc.GetProgressTable: %w", err)
	}

	changes := make([]*domain.GradeChange, 0, len(progressTable.Disciplines))
	if user.ProgressTable != nil {
		changes, err = compareProgressTables(user.ID, progressTable, user.ProgressTable)
		if err != nil && !errors.Is(err, ierrors.ErrProgressTableStructChanged) {
			return fmt.Errorf("compareProgressTables: %w", err)
		}
		if len(changes) == 0 && !errors.Is(err, ierrors.ErrProgressTableStructChanged) {
			return nil
		}
	}

	err = s.userSvc.UpdateProgressTable(ctx, user.ID, progressTable, changes)
	if err != nil {
		return fmt.Errorf("userSvc.UpdateProgressTable: %w", err)
	}

	return nil
}

func (s *svc) nextRetriesCount(userID int64) int {
	// сервер барса после падений может отдавать неожидаемое поведение
	// часто возникает, фиксим ретраями
	retriesCount := s.retriesCountCache.Get(userID)
	if retriesCount == nil || retriesCount.IsExpired() {
		s.retriesCountCache.Set(userID, 1, ttlcache.DefaultTTL)
		return 1
	}

	newRetriesCount := retriesCount.Value() + 1
	s.retriesCountCache.Set(
		userID,
		newRetriesCount,
		ttlcache.DefaultTTL,
	)

	return retriesCount.Value()
}

func (s *svc) Stop() error {
	if s.stopFunc == nil {
		return errors.New("service is not started")
	}

	s.stopFunc()
	return nil
}

func compareProgressTables(
	userID int64,
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
					UserID:       userID,
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
