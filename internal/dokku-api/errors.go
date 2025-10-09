package dokkuApi

import (
	"errors"
	"fmt"
)

// ErrAppNotFound is the sentinel error for missing Dokku applications.
var ErrAppNotFound = errors.New("app not found")

// NotFoundError indicates the target Dokku application/resource does not exist.
type NotFoundError struct {
	Command string
	Err     error
}

func (e *NotFoundError) Error() string {
	if e == nil {
		return ""
	}
	if e.Command != "" {
		return fmt.Sprintf("%s: not found: %v", e.Command, e.Err)
	}
	return fmt.Sprintf("not found: %v", e.Err)
}

func (e *NotFoundError) Unwrap() error { return e.Err }

// IsNotFoundError returns true when err is (or wraps) a NotFoundError.
func IsNotFoundError(err error) bool {
	if err == nil {
		return false
	}
	var nf *NotFoundError
	if errors.As(err, &nf) {
		return true
	}
	return errors.Is(err, ErrAppNotFound)
}
