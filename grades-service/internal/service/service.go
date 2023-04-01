package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/PuerkitoBio/goquery"
	"grades-service/internal/config"
	"grades-service/internal/entity/change"
	"grades-service/internal/entity/user"
	"grades-service/internal/events/model"
	"grades-service/pkg/client/bars"
	"grades-service/pkg/client/mq"
	"grades-service/pkg/logging"
	"net/http"
	"regexp"
	"strings"
	"sync"
	"time"
)

var (
	ErrStructurePtChanged = errors.New("the structure of the progress table has changed")
	replacedString        = regexp.MustCompile("\\s+")
	wg                    sync.WaitGroup
)

const defaultRequestExpiration = "5000"

type Service struct {
	usersStorage   user.Repository
	changesStorage change.Repository
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

func (s *Service) checkChanges(usr user.User) error {
	defer wg.Done()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)

	defer cancel()

	decryptedPassword, err := usr.DecryptPassword()
	if err != nil {
		return fmt.Errorf("decrypt a password: %s", err)
	}

	progressTableParser, err := s.getProgressTableWithAuthorize(ctx, usr.Username, decryptedPassword)
	if errors.Is(err, bars.ErrNoAuth) {
		if err2 := s.usersStorage.LogoutUser(ctx, usr.ID); err2 != nil {
			return fmt.Errorf("usersStorage (LogoutUser): %v", err)
		}

		if err2 := s.sendExpiredDataMessage(usr.ID); err2 != nil {
			return fmt.Errorf("send expired data message to UserID %d: %v", usr.ID, err2)
		}

		return nil
	} else if errors.Is(err, user.ErrIncorrectData) {
		s.logger.Warningf("received incorrect progress table for UserID: %d", usr.ID)
		s.logger.Debugf("UserID: %d, ProgressTableParser: %s", usr.ID, *progressTableParser)
		return nil
	} else if err != nil {
		s.logger.Debugf("UserID: %d\n Username: %s\n", usr.ID, usr.Username)
		return fmt.Errorf("getProgressTableWithAuthorize: %v", err)
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
		if err = s.sendChangeToTelegram(&c, usr.ID); err != nil {
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

func (s *Service) UpdateAndGetProgressTable(ctx context.Context, id int64) (*user.ProgressTable, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)

	defer cancel()

	usr, err := s.usersStorage.FindOne(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("users repository FindOne: %v", err)
	}

	if usr == nil || usr.Deleted == true {
		return nil, nil
	}

	decryptedPassword, err := usr.DecryptPassword()
	if err != nil {
		return nil, fmt.Errorf("decrypted password: %v", err)
	}

	progressTableParser, err := s.getProgressTableWithAuthorize(ctx, usr.Username, decryptedPassword)
	if errors.Is(err, bars.ErrNoAuth) {
		if err2 := s.usersStorage.LogoutUser(ctx, usr.ID); err2 != nil {
			return nil, fmt.Errorf("failed to LogoutUser due to error: %v", err)
		}

		if err2 := s.sendExpiredDataMessage(usr.ID); err2 != nil {
			return nil, fmt.Errorf("send expired data message: %v", err)
		}
	} else if errors.Is(err, user.ErrIncorrectData) {
		s.logger.Warningf("received incorrect progress table for UserID: %d", usr.ID)
		return nil, err
	} else if err != nil {
		s.logger.Debugf("UserID: %d\n Username: %s\n", usr.ID, usr.Username)
		return nil, err
	}

	if err = s.usersStorage.UpdateProgressTable(ctx, usr.ID, *progressTableParser); err != nil {
		return nil, err
	}

	return progressTableParser, nil
}

func (s *Service) GetProgressTableFromDB(ctx context.Context, id int64) (*user.ProgressTable, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)

	defer cancel()

	usr, err := s.usersStorage.FindOne(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("users repository FindOne: %v", err)
	}

	if usr == nil || usr.Deleted == true {
		return nil, nil
	}

	return &usr.ProgressTable, nil
}

func (s *Service) getProgressTableWithAuthorize(ctx context.Context, username, password string) (*user.ProgressTable, error) {
	client, err := s.authorizeUser(ctx, username, password)
	if err != nil {
		return nil, err
	}

	response, err := client.GetPage(ctx, http.MethodGet, s.cfg.Bars.URLs.MainPageURL, nil)
	if err != nil {
		return &user.ProgressTable{}, err
	}

	document, err := goquery.NewDocumentFromReader(response.Body)
	if err != nil {
		return &user.ProgressTable{}, err
	}

	ptLength := document.Find("tbody").Length()
	ptObject := &user.ProgressTable{Tables: make([]user.SubjectTable, ptLength)}

	if err = extractSubjectTablesData(document, ptObject); err != nil {
		return &user.ProgressTable{}, err
	}

	if err = extractSubjectTableNames(document, ptObject); err != nil {
		return &user.ProgressTable{}, err
	}

	// TODO validate on client in future library
	if err = ptObject.ValidateData(); err != nil {
		return &user.ProgressTable{}, err
	}

	return ptObject, nil
}

func (s *Service) authorizeUser(ctx context.Context, username, password string) (*bars.Client, error) {
	client := bars.NewClient(s.cfg.Bars.URLs.RegistrationURL)

	if err := client.Authorization(ctx, username, password); errors.Is(err, bars.ErrNoAuth) {
		return &bars.Client{}, err
	} else if err != nil {
		return &bars.Client{}, err
	}

	return client, nil
}

func (s *Service) sendChangeToTelegram(c *change.Change, id int64) error {
	response := model.SendMessageRequest{
		RequestID: id,
		Message:   c.String(),
		ParseMode: "Markdown",
	}

	responseBytes, err := json.Marshal(response)
	if err != nil {
		return fmt.Errorf("json marshal: %v", err)
	}

	return s.producer.Publish(s.cfg.RabbitMQ.Producer.TelegramExchange,
		s.cfg.RabbitMQ.Producer.TelegramMessagesKey, defaultRequestExpiration, responseBytes)
}

func (s *Service) sendExpiredDataMessage(id int64) error {
	response := model.SendMessageRequest{
		RequestID: id,
		Message:   s.cfg.Responses.Bars.ExpiredData,
	}

	responseBytes, err := json.Marshal(response)
	if err != nil {
		return fmt.Errorf("json marshal: %v", err)
	}

	return s.producer.Publish(s.cfg.RabbitMQ.Producer.TelegramExchange,
		s.cfg.RabbitMQ.Producer.TelegramMessagesKey, defaultRequestExpiration, responseBytes)
}

func compareProgressTables(pt *user.ProgressTable, newpt *user.ProgressTable) ([]change.Change, error) {
	tdc := make([]change.Change, 0)
	if len(pt.Tables) != len(newpt.Tables) {
		return tdc, ErrStructurePtChanged
	}
	for i, st := range pt.Tables {
		if len(st.Rows) != len(newpt.Tables[i].Rows) {
			return tdc, ErrStructurePtChanged
		}
		for j, strow := range st.Rows {
			if strow.Name != newpt.Tables[i].Rows[j].Name {
				return tdc, ErrStructurePtChanged
			}
			if strow.Grades != newpt.Tables[i].Rows[j].Grades &&
				!strings.HasPrefix(strow.Name, "Балл текущего контроля") {
				tdc = append(tdc, change.Change{
					Subject:      newpt.Tables[i].Name,
					ControlEvent: newpt.Tables[i].Rows[j].Name,
					OldGrade:     strow.Grades,
					NewGrade:     newpt.Tables[i].Rows[j].Grades,
				})
			}
		}
	}
	return tdc, nil
}

func extractSubjectTableNames(document *goquery.Document, ptObject *user.ProgressTable) error {
	var err error
	document.Find(".my-2").Find("div:first-child").Clone().Children().Remove().End().EachWithBreak(func(nameId int, name *goquery.Selection) bool {
		processedString := replacedString.ReplaceAllString(name.Text(), " ")
		if strings.HasPrefix(processedString, " ") {
			processedString = strings.Replace(processedString, " ", "", 1)
		}
		if isEmptyData(processedString) {
			err = fmt.Errorf("part of received data is empty. nameID: %d", nameId)
			return false
		}
		ptObject.Tables[nameId].Name = processedString
		return true
	})

	return err
}

func extractSubjectTablesData(document *goquery.Document, ptObject *user.ProgressTable) error {
	var (
		err  error
		flag = true
	)
	filterTrSelection := func(i int, tr *goquery.Selection) bool {
		trLen := tr.Find("td").Length()
		return trLen == 4 || trLen == 2
	}

	document.Find("tbody").EachWithBreak(func(tbodyId int, tbody *goquery.Selection) bool {

		trSelection := tbody.Find("tr").FilterFunction(filterTrSelection)

		stLength := trSelection.Length()
		stObject := user.SubjectTable{Name: "", Rows: make([]user.SubjectTableRow, stLength)}

		trSelection.EachWithBreak(func(trId int, tr *goquery.Selection) bool {
			strObject := user.SubjectTableRow{}
			tdSelection := tr.Find("td")
			tdSelection.EachWithBreak(func(tdId int, td *goquery.Selection) bool {
				processedString := replacedString.ReplaceAllString(td.Text(), " ")

				switch tdId {
				case 0:
					if isEmptyData(processedString) {
						err = fmt.Errorf("part of received data is empty. tdId: %d trId: %d tbodyId: %d", tdId, trId, tbodyId)
						flag = false
					}
					if strings.HasPrefix(processedString, " ") {
						tdNew := strings.Replace(processedString, " ", "", 1)
						strObject.Name = tdNew
					} else {
						strObject.Name = processedString
					}
				case tdSelection.Length() - 1:
					if processedString == " " {
						strObject.Grades = "отсутствует"
					} else if strings.HasPrefix(processedString, " ") {
						tdNew := strings.Replace(processedString, " ", "", 1)
						strObject.Grades = tdNew
					} else {
						strObject.Grades = processedString
					}
				}

				return flag
			})
			stObject.Rows[trId] = strObject

			return flag
		})
		ptObject.Tables[tbodyId] = stObject

		return flag
	})

	return err
}

func isEmptyData(data string) bool {
	return data == "" || data == " "
}

func NewService(cfg *config.Config, usersStorage user.Repository, changesStorage change.Repository,
	logger *logging.Logger, producer mq.Producer) *Service {
	return &Service{
		usersStorage:   usersStorage,
		changesStorage: changesStorage,
		cfg:            cfg,
		logger:         logger,
		producer:       producer,
	}
}
