package jsonl

import (
	"encoding/json"
	"errors"
	"io"
	"iter"
	"os"
	"path/filepath"
)

// File is an open JSON Lines file holding values of type T.
type File[T any] struct {
	*os.File
}

// Open opens path for reading and appending, creating the file and its
// parent directories if needed.
func Open[T any](path string) (*File[T], error) {
	return OpenFile[T](path, os.O_RDWR|os.O_CREATE|os.O_APPEND)
}

// OpenFile opens path with the given os.OpenFile flags, creating parent
// directories first. New files get mode 0666 before the umask.
func OpenFile[T any](path string, flag int) (*File[T], error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, err
	}
	f, err := os.OpenFile(path, flag, 0o666)
	if err != nil {
		return nil, err
	}
	return &File[T]{f}, nil
}

// Remove deletes the file by name. It does not close f.
func (f *File[T]) Remove() error {
	return os.Remove(f.Name())
}

// ReadLines returns an iterator over the file's values from the start, and
// a function that reports the first seek or decode error once iteration
// stops. Each iteration seeks to the start and leaves the file offset where
// reading stopped.
func (f *File[T]) ReadLines() (iter.Seq[T], func() error) {
	var iterErr error
	seq := func(yield func(T) bool) {
		iterErr = nil
		if _, err := f.Seek(0, io.SeekStart); err != nil {
			iterErr = err
			return
		}
		dec := json.NewDecoder(f.File)
		for {
			var v T
			err := dec.Decode(&v)
			if errors.Is(err, io.EOF) {
				return
			}
			if err != nil {
				iterErr = err
				return
			}
			if !yield(v) {
				return
			}
		}
	}
	return seq, func() error { return iterErr }
}

// Write appends v as one JSON line at the file offset; with Open the offset
// is always the end of the file.
func (f *File[T]) Write(v T) error {
	return json.NewEncoder(f.File).Encode(v)
}
