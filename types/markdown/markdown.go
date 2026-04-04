package markdown

import (
	"cmp"
	"errors"
	"io"
	"maps"
	"os"
	"sync/atomic"
	"time"
)

type loader[T any] interface {
	Write(w io.Writer) error
	Read(r io.Reader) (T, error)
}

func NewLoader[T any](path string, meta map[string]any, ttl time.Duration, l loader[T]) *Loader[T] {
	return &Loader[T]{
		loader: l,
		meta:   meta,
		path:   path,
		ttl:    ttl,
	}
}

type Loader[T any] struct {
	loader loader[T]
	meta   map[string]any
	path   string
	ttl    time.Duration
	val    atomic.Pointer[T]
}

func (u *Loader[T]) Load() T {
	return *u.val.Load()
}

func (u *Loader[T]) Sync() error {
	f, err := os.Open(u.path)
	if errors.Is(err, os.ErrNotExist) {
		return cmp.Or(
			u.Download(),
			u.Sync(),
		)
	}
	if err != nil {
		return err
	}
	defer func() {
		_ = f.Close()
	}()

	meta, reader, err := ParseFrontmatter(f)
	if err != nil {
		return err
	}

	if u.shouldUpdate(meta) {
		return cmp.Or(
			u.Download(),
			u.Sync(),
		)
	}

	t, err := u.loader.Read(reader)
	if err != nil {
		return err
	}
	u.val.Swap(&t)

	return nil
}

func (u *Loader[T]) Download() error {
	f, err := os.Create(u.path)
	if err != nil {
		return err
	}
	defer func() {
		_ = f.Close()
	}()

	meta := maps.Clone(u.meta)
	meta["created_at"] = time.Now().Format(time.RFC3339)
	err = WriteFrontmatter(meta, f)
	if err != nil {
		return err
	}

	return u.loader.Write(f)
}

func (u *Loader[T]) shouldUpdate(meta map[string]any) bool {
	s, ok := meta["created_at"].(string)
	if !ok {
		return false
	}

	t, err := time.Parse(time.RFC3339, s)
	if err != nil {
		return false
	}

	return time.Since(t) > u.ttl
}
