package testutil

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"strings"

	"github.com/google/go-cmp/cmp"
)

// Validate if the package name is valid.
var pkgRe = regexp.MustCompile(`^[a-zA-Z][a-zA-Z0-9_]*$`)

type dumper interface {
	Dump() ([]byte, error)
}

func dump(fileName string, dump dumper) (want, got []byte, err error) {
	got, err = dump.Dump()
	if err != nil {
		return
	}

	err = writeToNewFile(fileName, got)
	if err != nil {
		return
	}

	want, err = os.ReadFile(fileName)
	if err != nil {
		return
	}

	return
}

func typeName(v any) []string {
	t := reflect.TypeOf(v)
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}

	var res []string
	pkg := t.PkgPath()
	pkg = filepath.Base(pkg)
	pkg = strings.TrimSpace(pkg)
	if pkg != "" && pkgRe.MatchString(pkg) {
		res = append(res, pkg)
	}

	name := strings.TrimSpace(t.Name())
	if name != "" {
		res = append(res, name)
	}

	return res
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

func cmpDiff(x, y any, opts ...cmp.Option) error {
	if diff := cmp.Diff(x, y, opts...); diff != "" {
		return fmt.Errorf("\n  Snapshot(-)\n  Received(+):\n\n%s", diff)
	}

	return nil
}
