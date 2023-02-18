package app

import (
	"context"
	"errors"
	tele "gopkg.in/telebot.v3"
	"gopkg.in/telebot.v3/middleware"
	"os"
	"strconv"
	"time"
	"tracking-barsv1.1/internal/config"
	"tracking-barsv1.1/internal/entity/change"
	"tracking-barsv1.1/internal/entity/user"
	"tracking-barsv1.1/internal/service/bars"
	"tracking-barsv1.1/internal/service/telegram"
	storage "tracking-barsv1.1/internal/storage/postgresql"
	"tracking-barsv1.1/pkg/client/postgresql"
	"tracking-barsv1.1/pkg/logging"
)

type TelegramService interface {
	GetProgressTableByID(ctx context.Context, userID int64) (*user.ProgressTable, error)
	LogoutUserByID(ctx context.Context, userID int64) error
	DeleteUserByID(ctx context.Context, userID int64) error
	GetAllUsers(ctx context.Context, aq ...string) ([]user.User, error)
	GetUserByID(ctx context.Context, userID int64) (*user.User, error)
}

type BarsService interface {
	Start()
	Authorization(ctx context.Context, userID int64, username, password string) (bool, error)
	GetProgressTable(ctx context.Context, client bars.Client) (user.ProgressTable, error)
	CheckChanges(ctx context.Context, usr user.User)
}

type app struct {
	bot             *tele.Bot
	barsService     BarsService
	telegramService TelegramService
	usersStorage    user.Repository
	changesStorage  change.Repository
	cfg             *config.Config
	logger          *logging.Logger
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
		a.logger.Errorf("failed to connection to postgresql due error: %v", err)
		return nil, err
	}

	a.logger.Info("database initializing")
	a.usersStorage = storage.NewUsersPostgres(postgresqlClient, a.logger)
	a.changesStorage = storage.NewChangesPostgres(postgresqlClient, a.logger)

	a.logger.Info("telegram bot initializing")
	bot, err := a.createBot()
	if err != nil {
		a.logger.Errorf("failed to initializing a telegram bot due error: %v", err)
		return nil, err
	}
	a.bot = bot

	a.logger.Info("BARS service initializing")
	a.barsService = bars.NewService(cfg, a.usersStorage, a.changesStorage, a.logger, bot)

	a.telegramService = telegram.NewService(a.logger, a.cfg, a.usersStorage, a.changesStorage)

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

	adminGroup.Handle("/sendnewsall", a.handleSendnewsAllCommand)

	adminGroup.Handle("/sendnewsauth", a.handleSendNewsAuthCommand)

	adminGroup.Handle("/sendmsg", a.handleSendmsgCommand)

	adminGroup.Handle("/logoutuser", a.handleLogoutUserCommand)

	adminGroup.Handle("/deleteuser", a.handleDeleteUserCommand)

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
	a.logger.Errorf("failed to execute bot instructions: %v", err)
	a.logger.Debugf("Chat: %d, Message: %s", c.Sender().ID, c.Message().Text)

	if errors.Is(err, tele.ErrMessageNotModified) {
		if err2 := c.Send(a.cfg.Responses.BotError); err2 != nil {
			a.logger.Errorf("failed to send an error response message due error: %v", err2)
		}
		return
	}
}
