// Package assert provides utilities for building structured validation systems.
// It enables creating composable validation logic that can be used for
// API request validation, configuration validation, and other scenarios
// where structured error reporting is needed.
package assert

import (
	"cmp"
	"fmt"
	"reflect"
	"slices"
	"strings"

	"github.com/alextanhongpin/core/types/email"
)

// Is returns an empty string if the condition is true, otherwise returns the message.
// This is useful for conditional validation.
func Is(is bool, msg string, args ...any) string {
	if is {
		return ""
	}
	return fmt.Sprintf(msg, args...)
}

// Required validates that a value is non-zero and applies additional assertions.
// Returns a comma-separated string of validation errors.
func Required(v any, assertions ...string) string {
	var sb strings.Builder

	if IsZero(v) {
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
	if IsZero(v) {
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

// IsZero checks if a value is the zero value for its type.
func IsZero(v any) bool {
	if v == nil {
		return true
	}
	val := reflect.ValueOf(v)
	if slices.Contains([]reflect.Kind{reflect.Map, reflect.Slice, reflect.Array}, val.Kind()) {
		return val.Len() == 0
	}
	if val.IsZero() {
		return true
	}

	if val.Kind() == reflect.Pointer {
		return val.Elem().IsZero()
	}

	return false
}

// MinLength validates that a string has at least the specified length.
func MinLength(s string, min int) string {
	return Is(len(s) >= min, "must be at least %d characters", min)
}

// MaxLength validates that a string has at most the specified length.
func MaxLength(s string, max int) string {
	return Is(len(s) <= max, "must be at most %d characters", max)
}

// Range validates that a value is between min and max (inclusive).
func Range[T cmp.Ordered](v, lo, hi T) string {
	return Is(v >= lo && v <= hi, "must be between %v and %v", lo, hi)
}

// Email validates that a string is a valid email address.
func Email(s string) string {
	return Is(email.IsValid(s), "must be a valid email address")
}

// OneOf validates that a value is one of the allowed values.
func OneOf[T comparable](v T, allowed ...T) string {
	return Is(slices.Contains(allowed, v), "must be one of %v", allowed)
}
