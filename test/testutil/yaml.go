package testutil

import (
	"encoding/json"
	"testing"

	"github.com/alextanhongpin/core/internal"
	"github.com/alextanhongpin/core/test/testdump"
	"github.com/alextanhongpin/core/types/maputil"
	"github.com/google/go-cmp/cmp"
)

type YAMLOption[T any] func(*YamlOption[T])

type YamlOption[T any] struct {
	Dump     *testdump.YAMLOption[T]
	FileName string
}

func DumpYAML[T any](t *testing.T, v T, opts ...YAMLOption[T]) {
	t.Helper()

	o := new(YamlOption[T])
	o.Dump = new(testdump.YAMLOption[T])

	for _, opt := range opts {
		opt(o)
	}

	p := Path{
		Dir:      "testdata",
		FilePath: t.Name(),
		FileName: o.FileName,
		FileExt:  ".yaml",
	}

	if err := testdump.YAML(p.String(), v, o.Dump); err != nil {
		t.Fatal(err)
	}
}

func IgnoreKeys[T any](fields ...string) YAMLOption[T] {
	return func(o *YamlOption[T]) {
		o.Dump.Body = append(o.Dump.Body, internal.IgnoreMapEntries(fields...))
	}
}

func YAMLFileName[T any](name string) YAMLOption[T] {
	return func(o *YamlOption[T]) {
		o.FileName = name
	}
}

func YAMLCmpOption[T any](opts ...cmp.Option) YAMLOption[T] {
	return func(o *YamlOption[T]) {
		o.Dump.Body = append(o.Dump.Body, opts...)
	}
}

func MaskKeys[T any](fields ...string) YAMLOption[T] {
	return func(o *YamlOption[T]) {
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

func CompareYAML[T any](hook func(snapshot, received T) error) YAMLOption[T] {
	return func(o *YamlOption[T]) {
		o.Dump.Hooks = append(o.Dump.Hooks,
			testdump.CompareHook(hook))
	}
}
