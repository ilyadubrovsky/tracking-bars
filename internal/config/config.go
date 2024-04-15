package config

import "time"

const (
	BARSRegistrationPageURL = "https://bars.mpei.ru/bars_web/"
	BARSMainPageURL         = "https://bars.mpei.ru/bars_web/?sod=1"
)

type Config struct {
	Telegram Telegram
	Bars     Bars
	Postgres Postgres
}

type Bars struct {
	CronDelay          time.Duration
	CronWorkerPoolSize int
	EncryptionKey      string
}

type Telegram struct {
	BotToken        string
	LongPollerDelay time.Duration
	AdminID         int64
}

type Postgres struct {
	DSN string
}
