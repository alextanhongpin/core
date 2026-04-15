package cache

import (
	"bytes"
	"cmp"
	"context"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"sync"
	"time"
)

type Event struct {
	Name      string    `json:"name,omitzero"`
	Key       string    `json:"key,omitzero"`
	Val       []byte    `json:"val,omitzero"`
	ExpiresAt time.Time `json:"expiresAt,omitzero"`
}

func (e *Event) IsExpired() bool {
	return time.Since(e.ExpiresAt) > 0
}

func NewEvent(name, key string, val []byte, ttl time.Duration) *Event {
	return &Event{
		Name:      name,
		Key:       key,
		Val:       val,
		ExpiresAt: time.Now().Add(ttl),
	}
}

type File struct {
	file *os.File

	mu   sync.Mutex
	data map[string]*Event
}

var _ Storage[[]byte] = (*File)(nil)

// NewFile creates a new File instance with the provided File client.
func NewFile(path string) (*File, error) {
	data := make(map[string]*Event)
	if err := load(path, data); err != nil {
		return nil, err
	}

	f, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0o644)
	if err != nil {
		return nil, err
	}

	return &File{
		data: data,
		file: f,
	}, nil
}

func (f *File) Close() error {
	return f.file.Close()
}

func (f *File) Load(ctx context.Context, key string) ([]byte, error) {
	f.mu.Lock()
	val, err := f.load(key)
	f.mu.Unlock()
	if err != nil {
		return nil, err
	}

	return val.Val, nil
}

func (f *File) Store(ctx context.Context, key string, value []byte, ttl time.Duration) error {
	f.mu.Lock()
	err := f.save(key, value, ttl)
	f.mu.Unlock()
	return err
}

func (f *File) StoreOnce(ctx context.Context, key string, value []byte, ttl time.Duration) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	_, err := f.load(key)
	if err == nil {
		return ErrExists
	}
	if !errors.Is(err, ErrNotExist) {
		return err
	}

	return f.save(key, value, ttl)
}

// LoadOrStore returns the existing value for the key if present. Otherwise, it
// stores and returns the given value. The loaded result is true if the value
// was loaded, false if stored.
// Also see usecase here: https://github.com/golang/go/issues/33762#issuecomment-523757434
func (f *File) LoadOrStore(ctx context.Context, key string, value []byte, ttl time.Duration) (curr []byte, loaded bool, err error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	v, err := f.load(key)
	if err == nil {
		return v.Val, true, nil
	}

	if !errors.Is(err, ErrNotExist) {
		return nil, false, err
	}

	err = f.save(key, value, ttl)
	if err != nil {
		return nil, false, err
	}
	return value, false, nil
}

func (f *File) LoadOrStoreFunc(ctx context.Context, key string, getter func(context.Context, string) ([]byte, time.Duration, error)) (curr []byte, loaded bool, err error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	v, err := f.load(key)
	if err == nil {
		return v.Val, true, nil
	}

	if !errors.Is(err, ErrNotExist) {
		return nil, false, err
	}

	value, ttl, err := getter(ctx, key)
	if err != nil {
		return nil, false, err
	}

	err = f.save(key, value, ttl)
	if err != nil {
		return nil, false, err
	}
	return value, false, nil
}

// LoadAndDelete deletes the value for a key, returning the previous value if
// any. The loaded result reports whether the key was present.
// Also see usecase here: https://github.com/golang/go/issues/33762#issuecomment-523757434
func (f *File) LoadAndDelete(ctx context.Context, key string) (value []byte, err error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	v, err := f.load(key)
	if err != nil {
		return nil, err
	}
	err = f.delete(key)
	if err != nil {
		return nil, err
	}

	return v.Val, nil
}

// CompareAndDelete deletes the entry for key if its value is equal to old. The
// old value must be of a comparable type.
// If there is no current value for key in the map, CompareAndDelete returns
// false (even if the old value is the nil interface value).
func (f *File) CompareAndDelete(ctx context.Context, key string, old []byte) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	v, err := f.load(key)
	if err != nil {
		return err
	}

	if !bytes.Equal(v.Val, old) {
		return ErrNotExist
	}

	return f.delete(key)
}

// CompareAndSwap swaps the old and new values for key if the value stored in
// the map is equal to old. The old value must be of a comparable type.
func (f *File) CompareAndSwap(ctx context.Context, key string, old, value []byte, ttl time.Duration) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	v, err := f.load(key)
	// NOTE: This is to follow the redis implementation.
	if errors.Is(err, ErrNotExist) {
		return ErrNotExist
	}
	if err != nil {
		return err
	}

	if !bytes.Equal(v.Val, old) {
		return ErrNotExist
	}

	return f.save(key, value, ttl)
}

// Exists checks if a key exists in the cache.
func (f *File) Exists(ctx context.Context, key string) (bool, error) {
	_, err := f.Load(ctx, key)
	if errors.Is(err, ErrNotExist) {
		return false, nil
	}
	return err == nil, err
}

// TTL returns the remaining time to live for a key.
// Returns -1 if the key exists but has no expiration.
// Returns -2 if the key does not exist.
func (f *File) TTL(ctx context.Context, key string) (time.Duration, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	v, err := f.load(key)
	if errors.Is(err, ErrNotExist) {
		return -2, nil
	}
	if err != nil {
		return 0, err
	}
	return time.Until(v.ExpiresAt), nil
}

// Expire sets a timeout on a key. After the timeout has expired, the key will automatically be deleted.
func (f *File) Expire(ctx context.Context, key string, ttl time.Duration) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	v, err := f.load(key)
	if err != nil {
		return err
	}
	v.ExpiresAt = time.Now().Add(ttl)

	return nil
}

// Delete removes one or more keys from the cache.
func (f *File) Delete(ctx context.Context, keys ...string) (int64, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	var count int64
	for _, key := range keys {
		if _, ok := f.data[key]; ok {
			count++
			if err := f.delete(key); err != nil {
				return 0, err
			}
		}
	}
	return count, nil
}

func (f *File) Size(ctx context.Context) (int, error) {
	f.mu.Lock()
	count := len(f.data)
	f.mu.Unlock()

	return count, nil
}

func (f *File) load(key string) (*Event, error) {
	it, ok := f.data[key]
	if !ok {
		return nil, ErrNotExist
	}

	if it.IsExpired() {
		err := f.delete(key)
		if err != nil {
			return nil, err
		}

		return nil, ErrNotExist
	}

	return it, nil
}

func load(path string, data map[string]*Event) error {
	err := os.MkdirAll(filepath.Dir(path), 0o755)
	if err != nil {
		return err
	}

	f, err := os.Open(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	defer f.Close()

	tmpPath := path + ".tmp"
	tmp, err := os.Create(tmpPath)
	if err != nil {
		panic(err)
	}
	defer tmp.Close()

	dec := json.NewDecoder(f)
	enc := json.NewEncoder(tmp)
	for {
		var e Event
		err := dec.Decode(&e)
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return err
		}
		switch e.Name {
		case "set":
			if e.IsExpired() {
				continue
			}
			data[e.Key] = &e
		case "del":
			delete(data, e.Key)
			continue
		}
		if err := enc.Encode(e); err != nil {
			return err
		}
	}

	return cmp.Or(
		f.Close(),
		tmp.Close(),
		os.Rename(tmpPath, path),
	)
}

func (f *File) save(key string, val []byte, ttl time.Duration) error {
	return f.flush("set", key, val, ttl)
}

func (f *File) delete(key string) error {
	return f.flush("del", key, nil, 0)
}

func (f *File) flush(name, key string, val []byte, ttl time.Duration) error {
	evt := NewEvent(name, key, val, ttl)
	switch name {
	case "set":
		f.data[key] = evt
	case "del":
		delete(f.data, key)
	}
	b, err := json.Marshal(evt)
	if err != nil {
		return err
	}
	_, err = f.file.Write(b)
	if err != nil {
		return err
	}

	return nil
}
