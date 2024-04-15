package dbo

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/ilyadubrovsky/tracking-bars/internal/domain"
)

type ProgressTable struct {
	UserID        int64
	ProgressTable []byte
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

type progressTableData struct {
	DisciplinesData []disciplineData `json:"progress_table"`
}

type disciplineData struct {
	Name          string             `json:"name"`
	ControlEvents []controlEventData `json:"control_events"`
}

type controlEventData struct {
	Name  string `json:"name"`
	Grade string `json:"grade"`
}

func FromDomain(progressTable *domain.ProgressTable) (*ProgressTable, error) {
	data := progressTableData{
		DisciplinesData: make([]disciplineData, len(progressTable.Disciplines)),
	}
	for i, discipline := range progressTable.Disciplines {
		data.DisciplinesData[i].Name = discipline.Name
		controlEventsData := make([]controlEventData, len(discipline.ControlEvents))
		for j, controlEvent := range progressTable.Disciplines[i].ControlEvents {
			controlEventsData[j].Name = controlEvent.Name
			controlEventsData[j].Grade = controlEvent.Grade
		}
		data.DisciplinesData[i].ControlEvents = controlEventsData
	}

	progressTableBytes, err := json.Marshal(data)
	if err != nil {
		return nil, fmt.Errorf("json.Marshal: %w", err)
	}

	return &ProgressTable{
		UserID:        progressTable.UserID,
		ProgressTable: progressTableBytes,
	}, nil
}

func ToDomain(dboProgressTable *ProgressTable) (*domain.ProgressTable, error) {
	data := progressTableData{}
	if err := json.Unmarshal(dboProgressTable.ProgressTable, &data); err != nil {
		return nil, fmt.Errorf("json.Unmarshal: %w", err)
	}

	progressTable := &domain.ProgressTable{
		UserID:      dboProgressTable.UserID,
		Disciplines: make([]domain.Discipline, 0, len(data.DisciplinesData)),
	}
	for _, dboDiscipline := range data.DisciplinesData {
		controlEvents := make([]domain.ControlEvent, 0, len(dboDiscipline.ControlEvents))
		for _, dboControlEvent := range dboDiscipline.ControlEvents {
			controlEvents = append(controlEvents, domain.ControlEvent{
				Name:  dboControlEvent.Name,
				Grade: dboControlEvent.Grade,
			})
		}

		progressTable.Disciplines = append(progressTable.Disciplines, domain.Discipline{
			Name:          dboDiscipline.Name,
			ControlEvents: controlEvents,
		})
	}

	return progressTable, nil
}
