package config

import (
	"github.com/ilyakaznacheev/cleanenv"
	"sync"
	"user-service/pkg/logging"
)

type Config struct {
	Bars struct {
		URLs BarsURLs `yaml:"urls" env-required:"true"`
	} `yaml:"bars" env-required:"true"`
	Responses Responses `yaml:"responses" env-required:"true"`
	RabbitMQ  RabbitMQ  `yaml:"rabbitmq" env-required:"true"`
}

type RabbitMQ struct {
	Consumer struct {
		NewsRequests       string `yaml:"news_requests" env-required:"true"`
		AuthRequests       string `yaml:"auth_requests" env-required:"true"`
		LogoutRequests     string `yaml:"logout_requests" env-required:"true"`
		DeleteUserRequests string `yaml:"delete_user_requests" env-required:"true"`
		NewsWorkers        int    `yaml:"news_workers" env-required:"true"`
		AuthWorkers        int    `yaml:"auth_workers" env-required:"true"`
		LogoutWorkers      int    `yaml:"logout_workers" env-required:"true"`
		DeleteUserWorkers  int    `yaml:"delete_user_workers" env-required:"true"`
		PrefetchCount      int    `yaml:"prefetch_count" env-required:"true"`
	} `yaml:"consumer" env-required:"true"`
	Producer struct {
		TelegramMessages    string `yaml:"telegram_messages" env-required:"true"`
		TelegramExchange    string `yaml:"telegram_exchange" env-required:"true"`
		TelegramMessagesKey string `yaml:"telegram_messages_key" env-required:"true"`
	} `yaml:"producer" env-required:"true"`
}

type Bars struct {
	Error                   string `yaml:"error" env-required:"true"`
	WrongData               string `yaml:"wrong_data" env-required:"true"`
	NotAuthorized           string `yaml:"not_authorized" env-required:"true"`
	AlreadyAuthorized       string `yaml:"already_authorized" env-required:"true"`
	SuccessfulAuthorization string `yaml:"successful_authorization" env-required:"true"`
	SuccessfulLogout        string `yaml:"successful_logout" env-required:"true"`
}

type Responses struct {
	BotError string `yaml:"bot_error" env-required:"true"`
	Bars     Bars   `yaml:"bars" env-required:"true"`
}

type BarsURLs struct {
	URL             string `yaml:"url" env-required:"true"`
	RegistrationURL string `yaml:"registration_url" env-required:"true"`
	MainPageURL     string `yaml:"main_page_url" env-required:"true"`
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
