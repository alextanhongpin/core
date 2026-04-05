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

// rw defines the interface required for reading and writing the loader state.
// It must support reading content from an io.ReaderFrom context and writing/reading
// metadata via WriterTo/ReaderFrom mechanisms, assuming it wraps the final file I/O.
type rw interface {
	io.WriterTo
	io.ReaderFrom
}

// Loader manages the lifecycle, reading, writing, and syncing of markdown loader metadata.
type Loader struct {
	meta map[string]any // Metadata stored with the file (e.g., creation timestamps)
	path string         // Absolute path to the markdown file
	rw   rw             // The underlying IO mechanism for reading/writing the file content
	ttl  time.Duration  // Time-to-live for metadata refresh check
}

// NewLoader creates a new Loader instance.
func NewLoader(path string, meta map[string]any, ttl time.Duration, rw rw) *Loader {
	return &Loader{
		meta: meta,
		path: path,
		rw:   rw,
		ttl:  ttl,
	}
}

// Sync starts a background routine to periodically check if the metadata needs updating
// (i.e., if the underlying content has changed since the metadata was written).
// It returns a function that, when called, stops the background syncing process.
func (l *Loader) Sync() func() {
	var wg sync.WaitGroup

	// We use a context/cancellation pattern here in a real app, but for this scope,
	// we'll rely on the pattern of returning a cleanup function.
	ticker := time.NewTicker(l.ttl)

	// Start the background monitoring routine
	wg.Go(func() {
		defer ticker.Stop()

		for range ticker.C {
			if err := l.Load(); err != nil {
				// Log synchronization errors but do not crash the background routine
				slog.Default().Error("sync error while loading markdown loader", "error", err.Error())
			}
		}
	})

	// Return the cleanup function
	return func() {
		ticker.Stop()
		wg.Done() // Decrement counter when cleanup function is called
	}
}

// Load attempts to load the metadata and content from the file system.
// Returns an error if loading fails.
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

	// 1. Parse Frontmatter
	meta, reader, err := ParseFrontmatter(f)
	if err != nil {
		return err
	}

	// 2. Check if metadata indicates an update is needed
	if l.shouldUpdate(meta) {
		return cmp.Or(
			// If metadata is stale, save first to update it, then reload to get the new state.
			l.Save(),
			// After saving, we must reload to ensure we read the newly written metadata
			l.Load(),
		)
	}

	// 3. Read the main content using the buffered reader
	_, err = l.rw.ReadFrom(reader)
	return err
}

// Save writes the current metadata state and the content provided by rw.WriterTo to the file.
// It updates the 'created_at' field with the current timestamp.
func (l *Loader) Save() error {
	// Ensure the directory structure exists
	if err := os.MkdirAll(filepath.Dir(l.path), 0o755); err != nil {
		return err
	}

	// Create the file handle
	f, err := os.Create(l.path)
	if err != nil {
		return err
	}
	defer func() {
		_ = f.Close()
	}()

	// Update metadata
	meta := maps.Clone(l.meta)
	meta["created_at"] = time.Now().Format(time.RFC3339)

	// 1. Write the frontmatter header
	if err := WriteFrontmatter(f, meta); err != nil {
		return err
	}

	// 2. Write the main content
	_, err = l.rw.WriteTo(f)
	return err
}

// shouldUpdate checks if the metadata provided is older than the configured TTL.
// If the file doesn't contain a valid 'created_at' timestamp, it defaults to false.
func (l *Loader) shouldUpdate(meta map[string]any) bool {
	// Check if 'created_at' exists and is a string
	s, ok := meta["created_at"].(string)
	if !ok {
		return false
	}

	// Parse the timestamp
	t, err := time.Parse(time.RFC3339, s)
	if err != nil {
		return false // Cannot parse, assume no change or handle error elsewhere
	}

	// Check if the time elapsed since creation exceeds the TTL
	return time.Since(t) > l.ttl
}
