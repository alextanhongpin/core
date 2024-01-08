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
	Body         []cmp.Option
	IgnoreFields []string
	MaskFields   []string
}

// NOTE: Why using a type is bad - because if we serialize to structs, the keys
// that are removed won't be compared.
// Ideally, using map[string]any or just any should work better for snapshot
// testing.
func JSON[T any](rw readerWriter, t T, opt *JSONOption) error {
	if opt == nil {
		opt = new(JSONOption)
	}

	t, err := maskFieldsFromTags(t, "json", opt.MaskFields...)
	if err != nil {
		return err
	}

	b, err := MarshalJSON(t)
	if err != nil {
		return err
	}

	if err := rw.Write(b); err != nil {
		return err
	}

	received, err := UnmarshalJSON[any](b)
	if err != nil {
		return err
	}

	b, err = rw.Read()
	if err != nil {
		return err
	}

	snapshot, err := UnmarshalJSON[any](b)
	if err != nil {
		return err
	}

	opt.Body = append(opt.Body, ignoreFieldsFromTags(t, "json", opt.IgnoreFields...)...)
	cmp := &JSONComparer[any]{Body: opt.Body}
	return cmp.Compare(snapshot, received)
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

func maskFields[T any](t T, fields ...string) (T, error) {
	if len(fields) == 0 {
		return t, nil
	}

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
}

func ignoreFieldsFromTags[T any](v T, name string, fields ...string) []cmp.Option {
	var opts []cmp.Option

	kv := internal.GetStructTags(v, name, "cmp")
	for k, v := range kv {
		if slices.Contains(v, "ignore") {
			fields = append(fields, k)
		}
	}

	if len(fields) > 0 {
		cond := func(k string, v any) bool {
			for _, f := range fields {
				if f == k {
					return true
				}
			}

			return false
		}

		opts = append(opts, cmpopts.IgnoreMapEntries(cond))
	}

	return opts
}

func maskFieldsFromTags[T any](v T, name string, fields ...string) (T, error) {
	kv := internal.GetStructTags(v, name, "cmp")
	for k, v := range kv {
		if slices.Contains(v, "mask") {
			fields = append(fields, k)
		}
	}

	return maskFields(v, fields...)
}
