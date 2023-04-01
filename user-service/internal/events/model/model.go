package model

type SendMessageRequest struct {
	RequestID int64  `json:"request_id"`
	Message   string `json:"message"`
	ParseMode string `json:"parse_mode"`
}
