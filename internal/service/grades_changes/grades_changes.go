package grades_changes

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/PuerkitoBio/goquery"
	"github.com/ilyadubrovsky/tracking-bars/internal/config"
	"github.com/ilyadubrovsky/tracking-bars/internal/domain"
	ierrors "github.com/ilyadubrovsky/tracking-bars/internal/errors"
	"github.com/ilyadubrovsky/tracking-bars/internal/service"
	"github.com/ilyadubrovsky/tracking-bars/pkg/aes"
	"github.com/ilyadubrovsky/tracking-bars/pkg/bars"
	"gopkg.in/telebot.v3"
)

type svc struct {
	progressTableSvc  service.ProgressTable
	barsCredentialSvc service.BarsCredential
	telegramSvc       service.Telegram
	cfg               config.Bars
	stopFunc          func()
}

func NewService(
	progressTableSvc service.ProgressTable,
	barsCredentialSvc service.BarsCredential,
	telegramSvc service.Telegram,
	cfg config.Bars,
) *svc {
	return &svc{
		progressTableSvc:  progressTableSvc,
		barsCredentialSvc: barsCredentialSvc,
		telegramSvc:       telegramSvc,
		cfg:               cfg,
	}
}

func (s *svc) Start() {
	ctx, cancel := context.WithCancel(context.Background())
	s.stopFunc = cancel

	jobChan := make(chan *domain.BarsCredentials)
	go func() {
		for {
			select {
			case <-time.After(s.cfg.CronDelay):
				s.sendActualCredentials(ctx, jobChan)
			case <-ctx.Done():
				close(jobChan)
				return
			}
		}
	}()
	for i := 0; i < s.cfg.CronWorkerPoolSize; i++ {
		go s.checkChangesWorker(jobChan)
	}
}

func (s *svc) sendActualCredentials(
	ctx context.Context,
	credentialsChan chan<- *domain.BarsCredentials,
) {
	barsCredentials, err := s.barsCredentialSvc.GetAllAuthorized(ctx)
	if err != nil {
		// TODO logging
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
			// TODO logging with userID
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

	document, err := getGradesPageDocument(ctx, barsClient, credentials.Username, decryptedPassword)
	if errors.Is(err, bars.ErrAuthorizationFailed) {
		sendMsgErr := s.telegramSvc.SendMessageWithOpts(credentials.UserID, config.CredentialsExpired)
		if sendMsgErr != nil {
			return fmt.Errorf("telegramSvc.SendMessageWithOpts(credentialsExpired): %w", err)
		}

		deleteErr := s.barsCredentialSvc.Logout(ctx, credentials.UserID)
		if deleteErr != nil {
			return fmt.Errorf("barsCredentialSvc.Logout(authFailed): %w", err)
		}

		return nil
	}
	if errors.Is(err, ierrors.ErrWrongGradesPage) {
		sendMsgErr := s.telegramSvc.SendMessageWithOpts(credentials.UserID, config.GradesPageWrong)
		if sendMsgErr != nil {
			return fmt.Errorf("telegramSvc.SendMessageWithOpts(gradesPageWrong): %w", err)
		}

		deleteErr := s.barsCredentialSvc.Logout(ctx, credentials.UserID)
		if deleteErr != nil {
			return fmt.Errorf("barsCredentialSvc.Delete(wrongGradesPage): %w", err)
		}

		return nil
	}
	if err != nil {
		return fmt.Errorf("getGradesPageDocument: %w", err)
	}

	progressTable, err := extractProgressTable(document)
	if err != nil {
		return fmt.Errorf("extractProgressTable: %w", err)
	}
	progressTable.UserID = credentials.UserID

	oldProgressTable, err := s.progressTableSvc.GetProgressTable(ctx, credentials.UserID)
	if err != nil {
		return fmt.Errorf("progressTableSvc.GetProgressTable: %w", err)
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
				// TODO logging with PARAMS
				// TODO ретраи можно сделать, чтобы не терять изменения
				// или прихранивать их где-то
				continue
			}
		}
	}

	if err = s.progressTableSvc.Save(ctx, progressTable); err != nil {
		return fmt.Errorf("progressTableSvc.Save: %w", err)
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

func getGradesPageDocument(
	ctx context.Context,
	barsClient bars.Client,
	username string,
	password string,
) (*goquery.Document, error) {
	// TODO здесь может не всегда проходить авторизация сразу
	// т.к. сервер барса может падать, нужно предусмотреть систему ретраев перед удалением кредов
	err := barsClient.Authorization(ctx, username, password)
	if err != nil {
		return nil, fmt.Errorf("barsClient.Authorization: %w", err)
	}

	// TODO аналогично тут ретраи хотелось бы сделать
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

func isEmptyData(data string) bool {
	return data == "" || data == " "
}

func validateProgressTable(pt *domain.ProgressTable) error {
	if !utf8.ValidString(pt.String()) {
		return errors.New("progress table contains not utf8 characters")
	}

	return nil
}
