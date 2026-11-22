//go:build darwin || linux || freebsd || openbsd || netbsd || dragonfly

package flock_test

import (
	"errors"
	"github.com/alextanhongpin/core/sync/flock"
	"os"
	"path/filepath"
	"syscall"
	"testing"
)

func TestPublicationAndContention(t *testing.T) {
	var fs flock.FS
	name := filepath.Join(t.TempDir(), "value")
	entered, release := make(chan struct{}), make(chan struct{})
	done := make(chan error, 1)
	go func() {
		_, err := fs.ReadOrWrite(name, func() ([]byte, error) { close(entered); <-release; return []byte("complete"), nil })
		done <- err
	}()
	<-entered
	_, statErr := os.Stat(name)
	_, lockErr := fs.ReadOrWrite(name, func() ([]byte, error) { return nil, errors.New("unexpected callback") })
	close(release)
	if err := <-done; err != nil {
		t.Fatal(err)
	}
	if !errors.Is(statErr, os.ErrNotExist) {
		t.Fatalf("destination published early: %v", statErr)
	}
	if !errors.Is(lockErr, flock.ErrLocked) {
		t.Fatalf("contention: %v", lockErr)
	}
	got, err := fs.ReadOrWrite(name, func() ([]byte, error) { t.Fatal("cached file regenerated"); return nil, nil })
	if err != nil || string(got) != "complete" {
		t.Fatalf("%q %v", got, err)
	}
}
func TestFailedGenerationCanRetry(t *testing.T) {
	var fs flock.FS
	name := filepath.Join(t.TempDir(), "value")
	failure := errors.New("generation failed")
	_, err := fs.ReadOrWrite(name, func() ([]byte, error) { return nil, failure })
	if !errors.Is(err, failure) {
		t.Fatal(err)
	}
	if _, err := os.Stat(name); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("failed file remains: %v", err)
	}
	got, err := fs.ReadOrWrite(name, func() ([]byte, error) { return []byte("retry"), nil })
	if err != nil || string(got) != "retry" {
		t.Fatalf("%q %v", got, err)
	}
	files, err := filepath.Glob(filepath.Join(filepath.Dir(name), ".value-*"))
	if err != nil || len(files) != 0 {
		t.Fatalf("temporary files: %v %v", files, err)
	}
}

func TestExistingFileReadWithoutLock(t *testing.T) {
	var fs flock.FS
	name := filepath.Join(t.TempDir(), "value")
	if err := os.WriteFile(name, []byte("done"), 0o644); err != nil {
		t.Fatal(err)
	}
	got, err := fs.ReadOrWrite(name, func() ([]byte, error) { t.Fatal("callback ran"); return nil, nil })
	if err != nil || string(got) != "done" {
		t.Fatalf("%q %v", got, err)
	}
	if _, err := os.Stat(name + ".lock"); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("lock file created for existing file: %v", err)
	}
	// A lock held by someone else does not block reading a published file.
	held, err := os.OpenFile(name+".lock", os.O_CREATE|os.O_RDWR, 0o644)
	if err != nil {
		t.Fatal(err)
	}
	defer held.Close()
	if err := syscall.Flock(int(held.Fd()), syscall.LOCK_EX|syscall.LOCK_NB); err != nil {
		t.Fatal(err)
	}
	got, err = fs.ReadOrWrite(name, func() ([]byte, error) { t.Fatal("callback ran"); return nil, nil })
	if err != nil || string(got) != "done" {
		t.Fatalf("while locked: %q %v", got, err)
	}
	// An absent file still reports contention.
	_, err = fs.ReadOrWrite(name+"-other", func() ([]byte, error) { return []byte("x"), nil })
	if err != nil {
		t.Fatal(err) // different lock file, so no contention
	}
	if err := os.Remove(name); err != nil {
		t.Fatal(err)
	}
	if _, err := fs.ReadOrWrite(name, func() ([]byte, error) { return nil, nil }); !errors.Is(err, flock.ErrLocked) {
		t.Fatalf("absent file under held lock: %v", err)
	}
}
