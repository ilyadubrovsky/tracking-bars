package config

import (
	"fmt"
	"github.com/ilyakaznacheev/cleanenv"
	"sync"
)

type BarsURLs struct {
	URL             string `yaml:"url"`
	RegistrationURL string `yaml:"registration_url"`
	MainPageURL     string `yaml:"main_page_url"`
}

type Messages struct {
	BotError string `yaml:"bot_error"`
	Bars     Bars   `yaml:"bars"`
}

type Bars struct {
	Error                   string `yaml:"error"`
	WrongData               string `yaml:"wrong_data"`
	ExpiredData             string `yaml:"expired_data"`
	NotAuthorized           string `yaml:"not_authorized"`
	UnavailablePT           string `yaml:"unavailable_pt"`
	NoDataEntered           string `yaml:"no_data_entered"`
	EntryFormIgnored        string `yaml:"entry_form_ignored"`
	DataEnteredIncorrectly  string `yaml:"data_entered_incorrectly"`
	AlreadyAuthorized       string `yaml:"already_authorized"`
	SuccessfulAuthorization string `yaml:"successful_authorization"`
	SuccessfulLogout        string `yaml:"successful_logout"`
	DefaultAnswer           string `yaml:"default_answer"`
}

type Config struct {
	Bars struct {
		URLs                 BarsURLs `yaml:"urls"`
		ParserDelayInSeconds int      `yaml:"parser_delay_in_seconds"`
	} `yaml:"bars"`
	Messages Messages `yaml:"messages"`
}

var (
	once     sync.Once
	instance *Config
)

func GetConfig() (*Config, error) {
	var err error = nil

	once.Do(func() {
		instance = &Config{}
		if err = cleanenv.ReadConfig("configs/config.yml", instance); err != nil {
			help, _ := cleanenv.GetDescription(instance, nil)
			fmt.Println(help)
			return
		}
	})

	return instance, err
}
