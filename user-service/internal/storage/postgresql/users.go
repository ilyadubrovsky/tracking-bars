package storage

import (
	"context"
	"errors"
	"fmt"
	"github.com/jackc/pgconn"
	"github.com/jackc/pgx/v4"
	"time"
	"user-service/internal/entity/user"
	"user-service/pkg/logging"
)

const (
	users = "users"
)

type Client interface {
	Exec(ctx context.Context, sql string, arguments ...interface{}) (pgconn.CommandTag, error)
	Query(ctx context.Context, sql string, args ...interface{}) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...interface{}) pgx.Row
	Begin(ctx context.Context) (pgx.Tx, error)
}

type usersPostgres struct {
	client Client
	logger *logging.Logger
}

func NewUsersPostgres(client Client, logger *logging.Logger) *usersPostgres {
	return &usersPostgres{
		client: client,
		logger: logger,
	}
}

func (s *usersPostgres) Create(ctx context.Context, u user.User) error {
	q := fmt.Sprintf("INSERT INTO %s (id, username, password, progress_table, created_at, updated_at)"+
		"VALUES ($1, $2, $3, $4, $5, $6)", users)
	s.logger.Tracef("SQL: %s", q)

	_, err := s.client.Exec(ctx, q, u.ID, u.Username, u.Password, u.ProgressTable, time.Now(), time.Now())
	if err != nil {
		s.logger.Debugf("UserID: %d\n, Username:%s\n, Password:%s\n, Progress Table:%s\n", u.ID, u.Username, u.Password, u.ProgressTable)
		return err
	}

	return nil
}

func (s *usersPostgres) GetAllUsers(ctx context.Context, aq ...string) ([]user.User, error) {
	q := fmt.Sprintf("SELECT id, username, password, progress_table, deleted FROM %s ", users)

	if len(aq) == 1 {
		q += aq[0]
	} else if len(aq) > 1 {
		return nil, fmt.Errorf("the length of aq is equal to %d, which is greater than one", len(aq))
	}
	s.logger.Tracef("SQL: %s", q)

	usersRows, err := s.client.Query(ctx, q)
	if err != nil {
		return nil, err
	}

	usrs := make([]user.User, 0)
	for usersRows.Next() {
		var usr user.User

		err = usersRows.Scan(&usr.ID, &usr.Username, &usr.Password, &usr.ProgressTable, &usr.Deleted)
		if err != nil {
			return nil, err
		}

		usrs = append(usrs, usr)
	}

	if err = usersRows.Err(); err != nil {
		return nil, err
	}

	return usrs, nil
}

func (s *usersPostgres) AuthorizationCheck(ctx context.Context, id int64) (*bool, error) {
	q := fmt.Sprintf("SELECT id, username, password, progress_table, deleted FROM %s WHERE id = $1", users)
	s.logger.Tracef("SQL: %s", q)

	var usr user.User

	userRow := s.client.QueryRow(ctx, q, id)
	err := userRow.Scan(&usr.ID, &usr.Username, &usr.Password, &usr.ProgressTable, &usr.Deleted)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	return &usr.Deleted, nil
}

func (s *usersPostgres) Reauthorization(ctx context.Context, id int64, username string, password []byte, deleted bool) error {
	q := fmt.Sprintf("UPDATE %s SET username = $2, password = $3, deleted = $4, updated_at = $5 WHERE id = $1", users)
	s.logger.Tracef("SQL: %s", q)

	_, err := s.client.Exec(ctx, q, id, username, password, deleted, time.Now())
	if err != nil {
		s.logger.Debugf("UserID: %d\n, Username:%s\n, Password: %v\n, Deleted: %v\n", id, username, password, deleted)
		return err
	}

	return nil
}

func (s *usersPostgres) LogoutUser(ctx context.Context, id int64) error {
	q := fmt.Sprintf("UPDATE %s SET password = DEFAULT, progress_table = DEFAULT, deleted = true, "+
		"updated_at = $2 WHERE id = $1", users)
	s.logger.Tracef("SQL: %s", q)

	_, err := s.client.Exec(ctx, q, id, time.Now())
	if err != nil {
		s.logger.Debugf("UserID: %d\n", id)
		return err
	}

	return nil
}

func (s *usersPostgres) Delete(ctx context.Context, id int64) error {
	q := fmt.Sprintf("DELETE from %s WHERE id = $1", users)
	s.logger.Tracef("SQL: %s", q)

	_, err := s.client.Exec(ctx, q, id)
	if err != nil {
		return err
	}

	return nil
}
