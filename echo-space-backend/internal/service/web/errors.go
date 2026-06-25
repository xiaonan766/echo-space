package web

import (
	"errors"
)

type BusinessError struct {
	Info string
}

func (e *BusinessError) Error() string {
	return e.Info
}

type NotFoundError struct {
	Info string
}

func (e *NotFoundError) Error() string {
	return e.Info
}

func IsBusinessError(err error) (*BusinessError, bool) {
	var businessError *BusinessError
	if errors.As(err, &businessError) {
		return businessError, true
	}
	return nil, false
}

func IsNotFoundError(err error) (*NotFoundError, bool) {
	var notFoundError *NotFoundError
	if errors.As(err, &notFoundError) {
		return notFoundError, true
	}
	return nil, false
}
