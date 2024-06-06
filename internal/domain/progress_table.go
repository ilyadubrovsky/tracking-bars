package domain

import "fmt"

type ProgressTable struct {
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
	Name          string
	ControlEvents []ControlEvent
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
	Name  string
	Grade string
}

func (ce *ControlEvent) String() string {
	return fmt.Sprintf("%s\n*Оценка:* %s\n", ce.Name, ce.Grade)
}
