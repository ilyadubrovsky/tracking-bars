package domain

import "fmt"

type GradeChange struct {
	UserID       int64
	Discipline   string
	ControlEvent string
	OldGrade     string
	NewGrade     string
}

// TODO это явно не логика для домеина, нужно переделать
func (c *GradeChange) String() string {
	return fmt.Sprintf("*Получение изменение:*\n\n*Название дисциплины:*\n%s\n\n*Контрольное мероприятие:*\n%s\n\n*Старая оценка:*\n%s\n\n"+
		"*Новая оценка:*\n%s", c.Discipline, c.ControlEvent, c.OldGrade, c.NewGrade)
}
