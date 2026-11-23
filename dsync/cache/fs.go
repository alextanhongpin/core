package cache

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math/rand/v2"
	"os"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/redis/go-redis/v9/helper"
	"golang.org/x/sync/singleflight"
)

const indexFile = ".index"

// FS is a [Storage] that stores each value as its own file in a directory.
// File names are a 64-bit hash of the key; expiry times are kept in an
// ".index" JSON file in the same directory. It suits larger values (such as
// memoized results with [GoFunc]) that should survive restarts.
//
// FS is safe for concurrent use within one process but does no
// cross-process locking: only one FS may use a directory at a time. Each file
// is replaced atomically (temp file, fsync, rename), but a value file and the
// index are written separately, so a crash between them can leave a value
// without its expiry.
type FS struct {
	group  singleflight.Group
	root   *os.Root
	mu     sync.Mutex
	data   map[string]time.Time
	closed bool
}

var _ cache[[]byte] = (*FS)(nil)

// NewFS opens dir (creating it if needed) as an FS store.
func NewFS(dir string) (*FS, error) {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, err
	}
	root, err := os.OpenRoot(dir)
	if err != nil {
		return nil, err
	}
	data := make(map[string]time.Time)
	b, err := root.ReadFile(indexFile)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return nil, errors.Join(err, root.Close())
	}

	if len(b) > 0 {
		err = json.Unmarshal(b, &data)
		if err != nil {
			return nil, errors.Join(err, root.Close())
		}
	}
	if data == nil {
		data = make(map[string]time.Time)
	}

	return &FS{
		root: root,
		data: data,
	}, nil
}

// Close closes the directory. Calls after Close return [ErrClosed]. Close is
// idempotent.
func (f *FS) Close() error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.closed {
		return nil
	}
	f.closed = true
	return f.root.Close()
}

// Load returns the value for key, or [ErrNotExist].
func (f *FS) Load(ctx context.Context, key string) ([]byte, error) {
	f.mu.Lock()
	val, err := f.load(key)
	f.mu.Unlock()
	if err != nil {
		return nil, err
	}

	return val.Val, nil
}

// Store writes key's value. ttl <= 0 means no expiry.
func (f *FS) Store(ctx context.Context, key string, value []byte, ttl time.Duration) error {
	f.mu.Lock()
	err := f.save(key, value, ttl)
	f.mu.Unlock()
	return err
}

// StoreOnce stores value only if key does not exist; otherwise it returns
// [ErrExists].
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

// LoadOrCreate coalesces local fills per key without holding the storage
// mutex during create. Factories may access other keys, but must not recursively
// fill the same key. Followers share the leader's context and wait for it.
func (f *FS) LoadOrCreate(ctx context.Context, key string, create func(context.Context, string) ([]byte, time.Duration, error)) ([]byte, bool, error) {
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

// LoadAndDelete returns and deletes the value for key, or returns
// [ErrNotExist].
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

// CompareAndDelete deletes key only if its value equals old. It returns
// [ErrNotExist] if the key is missing or the value differs.
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

// CompareAndSwap sets key to value only if its current value equals old. It
// returns [ErrNotExist] if the key is missing or the value differs.
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

// Exists reports whether key exists and has not expired.
func (f *FS) Exists(ctx context.Context, key string) (bool, error) {
	_, err := f.Load(ctx, key)
	if errors.Is(err, ErrNotExist) {
		return false, nil
	}
	return err == nil, err
}

// TTL returns the remaining time to live for key: -1 if it has no expiry, -2
// if it does not exist.
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

// Expire sets a new TTL on an existing key; ttl <= 0 deletes it. It returns
// [ErrNotExist] if the key does not exist.
func (f *FS) Expire(ctx context.Context, key string, ttl time.Duration) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	v, err := f.load(key)
	if err != nil {
		return err
	}
	if ttl <= 0 {
		return f.delete(key)
	}
	f.data[v.Key] = time.Now().Add(ttl)
	return f.saveIndex()
}

// Delete removes keys and returns how many value files existed.
func (f *FS) Delete(ctx context.Context, keys ...string) (int64, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	var count int64
	for _, key := range keys {
		err := f.delete(key)
		if errors.Is(err, os.ErrNotExist) {
			continue
		}
		if err != nil {
			return count, err
		}
		count++
	}
	return count, nil
}

// Size returns the number of value files, including expired ones that have
// not been read (and thus removed) yet.
func (f *FS) Size(ctx context.Context) (int, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.closed {
		return 0, ErrClosed
	}
	dir, err := f.root.Open(".")
	if err != nil {
		return 0, err
	}
	defer dir.Close()
	entries, err := dir.ReadDir(-1)
	if err != nil {
		return 0, err
	}
	count := 0
	for _, entry := range entries {
		// Value files are named by a decimal hash; skip the index and temp
		// files, which start with a dot.
		if !entry.IsDir() && !strings.HasPrefix(entry.Name(), ".") {
			count++
		}
	}
	return count, nil
}

func (f *FS) load(key string) (*Event, error) {
	if f.closed {
		return nil, ErrClosed
	}
	original := key
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
		if err := f.delete(original); err != nil {
			return nil, err
		}
		return nil, ErrNotExist
	}

	return it, nil
}

func (f *FS) delete(key string) error {
	if f.closed {
		return ErrClosed
	}
	key = fmt.Sprint(helper.DigestString(key))
	if err := f.root.Remove(key); err != nil {
		return err
	}
	delete(f.data, key)
	return f.saveIndex()
}

func (f *FS) save(key string, value []byte, ttl time.Duration) error {
	key = fmt.Sprint(helper.DigestString(key))
	if f.closed {
		return ErrClosed
	}
	if ttl > 0 {
		f.data[key] = time.Now().Add(ttl)
	} else {
		delete(f.data, key)
	}
	return errors.Join(f.writeFile(key, value), f.saveIndex())
}

func (f *FS) saveIndex() error {
	b, err := json.Marshal(f.data)
	if err != nil {
		return err
	}
	return f.writeFile(indexFile, b)
}

// writeFile replaces name atomically: it writes a temp file, syncs it and
// renames it over name, so a crash never leaves a truncated value or index.
func (f *FS) writeFile(name string, b []byte) (err error) {
	tmp := fmt.Sprintf(".tmp-%x", rand.Uint64())
	file, err := f.root.OpenFile(tmp, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
	if err != nil {
		return err
	}
	defer func() {
		if err != nil {
			_ = f.root.Remove(tmp)
		}
	}()
	_, writeErr := file.Write(b)
	if err := errors.Join(writeErr, file.Sync(), file.Close()); err != nil {
		return err
	}
	return f.root.Rename(tmp, name)
}
