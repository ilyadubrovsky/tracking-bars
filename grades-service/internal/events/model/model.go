package model

import "github.com/ilyadubrovsky/bars"

type GetGradesResponse struct {
	RequestID       int64              `json:"request_id"`
	ProgressTable   bars.ProgressTable `json:"progress_table"`
	ResponseMessage string             `json:"response_message"`
	IsCallback      bool               `json:"is_callback"`
	CallbackData    string             `json:"callback_data"`
	MessageID       int                `json:"message_id"`
}

type SendMessageRequest struct {
	RequestID int64  `json:"request_id"`
	Message   string `json:"message"`
	ParseMode string `json:"parse_mode"`
}
