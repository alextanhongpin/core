package cache

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"sync"
	"time"

	"golang.org/x/sync/singleflight"
	"golang.org/x/sys/unix"
)

// Event is one entry as persisted by [File]: the key, its value and its
// expiry time (zero means no expiry).
type Event struct {
	Key       string    `json:"key,omitzero"`
	Val       []byte    `json:"val,omitzero"`
	ExpiresAt time.Time `json:"expiresAt,omitzero"`
}

// IsExpired reports whether the entry has an expiry that has passed.
func (e *Event) IsExpired() bool {
	return !e.ExpiresAt.IsZero() && time.Since(e.ExpiresAt) > 0
}

// NewEvent returns an Event holding a copy of val that expires after ttl
// (ttl <= 0 means never).
func NewEvent(key string, val []byte, ttl time.Duration) *Event {
	var expiresAt time.Time
	if ttl > 0 {
		expiresAt = time.Now().Add(ttl)
	}
	return &Event{
		Key:       key,
		Val:       slices.Clone(val),
		ExpiresAt: expiresAt,
	}
}

// File is a [Storage] that keeps all entries in memory and persists them as a
// single JSON file. Every write rewrites the whole file atomically (write to
// a temp file, fsync, rename), so it suits small caches such as local
// snapshots or CLI tools, not high write rates.
//
// While open, File holds an exclusive flock(2) on a sidecar "<path>.lock"
// file. A second NewFile for the same path, in this or another process,
// blocks until the first is closed. Always call Close.
type File struct {
	group  singleflight.Group
	path   string
	closed bool
	file   *os.File
	close  func() error

	// TODO: Add sync snapshot.
	mu   sync.Mutex
	data map[string]*Event
}

var _ cache[[]byte] = (*File)(nil)

// NewFile opens (or creates) the JSON cache file at path, creating parent
// directories as needed, and loads its entries. It blocks until it can take
// the exclusive lock on "<path>.lock".
func NewFile(path string) (*File, error) {
	f, close, err := lockFile(path)
	if err != nil {
		return nil, err
	}

	data := make(map[string]*Event)
	err = json.NewDecoder(f).Decode(&data)
	if err != nil && !errors.Is(err, io.EOF) {
		return nil, errors.Join(err, close())
	}
	if data == nil {
		data = make(map[string]*Event)
	}
	return &File{
		path:  path,
		file:  f,
		close: close,
		data:  data,
	}, nil
}

// Close releases the file lock. Calls after Close return [ErrClosed].
// Close is idempotent.
func (f *File) Close() error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.closed {
		return nil
	}
	f.closed = true
	return f.close()
}

// Load returns a copy of the value for key, or [ErrNotExist].
func (f *File) Load(ctx context.Context, key string) ([]byte, error) {
	f.mu.Lock()
	val, err := f.load(key)
	f.mu.Unlock()
	if err != nil {
		return nil, err
	}

	return slices.Clone(val.Val), nil
}

// Store sets key's value and persists the file. ttl <= 0 means no expiry.
func (f *File) Store(ctx context.Context, key string, value []byte, ttl time.Duration) error {
	f.mu.Lock()
	err := f.save(key, value, ttl)
	f.mu.Unlock()
	return err
}

// StoreOnce stores value only if key does not exist; otherwise it returns
// [ErrExists].
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
		return slices.Clone(v.Val), true, nil
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

// LoadOrCreate coalesces local fills per key without holding the storage
// mutex during create. Factories may access other keys, but must not recursively
// fill the same key. Followers share the leader's context and wait for it.
func (f *File) LoadOrCreate(ctx context.Context, key string, create func(context.Context, string) ([]byte, time.Duration, error)) ([]byte, bool, error) {
	if err := context.Cause(ctx); err != nil {
		return nil, false, err
	}
	curr, err := f.Load(ctx, key)
	if err == nil {
		return curr, true, nil
	}
	if !errors.Is(err, ErrNotExist) {
		return nil, false, err
	}
	if create == nil {
		return nil, false, errors.New("cache: create is required")
	}
	created := false
	result, err, _ := f.group.Do(key, func() (any, error) {
		curr, err := f.Load(ctx, key)
		if err == nil {
			return curr, nil
		}
		if !errors.Is(err, ErrNotExist) {
			return nil, err
		}
		value, ttl, err := create(ctx, key)
		if err != nil {
			return nil, err
		}
		if err := context.Cause(ctx); err != nil {
			return nil, err
		}
		curr, loaded, err := f.LoadOrStore(ctx, key, value, ttl)
		created = err == nil && !loaded
		return curr, err
	})
	if err != nil {
		return nil, false, err
	}
	return slices.Clone(result.([]byte)), !created, nil
}

// LoadAndDelete atomically returns and deletes the value for key, or returns
// [ErrNotExist].
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

// CompareAndDelete deletes key only if its value equals old. It returns
// [ErrNotExist] if the key is missing or the value differs.
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

// CompareAndSwap sets key to value only if its current value equals old. It
// returns [ErrNotExist] if the key is missing or the value differs.
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

// Exists reports whether key exists and has not expired.
func (f *File) Exists(ctx context.Context, key string) (bool, error) {
	_, err := f.Load(ctx, key)
	if errors.Is(err, ErrNotExist) {
		return false, nil
	}
	return err == nil, err
}

// TTL returns the remaining time to live for key: -1 if it has no expiry, -2
// if it does not exist.
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
	if v.ExpiresAt.IsZero() {
		return -1, nil
	}
	return time.Until(v.ExpiresAt), nil
}

// Expire sets a new TTL on an existing key; ttl <= 0 deletes it. It returns
// [ErrNotExist] if the key does not exist.
func (f *File) Expire(ctx context.Context, key string, ttl time.Duration) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	v, err := f.load(key)
	if err != nil {
		return err
	}
	if ttl <= 0 {
		return f.delete(key)
	}
	return f.save(key, v.Val, ttl)
}

// Delete removes keys and returns how many were present.
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

// Size returns the number of stored entries, including expired entries that
// have not been read (and thus removed) yet.
func (f *File) Size(ctx context.Context) (int, error) {
	f.mu.Lock()
	count := len(f.data)
	f.mu.Unlock()

	return count, nil
}

func (f *File) load(key string) (*Event, error) {
	if f.closed {
		return nil, ErrClosed
	}
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

func (f *File) save(key string, value []byte, ttl time.Duration) error {
	if f.closed {
		return ErrClosed
	}
	before := maps.Clone(f.data)
	f.data[key] = NewEvent(key, value, ttl)
	if err := f.flush(); err != nil {
		f.data = before
		return err
	}
	return nil
}
func (f *File) delete(key string) error {
	if f.closed {
		return ErrClosed
	}
	before := maps.Clone(f.data)
	delete(f.data, key)
	if err := f.flush(); err != nil {
		f.data = before
		return err
	}
	return nil
}
func (f *File) flush() error {
	file, err := os.CreateTemp(filepath.Dir(f.path), ".cache-*")
	if err != nil {
		return err
	}
	defer os.Remove(file.Name())
	encodeErr := json.NewEncoder(file).Encode(f.data)
	chmodErr := file.Chmod(0o644)
	// Sync before rename so a crash cannot leave an empty or partial file
	// behind the final name.
	syncErr := file.Sync()
	if err := errors.Join(encodeErr, chmodErr, syncErr, file.Close()); err != nil {
		return err
	}
	return os.Rename(file.Name(), f.path)
}

func lockFile(name string) (*os.File, func() error, error) {
	if err := os.MkdirAll(filepath.Dir(name), 0o755); err != nil {
		return nil, nil, err
	}
	lock, err := os.OpenFile(name+".lock", os.O_CREATE|os.O_RDWR, 0o644)
	if err != nil {
		return nil, nil, err
	}
	if err := unix.Flock(int(lock.Fd()), unix.LOCK_EX); err != nil {
		return nil, nil, errors.Join(err, lock.Close())
	}
	release := func() error { return errors.Join(unix.Flock(int(lock.Fd()), unix.LOCK_UN), lock.Close()) }
	file, err := os.OpenFile(name, os.O_CREATE|os.O_RDWR, 0o644)
	if err != nil {
		return nil, nil, errors.Join(err, release())
	}
	return file, sync.OnceValue(func() error { return errors.Join(file.Close(), release()) }), nil
}
