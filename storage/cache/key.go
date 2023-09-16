package cache

import (
	"bytes"
	"fmt"
	"html/template"
	"time"
)

type TypedKey[T any] struct {
	t   *template.Template
	TTL time.Duration
}

func NewTypedKey[T any](key string, ttl time.Duration) *TypedKey[T] {
	return &TypedKey[T]{
		t:   template.Must(template.New("").Parse(key)),
		TTL: ttl,
	}
}

func (k TypedKey[T]) Format(v T) string {
	var b bytes.Buffer
	if err := k.t.Execute(&b, v); err != nil {
		panic("cache: failed to execute template")
	}

	return b.String()
}

type Key struct {
	key string
	TTL time.Duration
}

func NewKey(key string, ttl time.Duration) *Key {
	return &Key{
		key: key,
		TTL: ttl,
	}
}

func (k Key) Format(args ...any) string {
	return fmt.Sprintf(string(k.key), args...)
}
