package templates

import (
	"bytes"
	"html/template"
	"log"
)

var t = template.New("")

type Key[T any] string

func NewKey[T any](name, template string) Key[T] {
	_, err := t.New(name).Parse(template)
	if err != nil {
		log.Fatalf("parse %q template failed", name)
	}

	return Key[T](name)
}

func (k Key[T]) Format(v T) string {
	var b bytes.Buffer
	if err := t.ExecuteTemplate(&b, string(k), v); err != nil {
		panic(err)
	}

	return b.String()
}
