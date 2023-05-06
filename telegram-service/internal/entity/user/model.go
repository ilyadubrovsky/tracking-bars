package user

import (
	"github.com/ilyadubrovsky/bars"
)

type User struct {
	ID            int64              `json:"id"`
	Username      string             `json:"username"`
	Password      []byte             `json:"password"`
	ProgressTable bars.ProgressTable `json:"progress_table"`
	Deleted       bool               `json:"deleted"`
}
