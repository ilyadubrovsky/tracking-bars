package telegram

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/ilyadubrovsky/tracking-bars/internal/config/answers"
	"github.com/ilyadubrovsky/tracking-bars/internal/domain"
	ierrors "github.com/ilyadubrovsky/tracking-bars/internal/errors"
	"github.com/ilyadubrovsky/tracking-bars/pkg/bars"
	"github.com/rs/zerolog/log"
	tele "gopkg.in/telebot.v3"
)

const (
	callbackProgressTable                        = "pt"
	callbackProgressTableBackOption              = "back"
	callbackProgressTableDisciplineDetailsOption = "show"
)

func (s *svc) handleOnCallback(c tele.Context) error {
	callbackData := strings.Replace(c.Callback().Data, "\f", "", -1)
	if strings.HasPrefix(callbackData, callbackProgressTable) {
		return s.handleProgressTableCallback(c)
	}

	return s.EditMessageWithOpts(c.Sender().ID, c.Message().ID, answers.BotError)
}

func (s *svc) handleProgressTableCallback(c tele.Context) error {
	logger := log.With().Fields(extractTelebotFields(c)).Logger()
	ctx := logger.WithContext(context.Background())

	callbackData := strings.Replace(c.Callback().Data, "\f", "", -1)
	usefulData := strings.TrimPrefix(callbackData, callbackProgressTable)
	if len(usefulData) == 0 {
		return s.EditMessageWithOpts(c.Sender().ID, c.Message().ID, answers.BotError)
	}

	user, err := s.userSvc.User(ctx, c.Sender().ID)
	if err != nil {
		logger.Error().Msgf("handleProgressTableCallback: %v", err.Error())
		return s.EditMessageWithOpts(c.Sender().ID, c.Message().ID, answers.BotError)
	}

	progressTable := user.ProgressTable
	if progressTable == nil || len(progressTable.Disciplines) == 0 {
		return s.EditMessageWithOpts(c.Sender().ID, c.Message().ID, answers.GradesPageUnavailable)
	}

	isHideControlEventsName := true
	if usefulData == callbackProgressTableBackOption {
		return s.EditMessageWithOpts(
			c.Sender().ID,
			c.Message().ID,
			generateDisciplineListMessage(progressTable.Disciplines),
			tele.ModeMarkdown,
			s.generateDisciplineListMarkup(progressTable),
		)
	}

	if strings.HasPrefix(usefulData, callbackProgressTableDisciplineDetailsOption) {
		isHideControlEventsName = false
		usefulData = strings.TrimPrefix(usefulData, callbackProgressTableDisciplineDetailsOption)
	}

	disciplineNumber, err := strconv.Atoi(usefulData)
	if err != nil {
		logger.Error().Msgf("handleProgressTableCallback: %v", err.Error())
		return s.EditMessageWithOpts(c.Sender().ID, c.Message().ID, answers.BotError)
	}
	if disciplineNumber > len(user.ProgressTable.Disciplines) || disciplineNumber <= 0 {
		return s.EditMessageWithOpts(c.Sender().ID, c.Message().ID, answers.GradesPageUnavailable)
	}

	return s.EditMessageWithOpts(
		c.Sender().ID,
		c.Message().ID,
		generateDisciplineInfoMessage(
			user.ProgressTable.Disciplines[disciplineNumber-1],
			isHideControlEventsName,
		),
		tele.ModeMarkdown,
		s.generateDisciplineMarkup(disciplineNumber, isHideControlEventsName),
	)
}

func (s *svc) handleStartCommand(c tele.Context) error {
	logger := log.With().Fields(extractTelebotFields(c)).Logger()
	ctx := logger.WithContext(context.Background())
	err := s.userSvc.Save(ctx, &domain.User{
		ID: c.Sender().ID,
	})
	if err != nil {
		err = fmt.Errorf("userSvc.Save: %w", err)
		logger.Error().Msgf("handleStartCommand: %v", err.Error())
		return s.SendMessageWithOpts(c.Sender().ID, answers.BotError)
	}

	return s.SendMessageWithOpts(c.Sender().ID, answers.Start)
}

func (s *svc) handleHelpCommand(c tele.Context) error {
	return s.SendMessageWithOpts(c.Sender().ID, answers.Help)
}

func (s *svc) handleAuthCommand(c tele.Context) error {
	logger := log.With().Fields(extractTelebotFields(c)).Logger()
	ctx := logger.WithContext(context.Background())

	if c.Message().Payload == "" {
		return s.SendMessageWithOpts(c.Sender().ID, answers.CredentialsNoEntered)
	}

	userCredentials := strings.Split(c.Message().Payload, " ")

	if len(userCredentials) != 2 {
		return s.SendMessageWithOpts(c.Sender().ID, answers.CredentialsFormIgnored)
	}

	username := userCredentials[0]
	password := userCredentials[1]

	if !isValidUserData(username) {
		return s.SendMessageWithOpts(c.Sender().ID, answers.CredentialsIncorrectly)
	}

	err := s.barsSvc.Authorization(ctx, c.Sender().ID, username, []byte(password))
	switch {
	case errors.Is(err, ierrors.ErrWrongGradesPage):
		return s.SendMessageWithOpts(c.Sender().ID, answers.GradesPageWrong)
	case errors.Is(err, bars.ErrAuthorizationFailed):
		return s.SendMessageWithOpts(c.Sender().ID, answers.CredentialsWrong)
	case errors.Is(err, ierrors.ErrAlreadyAuth):
		return s.SendMessageWithOpts(c.Sender().ID, answers.ClientAlreadyAuthorized)
	case err != nil:
		err = fmt.Errorf("barsSvc.Authorization: %w", err)
		logger.Error().Msgf("handleAuthCommand: %v", err.Error())
		return s.SendMessageWithOpts(c.Sender().ID, answers.BotError)
	}

	return s.SendMessageWithOpts(c.Sender().ID, answers.SuccessfulAuthorization)
}

func (s *svc) handleLogoutCommand(c tele.Context) error {
	logger := log.With().Fields(extractTelebotFields(c)).Logger()
	ctx := logger.WithContext(context.Background())

	err := s.barsSvc.Logout(ctx, c.Sender().ID)
	if err != nil {
		err = fmt.Errorf("barsSvc.Logout: %w", err)
		logger.Error().Msgf("handleLogoutCommand: %v", err.Error())
		return s.SendMessageWithOpts(c.Sender().ID, answers.BotError)
	}

	return s.SendMessageWithOpts(c.Sender().ID, answers.SuccessfulLogout)
}

func (s *svc) handleProgressTableCommand(c tele.Context) error {
	logger := log.With().Fields(extractTelebotFields(c)).Logger()
	ctx := logger.WithContext(context.Background())

	user, err := s.userSvc.User(ctx, c.Sender().ID)
	if err != nil {
		logger.Error().Msgf(
			"handleProgressTableCommand: %v",
			fmt.Errorf("progressTableSvc.User: %w", err).Error(),
		)
		return s.SendMessageWithOpts(c.Sender().ID, answers.BotError)
	}
	progressTable := user.ProgressTable
	if progressTable == nil || len(progressTable.Disciplines) == 0 {
		return s.SendMessageWithOpts(c.Sender().ID, answers.GradesPageUnavailable)
	}

	return s.SendMessageWithOpts(
		c.Sender().ID,
		generateDisciplineListMessage(progressTable.Disciplines),
		tele.ModeMarkdown,
		s.generateDisciplineListMarkup(progressTable),
	)
}

func (s *svc) handleGithubCommand(c tele.Context) error {
	return s.SendMessageWithOpts(c.Sender().ID, answers.Github, tele.ModeMarkdown)
}

func (s *svc) handleText(c tele.Context) error {
	return s.SendMessageWithOpts(c.Sender().ID, answers.Default)
}

func (s *svc) handleAdminEchoCommand(c tele.Context) error {
	input := strings.SplitN(c.Text(), " ", 2)
	if len(input) <= 1 {
		return s.SendMessageWithOpts(c.Sender().ID, answers.AdminInvalidArgument)
	}

	return s.SendMessageWithOpts(c.Sender().ID, input[1], tele.ModeMarkdown)
}

func (s *svc) handleAdminSendMessageAllCommand(c tele.Context) error {
	input := strings.SplitN(c.Text(), " ", 2)
	if len(input) <= 1 {
		return s.SendMessageWithOpts(c.Sender().ID, answers.AdminInvalidArgument)
	}

	logger := log.With().Int64("admin", c.Sender().ID).Logger()

	users, err := s.userSvc.Users(context.Background())
	if err != nil {
		return s.SendMessageWithOpts(c.Sender().ID, answers.BotError)
	}

	errCounter := 0
	for _, user := range users {
		sendErr := s.SendMessageWithOpts(user.ID, input[1])
		if sendErr != nil {
			errCounter++
			logger.Error().Int64("receiver", user.ID).Msg("failed to send message")
		}
	}

	return s.SendMessageWithOpts(
		c.Sender().ID,
		fmt.Sprintf("Разослано сообщение (успешно: %d, ошибок: %d)\n%s",
			len(users)-errCounter, errCounter, input[1]),
		tele.ModeMarkdown,
	)
}

func (s *svc) handleAdminSendMessageAuthCommand(c tele.Context) error {
	input := strings.SplitN(c.Text(), " ", 2)
	if len(input) <= 1 {
		return s.SendMessageWithOpts(c.Sender().ID, answers.AdminInvalidArgument)
	}

	logger := log.With().Int64("admin", c.Sender().ID).Logger()

	users, err := s.userSvc.Users(context.Background())
	if err != nil {
		return s.SendMessageWithOpts(c.Sender().ID, answers.BotError)
	}

	errCounter := 0
	for _, user := range users {
		sendErr := s.SendMessageWithOpts(user.ID, input[1])
		if sendErr != nil {
			errCounter++
			logger.
				Error().
				Int64("receiver", user.ID).
				Msg("failed to send message")
		}
	}

	return s.SendMessageWithOpts(
		c.Sender().ID,
		fmt.Sprintf("Разослано сообщение (успешно: %d, ошибок: %d)\n%s",
			len(users)-errCounter, errCounter, input[1]),
		tele.ModeMarkdown,
	)
}

func (s *svc) handleAdminSendMessageCommand(c tele.Context) error {
	input := strings.SplitN(c.Text(), " ", 3)
	if len(input) <= 2 {
		return s.SendMessageWithOpts(c.Sender().ID, answers.AdminInvalidArgument)
	}

	userID, err := strconv.Atoi(input[1])
	if err != nil {
		return s.SendMessageWithOpts(c.Sender().ID, answers.AdminInvalidArgument)
	}

	err = s.SendMessageWithOpts(int64(userID), input[2], tele.ModeMarkdown)
	if err != nil {
		return s.SendMessageWithOpts(c.Sender().ID, answers.BotError)
	}

	return s.SendMessageWithOpts(c.Sender().ID,
		fmt.Sprintf("Пользователю %d успешно отправлено сообщение:\n%s",
			userID, input[2]), tele.ModeMarkdown)
}

// TODO
/*
func (s *svc) handleAdminCountAuthorizedCommand(c tele.Context) error {
	count, err := s.barsCredentialsRepo.Count(context.Background())
	if err != nil {
		return s.SendMessageWithOpts(c.Sender().ID, answers.BotError)
	}

	return s.SendMessageWithOpts(
		c.Sender().ID,
		fmt.Sprintf("Количество авторизованных: %d", count),
	)
}
*/

func (s *svc) handleFixGradesCommand(c tele.Context) error {
	return s.SendMessageWithOpts(c.Sender().ID, answers.FixGrades, tele.ModeMarkdown)
}

func isValidUserData(username string) bool {
	var isStringAlphabeticAndBackslash = regexp.MustCompile(`^[a-zA-Z\\]+$`).MatchString
	if !isStringAlphabeticAndBackslash(username) {
		return false
	}
	return true
}

func extractTelebotFields(c tele.Context) map[string]interface{} {
	return map[string]interface{}{
		"sender":   c.Sender().ID,
		"username": c.Sender().Username,
	}
}

func generateDisciplineListMessage(disciplines []domain.Discipline) string {
	var b strings.Builder

	for i, discipline := range disciplines {
		b.WriteString(fmt.Sprintf("*%d:* %s\n\n", i+1, discipline.Name))
	}

	b.WriteString("Для просмотра оценок по определённому предмету, воспользуйтесь кнопочным меню.")

	return b.String()
}

func generateDisciplineInfoMessage(
	discipline domain.Discipline,
	isHideControlEventNames bool,
) string {
	var b strings.Builder

	b.WriteString(fmt.Sprintf("*Название дисциплины:*\n%s\n\n", discipline.Name))

	for i, ce := range discipline.ControlEvents {
		name := ce.Name
		if isHideControlEventNames &&
			(!(strings.HasPrefix(ce.Name, "Балл текущего контроля") ||
				strings.HasPrefix(ce.Name, "Итоговая оценка:") ||
				strings.HasPrefix(ce.Name, "Промежуточная аттестация"))) {
			name = fmt.Sprintf("КМ-%d", i+1)
		}
		b.WriteString(fmt.Sprintf("%s\n*Оценка:* %s\n\n", name, ce.Grade))
	}

	return b.String()
}

const buttonsCountInRowDisciplineList = 5

func (s *svc) generateDisciplineListMarkup(progressTable *domain.ProgressTable) *tele.ReplyMarkup {
	markup := s.bot.NewMarkup()

	rowsCount, buttonsCountInLastRow := 0, 0
	disciplinesCount := len(progressTable.Disciplines)
	if disciplinesCount >= buttonsCountInRowDisciplineList {
		remainder := disciplinesCount % buttonsCountInRowDisciplineList
		if remainder == 0 {
			rowsCount = disciplinesCount / buttonsCountInRowDisciplineList
			buttonsCountInLastRow = buttonsCountInRowDisciplineList
		} else {
			rowsCount = disciplinesCount/buttonsCountInRowDisciplineList + 1
			buttonsCountInLastRow = remainder
		}
	} else if disciplinesCount > 0 && disciplinesCount < buttonsCountInRowDisciplineList {
		rowsCount = 1
		buttonsCountInLastRow = disciplinesCount
	} else {
		markup.Inline()
		return markup
	}
	rows := make([]tele.Row, 0, rowsCount)

	for i := 0; i < rowsCount-1; i++ {
		row := make([]tele.Btn, 0, buttonsCountInRowDisciplineList)
		for j := 0; j < buttonsCountInRowDisciplineList; j++ {
			disciplineNumber := i*buttonsCountInRowDisciplineList + j + 1
			button := markup.Data(
				strconv.Itoa(disciplineNumber),
				fmt.Sprintf("pt%d", disciplineNumber),
			)
			row = append(row, button)
		}
		rows = append(rows, row)
	}

	row := make([]tele.Btn, 0, buttonsCountInLastRow)
	for j := 0; j < buttonsCountInLastRow; j++ {
		disciplineNumber := (rowsCount-1)*buttonsCountInRowDisciplineList + j + 1
		button := markup.Data(
			strconv.Itoa(disciplineNumber),
			fmt.Sprintf("pt%d", disciplineNumber),
		)
		row = append(row, button)
	}
	rows = append(rows, row)

	markup.Inline(rows...)

	return markup
}

func (s *svc) generateDisciplineMarkup(
	disciplineNumber int,
	isHideControlEventNames bool,
) *tele.ReplyMarkup {
	markup := s.bot.NewMarkup()

	backButton := markup.Data(
		"←",
		fmt.Sprintf("%s%s", callbackProgressTable, callbackProgressTableBackOption),
	)
	showOrHideButton := tele.Btn{}
	if isHideControlEventNames {
		// если isHide, значит скрываем детали и должны показать кнопку show
		showOrHideButton = markup.Data(
			"↓",
			fmt.Sprintf(
				"%s%s%d",
				callbackProgressTable,
				callbackProgressTableDisciplineDetailsOption,
				disciplineNumber,
			),
		)
	} else {
		showOrHideButton = markup.Data(
			"↑",
			fmt.Sprintf("%s%d", callbackProgressTable, disciplineNumber),
		)
	}

	markup.Inline([]tele.Btn{backButton, showOrHideButton})

	return markup
}
