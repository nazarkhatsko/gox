// Package errs holds the kinds of errors a program returns. Code
// that fails says what kind of failure it is; callers classify errors with
// errors.Is against a kind, and transports map kinds to their own codes
// without knowing where an error came from.
//
// A package with an error of its own declares it next to its code, with a
// kind:
//
//	var ErrBusy = errs.New(errs.ErrConflict, "flow is already queued or running")
//
// so errors.Is matches both ErrBusy and errs.ErrConflict. Kinds are generic;
// nothing here knows about projects, runs or drivers.
package errs

import (
	"errors"
	"fmt"
)

// Kinds.
var (
	// ErrNotFound: the thing asked for does not exist.
	ErrNotFound = errors.New("not found")
	// ErrInvalid: the caller's input is unusable (bad name, config, cron, ...).
	ErrInvalid = errors.New("invalid")
	// ErrConflict: the current state does not allow it (busy, already exists,
	// held by someone else).
	ErrConflict = errors.New("conflict")
	// ErrExhausted: no capacity left (a full queue, a limit reached).
	ErrExhausted = errors.New("exhausted")
	// ErrUnavailable: temporarily not served (shutting down); retry later.
	ErrUnavailable = errors.New("unavailable")
)

// New returns an error of kind with its own message, for a package's
// sentinel errors (var Err... = errs.New(...)).
func New(kind error, msg string) error {
	return &kindError{err: errors.New(msg), kind: kind}
}

// Newf returns an error of kind with a message built like fmt.Errorf (%w
// wraps).
func Newf(kind error, format string, args ...any) error {
	return &kindError{err: fmt.Errorf(format, args...), kind: kind}
}

// NotFound returns an ErrNotFound error with a message built like
// fmt.Errorf (%w wraps).
func NotFound(format string, args ...any) error {
	return Newf(ErrNotFound, format, args...)
}

// Invalid returns an ErrInvalid error with a message built like fmt.Errorf
// (%w wraps).
func Invalid(format string, args ...any) error {
	return Newf(ErrInvalid, format, args...)
}

// kindError is an error of a kind that keeps its own message: it matches
// the kind with errors.Is without appending the kind's text.
type kindError struct {
	err  error
	kind error
}

func (e *kindError) Error() string        { return e.err.Error() }
func (e *kindError) Unwrap() error        { return e.err }
func (e *kindError) Is(target error) bool { return target == e.kind }
