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

type YAMLOption interface {
	isYAML()
}

func DumpYAML[T any](t *testing.T, v T, opts ...YAMLOption) {
	t.Helper()

	o := new(yamlOption[T])
	o.Dump = new(testdump.YAMLOption[T])

	for _, opt := range opts {
		switch ot := opt.(type) {
		case YAMLCmpOption:
			o.Dump.Body = append(o.Dump.Body, ot...)
		case CmpOption:
			o.Dump.Body = append(o.Dump.Body, ot...)
		case FileName:
			o.FileName = string(ot)
		case yamlOptionHook[T]:
			ot(o)
		default:
			panic(fmt.Errorf("testutil: unhandled YAML option: %#v", opt))
		}
	}

	p := Path{
		Dir:      "testdata",
		FilePath: t.Name(),
		FileName: internal.Or(o.FileName, internal.TypeName(v)),
		FileExt:  ".yaml",
	}

	if err := testdump.YAML(p.String(), v, o.Dump); err != nil {
		t.Fatal(err)
	}
}

type yamlOptionHook[T any] func(*yamlOption[T])

func (yamlOptionHook[T]) isYAML() {}

type yamlOption[T any] struct {
	Dump     *testdump.YAMLOption[T]
	FileName string
}

type YAMLCmpOption []cmp.Option

func (x YAMLCmpOption) isYAML() {}

func IgnoreKeys(fields ...string) YAMLCmpOption {
	return YAMLCmpOption([]cmp.Option{internal.IgnoreMapEntries(fields...)})
}

func MaskKeys[T any](fields ...string) yamlOptionHook[T] {
	return func(o *yamlOption[T]) {
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

func InspectYAML[T any](hook func(snapshot, received T) error) yamlOptionHook[T] {
	return func(o *yamlOption[T]) {
		o.Dump.Hooks = append(o.Dump.Hooks,
			testdump.CompareHook(hook))
	}
}

func InterceptYAML[T any](hook func(T) (T, error)) yamlOptionHook[T] {
	return func(o *yamlOption[T]) {
		o.Dump.Hooks = append(o.Dump.Hooks,
			testdump.MarshalHook(hook))
	}
}
