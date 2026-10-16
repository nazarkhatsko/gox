// Package console renders human-facing CLI output: status lines, hints,
// key/value blocks and borderless tables. Colors come from a palette and are
// dropped automatically when the writer is not a terminal or NO_COLOR is set.
//
//	c := console.New(os.Stdout)
//	c.Success("saved %s", name) // ✓ saved config
//	c.Failure("lint failed")    // ✗ lint failed
//	c.Hint("run: task init")    //   run: task init
//	c.KeyValues(console.KV{Key: "Version", Value: "v0.1.0"})
package console

import (
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"
)

// ErrReported marks an error whose failure output was already printed;
// Error then prints nothing. Wrap it to keep the cause:
// fmt.Errorf("%w: %w", console.ErrReported, err).
var ErrReported = errors.New("reported")

// Palette holds the colors a Console renders with. A nil color leaves text
// in the terminal's default color.
type Palette struct {
	Accent lipgloss.TerminalColor // titles, warnings
	OK     lipgloss.TerminalColor // success marks
	Fail   lipgloss.TerminalColor // failure marks
	Text   lipgloss.TerminalColor // bold primary text
	Muted  lipgloss.TerminalColor // hints, keys, table headers
}

// DefaultPalette uses the 16 ANSI colors, so it follows the terminal's theme.
var DefaultPalette = Palette{
	Accent: lipgloss.ANSIColor(3),
	OK:     lipgloss.ANSIColor(2),
	Fail:   lipgloss.ANSIColor(1),
	Muted:  lipgloss.ANSIColor(8),
}

// Option configures a Console.
type Option func(*Console)

// WithPalette renders with p instead of DefaultPalette.
func WithPalette(p Palette) Option {
	return func(c *Console) { c.palette = p }
}

// WithColors forces colors on or off, e.g. for a --color flag, instead of
// detecting them from the writer and the environment.
func WithColors(on bool) Option {
	return func(c *Console) {
		if on {
			c.r.SetColorProfile(termenv.TrueColor)
		} else {
			c.r.SetColorProfile(termenv.Ascii)
		}
	}
}

// Console writes styled output to one writer. It is not safe for concurrent
// use.
type Console struct {
	w       io.Writer
	r       *lipgloss.Renderer
	palette Palette
}

// New returns a Console that writes to w.
func New(w io.Writer, opts ...Option) *Console {
	c := &Console{w: w, r: lipgloss.NewRenderer(w), palette: DefaultPalette}
	for _, opt := range opts {
		opt(c)
	}

	return c
}

// Success prints "✓ message".
func (c *Console) Success(format string, args ...any) {
	c.mark(c.OK("✓"), format, args...)
}

// Failure prints "✗ message".
func (c *Console) Failure(format string, args ...any) {
	c.mark(c.Fail("✗"), format, args...)
}

// Warn prints "! message".
func (c *Console) Warn(format string, args ...any) {
	c.mark(c.Accent("!"), format, args...)
}

// Hint prints a muted, indented line, e.g. a follow-up command.
func (c *Console) Hint(format string, args ...any) {
	fmt.Fprintln(c.w, "  "+c.Muted(fmt.Sprintf(format, args...)))
}

// Line prints s as a line.
func (c *Console) Line(s string) {
	fmt.Fprintln(c.w, s)
}

// Error prints err as a failure unless it wraps ErrReported. A nil err
// prints nothing.
func (c *Console) Error(err error) {
	if err == nil || errors.Is(err, ErrReported) {
		return
	}

	c.Failure("%s", err)
}

// KV is one row of a key/value block.
type KV struct {
	Key   string
	Value string
}

// KeyValues prints an indented block of rows with the keys muted and aligned.
func (c *Console) KeyValues(rows ...KV) {
	width := 0
	for _, r := range rows {
		width = max(width, lipgloss.Width(r.Key))
	}

	for _, r := range rows {
		fmt.Fprintf(c.w, "  %s  %s\n", c.Muted(pad(r.Key, width)), r.Value)
	}
}

// Table prints a borderless table with muted uppercase headers, indented by
// two spaces. Cells may hold styled text: columns align on visible width.
// Rows shorter than headers leave the last columns empty.
func (c *Console) Table(headers []string, rows [][]string) {
	widths := make([]int, len(headers))
	for i, h := range headers {
		widths[i] = lipgloss.Width(h)
	}

	for _, row := range rows {
		for i, cell := range row[:min(len(row), len(widths))] {
			widths[i] = max(widths[i], lipgloss.Width(cell))
		}
	}

	var b strings.Builder

	writeRow := func(cells []string, style func(string) string) {
		line := make([]string, len(widths))
		for i := range widths {
			cell := ""
			if i < len(cells) {
				cell = cells[i]
			}

			if i < len(widths)-1 {
				cell = pad(cell, widths[i])
			}

			line[i] = style(cell)
		}

		b.WriteString(strings.TrimRight("  "+strings.Join(line, "   "), " ") + "\n")
	}

	writeRow(headers, func(s string) string { return c.Muted(strings.ToUpper(s)) })

	for _, row := range rows {
		writeRow(row, func(s string) string { return s })
	}

	fmt.Fprint(c.w, b.String())
}

// Accent renders s in the accent color.
func (c *Console) Accent(s string) string { return c.style(c.palette.Accent).Render(s) }

// OK renders s in the success color.
func (c *Console) OK(s string) string { return c.style(c.palette.OK).Render(s) }

// Fail renders s in the failure color.
func (c *Console) Fail(s string) string { return c.style(c.palette.Fail).Render(s) }

// Muted renders s in the muted color.
func (c *Console) Muted(s string) string { return c.style(c.palette.Muted).Render(s) }

// Bold renders s bold in the text color.
func (c *Console) Bold(s string) string { return c.style(c.palette.Text).Bold(true).Render(s) }

// Title renders s bold in the accent color.
func (c *Console) Title(s string) string { return c.style(c.palette.Accent).Bold(true).Render(s) }

func (c *Console) mark(mark, format string, args ...any) {
	fmt.Fprintln(c.w, mark+" "+fmt.Sprintf(format, args...))
}

func (c *Console) style(color lipgloss.TerminalColor) lipgloss.Style {
	s := c.r.NewStyle()
	if color != nil {
		s = s.Foreground(color)
	}

	return s
}

func pad(s string, width int) string {
	if w := lipgloss.Width(s); w < width {
		return s + strings.Repeat(" ", width-w)
	}

	return s
}
