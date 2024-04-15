package errors

import "errors"

var (
	ErrAlreadyAuth   = errors.New("user is already authorized")
	ErrNotAuthorized = errors.New("user is not authorized")
)
