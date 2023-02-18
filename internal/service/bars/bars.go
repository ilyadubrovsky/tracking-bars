package bars

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/PuerkitoBio/goquery"
	tele "gopkg.in/telebot.v3"
	"io"
	"net/http"
	"os"
	"regexp"
	"strings"
	"sync"
	"time"
	"tracking-barsv1.1/internal/config"
	"tracking-barsv1.1/internal/entity/change"
	"tracking-barsv1.1/internal/entity/user"
	"tracking-barsv1.1/pkg/client/bars"
	"tracking-barsv1.1/pkg/logging"
	"tracking-barsv1.1/pkg/utils/aes"
)

var (
	wg                    sync.WaitGroup
	ErrStructurePtChanged = errors.New("the structure of the progress table has changed")
	replacedString        = regexp.MustCompile("\\s+")
)

type Client interface {
	GetPage(ctx context.Context, method string, url string, body io.Reader) (*http.Response, error)
	Authorization(ctx context.Context, username, password string) error
}

type Service struct {
	bot            *tele.Bot
	usersStorage   user.Repository
	changesStorage change.Repository
	cfg            *config.Config
	logger         *logging.Logger
}

func (s *Service) Start() {
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
			go s.CheckChanges(context.Background(), usr)
		}

		wg.Wait()

		duration := time.Since(startTime)
		s.logger.Tracef("BARS service: parsing completed in: %v", duration)
	}
}

func (s *Service) GetProgressTable(ctx context.Context, client Client) (user.ProgressTable, error) {
	response, err := client.GetPage(ctx, http.MethodGet, s.cfg.Bars.URLs.MainPageURL, nil)
	if err != nil {
		s.logger.Errorf("failed to get a page due error: %v", err)
		return user.ProgressTable{}, err
	}

	document, err := goquery.NewDocumentFromReader(response.Body)
	if err != nil {
		s.logger.Errorf("failed to create a document from the response body due error: %v", err)
		return user.ProgressTable{}, err
	}

	ptLength := document.Find("tbody").Length()
	ptObject := user.ProgressTable{Tables: make([]user.SubjectTable, ptLength)}

	retrieveSubjectTablesData(document, &ptObject)

	retrieveSubjectTableNames(document, &ptObject)

	if err = ptObject.ValidateData(); err == user.ErrIncorrectData {
		return user.ProgressTable{}, err
	}

	return ptObject, nil
}

// CheckChanges TODO fix leaked changes (attempt to resend)
func (s *Service) CheckChanges(ctx context.Context, usr user.User) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	defer wg.Done()

	decryptedPassword, err := aes.DecryptAES([]byte(os.Getenv("ENCRYPTION_KEY")), usr.Password)
	if err != nil {
		s.logger.Errorf("failed to decrypt a password (DecryptAES) due error: %s", err)
		return
	}

	client, err := s.authorizeUser(ctx, usr.Username, decryptedPassword)
	if errors.Is(err, bars.ErrNoAuth) {
		usrDTO := user.UpdateUserDTO{
			ID:            usr.ID,
			Username:      usr.Username,
			Password:      []byte{},
			ProgressTable: "{}",
			Deleted:       true,
		}
		if err2 := s.usersStorage.Update(ctx, usrDTO); err2 != nil {
			s.logger.Errorf("failed to Update due error: %s", err2)
			s.logger.Errorf("UserID: %d, Username: %s, Password: %x\n ProgressTable: %s\n", usrDTO.ID, usrDTO.Username, usrDTO.Password, usrDTO.ProgressTable)
			return
		}
		_, err2 := s.bot.Send(tele.ChatID(usr.ID), s.cfg.Responses.Bars.ExpiredData)
		if err2 != nil {
			s.logger.Tracef("UserID: %d\n Message: %s", usr.ID, s.cfg.Responses.Bars.ExpiredData)
			s.logger.Errorf("failed to send a message due error: %s", err2)
			return
		}
		return
	} else if err != nil {
		s.logger.Errorf("failed to Authorization method due error: %s", err)
		s.logger.Debugf("UserID: %d\n Username:%s\n Password:%x\n", usr.ID, usr.Username, usr.Password)
		return
	}

	progressTableParser, err := s.GetProgressTable(ctx, client)
	if err != nil {
		s.logger.Errorf("failed to GetProgressTable method due error: %s", err)
		return
	}

	tableDataChanges, err := compareProgressTables(usr.ProgressTable, progressTableParser)
	if err == ErrStructurePtChanged {
		if err2 := s.updateUserInDB(ctx, &usr, &progressTableParser); err2 != nil {
			s.logger.Errorf("failed to update user due error: %s", err2)
		}
		return
	} else if err != nil {
		s.logger.Errorf("failed to compare method due error: %s", err)
		return
	}

	if len(tableDataChanges) == 0 {
		return
	}

	if err = s.updateUserInDB(ctx, &usr, &progressTableParser); err != nil {
		s.logger.Errorf("%v", err)
		return
	}

	for _, tdc := range tableDataChanges {
		if err = s.createChangeInDB(ctx, usr.ID, &tdc); err != nil {
			s.logger.Errorf("%v", err)
			return
		}

		msg := fmt.Sprintf("*Получено изменение:*\n\n%s\n\n", tdc.String())
		_, err = s.bot.Send(tele.ChatID(usr.ID), msg, "Markdown")
		if err != nil {
			s.logger.Errorf("failed to send a message with the change to the user due error: %s", err)
			continue
		}
	}
}

func (s *Service) Authorization(ctx context.Context, userID int64, username, password string) (bool, error) {
	usr, err := s.usersStorage.FindOne(ctx, userID)
	if err != nil {
		s.logger.Errorf("failed to FindOne due error: %v", err)
		return false, err
	}

	client, err := s.authorizeUser(ctx, username, password)
	if errors.Is(err, bars.ErrNoAuth) {
		return false, nil
	} else if err != nil {
		s.logger.Tracef("UserID: %d, username: %s", userID, username)
		s.logger.Errorf("failed to authorizeUser method due error: %v", err)
		s.logger.Debugf("Username:%s\n Password: %x\n", username, password)
		return false, err
	}

	encryptedPassword, err := aes.EncryptAES([]byte(os.Getenv("ENCRYPTION_KEY")), []byte(password))
	if err != nil {
		s.logger.Tracef("UserID: %d, username: %s", userID, username)
		s.logger.Errorf("failed to encrypt a password due error: %v", err)
		return false, err
	}

	pt, err := s.GetProgressTable(ctx, client)
	if err != nil {
		s.logger.Tracef("UserID: %d, username: %s", userID, username)
		s.logger.Errorf("failed to GetProgressTable method due error: %v", err)
		return false, err
	}

	ptBytes, err := json.Marshal(pt)
	if err != nil {
		s.logger.Errorf("failed to marshal the structure received from the parser due error: %v", err)
		return false, err
	}

	if usr.Deleted == true {
		usrDTO := user.UpdateUserDTO{
			ID:            userID,
			Username:      username,
			Password:      encryptedPassword,
			ProgressTable: string(ptBytes),
			Deleted:       false,
		}
		err = s.usersStorage.Update(ctx, usrDTO)
	} else {
		usrDTO := user.CreateUserDTO{
			ID:            userID,
			Username:      username,
			Password:      encryptedPassword,
			ProgressTable: string(ptBytes),
		}
		err = s.usersStorage.Create(ctx, usrDTO)
	}

	if err != nil {
		s.logger.Errorf("failed to Create/Update due error: %v", err)
		return false, err
	}

	return true, nil
}

func (s *Service) authorizeUser(ctx context.Context, username, password string) (Client, error) {
	client := bars.NewClient(s.cfg.Bars.URLs.RegistrationURL)

	if err := client.Authorization(ctx, username, password); err == bars.ErrNoAuth {
		return &bars.Client{}, err
	} else if err != nil {
		s.logger.Tracef("username: %s", username)
		s.logger.Errorf("failed to Authorization client method due error: %s", err)
		return &bars.Client{}, err
	}

	return client, nil
}

func (s *Service) updateUserInDB(ctx context.Context, usr *user.User, pt *user.ProgressTable) error {
	ptBytes, err := json.Marshal(pt)
	if err != nil {
		return fmt.Errorf("failed to marshal the structure received from the parser due error: %v", err)
	}

	usrDTO := user.UpdateUserDTO{
		ID:            usr.ID,
		Username:      usr.Username,
		Password:      usr.Password,
		ProgressTable: string(ptBytes),
		Deleted:       false,
	}
	if err = s.usersStorage.Update(ctx, usrDTO); err != nil {
		return fmt.Errorf("failed to Update due error: %v", err)
	}

	return nil
}

func (s *Service) createChangeInDB(ctx context.Context, userID int64, tdc *change.Change) error {
	if err := s.changesStorage.Create(ctx, change.CreateChangeDTO{
		UserID:       userID,
		Subject:      tdc.Subject,
		ControlEvent: tdc.ControlEvent,
		OldGrade:     tdc.OldGrade,
		NewGrade:     tdc.NewGrade,
	}); err != nil {
		return fmt.Errorf("failed to Create due error: %v", err)
	}

	return nil
}

func NewService(cfg *config.Config, usersStorage user.Repository, changesStorage change.Repository, logger *logging.Logger, bot *tele.Bot) *Service {
	return &Service{
		usersStorage:   usersStorage,
		changesStorage: changesStorage,
		cfg:            cfg,
		logger:         logger,
		bot:            bot,
	}
}

func compareProgressTables(pt user.ProgressTable, newpt user.ProgressTable) ([]change.Change, error) {
	var tdc []change.Change
	if len(pt.Tables) != len(newpt.Tables) {
		return nil, ErrStructurePtChanged
	}
	for i, st := range pt.Tables {
		if len(st.Rows) != len(newpt.Tables[i].Rows) {
			return nil, ErrStructurePtChanged
		}
		for j, strow := range st.Rows {
			if strow.Name != newpt.Tables[i].Rows[j].Name {
				return nil, ErrStructurePtChanged
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

func retrieveSubjectTableNames(document *goquery.Document, ptObject *user.ProgressTable) {
	document.Find(".my-2").Find("div:first-child").Clone().Children().Remove().End().Each(func(nameId int, name *goquery.Selection) {
		processedString := replacedString.ReplaceAllString(name.Text(), " ")
		if strings.HasPrefix(processedString, " ") {
			ptObject.Tables[nameId].Name = strings.Replace(processedString, " ", "", 1)
		}
	})
}

func retrieveSubjectTablesData(document *goquery.Document, ptObject *user.ProgressTable) {
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
				processedString := replacedString.ReplaceAllString(td.Text(), " ")
				switch tdId {
				case 0:
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
			})
			stObject.Rows[trId] = strObject
		})
		ptObject.Tables[tbodyId] = stObject
	})
}
