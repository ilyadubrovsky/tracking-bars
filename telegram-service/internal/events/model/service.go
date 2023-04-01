package model

type Service interface {
	SendMessageWithOpts(id int64, msg string, opts ...interface{}) error
	EditMessageWithOpts(id int64, messageid int, msg string, opts ...interface{}) error
}
