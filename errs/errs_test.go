package errs

import (
	"errors"
	"fmt"
	"testing"
)

func TestNewMatchesItselfAndKind(t *testing.T) {
	errBusy := New(ErrConflict, "flow is busy")
	err := fmt.Errorf("run p/a: %w", errBusy)

	if !errors.Is(err, errBusy) || !errors.Is(err, ErrConflict) {
		t.Errorf("%v must match its sentinel and its kind", err)
	}

	if errors.Is(err, ErrNotFound) || errors.Is(err, New(ErrConflict, "flow is busy")) {
		t.Errorf("%v must not match another kind or another sentinel", err)
	}

	if err.Error() != "run p/a: flow is busy" {
		t.Errorf("message = %q, want the kind's text left out", err.Error())
	}
}

func TestNewfWraps(t *testing.T) {
	cause := errors.New("bad cron")
	err := Invalid("trigger: %w", cause)

	if !errors.Is(err, ErrInvalid) || !errors.Is(err, cause) {
		t.Errorf("%v must match its kind and the wrapped cause", err)
	}

	if err.Error() != "trigger: bad cron" {
		t.Errorf("message = %q", err.Error())
	}
}

func TestKindOfAWrappedKind(t *testing.T) {
	// A store's ErrNotFound is the kind itself; a service may wrap it again.
	err := NotFound("run #3: %w", fmt.Errorf("get run: %w", ErrNotFound))

	if !errors.Is(err, ErrNotFound) {
		t.Errorf("%v must match ErrNotFound", err)
	}
}
