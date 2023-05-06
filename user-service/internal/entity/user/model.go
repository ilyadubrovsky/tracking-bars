package user

import (
	"github.com/ilyadubrovsky/bars"
	"os"
	"user-service/pkg/utils/aes"
)

type User struct {
	ID            int64              `json:"id"`
	Username      string             `json:"username"`
	Password      []byte             `json:"password"`
	ProgressTable bars.ProgressTable `json:"progress_table"`
	Deleted       bool               `json:"deleted"`
}

func (u *User) EncryptPassword() error {
	encryptedPassword, err := aes.EncryptCFB([]byte(os.Getenv("ENCRYPTION_KEY")), u.Password)

	if err != nil {
		return err
	}

	u.Password = encryptedPassword

	return nil
}

func (u *User) DecryptPassword() (string, error) {
	return aes.DecryptCFB([]byte(os.Getenv("ENCRYPTION_KEY")), u.Password)
}

type CreateUserDTO struct {
	ID       int64  `json:"id"`
	Username string `json:"username"`
	Password string `json:"password"`
}

func NewUser(dto CreateUserDTO) User {
	return User{
		ID:            dto.ID,
		Username:      dto.Username,
		Password:      []byte(dto.Password),
		ProgressTable: bars.ProgressTable{},
		Deleted:       false,
	}
}
