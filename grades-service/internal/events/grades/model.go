package grades

type GetGradesRequest struct {
	RequestID    int64  `json:"request_id"`
	IsCallback   bool   `json:"is_callback"`
	CallbackData string `json:"callback_data"`
	MessageID    int    `json:"message_id"`
}
