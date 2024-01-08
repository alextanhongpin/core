package testdump

import (
	"encoding/json"

	"github.com/alextanhongpin/core/internal"
	"github.com/alextanhongpin/core/types/maputil"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"golang.org/x/exp/slices"
)

type JSONOption struct {
	Body []cmp.Option
}

// NOTE: Why using a type is bad - because if we serialize to structs, the keys
// that are removed won't be compared.
// Ideally, using map[string]any or just any should work better for snapshot
// testing.
func JSON[T any](rw readerWriter, t T, opt *JSONOption, hooks ...Hook[T]) error {
	if opt == nil {
		opt = new(JSONOption)
	}

	opt.Body = append(opt.Body, ignoreFieldsFromTags(t, "json")...)

	var s S[T] = &snapshot[T]{
		marshaler: MarshalFunc[T](MarshalJSON[T]),
		// This is only used for custom comparison. It does not benefit as much as
		// using map[string]any for comparison due to loss of information.
		unmarshaler:    UnmarshalFunc[T](UnmarshalJSON[T]),
		anyUnmarshaler: UnmarshalAnyFunc(UnmarshalJSON[any]),
		anyComparer:    CompareAnyFunc((&JSONComparer[any]{Body: opt.Body}).Compare),
	}

	s = Hooks[T](append(hooks, maskFieldsFromTags(t, "json")...)).Apply(s)

	return Snapshot(rw, t, s)
}

func MarshalJSON[T any](t T) ([]byte, error) {
	return internal.PrettyJSON(t)
}

// The problem is here - the unmarshalling actually causes a loss of data.
func UnmarshalJSON[T any](b []byte) (T, error) {
	var t T
	if err := json.Unmarshal(b, &t); err != nil {
		return t, err
	}

	return t, nil
}

type JSONComparer[T any] struct {
	Body []cmp.Option
}

func (c *JSONComparer[T]) Compare(snapshot, received T) error {
	return internal.ANSIDiff(snapshot, received, c.Body...)
}

func MaskFields[T any](fields ...string) Hook[T] {
	return MarshalHook(func(t T) (T, error) {
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
	})
}

func ignoreFieldsFromTags[T any](v T, name string) []cmp.Option {
	var opts []cmp.Option

	kv := internal.GetStructTags(v, name, "cmp")
	fields := make(map[string]bool)
	for k, v := range kv {
		if slices.Contains(v, "ignore") {
			fields[k] = true
		}
	}

	if len(fields) > 0 {
		cond := func(k string, v any) bool {
			return fields[k]
		}

		opts = append(opts, cmpopts.IgnoreMapEntries(cond))
	}

	return opts
}

func maskFieldsFromTags[T any](v T, name string) []Hook[T] {
	var hooks []Hook[T]
	kv := internal.GetStructTags(v, name, "cmp")
	var fields []string
	for k, v := range kv {
		if slices.Contains(v, "mask") {
			fields = append(fields, k)
		}
	}

	if len(fields) > 0 {
		hooks = append(hooks, MaskFields[T](fields...))
	}

	return hooks
}
