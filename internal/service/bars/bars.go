package bars

import (
	"TrackingBARSv2/internal/config"
	"TrackingBARSv2/internal/entity/change"
	"TrackingBARSv2/internal/entity/user"
	"TrackingBARSv2/pkg/client/bars"
	"TrackingBARSv2/pkg/logging"
	"TrackingBARSv2/pkg/utils/aes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/PuerkitoBio/goquery"
	"gopkg.in/telebot.v3"
	"net/http"
	"os"
	"regexp"
	"strings"
	"sync"
	"time"
)

var (
	wg                    sync.WaitGroup
	ErrStructurePtChanged = errors.New("the structure of the progress table has changed")
	replacedString        = regexp.MustCompile("\\s+")
)

type Service interface {
	Start()
	GetProgressTable(ctx context.Context, client bars.Client) (user.ProgressTable, error)
	CheckChanges(usr user.User)
}

type service struct {
	bot            *telebot.Bot
	usersStorage   user.Repository
	changesStorage change.Repository
	cfg            *config.Config
	logger         logging.Logger
}

func (s *service) Start() {
	delay := time.Duration(s.cfg.Bars.ParserDelayInSeconds) * time.Second
	s.logger.Info("BARS service: start")

	for {
		time.Sleep(delay)
		startTime := time.Now()

		usrs, err := s.usersStorage.FindAll(context.Background(), "WHERE deleted = false;")
		if err != nil {
			s.logger.Errorf("failed to FindAll due error: %s", err)
			continue
		}

		for _, usr := range usrs {
			wg.Add(1)
			go s.CheckChanges(usr)
		}

		wg.Wait()

		duration := time.Since(startTime)
		s.logger.Tracef("BARS service: parsing completed in: %v", duration)
	}
}

func (s *service) GetProgressTable(ctx context.Context, client bars.Client) (user.ProgressTable, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	response, err := client.GetPage(ctx, http.MethodGet, s.cfg.Bars.URLs.MainPageURL, nil)
	if err != nil {
		s.logger.Debugf("Status code: %d,\n Response body: %s\n Response header: %s", response.StatusCode, response.Body, response.Header)
		return user.ProgressTable{}, fmt.Errorf("failed to get a page due error: %s", err)
	}

	document, err := goquery.NewDocumentFromReader(response.Body)
	if err != nil {
		return user.ProgressTable{}, fmt.Errorf("failed to create a document from the response body due error: %s", err)
	}

	ptLength := document.Find("tbody").Length()
	ptObject := user.ProgressTable{Tables: make([]user.SubjectTable, ptLength)}

	getSubjectTableData(document, &ptObject)

	getSubjectTableNames(document, &ptObject)

	if err = ptObject.ValidateData(); err == user.ErrIncorrectData {
		return user.ProgressTable{}, err
	}

	return ptObject, nil
}

// CheckChanges TODO fix leaked changes (attempt to resend)
func (s *service) CheckChanges(usr user.User) {
	defer wg.Done()

	client := bars.NewClient(s.cfg.Bars.URLs.RegistrationURL)

	if err := s.authorizeUser(&usr, client); err != nil {
		return
	}

	progressTableParser, err := s.GetProgressTable(context.Background(), client)
	if err != nil {
		s.logger.Errorf("failed to GetProgressTable method due error: %s", err)
		return
	}

	tableDataChanges, err := compare(usr.ProgressTable, progressTableParser)
	if err == ErrStructurePtChanged {
		if err2 := s.updateUser(&usr, &progressTableParser); err2 != nil {
			s.logger.Errorf("failed to update user due error: %s", err)
		}
		return
	} else if err != nil {
		s.logger.Errorf("failed to compare method due error: %s", err)
		return
	}

	if len(tableDataChanges) == 0 {
		return
	}

	if err = s.updateUser(&usr, &progressTableParser); err != nil {
		return
	}

	for _, tdc := range tableDataChanges {
		if err = s.createChange(usr.ID, &tdc); err != nil {
			return
		}

		msg := fmt.Sprintf("*Получено изменение:*\n\n%s\n\n", tdc.String())
		_, err = s.bot.Send(telebot.ChatID(usr.ID), msg, "Markdown")
		if err != nil {
			s.logger.Errorf("failed to send a message with the change to the user due error: %s", err)
			continue
		}
	}
}

func (s *service) authorizeUser(usr *user.User, client bars.Client) error {
	decryptedPassword, err := aes.DecryptAES([]byte(os.Getenv("ENCRYPTION_KEY")), usr.Password)
	if err != nil {
		s.logger.Errorf("failed to decrypt a password (DecryptAES) due error: %s", err)
		return err
	}

	if err = client.Authorization(context.Background(), usr.Username, decryptedPassword); err == bars.ErrNoAuth {
		if err2 := s.usersStorage.Delete(context.Background(), usr.ID); err2 != nil {
			s.logger.Errorf("failed to Delete due error: %s", err)
			return err
		}

		_, err2 := s.bot.Send(telebot.ChatID(usr.ID), s.cfg.Messages.Bars.ExpiredData)
		if err2 != nil {
			s.logger.Debugf("UserID: %d\n Message: %s", usr.ID, s.cfg.Messages.Bars.ExpiredData)
			s.logger.Errorf("failed to send a message due error: %s", err2)
			return err2
		}
	} else if err != nil {
		s.logger.Errorf("failed to Authorization method due error: %s", err)
		s.logger.Debugf("UserID: %d\nUsername:%s\nPassword:%x\n", usr.ID, usr.Username, usr.Password)
		return err
	}

	return nil
}

func (s *service) updateUser(usr *user.User, pt *user.ProgressTable) error {
	ptBytes, err := json.Marshal(pt)
	if err != nil {
		s.logger.Errorf("failed to marshal the structure received from the parser due error: %s", err)
		return err
	}

	usrDTO := user.UpdateUserDTO{
		ID:            usr.ID,
		Username:      usr.Username,
		Password:      usr.Password,
		ProgressTable: string(ptBytes),
		Deleted:       false,
	}
	if err = s.usersStorage.Update(context.Background(), usrDTO); err != nil {
		return err
	}

	return nil
}

func (s *service) createChange(userID int64, tdc *change.Change) error {
	if err := s.changesStorage.Create(context.Background(), change.CreateChangeDTO{
		UserID:       userID,
		Subject:      tdc.Subject,
		ControlEvent: tdc.ControlEvent,
		OldGrade:     tdc.OldGrade,
		NewGrade:     tdc.NewGrade,
	}); err != nil {
		s.logger.Errorf("failed to Create due error: %s", err)
		return err
	}

	return nil
}

func NewService(cfg *config.Config, usersStorage user.Repository, changesStorage change.Repository, logger logging.Logger, bot *telebot.Bot) *service {
	return &service{
		usersStorage:   usersStorage,
		changesStorage: changesStorage,
		cfg:            cfg,
		logger:         logger,
		bot:            bot,
	}
}

func compare(pt user.ProgressTable, newpt user.ProgressTable) ([]change.Change, error) {
	var tdc []change.Change
	if len(pt.Tables) != len(newpt.Tables) {
		return nil, ErrStructurePtChanged
	}
	for i, st := range pt.Tables {
		if len(st.Rows) != len(newpt.Tables[i].Rows) {
			return nil, ErrStructurePtChanged
		}
		for j, strow := range st.Rows {
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

func getSubjectTableNames(document *goquery.Document, ptObject *user.ProgressTable) {
	document.Find(".my-2").Find("div:first-child").Clone().Children().Remove().End().Each(func(nameId int, name *goquery.Selection) {
		ptObject.Tables[nameId].Name = replacedString.ReplaceAllString(name.Text(), " ")
	})
}

func getSubjectTableData(document *goquery.Document, ptObject *user.ProgressTable) {
	filterTrSelection := func(i int, tr *goquery.Selection) bool {
		trLen := tr.Find("td").Length()
		return trLen == 4 || trLen == 2
	}

	document.Find("tbody").Each(func(tbodyId int, tbody *goquery.Selection) {

		trSelection := tbody.Find("tr").FilterFunction(filterTrSelection)

		stLength := trSelection.Length()
		stObject := user.SubjectTable{Name: "", Rows: make([]user.SubjectTableRow, stLength)}

		trSelection.Each(func(trId int, tr *goquery.Selection) {
			strObject := user.SubjectTableRow{}
			tdSelection := tr.Find("td")
			tdSelection.Each(func(tdId int, td *goquery.Selection) {
				tdNew := replacedString.ReplaceAllString(td.Text(), " ")
				switch tdId {
				case 0:
					strObject.Name = tdNew
				case tdSelection.Length() - 1:
					if tdNew == " " {
						strObject.Grades = "отсутствует"
					} else {
						strObject.Grades = tdNew
					}
				}
			})
			stObject.Rows[trId] = strObject
		})
		ptObject.Tables[tbodyId] = stObject
	})
}
