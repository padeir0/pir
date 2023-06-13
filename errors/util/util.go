package util

import (
	. "github.com/padeir0/pir/errors"
	et "github.com/padeir0/pir/errors/errorkind"
	sv "github.com/padeir0/pir/errors/severity"
)

func NewInternalSemanticError(debug string) *Error {
	return newInternalError(debug)
}

func newInternalError(message string) *Error {
	return &Error{
		Code:     et.InternalCompilerError,
		Severity: sv.InternalError,
		Message:  message,
	}
}
