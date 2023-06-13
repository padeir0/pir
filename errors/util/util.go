package util

import (
	. "github.com/padeir0/pir/errors"
)

func NewInternalSemanticError(debug string) *Error {
	return newInternalError(debug)
}

func newInternalError(message string) *Error {
	e := Error(message)
	return &e
}
