package user

import (
	"errors"
	"fmt"
	"grades-service/pkg/utils/aes"
	"os"
	"unicode/utf8"
)

var (
	ErrIncorrectData = errors.New("the received data is incorrect")
)

type User struct {
	ID            int64         `json:"id"`
	Username      string        `json:"username"`
	Password      []byte        `json:"password"`
	ProgressTable ProgressTable `json:"progress_table"`
	Deleted       bool          `json:"deleted"`
}

func (u *User) EncryptPassword() error {
	encryptedPassword, err := aes.EncryptAES([]byte(os.Getenv("ENCRYPTION_KEY")), u.Password)

	if err != nil {
		return err
	}

	u.Password = encryptedPassword

	return nil
}

func (u *User) DecryptPassword() (string, error) {
	return aes.DecryptAES([]byte(os.Getenv("ENCRYPTION_KEY")), u.Password)
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

func (pt *ProgressTable) ValidateData() error {
	if !utf8.ValidString(pt.String()) {
		return ErrIncorrectData
	}

	return nil
}
