package errors

import "errors"

var (
	ErrAlreadyAuth = errors.New("user already authorized")
)
