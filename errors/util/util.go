package util

import (
	. "pir/errors"
	et "pir/errors/errorkind"
	sv "pir/errors/severity"
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
