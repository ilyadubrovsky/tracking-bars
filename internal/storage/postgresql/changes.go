package storage

import (
	"TrackingBARSv2/internal/entity/change"
	"TrackingBARSv2/pkg/client/postgresql"
	"TrackingBARSv2/pkg/logging"
	"context"
	"fmt"
	"github.com/jackc/pgx/v4"
)

type changesStorage struct {
	client postgresql.Client
	logger logging.Logger
}

func NewChangesPostgres(client postgresql.Client, logger logging.Logger) change.Repository {
	return &changesStorage{
		client: client,
		logger: logger,
	}
}

func (s *changesStorage) Create(ctx context.Context, dto change.CreateChangeDTO) error {
	q := `INSERT INTO changes (user_id, subject, control_event, old_grade, new_grade) VALUES ($1, $2, $3, $4, $5)`
	if _, err := s.client.Exec(ctx, q, dto.UserID, dto.Subject, dto.ControlEvent, dto.OldGrade, dto.NewGrade); err != nil {
		s.logger.Tracef("SQL: %s", q)
		s.logger.Debugf("UserID: %d\nSubject: %s\nControl event: %s\n Old grade: %s\n New grade: %s", dto.UserID, dto.Subject, dto.ControlEvent, dto.OldGrade, dto.NewGrade)
		return fmt.Errorf("failed to Exec due error: %s", err)
	}
	return nil
}

func (s *changesStorage) FindAll(ctx context.Context) ([]change.Change, error) {
	q := `SELECT id, user_id, subject, control_event, old_grade, new_grade FROM changes`
	changesRows, err := s.client.Query(ctx, q)
	if err != nil {
		s.logger.Tracef("SQL: %s", q)
		s.logger.Debugf("Raw Values: %s", changesRows.RawValues())
		return nil, fmt.Errorf("failed to Query due error: %s", err)
	}

	cs := make([]change.Change, 0)
	for changesRows.Next() {
		var c change.Change
		if err = changesRows.Scan(&c.ID, &c.UserID, &c.Subject, &c.ControlEvent, &c.OldGrade, &c.NewGrade); err != nil {
			s.logger.Tracef("SQL: %s", q)
			s.logger.Debugf("Raw Values: %s", changesRows.RawValues())
			return nil, fmt.Errorf("failed to Scan when reading Rows due error: %s", err)
		}
		cs = append(cs, c)
	}

	if err = changesRows.Err(); err != nil {
		s.logger.Tracef("SQL: %s", q)
		s.logger.Debugf("Raw Values: %s", changesRows.RawValues())
		return nil, fmt.Errorf("failed to handle table rows due error : %s", err)
	}

	return cs, nil
}

func (s *changesStorage) FindOne(ctx context.Context, id int) (change.Change, error) {
	q := `SELECT id, user_id, subject, control_event, old_grade, new_grade FROM changes WHERE id = $1`

	var c change.Change
	changeRow := s.client.QueryRow(ctx, q, id)
	err := changeRow.Scan(&c.ID, &c.UserID, &c.Subject, &c.ControlEvent, &c.OldGrade, &c.NewGrade)
	if err == pgx.ErrNoRows {
		return change.Change{}, nil
	}
	if err != nil {
		s.logger.Debugf("SubjectID: %d", id)
		return change.Change{}, fmt.Errorf("failed to Scan when reading Rows: %s", err)
	}
	return c, nil
}

func (s *changesStorage) Update(ctx context.Context, dto change.UpdateChangeDTO) error {
	q := `UPDATE changes SET subject = $2, control_event = $3, old_grade = $4, new_grade = $5 WHERE id = $1`
	_, err := s.client.Exec(ctx, q, dto.Subject, dto.ControlEvent, dto.OldGrade, dto.NewGrade)
	if err != nil {
		s.logger.Tracef("SQL: %s", q)
		s.logger.Debugf("Subject: %s\n Control event: %s\n Old grade: %s\n New grade: %s\n", dto.Subject, dto.ControlEvent, dto.OldGrade, dto.NewGrade)
		return fmt.Errorf("failed to Exec due error: %s", err)
	}
	return nil
}

func (s *changesStorage) Delete(ctx context.Context, id int) error {
	q := `DELETE FROM changes WHERE id = $1`
	_, err := s.client.Exec(ctx, q, id)
	if err != nil {
		s.logger.Tracef("SQL: %s", q)
		s.logger.Debugf("ID: %d", id)
		return fmt.Errorf("failed to Exec due error: %s", err)
	}
	return nil
}
