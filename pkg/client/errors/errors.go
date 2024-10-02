package errors

import (
	"errors"
)

var (
	ErrTemporary = errors.New("temporary error")
	ErrPermanent = errors.New("permanent error")

	ErrUnreachable     = errors.New("unreachable code")
	ErrUnmarshalResult = errors.New("unmarshal result not set")

	ErrRequestCreation     = errors.New("request creation error")
	ErrBodyMarshalConflict = errors.New("body and marshal body conflict")

	ErrNoCookie  = errors.New("no cookie found")
	ErrNetwork   = errors.New("network error")
	ErrTimeout   = errors.New("timeout error")
	ErrBadStatus = errors.New("bad status code")

	ErrSingleFlight      = errors.New("single flight error")
	ErrRetryFailed       = errors.New("retry failed")
	ErrRateLimitExceeded = errors.New("rate limit exceeded")
	ErrInvalidTransport  = errors.New("invalid transport")
	ErrCircuitOpen       = errors.New("circuit breaker is open")
	ErrCircuitExhausted  = errors.New("circuit breaker is exhausted")
)

// IsTemporary returns true if the error is considered temporary and can be retried.
func IsTemporary(err error) bool {
	return errors.Is(err, ErrNetwork) || errors.Is(err, ErrTimeout) || errors.Is(err, ErrBadStatus) || errors.Is(err, ErrTemporary)
}

// Is reports whether any error in err's chain is an instance of target.
func Is(err, target error) bool {
	return errors.Is(err, target)
}
