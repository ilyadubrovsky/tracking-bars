package config

import (
	"github.com/ilyakaznacheev/cleanenv"
	"sync"
	"tracking-barsv1.1/pkg/logging"
)

type Config struct {
	Bars struct {
		URLs                 BarsURLs `yaml:"urls" env-required:"true"`
		ParserDelayInSeconds int      `yaml:"parser_delay_in_seconds" env-required:"true"`
	} `yaml:"bars" env-required:"true"`
	Responses Responses `yaml:"responses" env-required:"true"`
}

type Bars struct {
	Error                   string `yaml:"error" env-required:"true"`
	WrongData               string `yaml:"wrong_data" env-required:"true"`
	ExpiredData             string `yaml:"expired_data" env-required:"true"`
	NotAuthorized           string `yaml:"not_authorized" env-required:"true"`
	UnavailablePT           string `yaml:"unavailable_pt" env-required:"true"`
	NoDataEntered           string `yaml:"no_data_entered" env-required:"true"`
	EntryFormIgnored        string `yaml:"entry_form_ignored" env-required:"true"`
	DataEnteredIncorrectly  string `yaml:"data_entered_incorrectly" env-required:"true"`
	AlreadyAuthorized       string `yaml:"already_authorized" env-required:"true"`
	SuccessfulAuthorization string `yaml:"successful_authorization" env-required:"true"`
	SuccessfulLogout        string `yaml:"successful_logout" env-required:"true"`
	DefaultAnswer           string `yaml:"default_answer" env-required:"true"`
}

type BarsURLs struct {
	URL             string `yaml:"url" env-required:"true"`
	RegistrationURL string `yaml:"registration_url" env-required:"true"`
	MainPageURL     string `yaml:"main_page_url" env-required:"true"`
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
