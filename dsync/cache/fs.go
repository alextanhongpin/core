package cache

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/redis/go-redis/v9/helper"
)

const indexFile = ".index"

type FS struct {
	root *os.Root
	mu   sync.Mutex
	data map[string]time.Time
}

var _ cache[[]byte] = (*FS)(nil)

// NewFile creates a new FS instance with the provided FS client.
func NewFS(dir string) (*FS, error) {
	root, err := os.OpenRoot(dir)
	if err != nil {
		return nil, err
	}
	data := make(map[string]time.Time)
	b, err := root.ReadFile(indexFile)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return nil, err
	}

	if json.Valid(b) {
		err = json.Unmarshal(b, &data)
		if err != nil {
			return nil, err
		}
	}

	return &FS{
		root: root,
		data: data,
	}, nil
}

func (f *FS) Close() error {
	return nil
}

func (f *FS) Load(ctx context.Context, key string) ([]byte, error) {
	f.mu.Lock()
	val, err := f.load(key)
	f.mu.Unlock()
	if err != nil {
		return nil, err
	}

	return val.Val, nil
}

func (f *FS) Store(ctx context.Context, key string, value []byte, ttl time.Duration) error {
	f.mu.Lock()
	err := f.save(key, value, ttl)
	f.mu.Unlock()
	return err
}

func (f *FS) StoreOnce(ctx context.Context, key string, value []byte, ttl time.Duration) error {
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
func (f *FS) LoadOrStore(ctx context.Context, key string, value []byte, ttl time.Duration) (curr []byte, loaded bool, err error) {
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

func (f *FS) LoadOrCreate(ctx context.Context, key string, create func(context.Context, string) ([]byte, time.Duration, error)) (curr []byte, loaded bool, err error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	v, err := f.load(key)
	if err == nil {
		return v.Val, true, nil
	}

	if !errors.Is(err, ErrNotExist) {
		return nil, false, err
	}

	value, ttl, err := create(ctx, key)
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
func (f *FS) LoadAndDelete(ctx context.Context, key string) (value []byte, err error) {
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
func (f *FS) CompareAndDelete(ctx context.Context, key string, old []byte) error {
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
func (f *FS) CompareAndSwap(ctx context.Context, key string, old, value []byte, ttl time.Duration) error {
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
func (f *FS) Exists(ctx context.Context, key string) (bool, error) {
	_, err := f.Load(ctx, key)
	if errors.Is(err, ErrNotExist) {
		return false, nil
	}
	return err == nil, err
}

// TTL returns the remaining time to live for a key.
// Returns -1 if the key exists but has no expiration.
// Returns -2 if the key does not exist.
func (f *FS) TTL(ctx context.Context, key string) (time.Duration, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	v, err := f.load(key)
	if errors.Is(err, ErrNotExist) {
		return -2, nil
	}
	if err != nil {
		return 0, err
	}
	if v.ExpiresAt.IsZero() {
		return -1, nil
	}
	return time.Until(v.ExpiresAt), nil
}

// Expire sets a timeout on a key. After the timeout has expired, the key will automatically be deleted.
func (f *FS) Expire(ctx context.Context, key string, ttl time.Duration) error {
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
func (f *FS) Delete(ctx context.Context, keys ...string) (int64, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	var count int64
	for _, key := range keys {
		err := f.delete(key)
		if err == nil {
			count++
		}
		if errors.Is(err, os.ErrNotExist) {
			continue
		}
		return 0, err
	}
	return count, nil
}

func (f *FS) Size(ctx context.Context) (int, error) {
	f.mu.Lock()
	dir, err := f.root.Open(".")
	if err != nil {
		return 0, err
	}
	entries, err := dir.ReadDir(-1)
	if err != nil {
		return 0, err
	}
	var count int
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		count++
	}
	f.mu.Unlock()

	return count, nil
}

func (f *FS) load(key string) (*Event, error) {
	key = fmt.Sprint(helper.DigestString(key))
	b, err := f.root.ReadFile(key)
	if errors.Is(err, os.ErrNotExist) {
		return nil, ErrNotExist
	}
	if err != nil {
		return nil, err
	}

	it := &Event{
		Key:       key,
		Val:       b,
		ExpiresAt: f.data[key],
	}

	if it.IsExpired() {
		if err := f.delete(key); err != nil {
			return nil, err
		}
		return nil, ErrNotExist
	}

	return it, nil
}

func (f *FS) delete(key string) error {
	key = fmt.Sprint(helper.DigestString(key))
	delete(f.data, key)
	return errors.Join(f.root.RemoveAll(key), f.saveIndex())
}

func (f *FS) save(key string, value []byte, ttl time.Duration) error {
	key = fmt.Sprint(helper.DigestString(key))
	if ttl != 0 {
		f.data[key] = time.Now().Add(ttl)
	}
	return errors.Join(f.root.WriteFile(key, value, 0o644), f.saveIndex())
}

func (f *FS) saveIndex() error {
	b, err := json.Marshal(f.data)
	if err != nil {
		return err
	}
	return f.root.WriteFile(indexFile, b, 0o644)
}
