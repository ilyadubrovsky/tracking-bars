package domain

type BarsCredentials struct {
	ID       string
	UserID   int64
	Username string
	Password []byte
}
