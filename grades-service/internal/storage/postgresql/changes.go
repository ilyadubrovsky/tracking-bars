package storage

import (
	"context"
	"fmt"
	"grades-service/internal/entity/change"
	"grades-service/pkg/logging"
	"time"
)

const (
	changes = "changes"
)

type changesStorage struct {
	client Client
	logger *logging.Logger
}

func NewChangesPostgres(client Client, logger *logging.Logger) *changesStorage {
	return &changesStorage{
		client: client,
		logger: logger,
	}
}

func (s *changesStorage) Create(ctx context.Context, c change.Change) error {
	q := fmt.Sprintf("INSERT INTO %s (user_id, subject, control_event, old_grade, new_grade, created_at)"+
		"VALUES ($1, $2, $3, $4, $5, $6)", changes)
	s.logger.Tracef("SQL: %s", q)

	if _, err := s.client.Exec(ctx, q, c.UserID, c.Subject, c.ControlEvent, c.OldGrade, c.NewGrade, time.Now()); err != nil {
		s.logger.Debugf("UserID: %d\n Subject: %s\n Control event: %s\n Old grade: %s\n New grade: %s", c.UserID, c.Subject, c.ControlEvent, c.OldGrade, c.NewGrade)
		return err
	}

	return nil
}
