package jsonl

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"iter"
	"os"
	"path/filepath"
	"slices"
)

type JSONL[T any] struct {
	path string
}

func New[T any](path string) *JSONL[T] {
	if filepath.Ext(path) != ".jsonl" {
		panic(fmt.Errorf("path must have .jsonl extension: %q", path))
	}
	return &JSONL[T]{
		path: path,
	}
}

func (s *JSONL[T]) All() (iter.Seq[T], func() error) {
	var iterErr error
	seq := func(yield func(T) bool) {
		f, err := os.Open(s.path)
		if errors.Is(err, os.ErrNotExist) {
			return
		}
		if err != nil {
			iterErr = err
			return
		}

		defer func() {
			_ = f.Close()
		}()

		dec := json.NewDecoder(f)
		for {
			var v T
			err = dec.Decode(&v)
			if errors.Is(err, io.EOF) {
				break
			}
			if err != nil {
				iterErr = err
				break
			}
			if !yield(v) {
				break
			}
		}
	}

	return seq, func() error {
		return iterErr
	}
}

func (s *JSONL[T]) Head(n int) ([]T, error) {
	seq, stop := s.All()
	res := make([]T, 0, n)
	for row := range seq {
		res = append(res, row)
		if len(res) == n {
			break
		}
	}
	if err := stop(); err != nil {
		return nil, err
	}

	return res, nil
}

func (s *JSONL[T]) Load() ([]T, error) {
	seq, stop := s.All()
	return slices.Collect(seq), stop()
}

func (s *JSONL[T]) Remove() error {
	return os.Remove(s.path)
}

func (s *JSONL[T]) Store(msgs ...T) error {
	err := os.MkdirAll(filepath.Dir(s.path), 0o755)
	if err != nil {
		return err
	}
	f, err := os.OpenFile(s.path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return err
	}
	defer func() {
		_ = f.Close()
	}()

	enc := json.NewEncoder(f)
	for _, msg := range msgs {
		if err := enc.Encode(msg); err != nil {
			return err
		}
	}

	return nil
}

func (s *JSONL[T]) Tail(n int) ([]T, error) {
	seq, stop := s.All()
	pos := 0
	res := make([]T, n)

	for v := range seq {
		res[pos%n] = v
		pos++
	}
	if err := stop(); err != nil {
		return nil, err
	}

	if pos > n {
		i := pos % n
		return append(res[i:], res[:i]...), nil
	}

	return res[:pos], nil
}
