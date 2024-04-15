package telegram

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/ilyadubrovsky/tracking-bars/internal/config"
	"github.com/ilyadubrovsky/tracking-bars/internal/domain"
	ierrors "github.com/ilyadubrovsky/tracking-bars/internal/errors"
	"github.com/ilyadubrovsky/tracking-bars/pkg/bars"
	tele "gopkg.in/telebot.v3"
)

func (s *svc) handleCallback(c tele.Context) error {
	callbackData := strings.Replace(c.Callback().Data, "\f", "", -1)
	if callbackData[:2] != "pt" {
		return s.EditMessageWithOpts(c.Sender().ID, c.Message().ID, config.BotError)
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
	err := s.userSvc.Save(
		context.Background(),
		&domain.User{
			ID: c.Sender().ID,
		},
	)
	if err != nil {
		return s.SendMessageWithOpts(c.Sender().ID, config.StartError)
	}

	return s.SendMessageWithOpts(c.Sender().ID, config.Start)
}

func (s *svc) handleHelpCommand(c tele.Context) error {
	return s.SendMessageWithOpts(c.Sender().ID, config.Help)
}

// TODO пока не придумал че делать если /start не написал пользователь и не попал в таблицу users(
func (s *svc) handleAuthCommand(c tele.Context) error {
	if c.Message().Payload == "" {
		return s.SendMessageWithOpts(c.Sender().ID, config.CredentialsNoEntered)
	}

	userCredentials := strings.Split(c.Message().Payload, " ")

	if len(userCredentials) != 2 {
		return s.SendMessageWithOpts(c.Sender().ID, config.CredentialsFormIgnored)
	}

	username := userCredentials[0]
	password := userCredentials[1]

	if !isValidUserData(username) {
		return s.SendMessageWithOpts(c.Sender().ID, config.CredentialsIncorrectly)
	}

	err := s.barsSvc.Authorization(context.Background(), &domain.BarsCredentials{
		UserID:   c.Sender().ID,
		Username: username,
		Password: []byte(password),
	})
	switch {
	case errors.Is(err, ierrors.ErrWrongGradesPage):
		return s.SendMessageWithOpts(c.Sender().ID, config.GradesPageWrong)
	case errors.Is(err, bars.ErrAuthorizationFailed):
		return s.SendMessageWithOpts(c.Sender().ID, config.CredentialsWrong)
	case errors.Is(err, ierrors.ErrAlreadyAuth):
		return s.SendMessageWithOpts(c.Sender().ID, config.ClientAlreadyAuthorized)
	case err != nil:
		return s.SendMessageWithOpts(c.Sender().ID, config.BotError)
	}

	return s.SendMessageWithOpts(c.Sender().ID, config.SuccessfulAuthorization)
}

func (s *svc) handleLogoutCommand(c tele.Context) error {
	err := s.barsSvc.Logout(context.Background(), c.Sender().ID)
	if err != nil {
		return s.SendMessageWithOpts(c.Sender().ID, config.BotError)
	}

	return s.SendMessageWithOpts(c.Sender().ID, config.SuccessfulLogout)
}

func (s *svc) handleProgressTableCommand(c tele.Context) error {
	//RequestID:  c.Sender().ID,
	//IsCallback: false,

	// TODO get progress table by bars service
	// s.SendMessageWithOpts(c.Sender().ID, a.cfg.Responses.BotError)

	return s.SendMessageWithOpts(c.Sender().ID, "Команда пока не работает.")
}

func (s *svc) handleGithubCommand(c tele.Context) error {
	return s.SendMessageWithOpts(c.Sender().ID, config.Github, tele.ModeMarkdown)
}

func (s *svc) handleText(c tele.Context) error {
	return s.SendMessageWithOpts(c.Sender().ID, config.Default)
}

func (s *svc) handleAdminEchoCommand(c tele.Context) error {
	text := c.Message().Text
	message := strings.Replace(text, "/echo ", "", -1)

	return s.SendMessageWithOpts(c.Sender().ID, message, tele.ModeMarkdown)
}

func (s *svc) handleFixGradesCommand(c tele.Context) error {
	return s.SendMessageWithOpts(c.Sender().ID, config.FixGrades, tele.ModeMarkdown)
}

func (s *svc) handleAdminSendMessageCommand(c tele.Context) error {
	msg := strings.Split(c.Message().Text, " ")
	if len(msg) < 3 {
		return s.SendMessageWithOpts(c.Sender().ID, config.BotError)
	}

	userID, err := strconv.Atoi(msg[1])
	if err != nil {
		return s.SendMessageWithOpts(c.Sender().ID, config.BotError)
	}

	text := strings.Join(msg[2:], " ")
	if err = s.SendMessageWithOpts(int64(userID), text, tele.ModeMarkdown); err != nil {
		return err
	}

	return s.SendMessageWithOpts(c.Sender().ID,
		fmt.Sprintf("Пользователю %d успешно отправлено сообщение:\n%s", userID, text), tele.ModeMarkdown)
}

func isValidUserData(username string) bool {
	var isStringAlphabeticAndBackslash = regexp.MustCompile(`^[a-zA-Z\\]+$`).MatchString
	if !isStringAlphabeticAndBackslash(username) {
		return false
	}
	return true
}
