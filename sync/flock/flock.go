//go:build darwin || linux || freebsd || openbsd || netbsd || dragonfly

// Package flock coordinates file generation using advisory OS file locks.
package flock

import (
	"errors"
	"os"
	"path/filepath"
	"syscall"
)

var ErrLocked = errors.New("flock: locked")

// FS is safe for concurrent use. Its zero value is ready to use.
// All cooperating writers must use the same path and preserve its .lock file.
type FS struct{}

// ReadOrWrite returns an existing file or generates it under a nonblocking
// exclusive lock. Contention returns ErrLocked. Readers see only completed
// files, published through a rename in the same directory. Failed generation
// leaves the destination absent and can be retried. The callback runs under
// the file lock and must not reenter this method for the same path.
// Publication is atomic but does not promise durability across power loss.
func (f *FS) ReadOrWrite(name string, fn func() ([]byte, error)) (result []byte, err error) {
	lock, err := os.OpenFile(name+".lock", os.O_CREATE|os.O_RDWR, 0o644)
	if err != nil {
		return nil, err
	}
	defer func() { err = errors.Join(err, lock.Close()) }()
	if err := syscall.Flock(int(lock.Fd()), syscall.LOCK_EX|syscall.LOCK_NB); err != nil {
		if errors.Is(err, syscall.EWOULDBLOCK) {
			return nil, ErrLocked
		}
		return nil, err
	}
	defer func() { err = errors.Join(err, syscall.Flock(int(lock.Fd()), syscall.LOCK_UN)) }()
	b, err := os.ReadFile(name)
	if err == nil {
		return b, nil
	}
	if !errors.Is(err, os.ErrNotExist) {
		return nil, err
	}
	b, err = fn()
	if err != nil {
		return nil, err
	}
	file, err := os.CreateTemp(filepath.Dir(name), "."+filepath.Base(name)+"-*")
	if err != nil {
		return nil, err
	}
	defer os.Remove(file.Name())
	if err := file.Chmod(0o644); err != nil {
		return nil, errors.Join(err, file.Close())
	}
	_, writeErr := file.Write(b)
	if err := errors.Join(writeErr, file.Close()); err != nil {
		return nil, err
	}
	if err := os.Rename(file.Name(), name); err != nil {
		return nil, err
	}
	return b, nil
}
