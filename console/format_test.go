package console

import (
	"path/filepath"
	"testing"
	"time"
)

func TestDuration(t *testing.T) {
	cases := map[time.Duration]string{
		8 * time.Millisecond:              "8ms",
		1400 * time.Millisecond:           "1.4s",
		12 * time.Second:                  "12s",
		3*time.Minute + 12*time.Second:    "3m 12s",
		2*time.Hour + 5*time.Minute:       "2h 5m",
		76 * time.Hour:                    "3d 4h",
		5 * time.Second:                   "5s",
		59*time.Second + time.Millisecond: "59s",
	}

	for d, want := range cases {
		if got := Duration(d); got != want {
			t.Errorf("Duration(%v) = %q, want %q", d, got, want)
		}
	}
}

func TestRelative(t *testing.T) {
	now := time.Date(2026, 10, 16, 12, 0, 0, 0, time.Local)
	cases := map[time.Duration]string{
		0:                                "now",
		3*time.Minute + 12*time.Second:   "in 3m 12s",
		-(2*time.Minute + 5*time.Second): "2m 5s ago",
		-500 * time.Millisecond:          "just now",
		48 * time.Hour:                   "2026-10-18 12:00",
	}

	for d, want := range cases {
		if got := Relative(now.Add(d), now); got != want {
			t.Errorf("Relative(now%+v) = %q, want %q", d, got, want)
		}
	}
}

func TestShortPath(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	cases := map[string]string{
		home:                          "~",
		filepath.Join(home, "a", "b"): filepath.Join("~", "a", "b"),
		home + "other":                home + "other",
		"/etc/hosts":                  "/etc/hosts",
	}

	for in, want := range cases {
		if got := ShortPath(in); got != want {
			t.Errorf("ShortPath(%q) = %q, want %q", in, got, want)
		}
	}
}
