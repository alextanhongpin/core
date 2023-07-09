package internal

import (
	"errors"
	"fmt"
	"strings"

	"github.com/google/go-cmp/cmp"
)

func ANSIDiff(x, y any, opts ...cmp.Option) error {
	diff := cmp.Diff(x, y, opts...)
	if diff == "" {
		return nil
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

	return errors.New(strings.Join(lines, "\n"))
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
