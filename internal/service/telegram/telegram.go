package telegram

import (
	"context"
	"errors"
	"fmt"

	"github.com/ilyadubrovsky/tracking-bars/internal/config"
	"github.com/ilyadubrovsky/tracking-bars/internal/repository"
	"github.com/ilyadubrovsky/tracking-bars/internal/service"
	"github.com/rs/zerolog/log"
	tele "gopkg.in/telebot.v3"
	"gopkg.in/telebot.v3/middleware"
)

type svc struct {
	userSvc             service.User
	barsSvc             service.Bars
	barsCredentialsRepo repository.BarsCredentials
	bot                 *tele.Bot
	cfg                 config.Telegram
}

func NewService(
	userSvc service.User,
	barsSvc service.Bars,
	barsCredentialsRepo repository.BarsCredentials,
	cfg config.Telegram,
) (*svc, error) {
	bot, err := createBot(cfg)
	if err != nil {
		return nil, fmt.Errorf("createBot: %w", err)
	}

	s := &svc{
		userSvc:             userSvc,
		barsSvc:             barsSvc,
		barsCredentialsRepo: barsCredentialsRepo,
		bot:                 bot,
		cfg:                 cfg,
	}

	s.setBotSettings()

	return s, nil
}

func createBot(cfg config.Telegram) (*tele.Bot, error) {
	pref := tele.Settings{
		Token:  cfg.BotToken,
		Poller: &tele.LongPoller{Timeout: cfg.LongPollerDelay},
		OnError: func(err error, c tele.Context) {
			log.Error().Fields(extractTelebotFields(c)).
				Msgf("bot.OnError: %v", err.Error())
		},
	}

	abot, err := tele.NewBot(pref)
	if err != nil {
		return nil, fmt.Errorf("tele.NewBot: %w", err)
	}

	return abot, nil
}

func (s *svc) setBotSettings() {
	// TODO в общем-то нужен ratelimiter на все эти ручки
	s.bot.Handle(tele.OnCallback, s.handleCallback)

	s.bot.Handle("/start", s.handleStartCommand)

	s.bot.Handle("/help", s.handleHelpCommand)

	s.bot.Handle("/fixgrades", s.handleFixGradesCommand)

	s.bot.Handle("/auth", s.handleAuthCommand)

	s.bot.Handle("/logout", s.handleLogoutCommand)

	s.bot.Handle("/pt", s.handleProgressTableCommand)

	s.bot.Handle("/gh", s.handleGithubCommand)

	s.bot.Handle(tele.OnText, s.handleText)

	adminGroup := s.bot.Group()
	adminGroup.Use(
		middleware.Whitelist(
			s.cfg.AdminID,
		),
	)

	adminGroup.Handle("/aecho", s.handleAdminEchoCommand)

	adminGroup.Handle("/asmall", s.handleAdminSendMessageAllCommand)

	adminGroup.Handle("/asm", s.handleAdminSendMessageCommand)

	adminGroup.Handle("/acauth", s.handleAdminCountAuthorizedCommand)
}

func (s *svc) SendMessageWithOpts(id int64, message string, opts ...interface{}) error {
	chat := tele.ChatID(id)

	_, err := s.bot.Send(chat, message, opts...)

	return s.middlewareError(id, err)
}

func (s *svc) EditMessageWithOpts(id int64, messageID int, msg string, opts ...interface{}) error {
	_, err := s.bot.Edit(
		&editableMessage{
			messageID: messageID,
			chatID:    id,
		},
		msg,
		opts...,
	)

	if errors.Is(err, tele.ErrTrueResult) {
		err = s.SendMessageWithOpts(id, "TODO bot error")
	}

	return s.middlewareError(id, err)
}

func (s *svc) middlewareError(targetUserID int64, err error) error {
	if err == nil {
		return nil
	}

	if errors.As(err, &tele.ErrBlockedByUser) ||
		errors.As(err, &tele.ErrUserIsDeactivated) ||
		errors.As(err, &tele.ErrNotStartedByUser) {
		deleteErr := s.userSvc.Delete(context.Background(), targetUserID)
		if deleteErr != nil {
			log.Error().Int64("user", targetUserID).Msgf(
				"deleting user with received err %v failed: %v", err, deleteErr,
			)
		}
	}

	return err
}

func (s *svc) Start() {
	s.bot.Start()
}

func (s *svc) Stop() {
	s.bot.Stop()
}
