package app

import (
	"TrackingBARSv2/internal/config"
	"TrackingBARSv2/internal/entity/change"
	"TrackingBARSv2/internal/entity/user"
	"TrackingBARSv2/internal/service/bars"
	storage "TrackingBARSv2/internal/storage/postgresql"
	"TrackingBARSv2/pkg/client/postgresql"
	"TrackingBARSv2/pkg/logging"
	"context"
	tele "gopkg.in/telebot.v3"
	"gopkg.in/telebot.v3/middleware"
	"os"
	"strconv"
	"time"
)

type app struct {
	bot            *tele.Bot
	barsService    bars.Service
	usersStorage   user.Repository
	changesStorage change.Repository
	cfg            *config.Config
	logger         logging.Logger
}

type App interface {
	Run() error
}

func NewApp(cfg *config.Config) (App, error) {
	var a app

	a.cfg = cfg

	a.logger = logging.GetLogger()

	a.logger.Info("app initializing")

	a.logger.Info("postgresql client initializing")
	pgConfig := postgresql.NewPgConfig(os.Getenv("PG_USERNAME"), os.Getenv("PG_PASSWORD"),
		os.Getenv("PG_HOST"), os.Getenv("PG_PORT"), os.Getenv("PG_DATABASE"))
	postgresqlClient, err := postgresql.NewClient(context.Background(), pgConfig)
	if err != nil {
		a.logger.Errorf("failed to connection to postgresql due error: %s", err)
		return nil, err
	}

	a.logger.Info("database initializing")
	a.usersStorage = storage.NewUsersPostgres(postgresqlClient, a.logger)
	a.changesStorage = storage.NewChangesPostgres(postgresqlClient, a.logger)

	a.logger.Info("telegram bot initializing")
	bot, err := a.createBot()
	if err != nil {
		a.logger.Errorf("failed to initializing a telegram bot due error: %s", err)
		return nil, err
	}
	a.bot = bot

	a.logger.Info("BARS service initializing")
	a.barsService = bars.NewService(cfg, a.usersStorage, a.changesStorage, a.logger, bot)

	return &a, nil
}

func (a *app) createBot() (*tele.Bot, error) {
	pref := tele.Settings{
		Token:   os.Getenv("TELEGRAM_TOKEN"),
		Poller:  &tele.LongPoller{Timeout: 10 * time.Second},
		OnError: a.OnBotError,
	}

	abot, err := tele.NewBot(pref)
	if err != nil {
		return nil, err
	}

	adminGroup := abot.Group()
	adminID, err := strconv.Atoi(os.Getenv("ADMIN_ID"))
	if err != nil {
		return nil, err
	}
	adminGroup.Use(middleware.Whitelist(int64(adminID)))

	abot.Handle(tele.OnCallback, a.handleCallback)

	abot.Handle("/start", a.handleStartCommand)

	abot.Handle("/help", a.handleHelpCommand)

	abot.Handle("/auth", a.handleAuthCommand)

	abot.Handle("/logout", a.handleLogoutCommand)

	abot.Handle("/pt", a.handlePtCommand)

	abot.Handle("/gh", a.handleGhCommand)

	abot.Handle(tele.OnText, a.handleText)

	adminGroup.Handle("/echo", a.handleEchoCommand)

	adminGroup.Handle("/sendnews", a.handleSendnewsCommand)

	adminGroup.Handle("/sendmsg", a.handleSendmsgCommand)

	adminGroup.Handle("/deluser", a.handleDeluserCommand)

	return abot, nil
}

func (a *app) Run() error {
	a.logger.Info("app launching")

	a.logger.Info("BARS service: parser launching")
	go a.barsService.Start()

	a.logger.Info("telegram bot launching")
	a.bot.Start()

	return nil
}

func (a *app) OnBotError(err error, c tele.Context) {
	a.logger.Errorf("failed to execute bot instructions: %s", err)
	a.logger.Debugf("Chat: %d, Message: %s", c.Sender().ID, c.Message().Text)

	switch err.Error() {
	case tele.ErrMessageNotModified.Description:
		if err2 := c.Send(a.cfg.Messages.BotError); err2 != nil {
			a.logger.Errorf("failed to send an error response message due error: %s", err)
		}
	case tele.ErrBlockedByUser.Description, tele.ErrUserIsDeactivated.Description:
		if err2 := a.usersStorage.Delete(context.Background(), c.Sender().ID); err2 != nil {
			a.logger.Errorf("failed to Delete blocked user from database due error: %s", err)
		}
	}
}
