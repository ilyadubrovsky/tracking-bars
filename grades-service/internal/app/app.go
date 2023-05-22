package app

import (
	"context"
	"fmt"
	"github.com/streadway/amqp"
	"grades-service/internal/config"
	"grades-service/internal/events"
	"grades-service/internal/events/grades"
	"grades-service/internal/service"
	db "grades-service/internal/storage/postgresql"
	cache "grades-service/internal/storage/redis"
	"grades-service/pkg/client/mq"
	"grades-service/pkg/client/mq/rabbitmq"
	"grades-service/pkg/client/postgresql"
	"grades-service/pkg/client/redis"
	"grades-service/pkg/logging"
	"os"
	"os/signal"
	"syscall"
	"time"
)

const (
	shutdownTimeout = 5 * time.Second
)

type Service interface {
	ReceiveChanges()
}

type App struct {
	service        Service
	producer       mq.Producer
	gradesStrategy events.ProcessStrategy
	cfg            *config.Config
	logger         *logging.Logger
}

func Run(cfg *config.Config) error {
	var a App
	a.cfg = cfg

	a.logger = logging.GetLogger()

	pgConfig := postgresql.NewConfig(os.Getenv("PG_USERNAME"), os.Getenv("PG_PASSWORD"),
		os.Getenv("PG_HOST"), os.Getenv("PG_PORT"), os.Getenv("PG_DATABASE"))

	a.logger.Info("postgresql client initializing")
	postgresqlClient, err := postgresql.NewClient(context.Background(), pgConfig)
	if err != nil {
		return err
	}

	a.logger.Info("users storage initializing")
	usersStorage := db.NewUsersPostgres(postgresqlClient, a.logger)
	a.logger.Info("changes storage initializing")
	changesStorage := db.NewChangesPostgres(postgresqlClient, a.logger)
	redisConfig := redis.NewConfig(os.Getenv("REDIS_HOST"), os.Getenv("REDIS_PORT"),
		os.Getenv("REDIS_PASSWORD"))

	a.logger.Info("redis client initializing")
	redisClient, err := redis.NewClient(redisConfig)
	if err != nil {
		return err
	}

	a.logger.Info("grades cache initializing")
	gradesCache := cache.NewGradesRedis(redisClient, a.logger)

	RabbitURL := fmt.Sprintf("amqp://%s:%s@%s:%s/",
		os.Getenv("RABBIT_USERNAME"), os.Getenv("RABBIT_PASSWORD"),
		os.Getenv("RABBIT_HOST"), os.Getenv("RABBIT_PORT"))
	a.logger.Info("rabbitmq producer initializing")
	producer, err := rabbitmq.NewProducer(RabbitURL)
	if err != nil {
		return err
	}
	a.producer = producer

	a.logger.Info("producer: 'telegram' exchange initializing")
	if err = producer.DeclareExchange(a.cfg.RabbitMQ.Producer.TelegramExchange, amqp.ExchangeDirect,
		true, false, false); err != nil {
		a.logger.Fatalf("failed to declare an exchange due to error: %v", err)
	}

	a.declareAndBindQueue(a.cfg.RabbitMQ.Producer.TelegramExchange,
		a.cfg.RabbitMQ.Producer.GradesResponses, a.cfg.RabbitMQ.Producer.GradesResponsesKey)

	a.declareAndBindQueue(a.cfg.RabbitMQ.Producer.TelegramExchange,
		a.cfg.RabbitMQ.Producer.TelegramMessages, a.cfg.RabbitMQ.Producer.TelegramMessagesKey)

	a.logger.Info("service initializing")
	gradesService := service.NewService(cfg, usersStorage, changesStorage, gradesCache,
		a.logger, a.producer)
	a.service = gradesService

	a.logger.Info("grades process strategy initializing")
	a.gradesStrategy = grades.NewProcessStrategy(gradesService, a.cfg.Responses.Bars.NotAuthorized,
		a.cfg.Responses.BotError, a.cfg.Responses.Bars.PTNotProvided, a.cfg.Responses.Bars.WrongGradesPage)

	a.logger.Info("app launching")
	a.logger.Info("app: start consume")
	a.startConsume()
	a.logger.Info("bars service: receive changes")
	go a.service.ReceiveChanges()
	a.logger.Info("app started")

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	<-ctx.Done()

	a.logger.Info("app shutting down")
	postgresqlClient.Close()
	if err = redisClient.Close(); err != nil {
		a.logger.Errorf("redis client failed to close due to error: %v", err)
	}
	// TODO close RabbitMQ connections

	return nil
}

func (a *App) declareAndBindQueue(exchange, queue, key string) {
	a.logger.Infof("producer: '%s' queue initializing and binding", queue)
	if err := a.producer.DeclareAndBindQueue(exchange, queue, key); err != nil {
		a.logger.Fatalf("failed to declare and bind a queue and bind due to error: %v", err)
	}
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
