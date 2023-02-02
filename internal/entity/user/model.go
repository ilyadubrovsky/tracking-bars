package user

import (
	"errors"
	"fmt"
	"unicode/utf8"
)

var (
	ErrIncorrectData = errors.New("the received data is incorrect")
)

type User struct {
	ID            int64
	Username      string
	Password      []byte
	ProgressTable ProgressTable
	Deleted       bool
}

type ProgressTable struct {
	Tables []SubjectTable `json:"pt_table"`
}

func (pt *ProgressTable) String() string {
	result := ""
	for _, st := range pt.Tables {
		result += fmt.Sprintf("%s\n", st.String())
	}
	return result
}

type SubjectTable struct {
	Name string            `json:"st_name"`
	Rows []SubjectTableRow `json:"st_rows"`
}

func (st *SubjectTable) String() string {
	result := fmt.Sprintf("*Название дисциплины:*\n%s\n\n", st.Name)
	for _, str := range st.Rows {
		result += str.String() + "\n"
	}
	return result
}

type SubjectTableRow struct {
	Name   string `json:"str_name"`
	Grades string `json:"str_grades"`
}

func (str *SubjectTableRow) String() string {
	return fmt.Sprintf("%s\n*Оценка:* %s\n", str.Name, str.Grades)
}

type CreateUserDTO struct {
	ID            int64  `json:"id,omitempty"`
	Username      string `json:"username,omitempty"`
	Password      []byte `json:"password,omitempty"`
	ProgressTable string `json:"progress_table,omitempty"`
}

type UpdateUserDTO struct {
	ID            int64  `json:"id,omitempty"`
	Username      string `json:"username,omitempty"`
	Password      []byte `json:"password,omitempty"`
	ProgressTable string `json:"progress_table,omitempty"`
	Deleted       bool   `json:"deleted"`
}

func (pt *ProgressTable) ValidateData() error {
	if check := utf8.Valid([]byte(pt.String())); check == false {
		return ErrIncorrectData
	}
	return nil
}
