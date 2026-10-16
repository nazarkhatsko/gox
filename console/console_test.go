package console

import (
	"bytes"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
)

var ansi = regexp.MustCompile(`\x1b\[[0-9;]*m`)

func plain(t *testing.T) (*Console, *bytes.Buffer) {
	t.Helper()

	var b bytes.Buffer

	return New(&b), &b
}

func TestMarks(t *testing.T) {
	c, b := plain(t)
	c.Success("saved %s", "config")
	c.Failure("lint failed")
	c.Warn("deprecated %q", "-x")
	c.Hint("run: %s", "task init")
	c.Line("done")

	want := "✓ saved config\n✗ lint failed\n! deprecated \"-x\"\n  run: task init\ndone\n"
	if got := b.String(); got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestNoColorsWhenNotATerminal(t *testing.T) {
	c, b := plain(t)
	c.Success("ok")

	if ansi.MatchString(b.String()) {
		t.Errorf("unexpected color codes in %q", b.String())
	}
}

func TestWithColors(t *testing.T) {
	var on, off bytes.Buffer

	New(&on, WithColors(true)).Success("ok")
	New(&off, WithColors(false)).Success("ok")

	if !ansi.MatchString(on.String()) {
		t.Errorf("WithColors(true): no color codes in %q", on.String())
	}

	if ansi.MatchString(off.String()) {
		t.Errorf("WithColors(false): color codes in %q", off.String())
	}
}

func TestWithPalette(t *testing.T) {
	var b bytes.Buffer

	c := New(&b, WithColors(true), WithPalette(Palette{OK: lipgloss.Color("#00FF00")}))

	if got := c.OK("x"); !strings.Contains(got, "38;2;0;255;0") {
		t.Errorf("OK = %q, want true-color green", got)
	}

	if got := c.Fail("x"); got != "x" {
		t.Errorf("Fail with a nil color = %q, want plain text", got)
	}
}

func TestError(t *testing.T) {
	cases := map[string]struct {
		err  error
		want string
	}{
		"nil":      {nil, ""},
		"plain":    {errors.New("boom"), "✗ boom\n"},
		"reported": {fmt.Errorf("%w: %w", ErrReported, errors.New("boom")), ""},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			c, b := plain(t)
			c.Error(tc.err)

			if got := b.String(); got != tc.want {
				t.Errorf("got %q, want %q", got, tc.want)
			}
		})
	}
}

func TestKeyValues(t *testing.T) {
	c, b := plain(t)
	c.KeyValues(KV{Key: "Name", Value: "gox"}, KV{Key: "Version", Value: "v0.1.0"})

	want := "  Name     gox\n  Version  v0.1.0\n"
	if got := b.String(); got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestTable(t *testing.T) {
	c, b := plain(t)
	c.Table([]string{"name", "status"}, [][]string{
		{"httpx", "ok"},
		{"console", "new"},
		{"tag"},
	})

	want := "" +
		"  NAME      STATUS\n" +
		"  httpx     ok\n" +
		"  console   new\n" +
		"  tag\n"
	if got := b.String(); got != want {
		t.Errorf("got:\n%s\nwant:\n%s", got, want)
	}
}

func TestTableAlignsStyledCells(t *testing.T) {
	var b bytes.Buffer

	c := New(&b, WithColors(true))
	c.Table([]string{"name", "status"}, [][]string{
		{c.OK("httpx"), "ok"},
		{"console", "new"},
	})

	want := "" +
		"  NAME      STATUS\n" +
		"  httpx     ok\n" +
		"  console   new\n"
	if got := ansi.ReplaceAllString(b.String(), ""); got != want {
		t.Errorf("got:\n%s\nwant:\n%s", got, want)
	}
}

func TestTableNoTrailingSpaces(t *testing.T) {
	var b bytes.Buffer

	New(&b, WithColors(true)).Table([]string{"name", "status"}, [][]string{{"httpx", "a-long-status"}})

	for line := range strings.Lines(b.String()) {
		if plain := ansi.ReplaceAllString(line, ""); strings.HasSuffix(plain, " \n") {
			t.Errorf("trailing spaces in %q", plain)
		}
	}
}
