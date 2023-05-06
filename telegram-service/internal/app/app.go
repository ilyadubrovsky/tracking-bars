package app

import (
	"fmt"
	"github.com/streadway/amqp"
	tele "gopkg.in/telebot.v3"
	"gopkg.in/telebot.v3/middleware"
	"os"
	"strconv"
	"sync"
	"telegram-service/internal/config"
	"telegram-service/internal/events"
	"telegram-service/internal/events/grades"
	"telegram-service/internal/events/tgmessages"
	"telegram-service/internal/service"
	"telegram-service/pkg/client/mq"
	"telegram-service/pkg/client/mq/rabbitmq"
	"telegram-service/pkg/logging"
	"time"
)

type Service interface {
	SendMessageWithOpts(id int64, msg string, opts ...interface{}) error
	EditMessageWithOpts(id int64, messageid int, msg string, opts ...interface{}) error
}

type App struct {
	bot              *tele.Bot
	service          Service
	cfg              *config.Config
	logger           *logging.Logger
	producer         mq.Producer
	telegramStrategy events.ProcessStrategy
	gradesStrategy   events.ProcessStrategy
}

func NewApp(cfg *config.Config) (*App, error) {
	var a App
	a.cfg = cfg

	a.logger = logging.GetLogger()

	bot, err := a.createBot()
	if err != nil {
		return nil, err
	}
	a.bot = bot

	RabbitURL := fmt.Sprintf("amqp://%s:%s@%s:%s",
		os.Getenv("RABBIT_USERNAME"), os.Getenv("RABBIT_PASSWORD"),
		os.Getenv("RABBIT_HOST"), os.Getenv("RABBIT_PORT"))

	a.logger.Info("rabbitmq producer initializing")
	producer, err := rabbitmq.NewProducer(RabbitURL)
	if err != nil {
		a.logger.Fatalf("failed to create a new producer due to error: %v", err)
	}
	a.producer = producer

	a.logger.Infof("service initializing")
	a.service = service.NewService(a.logger, a.cfg, a.bot, producer)

	a.logger.Info("telegram process strategy initializing")
	a.telegramStrategy = tgmessages.NewProcessStrategy(a.service)

	a.logger.Info("grades process strategy initializing")
	a.gradesStrategy = grades.NewProcessStrategy(a.service, a.cfg.Responses.BotError, a.cfg.Responses.Bars.UnavailablePT)

	a.logger.Info("producer: 'user' exchange initializing")
	if err = producer.DeclareExchange(a.cfg.RabbitMQ.Producer.UserExchange, amqp.ExchangeDirect,
		true, false, false); err != nil {
		a.logger.Fatalf("failed to declare an exchange due to error: %v", err)
	}

	a.declareAndBindQueue(a.cfg.RabbitMQ.Producer.UserExchange,
		a.cfg.RabbitMQ.Producer.AuthRequests, a.cfg.RabbitMQ.Producer.AuthRequestsKey)

	a.declareAndBindQueue(a.cfg.RabbitMQ.Producer.UserExchange,
		a.cfg.RabbitMQ.Producer.LogoutRequests, a.cfg.RabbitMQ.Producer.LogoutRequestsKey)

	a.declareAndBindQueue(a.cfg.RabbitMQ.Producer.UserExchange,
		a.cfg.RabbitMQ.Producer.NewsRequests, a.cfg.RabbitMQ.Producer.NewsRequestsKey)

	a.declareAndBindQueue(a.cfg.RabbitMQ.Producer.UserExchange,
		a.cfg.RabbitMQ.Producer.DeleteUserRequests, a.cfg.RabbitMQ.Producer.DeleteUserRequestsKey)

	a.logger.Info("producer: 'grades' exchange initializing")
	if err = producer.DeclareExchange(a.cfg.RabbitMQ.Producer.GradesExchange, amqp.ExchangeDirect,
		true, false, false); err != nil {
		a.logger.Fatalf("failed to declare an exchange due to error: %v", err)
	}

	a.declareAndBindQueue(a.cfg.RabbitMQ.Producer.GradesExchange,
		a.cfg.RabbitMQ.Producer.GradesRequests, a.cfg.RabbitMQ.Producer.GradesRequestsKey)

	a.producer = producer
	return &a, nil
}

func (a *App) declareAndBindQueue(exchange, queue, key string) {
	a.logger.Infof("producer: '%s' qeueu initializing and binding", queue)
	if err := a.producer.DeclareAndBindQueue(exchange, queue, key); err != nil {
		a.logger.Fatalf("failed to declare and bind a queue due to error: %v", err)
	}
}

func (a *App) startConsume() {
	a.logger.Info("start consume")
	RabbitURL := fmt.Sprintf("amqp://%s:%s@%s:%s",
		os.Getenv("RABBIT_USERNAME"), os.Getenv("RABBIT_PASSWORD"),
		os.Getenv("RABBIT_HOST"), os.Getenv("RABBIT_PORT"))

	a.logger.Info("rabbitmq consumer initalizing")
	consumer, err := rabbitmq.NewConsumer(RabbitURL, a.cfg.RabbitMQ.Consumer.PrefetchCount)
	if err != nil {
		a.logger.Fatalf("failed to create a consumer due to error: %v", err)
	}

	if err = a.initializeConsume(consumer, a.cfg.RabbitMQ.Consumer.TelegramMessages,
		a.cfg.RabbitMQ.Consumer.TelegramWorkers, a.telegramStrategy); err != nil {
		a.logger.Fatalf("failed to initialize consume due to error: %v", err)
	}

	if err = a.initializeConsume(consumer, a.cfg.RabbitMQ.Consumer.GradesResponses,
		a.cfg.RabbitMQ.Consumer.GradesWorkers, a.gradesStrategy); err != nil {
		a.logger.Fatalf("failed to initialize consume due to error: %v", err)
	}
}

func (a *App) initializeConsume(consumer mq.Consumer, queue string, workers int, strategy events.ProcessStrategy) error {
	a.logger.Infof("consumer: '%s' queue initializing", queue)
	if err := consumer.DeclareQueue(queue, true, false, false); err != nil {
		return fmt.Errorf("declare queue: %v", err)
	}

	messages, err := consumer.Consume(queue, false, false)
	if err != nil {
		return fmt.Errorf("consume: %v", err)
	}

	for i := 0; i < workers; i++ {
		worker := events.NewWorker(a.logger, consumer, strategy, messages)
		go worker.Process()
		a.logger.Infof("queue '%s' worker [%d] started", queue, i+1)
	}

	return nil
}

func (a *App) createBot() (*tele.Bot, error) {
	pref := tele.Settings{
		Token:  os.Getenv("TELEGRAM_TOKEN"),
		Poller: &tele.LongPoller{Timeout: 60 * time.Second},
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

	abot.Handle("/sq", a.handleSqCommand)

	abot.Handle(tele.OnText, a.handleText)

	adminGroup.Handle("/echo", a.handleEchoCommand)

	adminGroup.Handle("/sendnewsall", a.handleSendnewsAllCommand)

	adminGroup.Handle("/sendnewsauth", a.handleSendNewsAuthCommand)

	adminGroup.Handle("/authuser", a.handleAuthUserCommand)

	adminGroup.Handle("/logoutuser", a.handleLogoutUserCommand)

	adminGroup.Handle("/deleteuser", a.handleDeleteUserCommand)

	adminGroup.Handle("/sendmsg", a.handleSendmsgCommand)

	return abot, nil
}

func (a *App) Run() {
	a.logger.Info("app launching")

	var wg sync.WaitGroup
	wg.Add(1)

	a.startConsume()

	a.logger.Info("telegram bot launching")
	a.bot.Start()

	wg.Wait()

}
