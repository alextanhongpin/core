// Package assert builds field validation messages. Each check returns an
// empty string when it passes and a message when it fails; Required and
// Optional join messages for one field, and Map drops the fields that passed.
package assert

import (
	"cmp"
	"fmt"
	"reflect"
	"slices"
	"strings"
	"unicode/utf8"

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

// IsZero reports whether v is nil, a zero value, an empty map or slice, or a
// pointer to a zero value.
func IsZero(v any) bool {
	if v == nil {
		return true
	}
	val := reflect.ValueOf(v)
	switch val.Kind() {
	case reflect.Map, reflect.Slice:
		return val.Len() == 0
	case reflect.Pointer:
		return val.IsNil() || val.Elem().IsZero()
	}
	return val.IsZero()
}

// MinLength validates that s has at least min characters (runes, not bytes).
func MinLength(s string, min int) string {
	return Is(utf8.RuneCountInString(s) >= min, "must be at least %d characters", min)
}

// MaxLength validates that s has at most max characters (runes, not bytes).
func MaxLength(s string, max int) string {
	return Is(utf8.RuneCountInString(s) <= max, "must be at most %d characters", max)
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
