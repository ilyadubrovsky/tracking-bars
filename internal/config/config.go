package config

import "time"

type Config struct {
	Telegram Telegram
	Bars     Bars
}

type Bars struct {
	CronDelay          time.Time
	CronWorkerPoolSize int
}

type Telegram struct {
	BotToken string
}

type Postgres struct {
	Host     string
	Port     string
	Username string
	Password string
}
