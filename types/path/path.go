package path

import (
	"cmp"
	"errors"
	"os"
	"path/filepath"
)

type Path string

func (p Path) Base() string {
	return filepath.Base(p.String())
}

func (p Path) Dir() string {
	return filepath.Dir(p.String())
}

func (p Path) Exist() bool {
	_, err := os.Stat(p.String())
	if errors.Is(err, os.ErrNotExist) {
		return false
	}
	return err == nil
}

func (p Path) Ext() string {
	return filepath.Ext(p.String())
}

func (p Path) Join(paths ...string) Path {
	return Path(filepath.Join(p.String(), filepath.Join(paths...)))
}

func (p Path) OpenFile(flag int) (*os.File, error) {
	if err := os.MkdirAll(p.Dir(), 0o755); err != nil {
		return nil, err
	}

	return os.OpenFile(p.String(), flag, 0o644)
}

func (p Path) ReadFile() ([]byte, error) {
	b, err := os.ReadFile(p.String())
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	return b, err
}

func (p Path) String() string {
	return string(p)
}

func (p Path) WriteFile(b []byte) error {
	return cmp.Or(
		os.MkdirAll(p.Dir(), 0o755),
		os.WriteFile(p.String(), b, 0o644),
	)
}
