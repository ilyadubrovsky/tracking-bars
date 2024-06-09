package bars

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"sync"
	"unicode/utf8"

	"github.com/PuerkitoBio/goquery"
	"github.com/ilyadubrovsky/tracking-bars/internal/config"
	"github.com/ilyadubrovsky/tracking-bars/internal/domain"
	ierrors "github.com/ilyadubrovsky/tracking-bars/internal/errors"
	"github.com/ilyadubrovsky/tracking-bars/internal/service"
	"github.com/ilyadubrovsky/tracking-bars/pkg/aes"
	"github.com/ilyadubrovsky/tracking-bars/pkg/bars"
)

// TODO здесь должнен быть пул клиентов, реализация с мьютексом медленная
// нужно сбрасывать клиента через Clear() после использования перед возвращением в пул
type svc struct {
	userSvc service.User
	cfg     config.Bars

	mu         sync.Mutex
	barsClient bars.Client
}

func NewService(
	userSvc service.User,
	cfg config.Bars,
) *svc {
	return &svc{
		userSvc:    userSvc,
		cfg:        cfg,
		barsClient: bars.NewClient(config.BARSRegistrationPageURL),
	}
}

func (s *svc) Authorization(
	ctx context.Context,
	userID int64,
	username string,
	password []byte,
) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	defer s.barsClient.Clear()

	user, err := s.userSvc.User(ctx, userID)
	if err != nil {
		return fmt.Errorf("barsCredentialsRepo.User: %w", err)
	}
	if user != nil && user.BarsCredentials != nil {
		return ierrors.ErrAlreadyAuth
	}

	progressTable, err := s.GetProgressTable(ctx, username, password, s.barsClient)
	if err != nil {
		return fmt.Errorf("svc.GetProgressTable: %w", err)
	}

	encryptedPassword, err := aes.Encrypt([]byte(s.cfg.EncryptionKey), password)
	if err != nil {
		return fmt.Errorf("aes.Encrypt (password): %w", err)
	}

	err = s.userSvc.Save(ctx, &domain.User{
		ID:            userID,
		ProgressTable: progressTable,
		BarsCredentials: &domain.BarsCredentials{
			Username: username,
			Password: encryptedPassword,
		},
	})
	if err != nil {
		return fmt.Errorf("userSvc.Save: %w", err)
	}

	return nil
}

func (s *svc) Logout(ctx context.Context, userID int64) error {
	err := s.userSvc.Delete(ctx, userID)
	if err != nil {
		return fmt.Errorf("userSvc.Delete: %w", err)
	}

	return nil
}

// TODO тут чета с разделением ответственности не то, надо поправить
func (s *svc) GetProgressTable(
	ctx context.Context,
	username string,
	password []byte,
	barsClient bars.Client,
) (*domain.ProgressTable, error) {
	if barsClient == nil {
		barsClient = bars.NewClient(config.BARSRegistrationPageURL)
	}

	err := barsClient.Authorization(ctx, username, string(password))
	if err != nil {
		return nil, fmt.Errorf("barsClient.Authorization: %w", err)
	}

	document, err := getGradesPageDocument(ctx, barsClient)
	if err != nil {
		return nil, fmt.Errorf("getGradesPageDocument: %w", err)
	}

	progressTable, err := extractProgressTable(document)
	if err != nil {
		return nil, fmt.Errorf("extractProgressTable: %w", err)
	}

	return progressTable, nil
}

func getGradesPageDocument(
	ctx context.Context,
	barsClient bars.Client,
) (*goquery.Document, error) {
	response, err := barsClient.MakeRequest(ctx, http.MethodGet, config.BARSGradesPageURL, nil)
	if err != nil {
		return nil, fmt.Errorf("barsClient.MakeRequest: %w", err)
	}

	document, err := goquery.NewDocumentFromReader(response.Body)
	if err != nil {
		return nil, fmt.Errorf("goquery.NewDocumentFromReader: %w", err)
	}

	if !isGradePage(document) {
		return nil, ierrors.ErrWrongGradesPage
	}

	return document, nil
}

func isGradePage(document *goquery.Document) bool {
	return document.Find("div#div-Student_SemesterSheet__Mark").Length() != 0
}

func extractProgressTable(document *goquery.Document) (*domain.ProgressTable, error) {
	if !isGradePage(document) {
		return nil, ierrors.ErrWrongGradesPage
	}

	disciplinesCount := document.Find("tbody").Length()
	progressTable := &domain.ProgressTable{
		Disciplines: make([]domain.Discipline, 0, disciplinesCount),
	}

	if err := extractDisciplinesData(document, progressTable); err != nil {
		return nil, fmt.Errorf("extractDisciplinesData: %w", err)
	}

	if err := extractDisciplineNames(document, progressTable); err != nil {
		return nil, fmt.Errorf("extractDisciplineNames: %w", err)
	}

	if err := validateProgressTable(progressTable); err != nil {
		return nil, fmt.Errorf("validateProgressTable: %w", err)
	}

	return progressTable, nil
}

func extractDisciplineNames(document *goquery.Document, pt *domain.ProgressTable) error {
	var err error
	document.Find(".my-2").
		Find("div:first-child").
		Clone().
		Children().
		Remove().
		End().
		EachWithBreak(func(nameId int, name *goquery.Selection) bool {
			processedName := regexp.MustCompile("\\s+").ReplaceAllString(name.Text(), " ")
			processedName = strings.TrimSuffix(processedName, " ")
			if strings.HasPrefix(processedName, " ") {
				processedName = strings.Replace(processedName, " ", "", 1)
			}
			if isEmptyData(processedName) {
				err = fmt.Errorf("part of received data is empty. nameID: %d", nameId)
				return false
			}
			pt.Disciplines[nameId].Name = processedName
			return true
		})

	return err
}

func extractDisciplinesData(
	document *goquery.Document,
	progressTable *domain.ProgressTable,
) error {
	var (
		err        error
		isContinue = true
	)
	filterTrSelection := func(i int, tr *goquery.Selection) bool {
		trLen := tr.Find("td").Length()
		return trLen == 4 || trLen == 2
	}

	document.Find("tbody").EachWithBreak(func(tbodyId int, tbody *goquery.Selection) bool {
		trSelection := tbody.Find("tr").FilterFunction(filterTrSelection)

		controlEventsCount := trSelection.Length()
		discipline := domain.Discipline{
			Name:          "",
			ControlEvents: make([]domain.ControlEvent, 0, controlEventsCount),
		}

		trSelection.EachWithBreak(func(trId int, tr *goquery.Selection) bool {
			controlEvent := domain.ControlEvent{}
			tdSelection := tr.Find("td")
			tdSelection.EachWithBreak(func(tdId int, td *goquery.Selection) bool {
				processedData := regexp.MustCompile("\\s+").ReplaceAllString(td.Text(), " ")
				processedData = strings.TrimSuffix(processedData, " ")

				switch tdId {
				case 0:
					if isEmptyData(processedData) {
						err = fmt.Errorf("part of received data is empty. "+
							"tdId: %d trId: %d tbodyId: %d", tdId, trId, tbodyId)
						isContinue = false
					}
					if strings.HasPrefix(processedData, " ") {
						processedData = strings.Replace(processedData, " ", "", 1)
					}
					controlEvent.Name = processedData
				case tdSelection.Length() - 1:
					if isEmptyData(processedData) {
						processedData = "отсутствует"
					} else if strings.HasPrefix(processedData, " ") {
						processedData = strings.Replace(processedData, " ", "", 1)
					}
					controlEvent.Grade = processedData
				}

				return isContinue
			})
			discipline.ControlEvents = append(discipline.ControlEvents, controlEvent)

			return isContinue
		})
		progressTable.Disciplines = append(progressTable.Disciplines, discipline)

		return isContinue
	})

	return err
}

func isEmptyData(data string) bool {
	return data == "" || data == " "
}

func validateProgressTable(pt *domain.ProgressTable) error {
	if !utf8.ValidString(pt.String()) {
		return errors.New("progress table contains not utf8 characters")
	}

	return nil
}
