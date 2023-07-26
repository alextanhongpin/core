package testutil

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/alextanhongpin/core/internal"
	"github.com/alextanhongpin/core/test/testdump"
	"github.com/alextanhongpin/core/types/maputil"
	"github.com/google/go-cmp/cmp"
)

type JSONOption interface {
	isJSON()
}

func DumpJSON[T any](t *testing.T, v T, opts ...JSONOption) {
	t.Helper()

	o := new(jsonOption[T])
	o.Dump = new(testdump.JSONOption[T])

	for _, opt := range opts {
		switch ot := opt.(type) {
		case JSONCmpOption:
			o.Dump.Body = append(o.Dump.Body, ot...)
		case CmpOption:
			o.Dump.Body = append(o.Dump.Body, ot...)
		case FileName:
			o.FileName = string(ot)
		case jsonOptionHook[T]:
			ot(o)
		default:
			panic(fmt.Errorf("testutil: unhandled JSON option: %#v", opt))
		}
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

type jsonOptionHook[T any] func(*jsonOption[T])

func (j jsonOptionHook[T]) isJSON() {}

type jsonOption[T any] struct {
	Dump     *testdump.JSONOption[T]
	FileName string
}

type JSONCmpOption []cmp.Option

func (x JSONCmpOption) isJSON() {}

func IgnoreFields(fields ...string) JSONCmpOption {
	return JSONCmpOption([]cmp.Option{internal.IgnoreMapEntries(fields...)})
}

func MaskFields[T any](fields ...string) jsonOptionHook[T] {
	return func(o *jsonOption[T]) {
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

func InspectJSON[T any](hook func(snapshot, received T) error) jsonOptionHook[T] {
	return func(o *jsonOption[T]) {
		o.Dump.Hooks = append(o.Dump.Hooks,
			testdump.CompareHook(hook))
	}
}

func InterceptJSON[T any](hook func(t T) (T, error)) jsonOptionHook[T] {
	return func(o *jsonOption[T]) {
		o.Dump.Hooks = append(o.Dump.Hooks,
			testdump.MarshalHook(hook))
	}
}
