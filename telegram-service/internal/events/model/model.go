package model

type LogoutRequest struct {
	RequestID int64 `json:"request_id"` // user who made the request and will receive an answer
	UserID    int64 `json:"user_id"`    // user who will logout
}

type AuthorizationRequest struct {
	RequestID int64  `json:"request_id"` // user who made the request and will receive an answer
	UserID    int64  `json:"user_id"`    // user who will authorization
	Username  string `json:"username"`
	Password  string `json:"password"`
}

type DeleteUserRequest struct {
	RequestID    int64 `json:"request_id"`    // user who made the request and will receive an answer
	UserID       int64 `json:"user_id"`       // user who will delete
	SendResponse bool  `json:"send_response"` // send response to request id?
}

type SendNewsRequest struct {
	RequestID int64  `json:"request_id"`
	Type      string `json:"type"`
	Message   string `json:"message"`
	ParseMode string `json:"parse_mode"`
}

type GetGradesRequest struct {
	RequestID    int64  `json:"request_id"`
	IsCallback   bool   `json:"is_callback"`
	CallbackData string `json:"callback_data"`
	MessageID    int    `json:"message_id"`
}
