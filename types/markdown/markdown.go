package markdown

import (
	"bytes"
	"errors"
	"io"
	"log/slog"
	"maps"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// Content produces a file's body with WriteTo and receives the body read
// back from disk with ReadFrom.
type Content interface {
	io.WriterTo
	io.ReaderFrom
}

// Loader caches generated content in a Markdown file. The file's frontmatter
// holds the configured metadata plus a created_at timestamp; when the file is
// missing or older than the TTL, Loader regenerates it from Content.WriteTo.
// A Loader is safe for concurrent use.
type Loader struct {
	mu      sync.Mutex
	meta    map[string]any
	path    string
	content Content
	ttl     time.Duration
}

// NewLoader returns a Loader for the file at path. meta is copied into the
// frontmatter of every saved file. A nonpositive ttl regenerates the file on
// every Load.
func NewLoader(path string, meta map[string]any, ttl time.Duration, content Content) *Loader {
	return &Loader{
		meta:    maps.Clone(meta),
		path:    path,
		content: content,
		ttl:     ttl,
	}
}

// Sync calls Load every ttl in a background goroutine, logging errors with
// slog. The returned function stops the goroutine and waits for it to exit;
// it is safe to call more than once. Sync panics if ttl is not positive.
func (l *Loader) Sync() (stop func()) {
	ticker := time.NewTicker(l.ttl)
	done := make(chan struct{})
	var wg sync.WaitGroup
	wg.Go(func() {
		defer ticker.Stop()
		for {
			select {
			case <-done:
				return
			case <-ticker.C:
				if err := l.Load(); err != nil {
					slog.Error("markdown: sync failed", "path", l.path, "error", err)
				}
			}
		}
	})

	var once sync.Once
	return func() {
		once.Do(func() { close(done) })
		wg.Wait()
	}
}

// Load regenerates the file if it is missing or expired, then passes the
// file's body to Content.ReadFrom.
func (l *Loader) Load() error {
	l.mu.Lock()
	defer l.mu.Unlock()

	meta, body, err := l.read()
	if errors.Is(err, os.ErrNotExist) || (err == nil && l.expired(meta)) {
		if err := l.save(); err != nil {
			return err
		}
		_, body, err = l.read()
	}
	if err != nil {
		return err
	}
	_, err = l.content.ReadFrom(body)
	return err
}

// Save regenerates the file from Content.WriteTo, creating parent
// directories as needed.
func (l *Loader) Save() error {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.save()
}

func (l *Loader) read() (map[string]any, io.Reader, error) {
	b, err := os.ReadFile(l.path)
	if err != nil {
		return nil, nil, err
	}
	return ParseFrontmatter[map[string]any](bytes.NewReader(b))
}

func (l *Loader) save() error {
	if err := os.MkdirAll(filepath.Dir(l.path), 0o755); err != nil {
		return err
	}
	f, err := os.Create(l.path)
	if err != nil {
		return err
	}
	defer f.Close()

	meta := maps.Clone(l.meta)
	if meta == nil {
		meta = make(map[string]any)
	}
	meta["created_at"] = time.Now().UTC().Format(time.RFC3339Nano)
	if err := WriteFrontmatter(f, meta); err != nil {
		return err
	}
	if _, err := l.content.WriteTo(f); err != nil {
		return err
	}
	return f.Close()
}

// expired reports whether meta's created_at is older than the TTL. A file
// without a parsable created_at never expires.
func (l *Loader) expired(meta map[string]any) bool {
	s, ok := meta["created_at"].(string)
	if !ok {
		return false
	}
	t, err := time.Parse(time.RFC3339Nano, s)
	if err != nil {
		return false
	}
	return time.Now().Sub(t) >= l.ttl
}
