package testutil

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"

	"github.com/google/go-cmp/cmp"
)

type dumper interface {
	Dump() ([]byte, error)
}

type comparer interface {
	Compare(snapshot, received []byte) error
}

type dumperAndComparer interface {
	dumper
	comparer
}

func Dump(fileName string, dnc dumperAndComparer) error {
	received, err := dnc.Dump()
	if err != nil {
		return err
	}

	err = writeToNewFile(fileName, received)
	if err != nil {
		return err
	}

	snapshot, err := os.ReadFile(fileName)
	if err != nil {
		return err
	}

	return dnc.Compare(snapshot, received)
}

func typeName(v any) string {
	t := reflect.TypeOf(v)
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}

	if t.Kind() == reflect.Struct {
		return t.String()
	}

	return t.Kind().String()
}

func writeToNewFile(name string, body []byte) error {
	f, err := os.OpenFile(name, os.O_RDONLY, 0644)
	if errors.Is(err, os.ErrNotExist) {
		dir := filepath.Dir(name)

		if err := os.MkdirAll(dir, 0700); err != nil && !os.IsExist(err) {
			return err
		} // Create your file

		return os.WriteFile(name, body, 0644)
	}
	if err != nil {
		return err
	}

	defer f.Close()

	return nil
}

func ansiDiff(x, y any, opts ...cmp.Option) error {
	if diff := cmp.Diff(x, y, opts...); diff != "" {
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

	return nil
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
