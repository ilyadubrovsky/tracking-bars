package user

import (
	"github.com/ilyadubrovsky/bars"
	"grades-service/pkg/utils/aes"
	"os"
)

type User struct {
	ID            int64              `json:"id"`
	Username      string             `json:"username"`
	Password      []byte             `json:"password"`
	ProgressTable bars.ProgressTable `json:"progress_table"`
	Deleted       bool               `json:"deleted"`
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
