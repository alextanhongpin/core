package request

import (
	"cmp"
	"encoding/base64"
	"fmt"
	"net/http"
	"net/url"
	"reflect"
	"strconv"
	"strings"
	"time"
)

type Value string

func QueryValue(r *http.Request, name string) Value {
	return Value(r.URL.Query().Get(name))
}

func PathValue(r *http.Request, name string) Value {
	return Value(r.PathValue(name))
}

func FormValue(r *http.Request, name string) Value {
	return Value(r.FormValue(name))
}

func HeaderValue(r *http.Request, name string) Value {
	return Value(r.Header.Get(name))
}

// QueryValues returns all values for a query parameter
func QueryValues(r *http.Request, name string) []Value {
	values := r.URL.Query()[name]
	result := make([]Value, len(values))
	for i, v := range values {
		result[i] = Value(v)
	}
	return result
}

func (v Value) String() string {
	return string(v)
}

func (v Value) Trim() Value {
	return Value(strings.TrimSpace(string(v)))
}

func (v Value) Lower() Value {
	return Value(strings.ToLower(string(v)))
}

func (v Value) Upper() Value {
	return Value(strings.ToUpper(string(v)))
}

func (v Value) StringOr(str string) string {
	return cmp.Or(v.String(), str)
}

// IsEmpty returns true if the value is empty or only whitespace
func (v Value) IsEmpty() bool {
	return strings.TrimSpace(string(v)) == ""
}

// Required returns an error if the value is empty
func (v Value) Required(fieldName string) error {
	if v.IsEmpty() {
		return fmt.Errorf("field %s is required", fieldName)
	}
	return nil
}

// Length validates the string length
func (v Value) Length(min, max int) error {
	length := len(strings.TrimSpace(string(v)))
	if length < min {
		return fmt.Errorf("value must be at least %d characters", min)
	}
	if length > max {
		return fmt.Errorf("value must be at most %d characters", max)
	}
	return nil
}

// Contains checks if the value contains a substring
func (v Value) Contains(substr string) bool {
	return strings.Contains(string(v), substr)
}

// HasPrefix checks if the value has a prefix
func (v Value) HasPrefix(prefix string) bool {
	return strings.HasPrefix(string(v), prefix)
}

// HasSuffix checks if the value has a suffix
func (v Value) HasSuffix(suffix string) bool {
	return strings.HasSuffix(string(v), suffix)
}

// Split splits the value by a separator
func (v Value) Split(sep string) []string {
	if v.IsEmpty() {
		return []string{}
	}
	return strings.Split(string(v), sep)
}

func (v Value) Int64() int64 {
	return toInt64(v.String())
}

func (v Value) Int64Or(n int64) int64 {
	return cmp.Or(v.Int64(), n)
}

// Int64Range validates that the int64 value is within a range
func (v Value) Int64Range(min, max int64) (int64, error) {
	val := v.Int64()
	if val < min || val > max {
		return 0, fmt.Errorf("value must be between %d and %d", min, max)
	}
	return val, nil
}

func (v Value) Int32() int32 {
	return toInt32(v.String())
}

func (v Value) Int32Or(n int32) int32 {
	return cmp.Or(v.Int32(), n)
}

// Int32Range validates that the int32 value is within a range
func (v Value) Int32Range(min, max int32) (int32, error) {
	val := v.Int32()
	if val < min || val > max {
		return 0, fmt.Errorf("value must be between %d and %d", min, max)
	}
	return val, nil
}

func (v Value) Int() int {
	return toInt(v.String())
}

func (v Value) IntOr(n int) int {
	return cmp.Or(v.Int(), n)
}

// IntRange validates that the int value is within a range
func (v Value) IntRange(min, max int) (int, error) {
	val := v.Int()
	if val < min || val > max {
		return 0, fmt.Errorf("value must be between %d and %d", min, max)
	}
	return val, nil
}

// Float64 parses the value as a float64
func (v Value) Float64() float64 {
	f, _ := strconv.ParseFloat(v.String(), 64)
	return f
}

// Float64Or returns the parsed float64 or a default value
func (v Value) Float64Or(n float64) float64 {
	if v.IsEmpty() {
		return n
	}
	return v.Float64()
}

// Float64Range validates that the float64 value is within a range
func (v Value) Float64Range(min, max float64) (float64, error) {
	val := v.Float64()
	if val < min || val > max {
		return 0, fmt.Errorf("value must be between %f and %f", min, max)
	}
	return val, nil
}

func (v Value) Bool() bool {
	return toBool(v.String())
}

// BoolOr returns the parsed bool or a default value
func (v Value) BoolOr(b bool) bool {
	if v.IsEmpty() {
		return b
	}
	return v.Bool()
}

// Time parses the value as time using the provided layout
func (v Value) Time(layout string) (time.Time, error) {
	if v.IsEmpty() {
		return time.Time{}, nil
	}
	return time.Parse(layout, v.String())
}

// TimeOr parses the value as time or returns a default value
func (v Value) TimeOr(layout string, defaultTime time.Time) time.Time {
	t, err := v.Time(layout)
	if err != nil {
		return defaultTime
	}
	return t
}

// RFC3339 parses the value as RFC3339 formatted time
func (v Value) RFC3339() (time.Time, error) {
	return v.Time(time.RFC3339)
}

// Date parses the value as a date (YYYY-MM-DD)
func (v Value) Date() (time.Time, error) {
	return v.Time("2006-01-02")
}

// URL parses the value as a URL
func (v Value) URL() (*url.URL, error) {
	if v.IsEmpty() {
		return nil, fmt.Errorf("URL cannot be empty")
	}
	return url.Parse(v.String())
}

// Email validates that the value is a valid email format
func (v Value) Email() error {
	email := v.Trim().String()
	if email == "" {
		return fmt.Errorf("email cannot be empty")
	}
	if !strings.Contains(email, "@") || !strings.Contains(email, ".") {
		return fmt.Errorf("invalid email format")
	}
	parts := strings.Split(email, "@")
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return fmt.Errorf("invalid email format")
	}
	return nil
}

func (v Value) FromBase64() Value {
	return Value(fromBase64(v.String()))
}

func (v Value) ToBase64() Value {
	return Value(toBase64(v.String()))
}

// Match checks if the value matches a pattern (simple glob-style matching)
func (v Value) Match(pattern string) bool {
	// Simple pattern matching - could be enhanced with regex
	if pattern == "*" {
		return true
	}
	if strings.HasPrefix(pattern, "*") && strings.HasSuffix(pattern, "*") {
		return strings.Contains(v.String(), pattern[1:len(pattern)-1])
	}
	if strings.HasPrefix(pattern, "*") {
		return strings.HasSuffix(v.String(), pattern[1:])
	}
	if strings.HasSuffix(pattern, "*") {
		return strings.HasPrefix(v.String(), pattern[:len(pattern)-1])
	}
	return v.String() == pattern
}

// InSlice checks if the value is in a slice of strings
func (v Value) InSlice(slice []string) bool {
	str := v.String()
	for _, s := range slice {
		if s == str {
			return true
		}
	}
	return false
}

// CSV parses comma-separated values and trims whitespace
func (v Value) CSV() []string {
	if v.IsEmpty() {
		return []string{}
	}
	parts := strings.Split(v.String(), ",")
	result := make([]string, len(parts))
	for i, part := range parts {
		result[i] = strings.TrimSpace(part)
	}
	return result
}

func toInt(s string) int {
	i, _ := strconv.Atoi(s)
	return i
}

func toInt32(s string) int32 {
	i, _ := strconv.ParseInt(s, 10, 32)
	return int32(i)
}

func toInt64(s string) int64 {
	i, _ := strconv.ParseInt(s, 10, 64)
	return i
}

func toBool(s string) bool {
	b, _ := strconv.ParseBool(s)
	return b
}

func fromBase64(s string) string {
	b, _ := base64.URLEncoding.DecodeString(s)
	return string(b)
}

func toBase64(s any) string {
	// Skip zero values. We don't want the user to accidentally use the cursor.
	if isZero(s) {
		return ""
	}

	return base64.URLEncoding.EncodeToString(fmt.Appendf(nil, "%v", s))
}

func isZero(a any) bool {
	return reflect.ValueOf(a).IsZero()
}
