package storage

import (
	"TrackingBARSv2/internal/entity/user"
	"TrackingBARSv2/pkg/client/postgresql"
	"TrackingBARSv2/pkg/logging"
	"context"
	"fmt"
	"github.com/jackc/pgx/v4"
)

type usersPostgres struct {
	client postgresql.Client
	logger logging.Logger
}

func NewUsersPostgres(client postgresql.Client, logger logging.Logger) user.Repository {
	return &usersPostgres{
		client: client,
		logger: logger,
	}
}

func (s *usersPostgres) Create(ctx context.Context, dto user.CreateUserDTO) error {
	q := `INSERT INTO users (id, username, password, progress_table) VALUES ($1, $2, $3, $4)`
	_, err := s.client.Exec(ctx, q, dto.ID, dto.Username, dto.Password, dto.ProgressTable)
	if err != nil {
		s.logger.Tracef("SQL: %s", q)
		s.logger.Debugf("ID: %d\nUsername: %s\n Password: %s\n", dto.ID, dto.Username, dto.Password)
		return fmt.Errorf("failed to Exec due error: %s", err)
	}
	return nil
}

func (s *usersPostgres) FindAll(ctx context.Context, aq ...string) ([]user.User, error) {
	q := `SELECT id, username, password, progress_table, deleted FROM users `
	if len(aq) == 1 {
		q += aq[0]
	} else if len(aq) > 1 {
		return nil, fmt.Errorf("the length of aq is equal to %d, which is greater than one", len(aq))
	}

	usersRows, err := s.client.Query(ctx, q)
	if err != nil {
		s.logger.Tracef("SQL: %s", q)
		s.logger.Debugf("Raw Values: %s", usersRows.RawValues())
		return nil, fmt.Errorf("failed to Query due error: %s", err)
	}

	usrs := make([]user.User, 0)
	for usersRows.Next() {
		var usr user.User

		err = usersRows.Scan(&usr.ID, &usr.Username, &usr.Password, &usr.ProgressTable, &usr.Deleted)
		if err != nil {
			s.logger.Tracef("SQL: %s", q)
			s.logger.Debugf("Raw Values: %s", usersRows.RawValues())
			return nil, fmt.Errorf("failed to Scan when reading Rows due error: %s", err)
		}

		usrs = append(usrs, usr)
	}

	if err = usersRows.Err(); err != nil {
		s.logger.Tracef("SQL: %s", q)
		s.logger.Debugf("Raw Values: %s", usersRows.RawValues())
		return nil, fmt.Errorf("failed to handle Rows due error: %s", err)
	}

	return usrs, nil
}

func (s *usersPostgres) FindOne(ctx context.Context, id int64) (user.User, error) {
	q := `SELECT id, username, password, progress_table, deleted FROM users WHERE id = $1`

	var usr user.User
	userRow := s.client.QueryRow(ctx, q, id)
	err := userRow.Scan(&usr.ID, &usr.Username, &usr.Password, &usr.ProgressTable, &usr.Deleted)
	if err == pgx.ErrNoRows {
		return user.User{}, nil
	}
	if err != nil {
		s.logger.Debugf("UserID: %d", id)
		return user.User{}, fmt.Errorf("failed to Scan when reading a Rows due error: %s", err)
	}
	return usr, nil
}

func (s *usersPostgres) Update(ctx context.Context, dto user.UpdateUserDTO) error {
	q := `UPDATE users SET username = $2, password = $3, progress_table = $4, deleted = $5 WHERE id = $1`
	_, err := s.client.Exec(ctx, q, dto.ID, dto.Username, dto.Password, dto.ProgressTable, dto.Deleted)
	if err != nil {
		s.logger.Tracef("SQL: %s", q)
		s.logger.Debugf("UserID: %d\n,Username:%s\n,Password:%s\n,Progress Table:%s\n", dto.ID, dto.Username, dto.Password, dto.ProgressTable)
		return fmt.Errorf("failed to Exec due error: %s", err)
	}
	return nil
}

func (s *usersPostgres) Delete(ctx context.Context, id int64) error {
	q := `UPDATE users SET password = DEFAULT, progress_table = DEFAULT, deleted = true WHERE id = $1`
	_, err := s.client.Exec(ctx, q, id)
	if err != nil {
		s.logger.Tracef("SQL: %s", q)
		s.logger.Debugf("UserID: %d", id)
		return fmt.Errorf("failed to Exec due error: %s", err)
	}
	return nil
}
