package config

import (
	"github.com/ilyakaznacheev/cleanenv"
	"sync"
	"telegram-service/pkg/logging"
)

type Config struct {
	Responses Responses `yaml:"responses" env-required:"true"`
	RabbitMQ  RabbitMQ  `yaml:"rabbitmq" env-required:"true"`
}

type RabbitMQ struct {
	Consumer struct {
		TelegramWorkers  int    `yaml:"telegram_workers" env-required:"true"`
		TelegramMessages string `yaml:"telegram_messages" env-required:"true"`
		GradesResponses  string `yaml:"grades_responses" env-required:"true"`
		GradesWorkers    int    `yaml:"grades_workers" env-required:"true"`
		PrefetchCount    int    `yaml:"prefetch_count" env-required:"true"`
	} `yaml:"consumer" env-required:"true"`
	Producer struct {
		UserExchange          string `yaml:"user_exchange" env-required:"true"`
		NewsRequests          string `yaml:"news_requests" env-required:"true"`
		NewsRequestsKey       string `yaml:"news_requests_key" env-required:"true"`
		AuthRequests          string `yaml:"auth_requests" env-required:"true"`
		AuthRequestsKey       string `yaml:"auth_requests_key" env-required:"true"`
		LogoutRequests        string `yaml:"logout_requests" env_required:"true"`
		LogoutRequestsKey     string `yaml:"logout_requests_key" env-required:"true"`
		DeleteUserRequests    string `yaml:"delete_user_requests" env_required:"true"`
		DeleteUserRequestsKey string `yaml:"delete_user_requests_key" env-required:"true"`
		GradesExchange        string `yaml:"grades_exchange" env-required:"true"`
		GradesRequests        string `yaml:"grades_requests" env-required:"true"`
		GradesRequestsKey     string `yaml:"grades_requests_key" env-required:"true"`
	} `yaml:"producer" env-required:"true"`
}

type Bars struct {
	UnavailablePT          string `yaml:"unavailable_pt" env-required:"true"`
	NoDataEntered          string `yaml:"no_data_entered" env-required:"true"`
	EntryFormIgnored       string `yaml:"entry_form_ignored" env-required:"true"`
	DataEnteredIncorrectly string `yaml:"data_entered_incorrectly" env-required:"true"`
	DefaultAnswer          string `yaml:"default_answer" env-required:"true"`
}

type Responses struct {
	BotError string `yaml:"bot_error" env-required:"true"`
	Bars     Bars   `yaml:"bars" env-required:"true"`
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
