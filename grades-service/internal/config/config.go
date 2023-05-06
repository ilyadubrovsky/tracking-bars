package config

import (
	"github.com/ilyakaznacheev/cleanenv"
	"grades-service/pkg/logging"
	"sync"
)

type Config struct {
	Bars struct {
		ParserDelayInSeconds int `yaml:"parser_delay_in_seconds" env-required:"true"`
		CountOfParsers       int `yaml:"count_of_parsers" env-required:"true"`
	} `yaml:"bars" env-required:"true"`
	Responses Responses `yaml:"responses" env-required:"true"`
	RabbitMQ  RabbitMQ  `yaml:"rabbitmq" env-required:"true"`
}

type Bars struct {
	ExpiredData   string `yaml:"expired_data" env-required:"true"`
	NotAuthorized string `yaml:"not_authorized" env-required:"true"`
	UnavailablePT string `yaml:"unavailable_pt" env-required:"true"`
	DefaultAnswer string `yaml:"default_answer" env-required:"true"`
}

type Responses struct {
	BotError string `yaml:"bot_error" env-required:"true"`
	Bars     Bars   `yaml:"bars" env-required:"true"`
}

type RabbitMQ struct {
	Consumer struct {
		GradesRequests string `yaml:"grades_requests" env-required:"true"`
		GradesWorkers  int    `yaml:"grades_workers" env-required:"true"`
		PrefetchCount  int    `yaml:"prefetch_count" env-required:"true"`
	} `yaml:"consumer" env-required:"true"`
	Producer struct {
		TelegramExchange    string `yaml:"telegram_exchange" env-required:"true"`
		TelegramMessages    string `yaml:"telegram_messages" env-required:"true"`
		TelegramMessagesKey string `yaml:"telegram_messages_key" env-required:"true"`
		GradesResponses     string `yaml:"grades_responses" env-required:"true"`
		GradesResponsesKey  string `yaml:"grades_responses_key" env-required:"true"`
	} `yaml:"producer" env-required:"true"`
}

var (
	once     sync.Once
	instance *Config
)

func GetConfig() (*Config, error) {
	var err error = nil

	once.Do(func() {
		logger := logging.GetLogger()
		logger.Info("read application configuration")
		instance = &Config{}
		if err = cleanenv.ReadConfig("configs/config.yml", instance); err != nil {
			help, _ := cleanenv.GetDescription(instance, nil)
			logger.Debug(help)
			return
		}
	})

	return instance, err
}
