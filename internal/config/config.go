package config

import "time"

type Config struct {
	Telegram Telegram
	Bars     Bars
	Postgres Postgres
}

type Bars struct {
	CronDelay                time.Time
	CronWorkerPoolSize       int
	CredentialsEncryptionKey string
}

type Telegram struct {
	BotToken string
}

type Postgres struct {
	DSN string
}
