package app

import (
	"context"
	"fmt"
	"github.com/streadway/amqp"
	"grades-service/internal/config"
	"grades-service/internal/entity/change"
	"grades-service/internal/entity/user"
	"grades-service/internal/events"
	"grades-service/internal/events/grades"
	"grades-service/internal/service"
	storage "grades-service/internal/storage/postgresql"
	"grades-service/pkg/client/mq"
	"grades-service/pkg/client/mq/rabbitmq"
	"grades-service/pkg/client/postgresql"
	"grades-service/pkg/logging"
	"os"
	"sync"
)

type Service interface {
	ReceiveChanges()
}

type App struct {
	service        Service
	usersStorage   user.Repository
	changesStorage change.Repository
	producer       mq.Producer
	gradesStrategy events.ProcessStrategy
	cfg            *config.Config
	logger         *logging.Logger
}

func NewApp(cfg *config.Config) (*App, error) {
	var a App
	a.cfg = cfg

	a.logger = logging.GetLogger()

	pgConfig := postgresql.NewPgConfig(os.Getenv("PG_USERNAME"), os.Getenv("PG_PASSWORD"),
		os.Getenv("PG_HOST"), os.Getenv("PG_PORT"), os.Getenv("PG_DATABASE"))

	a.logger.Info("pottgresql client initializing")
	postgresqlClient, err := postgresql.NewClient(context.Background(), pgConfig)
	if err != nil {
		return nil, err
	}

	a.logger.Info("users storage initializing")
	a.usersStorage = storage.NewUsersPostgres(postgresqlClient, a.logger)
	a.logger.Info("changes storage initializing")
	a.changesStorage = storage.NewChangesPostgres(postgresqlClient, a.logger)

	RabbitURL := fmt.Sprintf("amqp://%s:%s@%s:%s/",
		os.Getenv("RABBIT_USERNAME"), os.Getenv("RABBIT_PASSWORD"),
		os.Getenv("RABBIT_HOST"), os.Getenv("RABBIT_PORT"))
	a.logger.Info("rabbitmq producer initializing")
	producer, err := rabbitmq.NewProducer(RabbitURL)
	if err != nil {
		return nil, err
	}

	a.logger.Info("producer: 'telegram' exchange initializing")
	if err = producer.DeclareExchange(a.cfg.RabbitMQ.Producer.TelegramExchange, amqp.ExchangeDirect,
		true, false, false); err != nil {
		a.logger.Fatalf("failed to declare an exchange due to error: %v", err)
	}

	a.logger.Info("producer: 'grades responses' queue initializing and binding")
	if err = producer.DeclareAndBindQueue(a.cfg.RabbitMQ.Producer.TelegramExchange,
		a.cfg.RabbitMQ.Producer.GradesResponses, a.cfg.RabbitMQ.Producer.GradesResponsesKey); err != nil {
		a.logger.Fatalf("failed to declare and bind a queue and bind due to error: %v", err)
	}

	a.logger.Info("producer: 'telegram messages' queue initializing and binding")
	if err = producer.DeclareAndBindQueue(a.cfg.RabbitMQ.Producer.TelegramExchange,
		a.cfg.RabbitMQ.Producer.TelegramMessages, a.cfg.RabbitMQ.Producer.TelegramMessagesKey); err != nil {
		a.logger.Fatalf("failed to declare and bind a queue and bind due to error: %v", err)
	}

	a.producer = producer

	a.logger.Info("service initializing")
	gradesService := service.NewService(cfg, a.usersStorage, a.changesStorage, a.logger, a.producer)

	a.service = gradesService

	a.logger.Info("grades process strategy initializing")
	a.gradesStrategy = grades.NewProcessStrategy(gradesService, a.cfg.Responses.Bars.UnavailablePT,
		a.cfg.Responses.Bars.NotAuthorized, a.cfg.Responses.BotError)

	return &a, nil
}

func (a *App) startConsume() {
	a.logger.Info("start consume")

	RabbitURL := fmt.Sprintf("amqp://%s:%s@%s:%s/",
		os.Getenv("RABBIT_USERNAME"), os.Getenv("RABBIT_PASSWORD"),
		os.Getenv("RABBIT_HOST"), os.Getenv("RABBIT_PORT"))

	a.logger.Info("rabbitmq grades consumer initializing")
	consumer, err := rabbitmq.NewConsumer(RabbitURL, a.cfg.RabbitMQ.Consumer.PrefetchCount)
	if err != nil {
		a.logger.Fatalf("failed to create a consumer due to error: %v", err)
	}

	if err = a.initializeConsume(consumer, a.cfg.RabbitMQ.Consumer.GradesRequests, a.cfg.RabbitMQ.Producer.TelegramExchange,
		a.cfg.RabbitMQ.Producer.GradesResponsesKey, a.cfg.RabbitMQ.Consumer.GradesWorkers, a.gradesStrategy); err != nil {
		a.logger.Fatalf("failed to initialize consume due to error: %v", err)
	}
}

func (a *App) initializeConsume(consumer mq.Consumer, queue, exchange, key string, workers int, strategy events.ProcessStrategy) error {
	a.logger.Infof("consumer: '%s' queue initializing", queue)
	if err := consumer.DeclareQueue(queue, true, false, false); err != nil {
		return fmt.Errorf("declare queue: %v", err)
	}

	messages, err := consumer.Consume(queue, false, false)
	if err != nil {
		return fmt.Errorf("consume: %v", err)
	}

	for i := 0; i < workers; i++ {
		worker := events.NewWorker(a.logger, consumer, a.producer, exchange, key, strategy, messages)
		go worker.Process()
		a.logger.Infof("queue '%s' worker [%d] started", queue, i+1)
	}

	return nil
}

func (a *App) Run() {
	a.logger.Info("app launching")

	var wg sync.WaitGroup
	wg.Add(1)

	a.startConsume()

	go a.service.ReceiveChanges()

	wg.Wait()
}
