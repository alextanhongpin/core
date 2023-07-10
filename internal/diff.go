package internal

import (
	"fmt"
	"strings"

	"github.com/google/go-cmp/cmp"
)

type DiffError struct {
	ansi  string
	text  string
	color bool
}

func (d *DiffError) Error() string {
	if d.color {
		return d.ansi
	}

	return d.text
}

func (d *DiffError) Ansi() string {
	return d.ansi
}

func (d *DiffError) Text() string {
	return d.text
}

func (d *DiffError) SetColor(color bool) {
	d.color = color
}

func ANSIDiff(x, y any, opts ...cmp.Option) error {
	diff := cmp.Diff(x, y, opts...)
	if diff == "" {
		return nil
	}

	return &DiffError{
		ansi:  ansiDiff(diff),
		text:  textDiff(diff),
		color: true,
	}
}

func ansiDiff(diff string) string {
	if diff == "" {
		return ""
	}

	// TODO: Option to disable.
	lines := strings.Split(diff, "\n")
	for i, line := range lines {
		switch {
		case strings.HasPrefix(line, "-"):
			lines[i] = red(line)
		case strings.HasPrefix(line, "+"):
			lines[i] = green(line)
		}
	}

	header := []string{
		"\n",
		red("  Snapshot(-)"),
		green("  Received(+)"),
		"\n",
	}
	lines = append(header, lines...)

	return strings.Join(lines, "\n")
}

func textDiff(diff string) string {
	if diff == "" {
		return ""
	}

	header := []string{
		"\n",
		"  Snapshot(-)",
		"  Received(+)",
		"\n",
	}
	lines := strings.Split(diff, "\n")
	lines = append(header, lines...)

	return strings.Join(lines, "\n")
}

func escapeCode(code int) string {
	return fmt.Sprintf("\x1b[%dm", code)
}

func color(code int, s string) string {
	return fmt.Sprintf("%s%s%s", escapeCode(code), s, escapeCode(0))
}

func red(s string) string {
	return color(31, s)
}

func green(s string) string {
	return color(32, s)
}
