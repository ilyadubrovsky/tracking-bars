package dbo

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/ilyadubrovsky/tracking-bars/internal/domain"
)

type GradeChange struct {
	ID        int64
	UserID    int64
	Data      []byte
	CreatedAt time.Time
}

func (dbo *GradeChange) ToDomain() (*domain.GradeChange, error) {
	data := gradeChangeData{}
	if err := json.Unmarshal(dbo.Data, &data); err != nil {
		return nil, fmt.Errorf("json.Unmarshal: %w", err)
	}

	return &domain.GradeChange{
		ID:           dbo.ID,
		UserID:       dbo.UserID,
		Discipline:   data.Discipline,
		ControlEvent: data.ControlEvent,
		OldGrade:     data.OldGrade,
		NewGrade:     data.NewGrade,
	}, nil
}

type gradeChangeData struct {
	Discipline   string `json:"discipline"`
	ControlEvent string `json:"control_event"`
	OldGrade     string `json:"old_grade"`
	NewGrade     string `json:"new_grade"`
}

func GradeChangeDataFromDomain(gradeChange *domain.GradeChange) ([]byte, error) {
	bytes, err := json.Marshal(&gradeChangeData{
		Discipline:   gradeChange.Discipline,
		ControlEvent: gradeChange.ControlEvent,
		OldGrade:     gradeChange.OldGrade,
		NewGrade:     gradeChange.NewGrade,
	})
	if err != nil {
		return nil, fmt.Errorf("json.Marshal: %w", err)
	}

	return bytes, nil
}
