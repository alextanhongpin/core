package testutil

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"

	"github.com/google/go-cmp/cmp"
)

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

func isStruct(v any) bool {
	return reflect.TypeOf(v).Kind() == reflect.Struct
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
		return fmt.Errorf("want(+), got(-):\n\n%s", diff)
	}

	return nil
}
