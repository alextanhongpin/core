package fstring

import (
	"bytes"
	"encoding/json"
	"os"
	"text/template"
)

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

type FString[T any] string

func (f FString[T]) FormatFunc(v T, fm template.FuncMap) (string, error) {
	var b bytes.Buffer
	t := template.Must(template.New("").Funcs(fm).Parse(f.String()))
	err := t.Execute(&b, v)
	return b.String(), err
}

func (f FString[T]) Format(v T) (string, error) {
	var b bytes.Buffer
	t := template.Must(template.New("").Parse(f.String()))
	err := t.Execute(&b, v)
	return b.String(), err
}

func (f FString[T]) String() string {
	return string(f)
}

func (f FString[T]) Bytes() []byte {
	return []byte(f)
}

// TODO: Rename to ReadFString
func FStringFromFile[T any](path string) (FString[T], error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}

	return FString[T](string(b)), nil
}
