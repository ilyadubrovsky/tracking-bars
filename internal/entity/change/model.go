package change

import "fmt"

type Change struct {
	UserID       int64  `json:"user_id"`
	ID           int64  `json:"id"`
	Subject      string `json:"subject"`
	ControlEvent string `json:"control_event"`
	OldGrade     string `json:"old_grade"`
	NewGrade     string `json:"new_grade"`
}

func (c *Change) String() string {
	return fmt.Sprintf("*Название дисциплины:*\n%s\n\n*Контрольное мероприятие:*\n%s\n\n*Старая оценка:*\n%s\n\n"+
		"*Новая оценка:*\n%s", c.Subject, c.ControlEvent, c.OldGrade, c.NewGrade)
}

type CreateChangeDTO struct {
	UserID       int64  `json:"user_id,omitempty"`
	Subject      string `json:"subject,omitempty"`
	ControlEvent string `json:"control_event,omitempty"`
	OldGrade     string `json:"old_grade,omitempty"`
	NewGrade     string `json:"new_grade,omitempty"`
}

type UpdateChangeDTO struct {
	Subject      string `json:"subject"`
	ControlEvent string `json:"control_event"`
	OldGrade     string `json:"old_grade"`
	NewGrade     string `json:"new_grade"`
}
