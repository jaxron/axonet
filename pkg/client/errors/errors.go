package errors

import (
	"errors"
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
)

// IsTemporary returns true if the error is considered temporary and can be retried.
func IsTemporary(err error) bool {
	return (errors.Is(err, ErrNetwork) ||
		errors.Is(err, ErrTimeout) ||
		errors.Is(err, ErrBadStatus) ||
		errors.Is(err, ErrTemporary)) &&
		!errors.Is(err, ErrPermanent)
}

// Is reports whether any error in err's chain is an instance of target.
func Is(err, target error) bool {
	return errors.Is(err, target)
}
