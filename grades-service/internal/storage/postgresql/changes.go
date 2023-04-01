package storage

import (
	"context"
	"grades-service/internal/entity/change"
	"grades-service/pkg/client/postgresql"
	"grades-service/pkg/logging"
)

type changesStorage struct {
	client postgresql.Client
	logger *logging.Logger
}

func NewChangesPostgres(client postgresql.Client, logger *logging.Logger) change.Repository {
	return &changesStorage{
		client: client,
		logger: logger,
	}
}

func (s *changesStorage) Create(ctx context.Context, c change.Change) error {
	q := `INSERT INTO changes (user_id, subject, control_event, old_grade, new_grade) VALUES ($1, $2, $3, $4, $5)`
	s.logger.Tracef("SQL: %s", q)

	if _, err := s.client.Exec(ctx, q, c.UserID, c.Subject, c.ControlEvent, c.OldGrade, c.NewGrade); err != nil {
		s.logger.Debugf("UserID: %d\n Subject: %s\n Control event: %s\n Old grade: %s\n New grade: %s", c.UserID, c.Subject, c.ControlEvent, c.OldGrade, c.NewGrade)
		return err
	}

	return nil
}
