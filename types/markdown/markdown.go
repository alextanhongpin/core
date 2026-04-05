package markdown

import (
	"cmp"
	"errors"
	"io"
	"log/slog"
	"maps"
	"os"
	"path/filepath"
	"sync"
	"time"
)

type rw interface {
	io.WriterTo
	io.ReaderFrom
}

func NewLoader(path string, meta map[string]any, ttl time.Duration, rw rw) *Loader {
	return &Loader{
		rw:   rw,
		meta: meta,
		path: path,
		ttl:  ttl,
	}
}

type Loader struct {
	meta map[string]any
	path string
	rw   rw
	ttl  time.Duration
}

func (l *Loader) Sync() func() {
	var wg sync.WaitGroup

	wg.Go(func() {
		t := time.NewTicker(l.ttl)
		defer t.Stop()
		for range t.C {
			err := l.Load()
			if err != nil {
				slog.Default().Error("sync error", "err", err.Error())
			}
		}
	})

	return wg.Wait
}

func (l *Loader) Load() error {
	f, err := os.Open(l.path)
	if errors.Is(err, os.ErrNotExist) {
		return cmp.Or(
			l.Save(),
			l.Load(),
		)
	}
	if err != nil {
		return err
	}
	defer func() {
		_ = f.Close()
	}()

	meta, reader, err := ParseFrontmatter(f)
	if err != nil {
		return err
	}

	if l.shouldUpdate(meta) {
		return cmp.Or(
			l.Save(),
			l.Load(),
		)
	}

	_, err = l.rw.ReadFrom(reader)
	if err != nil {
		return err
	}

	return nil
}

func (l *Loader) Save() error {
	err := os.MkdirAll(filepath.Dir(l.path), 0o755)
	if err != nil {
		return err
	}
	f, err := os.Create(l.path)
	if err != nil {
		return err
	}
	defer func() {
		_ = f.Close()
	}()

	meta := maps.Clone(l.meta)
	meta["created_at"] = time.Now().Format(time.RFC3339)
	err = WriteFrontmatter(meta, f)
	if err != nil {
		return err
	}

	_, err = l.rw.WriteTo(f)
	return err
}

func (l *Loader) shouldUpdate(meta map[string]any) bool {
	s, ok := meta["created_at"].(string)
	if !ok {
		return false
	}

	t, err := time.Parse(time.RFC3339, s)
	if err != nil {
		return false
	}

	return time.Since(t) > l.ttl
}
