package errors

import "errors"

var (
	ErrAlreadyAuth                = errors.New("user is already authorized")
	ErrProgressTableStructChanged = errors.New("progress table structure has been changed")
	ErrWrongGradesPage            = errors.New("wrong grades page")
)
