package news

type SendNewsRequest struct {
	RequestID int64  `json:"request_id"`
	Type      string `json:"type"`
	Message   string `json:"message"`
	ParseMode string `json:"parse_mode"`
}
