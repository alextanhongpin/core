package testutil

import (
	"encoding/json"
	"testing"

	"github.com/alextanhongpin/core/internal"
	"github.com/alextanhongpin/core/test/testdump"
	"github.com/alextanhongpin/core/types/maputil"
	"github.com/google/go-cmp/cmp"
)

type JSONOption[T any] func(*JsonOption[T])

type JsonOption[T any] struct {
	Dump     *testdump.JSONOption[T]
	FileName string
}

func DumpJSON[T any](t *testing.T, v T, opts ...JSONOption[T]) {
	t.Helper()

	o := new(JsonOption[T])
	o.Dump = new(testdump.JSONOption[T])

	for _, opt := range opts {
		opt(o)
	}

	p := Path{
		Dir:      "testdata",
		FilePath: t.Name(),
		FileName: internal.Or(o.FileName, internal.TypeName(v)),
		FileExt:  ".json",
	}

	if err := testdump.JSON(p.String(), v, o.Dump); err != nil {
		t.Fatal(err)
	}
}

func IgnoreFields[T any](fields ...string) JSONOption[T] {
	return func(o *JsonOption[T]) {
		o.Dump.Body = append(o.Dump.Body, internal.IgnoreMapEntries(fields...))
	}
}

func JSONFileName[T any](name string) JSONOption[T] {
	return func(o *JsonOption[T]) {
		o.FileName = name
	}
}

func JSONCmpOption[T any](opts ...cmp.Option) JSONOption[T] {
	return func(o *JsonOption[T]) {
		o.Dump.Body = append(o.Dump.Body, opts...)
	}
}

func MaskFields[T any](fields ...string) JSONOption[T] {
	return func(o *JsonOption[T]) {
		o.Dump.Hooks = append(o.Dump.Hooks, testdump.MarshalHook(func(t T) (T, error) {
			b, err := json.Marshal(t)
			if err != nil {
				return t, err
			}

			bb, err := maputil.MaskBytes(b, fields...)
			if err != nil {
				return t, err
			}

			var tt T
			if err := json.Unmarshal(bb, &tt); err != nil {
				return tt, err
			}

			return tt, nil
		}))
	}
}

func InspectJSON[T any](hook func(snapshot, received T) error) JSONOption[T] {
	return func(o *JsonOption[T]) {
		o.Dump.Hooks = append(o.Dump.Hooks,
			testdump.CompareHook(hook))
	}
}

func InterceptJSON[T any](hook func(t T) (T, error)) JSONOption[T] {
	return func(o *JsonOption[T]) {
		o.Dump.Hooks = append(o.Dump.Hooks,
			testdump.MarshalHook(hook))
	}
}
