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

func (s *svc) handleCallback(c tele.Context) error {
	callbackData := strings.Replace(c.Callback().Data, "\f", "", -1)
	if callbackData[:2] != "pt" {
		return s.EditMessageWithOpts(c.Sender().ID, c.Message().ID, answers.BotError)
	}

	// TODO bars get grades request
	//RequestID:    c.Sender().ID,
	//IsCallback:   true,
	//CallbackData: callbackData[2:],
	//MessageID:    c.Message().ID,

	// if err != nil
	//return s.EditMessageWithOpts(c.Sender().ID, c.Message().ID, answers.BotError)

	return nil
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

// TODO пока не придумал че делать если /start не написал пользователь и не попал в таблицу users(
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

	err := s.barsSvc.Authorization(ctx, &domain.BarsCredentials{
		UserID:   c.Sender().ID,
		Username: username,
		Password: []byte(password),
	})
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
	//RequestID:  c.Sender().ID,
	//IsCallback: false,

	// TODO get progress table by bars service
	// s.SendMessageWithOpts(c.Sender().ID, a.cfg.Responses.BotError)

	return s.SendMessageWithOpts(c.Sender().ID, "Команда пока не работает.")
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

	return s.SendMessageWithOpts(c.Sender().ID, input[1], tele.ModeMarkdownV2)
}

func (s *svc) handleAdminSendMessageAllCommand(c tele.Context) error {
	input := strings.SplitN(c.Text(), " ", 2)
	if len(input) <= 1 {
		return s.SendMessageWithOpts(c.Sender().ID, answers.AdminInvalidArgument)
	}

	logger := log.With().Int64("admin", c.Sender().ID).Logger()

	users, err := s.userSvc.GetAll(context.Background())
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
