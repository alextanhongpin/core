package jsonl

import (
	"encoding/json"
	"errors"
	"io"
	"iter"
	"os"
	"path/filepath"
)

type File[T any] struct {
	*os.File
}

func Open[T any](path string) (*File[T], error) {
	return OpenFile[T](path, os.O_RDWR|os.O_CREATE|os.O_APPEND)
}

func OpenFile[T any](path string, mode int) (*File[T], error) {
	err := os.MkdirAll(filepath.Dir(path), 0o755)
	if err != nil {
		return nil, err
	}

	f, err := os.OpenFile(path, mode, 0666)
	if err != nil {
		return nil, err
	}

	return &File[T]{f}, nil
}

func (f *File[T]) Remove() error {
	return os.Remove(f.Name())
}

func (f *File[T]) ReadLines() (iter.Seq[T], func() error) {
	var iterErr error
	return func(yield func(T) bool) {
			_, iterErr = f.File.Seek(io.SeekStart, 0)
			if iterErr != nil {
				return
			}
			dec := json.NewDecoder(f.File)
			for {
				var v T
				err := dec.Decode(&v)
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
		}, func() error {
			return iterErr
		}
}

func (f *File[T]) Write(v T) error {
	return json.NewEncoder(f.File).Encode(v)
}
