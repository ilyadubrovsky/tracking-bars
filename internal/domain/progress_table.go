package domain

import "fmt"

type ProgressTable struct {
	UserID      int64
	Disciplines []Discipline
}

func (pt *ProgressTable) String() string {
	str := ""
	for _, discipline := range pt.Disciplines {
		str += fmt.Sprintf("%s\n", discipline.String())
	}
	return str
}

type Discipline struct {
	Name          string         `json:"name"`
	ControlEvents []ControlEvent `json:"control_events"`
}

// TODO это явно не логика для домеина, нужно переделать
func (d *Discipline) String() string {
	str := fmt.Sprintf("*Название дисциплины:*\n%s\n\n", d.Name)
	for _, ce := range d.ControlEvents {
		str += fmt.Sprintf("%s\n", ce.String())
	}
	return str
}

type ControlEvent struct {
	Name  string `json:"name"`
	Grade string `json:"grade"`
}

func (ce *ControlEvent) String() string {
	return fmt.Sprintf("%s\n*Оценка:* %s\n", ce.Name, ce.Grade)
}
