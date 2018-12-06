package client

import (
	"github.com/juju/errors"
)

type ErrorResourceTypeNotFound struct {
	message string
}

func NewTypeNotFoundError(message string) ErrorResourceTypeNotFound {
	return ErrorResourceTypeNotFound{message: message}
}

func (e ErrorResourceTypeNotFound) Error() string {
	return e.message
}

func IsResourceTypeNotFound(err error) bool {
	_, ok := errors.Cause(err).(ErrorResourceTypeNotFound)
	return ok
}
