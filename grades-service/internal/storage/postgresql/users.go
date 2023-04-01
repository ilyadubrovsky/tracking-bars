package storage

import (
	"context"
	"errors"
	"fmt"
	"github.com/jackc/pgx/v4"
	"grades-service/internal/entity/user"
	"grades-service/pkg/client/postgresql"
	"grades-service/pkg/logging"
)

type usersPostgres struct {
	client postgresql.Client
	logger *logging.Logger
}

func NewUsersPostgres(client postgresql.Client, logger *logging.Logger) user.Repository {
	return &usersPostgres{
		client: client,
		logger: logger,
	}
}

func (s *usersPostgres) FindAll(ctx context.Context, aq ...string) ([]user.User, error) {
	q := `SELECT id, username, password, progress_table, deleted FROM users `
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

func (s *usersPostgres) FindOne(ctx context.Context, id int64) (*user.User, error) {
	q := `SELECT id, username, password, progress_table, deleted FROM users WHERE id = $1`
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

	return &usr, nil
}

func (s *usersPostgres) UpdateProgressTable(ctx context.Context, id int64, table user.ProgressTable) error {
	q := `UPDATE users SET progress_table = $2 WHERE id = $1`
	s.logger.Tracef("SQL: %s", q)

	_, err := s.client.Exec(ctx, q, id, table)
	if err != nil {
		s.logger.Debugf("UserID: %d\n, Progress Table:%s\n", id, table)
		return err
	}

	return nil
}

func (s *usersPostgres) LogoutUser(ctx context.Context, id int64) error {
	q := `UPDATE users SET password = DEFAULT, progress_table = DEFAULT, deleted = true WHERE id = $1`
	s.logger.Tracef("SQL: %s", q)

	_, err := s.client.Exec(ctx, q, id)
	if err != nil {
		s.logger.Debugf("UserID: %d\n", id)
		return err
	}

	return nil
}
