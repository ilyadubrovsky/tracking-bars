package domain

// TODO разделить на Password и RawPassword
type BarsCredentials struct {
	Username string
	Password []byte
}
