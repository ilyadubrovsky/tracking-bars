package telegram

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/ilyadubrovsky/tracking-bars/internal/domain"
	ierrors "github.com/ilyadubrovsky/tracking-bars/internal/errors"
	"github.com/ilyadubrovsky/tracking-bars/internal/service/telegram/answers"
	"github.com/ilyadubrovsky/tracking-bars/pkg/bars"
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
	err := s.userSvc.Save(
		context.Background(),
		&domain.User{
			ID: c.Sender().ID,
		},
	)
	if err != nil {
		return s.SendMessageWithOpts(c.Sender().ID, answers.StartError)
	}

	return s.SendMessageWithOpts(c.Sender().ID, answers.Start)
}

func (s *svc) handleHelpCommand(c tele.Context) error {
	return s.SendMessageWithOpts(c.Sender().ID, answers.Help)
}

func (s *svc) handleAuthCommand(c tele.Context) error {
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

	err := s.barsCredentialSvc.Authorization(context.Background(), &domain.BarsCredentials{
		UserID:   c.Sender().ID,
		Username: username,
		Password: []byte(password),
	})
	switch {
	case errors.Is(err, bars.ErrWrongGradesPage):
		return s.SendMessageWithOpts(c.Sender().ID, answers.GradesPageWrong)
	case errors.Is(err, bars.ErrAuthorizationFailed):
		return s.SendMessageWithOpts(c.Sender().ID, answers.CredentialsWrong)
	case errors.Is(err, ierrors.ErrAlreadyAuth):
		return s.SendMessageWithOpts(c.Sender().ID, answers.ClientAlreadyAuthorized)
	case err != nil:
		return s.SendMessageWithOpts(c.Sender().ID, answers.BotError)
	}

	return s.SendMessageWithOpts(c.Sender().ID, answers.SuccessfulAuthorization)
}

func (s *svc) handleLogoutCommand(c tele.Context) error {
	err := s.barsCredentialSvc.Logout(context.Background(), c.Sender().ID)
	switch {
	case errors.Is(err, ierrors.ErrNotAuthorized):
		return s.SendMessageWithOpts(c.Sender().ID, answers.ClientNotAuthorized)
	case err != nil:
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
	text := c.Message().Text
	message := strings.Replace(text, "/echo ", "", -1)

	return s.SendMessageWithOpts(c.Sender().ID, message, tele.ModeMarkdown)
}

func (s *svc) handleFixGradesCommand(c tele.Context) error {
	return s.SendMessageWithOpts(c.Sender().ID, answers.FixGrades, tele.ModeMarkdown)
}

func (s *svc) handleAdminSendMessageCommand(c tele.Context) error {
	msg := strings.Split(c.Message().Text, " ")
	if len(msg) < 3 {
		return s.SendMessageWithOpts(c.Sender().ID, answers.BotError)
	}

	userID, err := strconv.Atoi(msg[1])
	if err != nil {
		return s.SendMessageWithOpts(c.Sender().ID, answers.BotError)
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