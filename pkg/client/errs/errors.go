package errs

import (
	"errors"
	"fmt"
)

var (
	ErrTemporary   = errors.New("temporary error")
	ErrPermanent   = errors.New("permanent error")
	ErrUnreachable = errors.New("unreachable code")

	ErrRequestCreation     = errors.New("request creation error")
	ErrBodyMarshalConflict = errors.New("body and marshal body conflict")

	ErrNetwork   = errors.New("network error")
	ErrTimeout   = errors.New("timeout error")
	ErrBadStatus = errors.New("bad status code")

	ErrEmptyResponseBody = errors.New("response body is empty")
)

// StatusError wraps ErrBadStatus with the actual HTTP status code.
type StatusError struct {
	StatusCode int
}

func (e *StatusError) Error() string {
	return fmt.Sprintf("bad status code: %d", e.StatusCode)
}

func (e *StatusError) Unwrap() error {
	return ErrBadStatus
}

// IsTemporary returns true if the error is considered temporary and can be retried.
func IsTemporary(err error) bool {
	return (errors.Is(err, ErrNetwork) ||
		errors.Is(err, ErrTimeout) ||
		errors.Is(err, ErrTemporary)) &&
		!errors.Is(err, ErrPermanent)
}

// Is reports whether any error in err's chain is an instance of target.
func Is(err, target error) bool {
	return errors.Is(err, target)
}
