package apperror

import "errors"

var (
	ErrAlreadyAuthorized = errors.New("user already authorized")
	ErrNotAuthorized     = errors.New("user not authorized")
)
