package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/ilyadubrovsky/bars"
	"grades-service/internal/config"
	"grades-service/internal/entity/change"
	"grades-service/internal/entity/user"
	"grades-service/internal/events/model"
	"grades-service/pkg/client/mq"
	"grades-service/pkg/logging"
	"strings"
	"sync"
	"time"
	"unicode/utf8"
)

var (
	ErrIncorrectData      = errors.New("the received data is incorrect")
	ErrStructurePtChanged = errors.New("the structure of the progress table has changed")
	wg                    sync.WaitGroup
)

const defaultRequestExpiration = "5000"

type changesRepository interface {
	Create(ctx context.Context, c change.Change) error
}

type gradesCache interface {
	Set(ctx context.Context, key string, pt *bars.ProgressTable) error
	Get(ctx context.Context, key string) (*bars.ProgressTable, error)
}

type userRepository interface {
	FindAll(ctx context.Context, aq ...string) ([]user.User, error)
	FindOne(ctx context.Context, id int64) (*user.User, error)
	UpdateProgressTable(ctx context.Context, id int64, table bars.ProgressTable) error
	LogoutUser(ctx context.Context, id int64) error
}

type Service struct {
	usersStorage   userRepository
	changesStorage changesRepository
	gradesCache    gradesCache
	cfg            *config.Config
	logger         *logging.Logger
	producer       mq.Producer
}

func (s *Service) ReceiveChanges() {
	delay := time.Duration(s.cfg.Bars.ParserDelayInSeconds) * time.Second
	s.logger.Info("BARS service: start receive changes")

	jobChannel := make(chan user.User)

	defer close(jobChannel)

	for i := 0; i < s.cfg.Bars.CountOfParsers; i++ {
		s.logger.Infof("grades parser [%d] started", i+1)
		go func() {
			for usr := range jobChannel {
				if err := s.checkChanges(usr); err != nil {
					s.logger.Errorf("failed to checkChanges due to error: %v", err)
				}
			}
		}()
	}

	for {
		time.Sleep(delay)

		s.logger.Infof("BARS service: parsing")

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)

		startTime := time.Now()

		usrs, err := s.usersStorage.FindAll(ctx, "WHERE deleted = false;")

		cancel()

		if err != nil {
			s.logger.Errorf("failed to FindAll due to error: %v", err)
			continue
		}

		for i := range usrs {
			wg.Add(1)
			jobChannel <- usrs[i]
		}

		wg.Wait()

		duration := time.Since(startTime)
		s.logger.Infof("BARS service: parsing completed in %v for %d users", duration, len(usrs))
	}
}

func (s *Service) GetProgressTableByRequest(ctx context.Context, id int64) (*bars.ProgressTable, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	usr, err := s.getUserFromDB(ctx, id)
	if err != nil {
		return nil, err
	}
	if usr == nil {
		return nil, nil
	}

	decryptedPassword, err := usr.DecryptPassword()
	if err != nil {
		return nil, fmt.Errorf("decrypt password: %v", err)
	}

	progressTableParser, err := s.authorizeAndGetProgressTable(ctx, usr.Username, decryptedPassword)
	if err != nil {
		return nil, s.handleBarsError(ctx, err, usr.ID)
	}

	if err = s.usersStorage.UpdateProgressTable(ctx, usr.ID, *progressTableParser); err != nil {
		return nil, err
	}
	if err = s.gradesCache.Set(ctx, fmt.Sprint(usr.ID), progressTableParser); err != nil {
		return nil, err
	}

	return progressTableParser, nil
}

func (s *Service) GetProgressTableFromDB(ctx context.Context, id int64) (*bars.ProgressTable, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	ptCache, err := s.gradesCache.Get(ctx, fmt.Sprint(id))
	if err != nil {
		return nil, err
	}
	if ptCache != nil {
		return ptCache, nil
	}

	usr, err := s.getUserFromDB(ctx, id)
	if err != nil {
		return nil, err
	}
	if usr != nil {
		if len(usr.ProgressTable.Tables) != 0 {
			if err = s.gradesCache.Set(ctx, fmt.Sprint(usr.ID), &usr.ProgressTable); err != nil {
				return nil, err
			}

		}
		return &usr.ProgressTable, nil
	}

	return nil, err
}

func (s *Service) checkChanges(usr user.User) error {
	defer wg.Done()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)

	defer cancel()

	decryptedPassword, err := usr.DecryptPassword()
	if err != nil {
		return fmt.Errorf("decrypt a password: %s", err)
	}

	progressTableParser, err := s.authorizeAndGetProgressTable(ctx, usr.Username, decryptedPassword)
	if err != nil {
		if errors.Is(err, bars.ErrWrongGradesPage) {
			if err2 := s.sendTelegramMessage(usr.ID, s.cfg.Responses.Bars.WrongGradesPage, ""); err2 != nil {
				return fmt.Errorf("send wrong grades page message to UserID %d: %v", usr.ID, err2)
			}
		}

		return s.handleBarsError(ctx, err, usr.ID)
	}

	tableDataChanges, err := compareProgressTables(&usr.ProgressTable, progressTableParser)
	if errors.Is(err, ErrStructurePtChanged) || len(tableDataChanges) != 0 {
		if err2 := s.usersStorage.UpdateProgressTable(ctx, usr.ID, *progressTableParser); err2 != nil {
			return fmt.Errorf("usersStorage (UpdateProgressTable): %v", err)
		}
	}

	if len(tableDataChanges) == 0 {
		return nil
	}

	for _, c := range tableDataChanges {
		if err = s.sendTelegramMessage(usr.ID, c.String(), "Markdown"); err != nil {
			s.logger.Errorf("failed to send a change to telegram user (id: %d) due to error: %v", usr.ID, err)
			continue
		}

		c.UserID = usr.ID

		if err = s.changesStorage.Create(ctx, c); err != nil {
			s.logger.Errorf("changes repository: failed to Create due to error: %v", err)
		}
	}

	return nil
}

func (s *Service) authorizeAndGetProgressTable(ctx context.Context, username, password string) (*bars.ProgressTable, error) {
	client := bars.NewClient()

	if err := client.Authorization(ctx, username, password); err != nil {
		return nil, err
	}

	pt, err := client.GetProgressTable(ctx)
	if err != nil {
		return nil, err
	}

	if err = validateProgressTable(pt); err != nil {
		return nil, err
	}

	return pt, nil
}

func (s *Service) handleBarsError(ctx context.Context, err error, id int64) error {
	switch err {
	case bars.ErrWrongGradesPage:
		if err2 := s.usersStorage.LogoutUser(ctx, id); err2 != nil {
			s.logger.Errorf("usersStorage: failed to LogoutUser (UserID: %d) in ErrWrongGradesPage case due to error: %v", id, err2)
			return err2
		}
	case bars.ErrNoAuth:
		if err2 := s.usersStorage.LogoutUser(ctx, id); err2 != nil {
			s.logger.Errorf("usersStorage: failed to LogoutUser (UserID: %d) in ErrNoAuth case due to error: %v", id, err2)
			return err2
		}

		if err2 := s.sendTelegramMessage(id, s.cfg.Responses.Bars.ExpiredData, ""); err2 != nil {
			s.logger.Errorf("failed to send expired data message to UserID %d due to error: %v", id, err2)
			return err2
		}
	case ErrIncorrectData:
		s.logger.Warningf("received incorrect progress table for UserID: %d", id)
	default:
		s.logger.Errorf("failed to execute BARS client method for UserID: %d due to error: %v", id, err)
	}

	return err
}

func (s *Service) sendTelegramMessage(id int64, message string, parsemode string) error {
	response := model.SendMessageRequest{
		RequestID: id,
		Message:   message,
		ParseMode: parsemode,
	}

	responseBytes, err := json.Marshal(response)
	if err != nil {
		return fmt.Errorf("json marshal: %v", err)
	}

	return s.producer.Publish(s.cfg.RabbitMQ.Producer.TelegramExchange,
		s.cfg.RabbitMQ.Producer.TelegramMessagesKey, defaultRequestExpiration, responseBytes)
}

func (s *Service) getUserFromDB(ctx context.Context, id int64) (*user.User, error) {
	usr, err := s.usersStorage.FindOne(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("users repository FindOne: %v", err)
	}

	if usr == nil || usr.Deleted == true {
		return nil, nil
	}

	return usr, nil
}

// TODO make it more efficient
func compareProgressTables(pt *bars.ProgressTable, newpt *bars.ProgressTable) ([]change.Change, error) {
	tdc := make([]change.Change, 0)
	if len(pt.Tables) != len(newpt.Tables) {
		return tdc, ErrStructurePtChanged
	}
	for i, st := range pt.Tables {
		if len(st.ControlEvents) != len(newpt.Tables[i].ControlEvents) {
			return tdc, ErrStructurePtChanged
		}
		for j, strow := range st.ControlEvents {
			if strow.Name != newpt.Tables[i].ControlEvents[j].Name {
				return tdc, ErrStructurePtChanged
			}
			if strow.Grades != newpt.Tables[i].ControlEvents[j].Grades &&
				!strings.HasPrefix(strow.Name, "Балл текущего контроля") {
				tdc = append(tdc, change.Change{
					Subject:      newpt.Tables[i].Name,
					ControlEvent: newpt.Tables[i].ControlEvents[j].Name,
					OldGrade:     strow.Grades,
					NewGrade:     newpt.Tables[i].ControlEvents[j].Grades,
				})
			}
		}
	}
	return tdc, nil
}

func validateProgressTable(pt *bars.ProgressTable) error {
	if !utf8.ValidString(pt.String()) {
		return ErrIncorrectData
	} else {
		return nil
	}
}

func NewService(cfg *config.Config, usersStorage userRepository, changesStorage changesRepository,
	gradesCache gradesCache, logger *logging.Logger, producer mq.Producer) *Service {
	return &Service{
		usersStorage:   usersStorage,
		changesStorage: changesStorage,
		gradesCache:    gradesCache,
		cfg:            cfg,
		logger:         logger,
		producer:       producer,
	}
}
