package cache

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
)

// FS is a file-system backed cache.
type FS struct {
	dir string
}

// NewFS creates a new FS cache storing entries in the given directory.
func NewFS(dir string) *FS {
	return &FS{dir: dir}
}

// Get retrieves a value from the cache by key.
func (f *FS) Get(key string) ([]byte, bool) {
	hash := hex.EncodeToString(sha256.Sum256([]byte(key)))
	path := filepath.Join(f.dir, hash+".dat")
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, false
	}
	return data, true
}

// Set stores a value in the cache by key.
func (f *FS) Set(key string, value []byte) error {
	hash := hex.EncodeToString(sha256.Sum256([]byte(key)))
	path := filepath.Join(f.dir, hash+".dat")
	if err := os.MkdirAll(f.dir, 0755); err != nil {
		return err
	}
	return os.WriteFile(path, value, 0644)
}

// Delete removes a value from the cache by key.
func (f *FS) Delete(key string) error {
	hash := hex.EncodeToString(sha256.Sum256([]byte(key)))
	path := filepath.Join(f.dir, hash+".dat")
	return os.Remove(path)
}
