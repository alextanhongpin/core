// Package assert provides utilities for building structured validation systems.
// It enables creating composable validation logic that can be used for
// API request validation, configuration validation, and other scenarios
// where structured error reporting is needed.
package assert

import (
	"fmt"
	"reflect"
	"regexp"
	"strings"
)

var (
	// Common email validation regex
	emailRegex = regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)
)

// Is returns an empty string if the condition is true, otherwise returns the message.
// This is useful for conditional validation.
func Is(is bool, msg string) string {
	if is {
		return ""
	}
	return msg
}

// Required validates that a value is non-zero and applies additional assertions.
// Returns a comma-separated string of validation errors.
func Required(v any, assertions ...string) string {
	var sb strings.Builder

	if isZero(v) {
		sb.WriteString("required")
		sb.WriteString(", ")
	}

	for _, s := range assertions {
		if s == "" {
			continue
		}
		sb.WriteString(s)
		sb.WriteString(", ")
	}

	return strings.TrimSuffix(sb.String(), ", ")
}

// Optional applies assertions only if the value is non-zero.
// If the value is zero, returns empty string (no validation errors).
func Optional(v any, assertions ...string) string {
	if isZero(v) {
		return ""
	}
	return Required(v, assertions...)
}

// Map filters out empty validation messages from the map.
// Returns a new map containing only non-empty validation errors.
func Map(kv map[string]string) map[string]string {
	res := make(map[string]string)
	for k, v := range kv {
		if v == "" {
			continue
		}
		res[k] = v
	}
	return res
}

// MinLength validates that a string has at least the specified length.
func MinLength(s string, min int) string {
	return Is(len(s) >= min, fmt.Sprintf("must be at least %d characters", min))
}

// MaxLength validates that a string has at most the specified length.
func MaxLength(s string, max int) string {
	return Is(len(s) <= max, fmt.Sprintf("must be at most %d characters", max))
}

// Range validates that a value is between min and max (inclusive).
func Range[T comparable](v, min, max T) string {
	rv := reflect.ValueOf(v)
	rmin := reflect.ValueOf(min)
	rmax := reflect.ValueOf(max)

	switch rv.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		vi, mini, maxi := rv.Int(), rmin.Int(), rmax.Int()
		return Is(vi >= mini && vi <= maxi, fmt.Sprintf("must be between %d and %d", mini, maxi))
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		vi, mini, maxi := rv.Uint(), rmin.Uint(), rmax.Uint()
		return Is(vi >= mini && vi <= maxi, fmt.Sprintf("must be between %d and %d", mini, maxi))
	case reflect.Float32, reflect.Float64:
		vi, mini, maxi := rv.Float(), rmin.Float(), rmax.Float()
		return Is(vi >= mini && vi <= maxi, fmt.Sprintf("must be between %g and %g", mini, maxi))
	case reflect.String:
		vi, mini, maxi := rv.String(), rmin.String(), rmax.String()
		return Is(vi >= mini && vi <= maxi, fmt.Sprintf("must be between %s and %s", mini, maxi))
	}
	return ""
}

// Email validates that a string is a valid email address.
func Email(s string) string {
	return Is(emailRegex.MatchString(s), "must be a valid email address")
}

// OneOf validates that a value is one of the allowed values.
func OneOf[T comparable](v T, allowed ...T) string {
	for _, a := range allowed {
		if v == a {
			return ""
		}
	}
	return fmt.Sprintf("must be one of: %v", allowed)
}

// isZero checks if a value is the zero value for its type.
func isZero(v any) bool {
	return v == nil || reflect.ValueOf(v).IsZero()
}
