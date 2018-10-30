package client

import "github.com/juju/errors"

var (
	ErrTypeNotFind = errors.New("find apiResource for type error")
)

func IsResourceTypeNotFindError(err error) bool {
	return errors.Cause(err) == ErrTypeNotFind
}
