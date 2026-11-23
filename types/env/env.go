// Package env loads environment variables into typed values. Load returns
// errors wrapping ErrNotSet or ErrParseFailed, LoadOr falls back to a default,
// and MustLoad panics. Values are trimmed of surrounding whitespace.
package env

import (
	"errors"
	"fmt"
	"os"
	"reflect"
	"strconv"
	"strings"
	"time"
)

var (
	// ErrNotSet is returned when a required environment variable is not set
	ErrNotSet = errors.New("env: variable not set")
	// ErrParseFailed is returned when parsing an environment variable fails
	ErrParseFailed = errors.New("env: parse failed")
)

// Parseable defines the types that can be parsed from environment variables
type Parseable interface {
	~string | ~bool | ~int | ~int8 | ~int16 | ~int32 | ~int64 |
		~uint | ~uint8 | ~uint16 | ~uint32 | ~uint64 | ~uintptr |
		~float32 | ~float64 | ~complex64 | ~complex128
}

// Parse converts str to T using strconv. The whole string must be a valid
// value: "12abc" is an error, and a string type receives str unchanged.
// Integers accept base prefixes (0x, 0o, 0b) and underscores; booleans
// accept the values strconv.ParseBool accepts.
func Parse[T Parseable](str string) (T, error) {
	var v T
	rv := reflect.ValueOf(&v).Elem()
	var err error
	switch rv.Kind() {
	case reflect.String:
		rv.SetString(str)
	case reflect.Bool:
		var b bool
		b, err = strconv.ParseBool(str)
		rv.SetBool(b)
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		var n int64
		n, err = strconv.ParseInt(str, 0, rv.Type().Bits())
		rv.SetInt(n)
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		var n uint64
		n, err = strconv.ParseUint(str, 0, rv.Type().Bits())
		rv.SetUint(n)
	case reflect.Float32, reflect.Float64:
		var f float64
		f, err = strconv.ParseFloat(str, rv.Type().Bits())
		rv.SetFloat(f)
	case reflect.Complex64, reflect.Complex128:
		var c complex128
		c, err = strconv.ParseComplex(str, rv.Type().Bits())
		rv.SetComplex(c)
	}
	if err != nil {
		var zero T
		return zero, fmt.Errorf("%w: %w", ErrParseFailed, err)
	}
	return v, nil
}

// MustLoad reads an environment variable and parses it to type T.
// Panics if the variable is not set or cannot be parsed.
// Use this for required configuration that should fail fast.
func MustLoad[T Parseable](name string) T {
	v, err := Load[T](name)
	if err != nil {
		panic(err)
	}

	return v
}

// Load reads an environment variable and parses it to type T.
// Returns the parsed value and an error if the variable is not set or cannot be parsed.
// Use this for optional configuration or when you want to handle errors gracefully.
func Load[T Parseable](name string) (T, error) {
	var zero T
	s, err := lookupEnv(name)
	if err != nil {
		return zero, err
	}

	v, err := Parse[T](strings.TrimSpace(s))
	if err != nil {
		return zero, fmt.Errorf("%w: variable %s", err, name)
	}

	return v, nil
}

// LoadOr reads an environment variable and parses it to type T.
// Returns the default value if the variable is not set or cannot be parsed.
func LoadOr[T Parseable](name string, defaultValue T) T {
	v, err := Load[T](name)
	if err != nil {
		return defaultValue
	}
	return v
}

// MustLoadDuration reads an environment variable and parses it as a time.Duration.
// Panics if the variable is not set or cannot be parsed.
func MustLoadDuration(name string) time.Duration {
	d, err := LoadDuration(name)
	if err != nil {
		panic(err)
	}

	return d
}

// LoadDuration reads an environment variable and parses it as a time.Duration.
// Returns an error if the variable is not set or cannot be parsed.
func LoadDuration(name string) (time.Duration, error) {
	s, err := lookupEnv(name)
	if err != nil {
		return 0, err
	}

	d, err := time.ParseDuration(strings.TrimSpace(s))
	if err != nil {
		return 0, fmt.Errorf("%w: variable %s: %s", ErrParseFailed, name, err)
	}

	return d, nil
}

// LoadDurationOr reads an environment variable and parses it as a time.Duration.
// Returns the default value if the variable is not set or cannot be parsed.
func LoadDurationOr(name string, defaultValue time.Duration) time.Duration {
	d, err := LoadDuration(name)
	if err != nil {
		return defaultValue
	}
	return d
}

// MustLoadTime reads an environment variable and parses it as a time.Time using
// the specified layout.
func MustLoadTime(name, layout string) time.Time {
	t, err := LoadTime(name, layout)
	if err != nil {
		panic(err)
	}

	return t
}

// LoadTime reads an environment variable and parses it as a time.Time using the
// specified layout.
func LoadTime(name, layout string) (time.Time, error) {
	s, err := lookupEnv(name)
	if err != nil {
		return time.Time{}, err
	}

	t, err := time.Parse(layout, strings.TrimSpace(s))
	if err != nil {
		return time.Time{}, fmt.Errorf("%w: variable %s: %s", ErrParseFailed, name, err)
	}

	return t, nil
}

// LoadTimeOr reads an environment variable and parses it as a
// time.Time using the specified layout.
func LoadTimeOr(name, layout string, defaultValue time.Time) time.Time {
	t, err := LoadTime(name, layout)
	if err != nil {
		return defaultValue
	}
	return t
}

// MustLoadSlice reads an environment variable and parses it as a slice of type T.
// The string is split by the separator and each element is parsed.
// Panics if the variable is not set or cannot be parsed.
func MustLoadSlice[T Parseable](name string, sep string) []T {
	res, err := LoadSlice[T](name, sep)
	if err != nil {
		panic(err)
	}

	return res
}

// LoadSlice reads an environment variable and parses it as a slice of type T.
// The string is split by the separator and each element is parsed.
// Returns an error if the variable is not set or cannot be parsed.
func LoadSlice[T Parseable](name string, sep string) ([]T, error) {
	v, err := lookupEnv(name)
	if err != nil {
		return nil, err
	}

	if strings.TrimSpace(v) == "" {
		return []T{}, nil
	}
	vs := strings.Split(v, sep)
	res := make([]T, len(vs))
	for i, s := range vs {
		v, err := Parse[T](strings.TrimSpace(s))
		if err != nil {
			return nil, fmt.Errorf("%w: variable %s at index %d", err, name, i)
		}
		res[i] = v
	}

	return res, nil
}

// LoadSliceOr reads an environment variable and parses it as a slice of type T.
// Returns the default value if the variable is not set or cannot be parsed.
func LoadSliceOr[T Parseable](name string, sep string, defaultValue []T) []T {
	v, err := LoadSlice[T](name, sep)
	if err != nil {
		return defaultValue
	}
	return v
}

// MustExist checks that all specified environment variables are set.
// Panics if any variable is missing. Useful for startup validation.
func MustExist(names ...string) {
	var missing []string
	for _, name := range names {
		if _, ok := os.LookupEnv(name); !ok {
			missing = append(missing, name)
		}
	}
	if len(missing) > 0 {
		panic(fmt.Errorf("%w: %s", ErrNotSet, strings.Join(missing, ", ")))
	}
}

// Exists checks if an environment variable is set (even if empty).
func Exists(name string) bool {
	_, ok := os.LookupEnv(name)
	return ok
}

// IsSet checks if an environment variable is set and non-empty.
func IsSet(name string) bool {
	v, ok := os.LookupEnv(name)
	return ok && v != ""
}

// lookupEnv looks up an environment variable and returns an error if not set.
func lookupEnv(name string) (string, error) {
	v, ok := os.LookupEnv(name)
	if !ok {
		return "", fmt.Errorf("%w: %q", ErrNotSet, name)
	}
	return v, nil
}
