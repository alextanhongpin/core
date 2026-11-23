// Package request handles parsing and validation of HTTP request bodies.
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

// DecodeJSON decodes the JSON request body into v.
//
// The body is limited to MaxBodySize bytes (WithMaxBodySize overrides it);
// larger bodies fail with ErrBodyTooLarge. An empty body fails with
// ErrEmptyBody when WithRequired is set and otherwise leaves v unchanged.
// After decoding, or for an empty body, v.Bind(r) and then v.Validate() are
// called if v implements them, so an empty body cannot skip validation. The
// body is restored so it can be read again.
func DecodeJSON(r *http.Request, v any, opts ...DecodeOption) error {
	options := &DecodeOptions{
		MaxBodySize: MaxBodySize,
		Required:    false,
	}

	for _, opt := range opts {
		opt(options)
	}

	b, err := readAtMost(r, options.MaxBodySize)
	if err != nil {
		if errors.Is(err, ErrBodyTooLarge) {
			return &BodyError{Body: b, err: ErrBodyTooLarge}
		}
		return err
	}

	if len(b) == 0 {
		if options.Required {
			return &BodyError{Body: b, err: ErrEmptyBody}
		}
	} else {
		if !json.Valid(b) {
			return &BodyError{Body: b, err: ErrInvalidJSON}
		}

		if err := json.Unmarshal(b, v); err != nil {
			return &BodyError{Body: b, err: err}
		}
	}

	if binder, ok := v.(bindable); ok {
		if err := binder.Bind(r); err != nil {
			return err
		}
	}

	if validator, ok := v.(validatable); ok {
		return validator.Validate()
	}

	return nil
}

// readAtMost reads the whole body if it is at most limit bytes and restores
// it. For a larger body it returns the first limit bytes and ErrBodyTooLarge,
// and r.Body still yields the complete original stream.
func readAtMost(r *http.Request, limit int64) ([]byte, error) {
	if r.Body == nil || r.Body == http.NoBody {
		return nil, nil
	}

	b, err := io.ReadAll(io.LimitReader(r.Body, limit+1))
	if err != nil {
		return nil, err
	}

	if int64(len(b)) > limit {
		r.Body = readCloser{io.MultiReader(bytes.NewReader(b), r.Body), r.Body}
		return b[:limit], ErrBodyTooLarge
	}

	r.Body = io.NopCloser(bytes.NewReader(b))
	return b, nil
}

type readCloser struct {
	io.Reader
	io.Closer
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
	if rv.Kind() != reflect.Pointer || rv.Elem().Kind() != reflect.Struct {
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
		// Parse with the field's size so out-of-range values fail instead of
		// silently wrapping.
		intVal, err := strconv.ParseInt(value, 10, field.Type().Bits())
		if err != nil {
			return err
		}
		field.SetInt(intVal)
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		uintVal, err := strconv.ParseUint(value, 10, field.Type().Bits())
		if err != nil {
			return err
		}
		field.SetUint(uintVal)
	case reflect.Float32, reflect.Float64:
		floatVal, err := strconv.ParseFloat(value, field.Type().Bits())
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
