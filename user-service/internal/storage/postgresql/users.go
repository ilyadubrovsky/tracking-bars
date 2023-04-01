package storage

import (
	"context"
	"errors"
	"fmt"
	"github.com/jackc/pgx/v4"
	"user-service/internal/entity/user"
	"user-service/pkg/client/postgresql"
	"user-service/pkg/logging"
)

/*
Create - Create with FULL USER
AuthorizationCheck - Get with ID, Deleted
 - GetAll with ID, Username, Password, ProgressTable
AuthorizationUser - Update username, password, deleted
LogoutUser - Update Password, ProgressTable, Deleted (w/o username) when logout
*/

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

func (s *usersPostgres) Create(ctx context.Context, u user.User) error {
	q := `INSERT INTO users (id, username, password, progress_table) VALUES ($1, $2, $3, $4)`
	s.logger.Tracef("SQL: %s", q)

	_, err := s.client.Exec(ctx, q, u.ID, u.Username, u.Password, u.ProgressTable)
	if err != nil {
		s.logger.Debugf("UserID: %d\n, Username:%s\n, Password:%s\n, Progress Table:%s\n", u.ID, u.Username, u.Password, u.ProgressTable)
		return err
	}

	return nil
}

func (s *usersPostgres) GetAllUsersID(ctx context.Context, aq ...string) ([]user.User, error) {
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

func (s *usersPostgres) AuthorizationCheck(ctx context.Context, id int64) (*bool, error) {
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

	return &usr.Deleted, nil
}

func (s *usersPostgres) Reauthorization(ctx context.Context, id int64, username string, password []byte, deleted bool) error {
	q := `UPDATE users SET username = $2, password = $3, deleted = $4 WHERE id = $1`
	s.logger.Tracef("SQL: %s", q)

	_, err := s.client.Exec(ctx, q, id, username, password, deleted)
	if err != nil {
		s.logger.Debugf("UserID: %d\n, Username:%s\n, Password: %v\n, Deleted: %v\n", id, username, password, deleted)
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

func (s *usersPostgres) Delete(ctx context.Context, id int64) error {
	q := `DELETE from users WHERE id = $1`
	s.logger.Tracef("SQL: %s", q)

	_, err := s.client.Exec(ctx, q, id)
	if err != nil {
		return err
	}

	return nil
}
