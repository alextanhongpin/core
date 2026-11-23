// Package request provides utilities for HTTP request parsing, validation, and manipulation.
package request

import (
	"cmp"
	"encoding/base64"
	"fmt"
	"math"
	"net/http"
	"net/url"
	"reflect"
	"slices"
	"strconv"
	"strings"
	"time"
)

// Value represents a string value extracted from HTTP requests with type conversion and validation utilities.
//
// This type wraps string values from various HTTP request sources (query parameters, path values,
// form data, headers) and provides a rich set of methods for type conversion, validation,
// and transformation.
//
// The Value type is designed to make HTTP parameter handling type-safe and convenient,
// reducing boilerplate code for common operations like parsing integers, validating
// email addresses, or checking required fields.
//
// Example usage:
//
//	// Extract and validate query parameters
//	userID := request.QueryValue(r, "user_id").IntOr(0)
//	email := request.QueryValue(r, "email")
//	if err := email.Required("email"); err != nil {
//		return err
//	}
//	if err := email.Email(); err != nil {
//		return err
//	}
//
//	// Extract and convert path parameters
//	id := request.PathValue(r, "id").MustInt()
//	name := request.PathValue(r, "name").Trim().String()
type Value string

// QueryValue extracts a query parameter value from the HTTP request.
//
// This function retrieves the value of a query parameter by name from the request URL.
// If the parameter is not present, it returns an empty Value.
//
// Parameters:
//   - r: The HTTP request to extract the parameter from
//   - name: The name of the query parameter
//
// Returns:
//   - A Value containing the query parameter value
//
// Example:
//
//	// GET /users?page=2&limit=10
//	page := request.QueryValue(r, "page")     // "2"
//	limit := request.QueryValue(r, "limit")   // "10"
//	missing := request.QueryValue(r, "sort")  // ""
func QueryValue(r *http.Request, name string) Value {
	return Value(r.URL.Query().Get(name))
}

// PathValue extracts a path parameter value from the HTTP request.
//
// This function retrieves the value of a path parameter by name from the request.
// Path parameters are typically defined in route patterns (e.g., "/users/{id}").
//
// Parameters:
//   - r: The HTTP request to extract the parameter from
//   - name: The name of the path parameter
//
// Returns:
//   - A Value containing the path parameter value
//
// Example:
//
//	// Route: /users/{id}
//	// Request: GET /users/123
//	userID := request.PathValue(r, "id")  // "123"
func PathValue(r *http.Request, name string) Value {
	return Value(r.PathValue(name))
}

// FormValue extracts a form parameter value from the HTTP request.
//
// This function retrieves the value of a form field by name from POST form data.
// It works with both application/x-www-form-urlencoded and multipart/form-data.
//
// Parameters:
//   - r: The HTTP request to extract the parameter from
//   - name: The name of the form field
//
// Returns:
//   - A Value containing the form field value
//
// Example:
//
//	// POST with Content-Type: application/x-www-form-urlencoded
//	// Body: name=John&email=john@example.com
//	name := request.FormValue(r, "name")    // "John"
//	email := request.FormValue(r, "email")  // "john@example.com"
func FormValue(r *http.Request, name string) Value {
	return Value(r.FormValue(name))
}

// HeaderValue extracts a header value from the HTTP request.
//
// This function retrieves the value of an HTTP header by name. Header names
// are case-insensitive according to HTTP specifications.
//
// Parameters:
//   - r: The HTTP request to extract the header from
//   - name: The name of the header (case-insensitive)
//
// Returns:
//   - A Value containing the header value
//
// Example:
//
//	userAgent := request.HeaderValue(r, "User-Agent")
//	authToken := request.HeaderValue(r, "Authorization")
//	contentType := request.HeaderValue(r, "Content-Type")
func HeaderValue(r *http.Request, name string) Value {
	return Value(r.Header.Get(name))
}

// QueryValues returns all values for a query parameter (for parameters that can have multiple values).
//
// This function is useful when a query parameter can appear multiple times in the URL,
// such as ?tags=go&tags=http&tags=web.
//
// Parameters:
//   - r: The HTTP request to extract the parameters from
//   - name: The name of the query parameter
//
// Returns:
//   - A slice of Values containing all values for the parameter
//
// Example:
//
//	// GET /items?tags=go&tags=http&tags=web
//	tags := request.QueryValues(r, "tags")  // ["go", "http", "web"]
//	for _, tag := range tags {
//		fmt.Println(tag.String())
//	}
func QueryValues(r *http.Request, name string) []Value {
	values := r.URL.Query()[name]
	result := make([]Value, len(values))
	for i, v := range values {
		result[i] = Value(v)
	}
	return result
}

// String returns the underlying string value.
//
// This is the most basic conversion method, returning the Value as a string
// without any processing or validation.
//
// Returns:
//   - The string representation of the value
func (v Value) String() string {
	return string(v)
}

// Trim returns a new Value with leading and trailing whitespace removed.
//
// This method is commonly used to clean up user input from forms and query parameters
// where users might accidentally include leading or trailing spaces.
//
// Returns:
//   - A new Value with whitespace trimmed
//
// Example:
//
//	value := Value("  hello world  ")
//	trimmed := value.Trim()  // "hello world"
func (v Value) Trim() Value {
	return Value(strings.TrimSpace(string(v)))
}

// Lower returns a new Value with all Unicode letters mapped to their lower case.
//
// This method is useful for case-insensitive comparisons and normalization
// of user input.
//
// Returns:
//   - A new Value with all characters converted to lowercase
//
// Example:
//
//	value := Value("Hello World")
//	lower := value.Lower()  // "hello world"
func (v Value) Lower() Value {
	return Value(strings.ToLower(string(v)))
}

// Upper returns a new Value with all Unicode letters mapped to their upper case.
//
// This method is useful for normalization of codes, identifiers, or other
// values that should be stored in uppercase.
//
// Returns:
//   - A new Value with all characters converted to uppercase
//
// Example:
//
//	value := Value("hello world")
//	upper := value.Upper()  // "HELLO WORLD"
func (v Value) Upper() Value {
	return Value(strings.ToUpper(string(v)))
}

// StringOr returns the string value or a default string if the value is empty.
//
// This method provides a convenient way to handle optional parameters with
// default values, using Go's generic cmp.Or function for clean null-coalescing.
//
// Parameters:
//   - str: The default value to return if the Value is empty
//
// Returns:
//   - The Value's string if non-empty, otherwise the default string
//
// Example:
//
//	page := request.QueryValue(r, "page").StringOr("1")
//	sort := request.QueryValue(r, "sort").StringOr("created_at")
func (v Value) StringOr(str string) string {
	return cmp.Or(v.String(), str)
}

// IsEmpty returns true if the value is empty or contains only whitespace.
//
// This method is useful for validation where both empty strings and
// whitespace-only strings should be considered invalid.
//
// Returns:
//   - true if the value is empty or whitespace-only, false otherwise
//
// Example:
//
//	Value("").IsEmpty()       // true
//	Value("   ").IsEmpty()    // true
//	Value("hello").IsEmpty()  // false
func (v Value) IsEmpty() bool {
	return strings.TrimSpace(string(v)) == ""
}

// Required returns an error if the value is empty or whitespace-only.
//
// This method provides a standard way to validate required fields with
// consistent error messages that include the field name.
//
// Parameters:
//   - fieldName: The name of the field being validated (used in error messages)
//
// Returns:
//   - An error if the value is empty, nil if the value is present
//
// Example:
//
//	email := request.QueryValue(r, "email")
//	if err := email.Required("email"); err != nil {
//		return fmt.Errorf("validation failed: %w", err)
//	}
func (v Value) Required(fieldName string) error {
	if v.IsEmpty() {
		return fmt.Errorf("field %s is required", fieldName)
	}
	return nil
}

// Length validates that the string length is within the specified range.
//
// This method checks that the trimmed length of the value falls within
// the specified minimum and maximum bounds (inclusive).
//
// Parameters:
//   - min: The minimum allowed length (inclusive)
//   - max: The maximum allowed length (inclusive)
//
// Returns:
//   - An error if the length is outside the valid range, nil if valid
//
// Example:
//
//	name := request.FormValue(r, "name")
//	if err := name.Length(2, 50); err != nil {
//		return fmt.Errorf("invalid name: %w", err)
//	}
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

// Contains checks if the value contains the specified substring.
//
// This method performs a case-sensitive substring search within the value.
//
// Parameters:
//   - substr: The substring to search for
//
// Returns:
//   - true if the substring is found, false otherwise
//
// Example:
//
//	userAgent := request.HeaderValue(r, "User-Agent")
//	isMobile := userAgent.Contains("Mobile")
func (v Value) Contains(substr string) bool {
	return strings.Contains(string(v), substr)
}

// HasPrefix checks if the value starts with the specified prefix.
//
// This method performs a case-sensitive prefix check, useful for
// validating URL schemes, token types, or other formatted values.
//
// Parameters:
//   - prefix: The prefix to check for
//
// Returns:
//   - true if the value starts with the prefix, false otherwise
//
// Example:
//
//	auth := request.HeaderValue(r, "Authorization")
//	isBearer := auth.HasPrefix("Bearer ")
func (v Value) HasPrefix(prefix string) bool {
	return strings.HasPrefix(string(v), prefix)
}

// HasSuffix checks if the value ends with the specified suffix.
//
// This method performs a case-sensitive suffix check, useful for
// validating file extensions, domain names, or other formatted values.
//
// Parameters:
//   - suffix: The suffix to check for
//
// Returns:
//   - true if the value ends with the suffix, false otherwise
//
// Example:
//
//	filename := request.FormValue(r, "filename")
//	isImage := filename.HasSuffix(".jpg") || filename.HasSuffix(".png")
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

// Int64 parses the value as an int64, returning 0 if it is not a valid
// integer in range.
func (v Value) Int64() int64 {
	n, _ := v.parseInt(64)
	return n
}

// Int64Or returns the parsed int64, or n if the value is empty or invalid.
// An explicit "0" returns 0.
func (v Value) Int64Or(n int64) int64 {
	i, err := v.parseInt(64)
	if err != nil {
		return n
	}
	return i
}

// Int64RangeOr returns the int64 value within a range or a default value if out of range.
func (v Value) Int64RangeOr(min, max, def int64) int64 {
	val, err := v.Int64Range(min, max)
	if err != nil {
		return def
	}
	return val
}

// Int64Range validates that the int64 value is within a range
func (v Value) Int64Range(min, max int64) (int64, error) {
	val, err := v.parseInt(64)
	if err != nil {
		return 0, err
	}
	if val < min || val > max {
		return 0, fmt.Errorf("value must be between %d and %d", min, max)
	}
	return val, nil
}

// Int32 parses the value as an int32, returning 0 if it is not a valid
// integer in range.
func (v Value) Int32() int32 {
	n, _ := v.parseInt(32)
	return int32(n)
}

// Int32Or returns the parsed int32, or n if the value is empty or invalid.
func (v Value) Int32Or(n int32) int32 {
	i, err := v.parseInt(32)
	if err != nil {
		return n
	}
	return int32(i)
}

// Int32RangeOr returns the int32 value within a range or a default value if out of range.
func (v Value) Int32RangeOr(min, max, def int32) int32 {
	val, err := v.Int32Range(min, max)
	if err != nil {
		return def
	}
	return val
}

// Int32Range validates that the int32 value is within a range
func (v Value) Int32Range(min, max int32) (int32, error) {
	n, err := v.parseInt(32)
	if err != nil {
		return 0, err
	}
	val := int32(n)
	if val < min || val > max {
		return 0, fmt.Errorf("value must be between %d and %d", min, max)
	}
	return val, nil
}

// Int parses the value as an int, returning 0 if it is not a valid integer
// in range.
func (v Value) Int() int {
	n, _ := v.parseInt(strconv.IntSize)
	return int(n)
}

// IntOr returns the parsed int, or n if the value is empty or invalid.
// An explicit "0" returns 0.
func (v Value) IntOr(n int) int {
	i, err := v.parseInt(strconv.IntSize)
	if err != nil {
		return n
	}
	return int(i)
}

func (v Value) IntRangeOr(min, max, def int) int {
	val, err := v.IntRange(min, max)
	if err != nil {
		return def
	}
	return val
}

// IntRange validates that the int value is within a range
func (v Value) IntRange(min, max int) (int, error) {
	n, err := v.parseInt(strconv.IntSize)
	if err != nil {
		return 0, err
	}
	val := int(n)
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

// Float64Or returns the parsed float64, or n if the value is empty, invalid,
// NaN or infinite.
func (v Value) Float64Or(n float64) float64 {
	f, err := v.parseFloat()
	if err != nil {
		return n
	}
	return f
}

// Float64RangeOr returns the float64 value within a range or a default value if out of range.
func (v Value) Float64RangeOr(min, max, def float64) float64 {
	val, err := v.Float64Range(min, max)
	if err != nil {
		return def
	}
	return val
}

// Float64Range validates that the float64 value is within a range
func (v Value) Float64Range(min, max float64) (float64, error) {
	val, err := v.parseFloat()
	if err != nil {
		return 0, err
	}
	if val < min || val > max {
		return 0, fmt.Errorf("value must be between %f and %f", min, max)
	}
	return val, nil
}

func (v Value) Bool() bool {
	return toBool(v.String())
}

// BoolOr returns the parsed bool, or b if the value is empty or invalid.
func (v Value) BoolOr(b bool) bool {
	parsed, err := strconv.ParseBool(v.String())
	if err != nil {
		return b
	}
	return parsed
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
	return slices.Contains(slice, str)
}

func (v Value) InSliceOr(slice []string, def string) string {
	str := v.String()
	if slices.Contains(slice, str) {
		return str
	}
	return def
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

// parseInt parses the value strictly; out-of-range values are errors rather
// than being clamped.
func (v Value) parseInt(bitSize int) (int64, error) {
	n, err := strconv.ParseInt(v.String(), 10, bitSize)
	if err != nil {
		return 0, fmt.Errorf("value %q is not a valid integer", v.String())
	}
	return n, nil
}

// parseFloat parses the value as a finite float64.
func (v Value) parseFloat() (float64, error) {
	f, err := strconv.ParseFloat(v.String(), 64)
	if err != nil || math.IsNaN(f) || math.IsInf(f, 0) {
		return 0, fmt.Errorf("value %q is not a valid number", v.String())
	}
	return f, nil
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
