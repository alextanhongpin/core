//go:build darwin || linux || freebsd || openbsd || netbsd || dragonfly

package flock_test

import (
	"errors"
	"github.com/alextanhongpin/core/sync/flock"
	"os"
	"path/filepath"
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
