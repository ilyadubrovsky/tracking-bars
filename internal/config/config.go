package config

import (
	"fmt"
	"time"

	"github.com/ilyakaznacheev/cleanenv"
)

const (
	BARSRegistrationPageURL = "https://bars.mpei.ru/bars_web/"
	BARSMainPageURL         = "https://bars.mpei.ru/bars_web/?sod=1"
	BARSGradesPageURL       = "https://bars.mpei.ru/bars_web/"
)

type Config struct {
	Telegram Telegram
	Bars     Bars
	Postgres Postgres
}

func NewConfig() (*Config, error) {
	cfg := &Config{}

	if err := cleanenv.ReadEnv(cfg); err != nil {
		return nil, fmt.Errorf("cleanenv.ReadEnv: %w", err)
	}

	return cfg, nil
}

type Bars struct {
	CronDelay                       time.Duration `env:"BARS_CRON_DELAY" env-default:"15m"`
	CronWorkerDelay                 time.Duration `env:"BARS_CRON_WORKER_DELAY" env-default:"10s"`
	CronWorkerPoolSize              int           `env:"BARS_CRON_WORKER_POOL_SIZE" env-default:"5"`
	AuthorizationFailedRetriesCount int           `env:"BARS_AUTHORIZATION_FAILED_RETRIES_COUNT" env-default:"3"`
	EncryptionKey                   string        `env:"BARS_ENCRYPTION_KEY"`
	OutboxCronDelay                 time.Duration `env:"BARS_OUTBOX_CRON_DELAY" env-default:"5m"`
}

type Telegram struct {
	BotToken        string        `env:"TELEGRAM_BOT_TOKEN"`
	LongPollerDelay time.Duration `env:"TELEGRAM_LONG_POLLER_DELAY" env-default:"60s"`
	AdminID         int64         `env:"TELEGRAM_ADMIN_ID"`
}

type Postgres struct {
	DSN string `env:"POSTGRES_DSN" env-default:"postgresql://postgres:postgres@localhost:5432/tracking-bars?sslmode=disable&timezone=UTC"`
}
