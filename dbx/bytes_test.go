package dbx

import "testing"

func TestNonNil(t *testing.T) {
	if got := NonNil(nil); got == nil || len(got) != 0 {
		t.Errorf("NonNil(nil) = %#v, want an empty slice", got)
	}

	b := []byte("x")
	if got := NonNil(b); string(got) != "x" {
		t.Errorf("NonNil(%q) = %q", b, got)
	}
}
