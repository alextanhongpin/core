package internal

import (
	"errors"
	"os"
	"path/filepath"
)

// WriteFile writes a file to the filesystem.
func WriteFile(name string, body []byte, overwrite bool) error {
	f, err := os.OpenFile(name, os.O_RDONLY, 0644)
	if errors.Is(err, os.ErrNotExist) || overwrite {
		dir := filepath.Dir(name)

		if err := os.MkdirAll(dir, 0700); err != nil && !os.IsExist(err) {
			return err
		} // Create your file

		return os.WriteFile(name, body, 0644)
	}
	if err != nil {
		return err
	}

	defer f.Close()

	return nil
}
