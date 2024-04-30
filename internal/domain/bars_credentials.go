package domain

// TODO разделить на Password и RawPassword
type BarsCredentials struct {
	UserID   int64
	Username string
	Password []byte
}
