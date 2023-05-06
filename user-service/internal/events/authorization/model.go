package authorization

type AuthorizationRequest struct {
	RequestID int64  `json:"request_id"` // user who made the request and will receive an answer
	UserID    int64  `json:"user_id"`    // user who will authorization
	Username  string `json:"username"`
	Password  string `json:"password"`
}
