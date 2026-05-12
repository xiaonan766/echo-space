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

func IsBusinessError(err error) (*BusinessError, bool) {
	var businessError *BusinessError
	if errors.As(err, &businessError) {
		return businessError, true
	}
	return nil, false
}
