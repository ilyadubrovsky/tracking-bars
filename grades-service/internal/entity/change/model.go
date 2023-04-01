package change

import "fmt"

type Change struct {
	ID           int64  `json:"id"`
	UserID       int64  `json:"user_id"`
	Subject      string `json:"subject"`
	ControlEvent string `json:"control_event"`
	OldGrade     string `json:"old_grade"`
	NewGrade     string `json:"new_grade"`
}

func (c *Change) String() string {
	return fmt.Sprintf("*Получение изменение:*\n\n*Название дисциплины:*\n%s\n\n*Контрольное мероприятие:*\n%s\n\n*Старая оценка:*\n%s\n\n"+
		"*Новая оценка:*\n%s", c.Subject, c.ControlEvent, c.OldGrade, c.NewGrade)
}
