package config

import "time"

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
}

type Postgres struct {
	DSN string
}
