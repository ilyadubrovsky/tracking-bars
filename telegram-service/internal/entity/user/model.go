package user

import (
	"fmt"
)

type User struct {
	ID            int64         `json:"id"`
	Username      string        `json:"username"`
	Password      []byte        `json:"password"`
	ProgressTable ProgressTable `json:"progress_table"`
	Deleted       bool          `json:"deleted"`
}

type ProgressTable struct {
	Tables []SubjectTable `json:"tables"`
}

func (pt *ProgressTable) String() string {
	result := ""
	for _, st := range pt.Tables {
		result += fmt.Sprintf("%s\n", st.String())
	}
	return result
}

type SubjectTable struct {
	Name string            `json:"name"`
	Rows []SubjectTableRow `json:"rows"`
}

func (st *SubjectTable) String() string {
	result := fmt.Sprintf("*Название дисциплины:*\n%s\n\n", st.Name)
	for _, str := range st.Rows {
		result += str.String() + "\n"
	}
	return result
}

type SubjectTableRow struct {
	Name   string `json:"name"`
	Grades string `json:"grades"`
}

func (str *SubjectTableRow) String() string {
	return fmt.Sprintf("%s\n*Оценка:* %s\n", str.Name, str.Grades)
}
