package testutil

import (
	"fmt"
	"testing"

	"github.com/alextanhongpin/core/internal"
	"github.com/alextanhongpin/core/test/testdump"
	"github.com/google/go-cmp/cmp"
)

type JSONOption interface {
	isJSON()
}

func DumpJSON[T any](t *testing.T, v T, opts ...JSONOption) {
	t.Helper()

	var fileName string
	var hooks []testdump.Hook[T]
	jsonOpt := new(testdump.JSONOption)

	for _, opt := range opts {
		switch o := opt.(type) {
		case JSONCmpOption:
			jsonOpt.Body = append(jsonOpt.Body, o...)
		case CmpOption:
			jsonOpt.Body = append(jsonOpt.Body, o...)
		case FileName:
			fileName = string(o)
		case *jsonHookOption[T]:
			hooks = append(hooks, o.hook)
		default:
			panic(fmt.Errorf("testutil: unhandled JSON option: %#v", opt))
		}
	}

	p := Path{
		Dir:      "testdata",
		FilePath: t.Name(),
		FileName: internal.Or(fileName, internal.TypeName(v)),
		FileExt:  ".json",
	}

	if err := testdump.JSON[T](testdump.NewFile(p.String()), v, jsonOpt, hooks...); err != nil {
		t.Fatal(err)
	}
}

type jsonHookOption[T any] struct {
	hook testdump.Hook[T]
}

func (j jsonHookOption[T]) isJSON() {}

type JSONCmpOption []cmp.Option

func (x JSONCmpOption) isJSON() {}

func IgnoreFields(fields ...string) JSONCmpOption {
	return JSONCmpOption([]cmp.Option{internal.IgnoreMapEntries(fields...)})
}

func MaskFields[T any](fields ...string) *jsonHookOption[T] {
	return &jsonHookOption[T]{
		hook: testdump.MaskFields[T](fields...),
	}
}

func InspectJSON[T any](hook func(snapshot, received T) error) *jsonHookOption[T] {
	return &jsonHookOption[T]{
		hook: testdump.CompareHook(hook),
	}
}

func InterceptJSON[T any](hook func(t T) (T, error)) *jsonHookOption[T] {
	return &jsonHookOption[T]{
		hook: testdump.MarshalHook(hook),
	}
}
