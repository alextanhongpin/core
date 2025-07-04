// Package structs provides utilities for struct reflection, validation, and analysis.
// It offers type introspection, field validation, and struct manipulation helpers.
package structs

import (
	"encoding/json"
	"fmt"
	"maps"
	"reflect"
	"slices"
	"strings"
)

// Type returns the full type name of the value including package information.
// Returns "nil" for nil values.
func Type(v any) string {
	if v == nil {
		return "nil"
	}
	return reflect.TypeOf(v).String()
}

// PkgName returns the package-qualified type name of the value.
// For structs, returns the full package.Type format.
// For other types, returns the kind name.
func PkgName(v any) string {
	if v == nil {
		return "nil"
	}

	t := reflect.TypeOf(v)

	// Handle pointers by getting the element type
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}

	// For struct types, return the full package.Type name
	if t.Kind() == reflect.Struct {
		return t.String()
	}

	// For other types, return the kind name
	return t.Kind().String()
}

// Name returns the type name without the package prefix.
// For "package.TypeName", returns just "TypeName".
func Name(v any) string {
	pkgName := PkgName(v)
	parts := strings.Split(pkgName, ".")
	return parts[len(parts)-1]
}

// Kind returns the underlying kind of the value.
func Kind(v any) reflect.Kind {
	if v == nil {
		return reflect.Invalid
	}

	t := reflect.TypeOf(v)
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}

	return t.Kind()
}

// IsStruct returns true if the value is a struct or pointer to struct.
func IsStruct(v any) bool {
	return Kind(v) == reflect.Struct
}

// IsPointer returns true if the value is a pointer.
func IsPointer(v any) bool {
	if v == nil {
		return false
	}
	return reflect.TypeOf(v).Kind() == reflect.Ptr
}

// IsNil returns true if the value is nil or a nil pointer.
func IsNil(v any) bool {
	if v == nil {
		return true
	}

	val := reflect.ValueOf(v)
	switch val.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface,
		reflect.Map, reflect.Ptr, reflect.Slice:
		return val.IsNil()
	default:
		return false
	}
}

// NonZero validates that all fields in a struct are non-zero values.
// It recursively checks nested structs, slices, and maps.
// Returns a FieldError indicating the first empty field found.
func NonZero(v any) error {
	if v == nil {
		return newFieldError("nil")
	}

	data, err := toMap(v)
	if err != nil {
		return fmt.Errorf("failed to convert to map: %w", err)
	}

	return validateNonZero(PkgName(v), data)
}

// GetFields returns a map of field names to their values for a struct.
// Only works with struct types.
func GetFields(v any) (map[string]any, error) {
	if !IsStruct(v) {
		return nil, fmt.Errorf("value is not a struct")
	}

	t := reflect.TypeOf(v)
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}

	result := make(map[string]any)
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		result[field.Name] = reflect.ValueOf(v).Field(i).Interface()
	}

	return result, nil
}

// HasField checks if a struct has a field with the given name.
func HasField(v any, fieldName string) bool {
	if !IsStruct(v) {
		return false
	}

	t := reflect.TypeOf(v)
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}

	_, found := t.FieldByName(fieldName)
	return found
}

// GetFieldValue returns the value of a field by name.
func GetFieldValue(v any, fieldName string) (any, error) {
	if !IsStruct(v) {
		return nil, fmt.Errorf("value is not a struct")
	}

	val := reflect.ValueOf(v)
	if val.Kind() == reflect.Ptr {
		val = val.Elem()
	}

	field := val.FieldByName(fieldName)
	if !field.IsValid() {
		return nil, fmt.Errorf("field %q not found", fieldName)
	}

	return field.Interface(), nil
}

// GetFieldNames returns all field names in a struct.
func GetFieldNames(v any) ([]string, error) {
	if !IsStruct(v) {
		return nil, fmt.Errorf("value is not a struct")
	}

	t := reflect.TypeOf(v)
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}

	var names []string
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		if field.IsExported() {
			names = append(names, field.Name)
		}
	}

	return names, nil
}

// GetTags returns all struct tags for a given tag key.
func GetTags(v any, tagKey string) (map[string]string, error) {
	if !IsStruct(v) {
		return nil, fmt.Errorf("value is not a struct")
	}

	t := reflect.TypeOf(v)
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}

	tags := make(map[string]string)
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		if field.IsExported() {
			if tag := field.Tag.Get(tagKey); tag != "" {
				tags[field.Name] = tag
			}
		}
	}

	return tags, nil
}

// Clone creates a deep copy of a struct using JSON marshaling/unmarshaling.
// Note: This only works for JSON-serializable fields.
func Clone[T any](v T) (T, error) {
	data, err := json.Marshal(v)
	if err != nil {
		var zero T
		return zero, fmt.Errorf("failed to marshal: %w", err)
	}

	var clone T
	if err := json.Unmarshal(data, &clone); err != nil {
		return clone, fmt.Errorf("failed to unmarshal: %w", err)
	}

	return clone, nil
}

// Helper functions

// validateNonZero recursively validates that all fields are non-zero.
func validateNonZero(path string, v any) error {
	switch data := v.(type) {
	case map[string]any:
		if len(data) == 0 {
			return newFieldError(path)
		}

		keys := slices.Sorted(maps.Keys(data))
		for _, key := range keys {
			value := data[key]
			if err := validateNonZero(joinPath(path, key), value); err != nil {
				return err
			}
		}
	case []any:
		if len(data) == 0 {
			return newFieldError(path)
		}
		for i, value := range data {
			if err := validateNonZero(joinPath(path, fmt.Sprintf("[%d]", i)), value); err != nil {
				return err
			}
		}
	default:
		if isEmpty(data) {
			return newFieldError(path)
		}
	}
	return nil
}

// joinPath joins path components with dots, handling empty components.
func joinPath(components ...string) string {
	var parts []string
	for _, component := range components {
		if component != "" {
			parts = append(parts, component)
		}
	}
	return strings.Join(parts, ".")
}

// isEmpty checks if a value is considered empty.
func isEmpty(v any) bool {
	return v == nil || v == 0.0 || v == false || v == ""
}

// toMap converts a value to a map using JSON marshaling/unmarshaling.
func toMap(v any) (any, error) {
	data, err := json.Marshal(v)
	if err != nil {
		return nil, err
	}

	var result any
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, err
	}

	return result, nil
}

// FieldError represents an error for a specific field.
type FieldError struct {
	Path  string
	Field string
}

// newFieldError creates a new FieldError from a path.
func newFieldError(path string) *FieldError {
	parts := strings.Split(path, ".")
	field := parts[len(parts)-1]
	return &FieldError{
		Path:  path,
		Field: field,
	}
}

// Error returns the error message.
func (fe *FieldError) Error() string {
	return fmt.Sprintf("field %q is empty", fe.Field)
}

// String returns a string representation of the error.
func (fe *FieldError) String() string {
	return fmt.Sprintf("FieldError{Path: %q, Field: %q}", fe.Path, fe.Field)
}
