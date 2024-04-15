package postgresql

import (
	"context"
	"fmt"
	"github.com/jackc/pgx/v4/pgxpool"
)

type pgConfig struct {
	Username string
	Password string
	Host     string
	Port     string
	Database string
}

func NewConfig(username, password, host, port, database string) *pgConfig {
	return &pgConfig{
		Username: username,
		Password: password,
		Host:     host,
		Port:     port,
		Database: database,
	}
}

func NewClient(ctx context.Context, cfg *pgConfig) (*pgxpool.Pool, error) {
	fmt.Println(cfg)
	connString := fmt.Sprintf("postgres://%s:%s@%s:%s/%s", cfg.Username, cfg.Password, cfg.Host, cfg.Port, cfg.Database)
	pool, err := pgxpool.Connect(ctx, connString)
	if err != nil {

		return nil, err
	}
	if err = pool.Ping(ctx); err != nil {
		return nil, err
	}
	return pool, nil
}
