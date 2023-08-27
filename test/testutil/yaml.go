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

	var fileName string
	var hooks []testdump.Hook[T]
	yamlOpt := new(testdump.YAMLOption)

	for _, opt := range opts {
		switch o := opt.(type) {
		case YAMLCmpOption:
			yamlOpt.Body = append(yamlOpt.Body, o...)
		case CmpOption:
			yamlOpt.Body = append(yamlOpt.Body, o...)
		case FileName:
			fileName = string(o)
		case *yamlHookOption[T]:
			hooks = append(hooks, o.hook)
		default:
			panic(fmt.Errorf("testutil: unhandled YAML option: %#v", opt))
		}
	}

	p := Path{
		Dir:      "testdata",
		FilePath: t.Name(),
		FileName: internal.Or(fileName, internal.TypeName(v)),
		FileExt:  ".yaml",
	}

	if err := testdump.YAML(testdump.NewFile(p.String()), v, yamlOpt, hooks...); err != nil {
		t.Fatal(err)
	}
}

type yamlHookOption[T any] struct {
	hook testdump.Hook[T]
}

func (yamlHookOption[T]) isYAML() {}

type YAMLCmpOption []cmp.Option

func (x YAMLCmpOption) isYAML() {}

func IgnoreKeys(fields ...string) YAMLCmpOption {
	return YAMLCmpOption([]cmp.Option{internal.IgnoreMapEntries(fields...)})
}

func MaskKeys[T any](fields ...string) *yamlHookOption[T] {
	return &yamlHookOption[T]{
		hook: testdump.MarshalHook(func(t T) (T, error) {
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
		}),
	}
}

func InspectYAML[T any](hook func(snapshot, received T) error) *yamlHookOption[T] {
	return &yamlHookOption[T]{
		hook: testdump.CompareHook(hook),
	}
}

func InterceptYAML[T any](hook func(T) (T, error)) *yamlHookOption[T] {
	return &yamlHookOption[T]{
		hook: testdump.MarshalHook(hook),
	}
}
