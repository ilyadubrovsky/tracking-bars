package deleteuser

type DeleteUserRequest struct {
	RequestID    int64 `json:"request_id"`    // user who made the request and will receive an answer
	UserID       int64 `json:"user_id"`       // user who will deleteuser
	SendResponse bool  `json:"send_response"` // send response to request id?
}
