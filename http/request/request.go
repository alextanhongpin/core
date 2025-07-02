// package request handles the parsing and validation for the request body.
package request

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"reflect"
	"strconv"
	"strings"
)

var (
	ErrInvalidJSON     = errors.New("request: invalid json")
	ErrInvalidFormData = errors.New("request: invalid form data")
	ErrBodyTooLarge    = errors.New("request: request body too large")
	ErrEmptyBody       = errors.New("request: empty request body")
)

const (
	// MaxBodySize is the default maximum request body size (10MB)
	MaxBodySize = 10 << 20
)

type BodyError struct {
	Body []byte
	err  error
}

func (b *BodyError) Unwrap() error {
	return b.err
}

func (b *BodyError) Error() string {
	return b.err.Error()
}

type validatable interface {
	Validate() error
}

type bindable interface {
	Bind(*http.Request) error
}

// DecodeOptions configures request decoding behavior
type DecodeOptions struct {
	MaxBodySize int64
	Required    bool
}

// DecodeOption is a functional option for configuring DecodeOptions
type DecodeOption func(*DecodeOptions)

// WithMaxBodySize sets the maximum allowed body size
func WithMaxBodySize(size int64) DecodeOption {
	return func(o *DecodeOptions) {
		o.MaxBodySize = size
	}
}

// WithRequired indicates that the request body is required
func WithRequired() DecodeOption {
	return func(o *DecodeOptions) {
		o.Required = true
	}
}

// DecodeJSON decodes JSON request body with validation and size limits
func DecodeJSON(r *http.Request, v any, opts ...DecodeOption) error {
	options := &DecodeOptions{
		MaxBodySize: MaxBodySize,
		Required:    false,
	}

	for _, opt := range opts {
		opt(options)
	}

	// Limit the reader to prevent huge requests
	limitedReader := io.LimitReader(r.Body, options.MaxBodySize+1)
	b, err := io.ReadAll(limitedReader)
	if err != nil {
		return err
	}

	if int64(len(b)) > options.MaxBodySize {
		return &BodyError{Body: b[:options.MaxBodySize], err: ErrBodyTooLarge}
	}

	if len(b) == 0 && options.Required {
		return &BodyError{Body: b, err: ErrEmptyBody}
	}

	if len(b) == 0 {
		return nil
	}

	// Restore the body for potential reuse
	r.Body = io.NopCloser(bytes.NewBuffer(bytes.Clone(b)))

	if !json.Valid(b) {
		return &BodyError{Body: b, err: ErrInvalidJSON}
	}

	if err := json.Unmarshal(b, v); err != nil {
		return &BodyError{Body: b, err: err}
	}

	// Check if the struct implements bindable interface
	if binder, ok := v.(bindable); ok {
		if err := binder.Bind(r); err != nil {
			return err
		}
	}

	// Check if the struct implements validatable interface
	if validator, ok := v.(validatable); ok {
		return validator.Validate()
	}

	return nil
}

// DecodeForm parses form data into a struct using reflection
func DecodeForm(r *http.Request, v any) error {
	if err := r.ParseForm(); err != nil {
		return &BodyError{err: fmt.Errorf("%w: %v", ErrInvalidFormData, err)}
	}

	return bindFormToStruct(r.Form, v)
}

// DecodeQuery parses query parameters into a struct using reflection
func DecodeQuery(r *http.Request, v any) error {
	return bindFormToStruct(r.URL.Query(), v)
}

// bindFormToStruct binds form values to struct fields
func bindFormToStruct(values url.Values, v any) error {
	rv := reflect.ValueOf(v)
	if rv.Kind() != reflect.Ptr || rv.Elem().Kind() != reflect.Struct {
		return errors.New("request: destination must be a pointer to struct")
	}

	rv = rv.Elem()
	rt := rv.Type()

	for i := 0; i < rv.NumField(); i++ {
		field := rv.Field(i)
		fieldType := rt.Field(i)

		if !field.CanSet() {
			continue
		}

		tag := fieldType.Tag.Get("form")
		if tag == "" {
			tag = fieldType.Tag.Get("json")
		}
		if tag == "" || tag == "-" {
			continue
		}

		// Handle comma-separated tag options (e.g., "name,omitempty")
		tagName := strings.Split(tag, ",")[0]
		value := values.Get(tagName)

		if err := setFieldValue(field, value); err != nil {
			return fmt.Errorf("request: failed to set field %s: %w", fieldType.Name, err)
		}
	}

	// Check if the struct implements validatable interface
	if validator, ok := v.(validatable); ok {
		return validator.Validate()
	}

	return nil
}

// setFieldValue sets a struct field value from a string
func setFieldValue(field reflect.Value, value string) error {
	if value == "" {
		return nil
	}

	switch field.Kind() {
	case reflect.String:
		field.SetString(value)
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		intVal, err := strconv.ParseInt(value, 10, 64)
		if err != nil {
			return err
		}
		field.SetInt(intVal)
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		uintVal, err := strconv.ParseUint(value, 10, 64)
		if err != nil {
			return err
		}
		field.SetUint(uintVal)
	case reflect.Float32, reflect.Float64:
		floatVal, err := strconv.ParseFloat(value, 64)
		if err != nil {
			return err
		}
		field.SetFloat(floatVal)
	case reflect.Bool:
		boolVal, err := strconv.ParseBool(value)
		if err != nil {
			return err
		}
		field.SetBool(boolVal)
	case reflect.Slice:
		// Handle slices (e.g., []string for multiple values)
		if field.Type().Elem().Kind() == reflect.String {
			values := strings.Split(value, ",")
			slice := reflect.MakeSlice(field.Type(), len(values), len(values))
			for i, v := range values {
				slice.Index(i).SetString(strings.TrimSpace(v))
			}
			field.Set(slice)
		}
	default:
		return fmt.Errorf("unsupported field type: %v", field.Kind())
	}

	return nil
}
