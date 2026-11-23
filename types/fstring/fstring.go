package fstring

import (
	"bytes"
	"encoding/json"
	"os"
	"text/template"
)

// FuncMap holds optional template functions: json encodes a value and
// parse_json decodes a JSON string. Pass it to FormatFunc.
var FuncMap = template.FuncMap{
	"json": func(v any) (string, error) {
		b, err := json.Marshal(v)
		return string(b), err
	},
	"parse_json": func(s string) (any, error) {
		var a any
		err := json.Unmarshal([]byte(s), &a)
		return a, err
	},
}

// FString is a text/template source whose data has type T.
type FString[T any] string

// Format executes the template with v as its data.
func (f FString[T]) Format(v T) (string, error) {
	return f.FormatFunc(v, nil)
}

// FormatFunc executes the template with v as its data and fm available as
// template functions. It returns an error, and no output, if the template
// fails to parse or execute.
func (f FString[T]) FormatFunc(v T, fm template.FuncMap) (string, error) {
	t, err := template.New("").Funcs(fm).Parse(string(f))
	if err != nil {
		return "", err
	}
	var b bytes.Buffer
	if err := t.Execute(&b, v); err != nil {
		return "", err
	}
	return b.String(), nil
}

// String returns the template source.
func (f FString[T]) String() string {
	return string(f)
}

// Bytes returns the template source.
func (f FString[T]) Bytes() []byte {
	return []byte(f)
}

// ReadFile reads a template source from path.
func ReadFile[T any](path string) (FString[T], error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return FString[T](b), nil
}
