package service

// TODO тут нужно будет завязаться на опции сервиса, а не телебота
type Telegram interface {
	SendMessageWithOpts(id int64, message string, opts ...interface{}) error
	EditMessageWithOpts(id int64, messageID int, msg string, opts ...interface{}) error
	Start()
	Stop()
}
