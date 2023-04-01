package grades

import "telegram-service/internal/entity/user"

type GetGradesResponse struct {
	RequestID       int64              `json:"request_id"`
	ProgressTable   user.ProgressTable `json:"progress_table"`
	ResponseMessage string             `json:"response_message"`
	IsCallback      bool               `json:"is_callback"`
	CallbackData    string             `json:"callback_data"`
	MessageID       int                `json:"message_id"`
}
