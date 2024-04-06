package testdump

import (
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"strings"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/cuecontext"
	"cuelang.org/go/cue/load"
	"github.com/alextanhongpin/core/internal"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
)

type JSONOption struct {
	Body         []cmp.Option
	IgnoreFields []string
	MaskFields   []string
	CUEOption    *CUEOption
}

type CUEOption struct {
	Schema     string       // The raw schema.
	SchemaPath string       // Path to the cue file.
	Options    []cue.Option // List of options
}

func (o *CUEOption) Validate(data []byte) error {
	bothEmpty := o.Schema == "" && o.SchemaPath == ""
	if bothEmpty {
		// Nothing to validate.
		return nil
	}

	bothPresent := o.Schema != "" && o.SchemaPath != ""
	if bothPresent {
		return errors.New("cannot specify both schema and schema path")
	}

	if len(o.Options) == 0 {
		// Concrete allows us to check for required fields.
		o.Options = append(o.Options, cue.Concrete(true))
	}

	ctx := cuecontext.New()

	var schema cue.Value
	if o.Schema != "" {
		schema = ctx.CompileString(o.Schema)
	} else if o.SchemaPath != "" {
		bins := load.Instances([]string{o.SchemaPath}, nil)
		schema = ctx.BuildInstance(bins[0])
	}
	value := ctx.CompileString(string(data))
	return schema.Unify(value).Validate(o.Options...)
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

	// Validate the current data.
	if opt.CUEOption != nil {
		if err := opt.CUEOption.Validate(b); err != nil {
			return fmt.Errorf("cueError: %w", err)
		}
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

	// Validate the snapshot data.
	if opt.CUEOption != nil {
		if err := opt.CUEOption.Validate(b); err != nil {
			return fmt.Errorf("cueError: %w", err)
		}
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

func ignoreFieldsFromTags[T any](v T, name string, fields ...string) []cmp.Option {
	var opts []cmp.Option

	_ = internal.GetStructTags(v, func(f reflect.StructField) error {
		fname := f.Tag.Get(name)
		if fname == "" {
			fname = f.Name
		}
		tags := strings.Split(f.Tag.Get("cmp"), ",")
		for _, t := range tags {
			if t == "ignore" {
				fields = append(fields, fname)
				break
			}
		}

		return nil
	})

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

// maskFieldsFromTags mask the fields based on the tag name.
func maskFieldsFromTags[T any](v T, tag string, fields ...string) (T, error) {
	var mask func(ori, a any) any
	mask = func(ori any, a any) any {
		rt := reflect.ValueOf(ori).Type()
		if rt.Kind() == reflect.Ptr {
			rt = rt.Elem()
		}

		switch m := a.(type) {
		case map[string]any:

			if rt.Kind() == reflect.Struct {
				for _, f := range reflect.VisibleFields(rt) {
					if f.Tag.Get("mask") != "true" {
						el := reflect.New(f.Type).Elem().Interface()
						name := f.Tag.Get(tag)
						if v, ok := m[name]; ok {
							m[name] = mask(el, v)
						}

						name = f.Name
						if v, ok := m[name]; ok {
							m[name] = mask(el, v)
						}

						continue
					}

					name := f.Tag.Get(tag)
					if _, ok := m[name]; ok {
						m[name] = "[REDACTED]"
					}

					name = f.Name
					if _, ok := m[name]; ok {
						m[name] = "[REDACTED]"
					}
				}

				for k, v := range m {
					f, ok := rt.FieldByName(k)
					if !ok {
						continue
					}

					el := reflect.New(f.Type).Elem().Interface()
					m[k] = mask(el, v)
				}
			}

			for _, f := range fields {
				if _, ok := m[f]; ok {
					m[f] = "[REDACTED]"
				}
			}
			return m
		case []any:
			res := make([]any, len(m))

			// Array or slice.
			rt = rt.Elem()
			el := reflect.New(rt).Elem().Interface()
			for i, v := range m {
				res[i] = mask(el, v)
			}
			return res
		default:
			return a
		}
	}

	b, err := json.Marshal(v)
	if err != nil {
		return v, err
	}
	var a any
	if err := json.Unmarshal(b, &a); err != nil {
		return v, err
	}

	b, err = json.Marshal(mask(v, a))
	if err != nil {
		return v, err
	}

	var t T
	return t, json.Unmarshal(b, &t)
}
