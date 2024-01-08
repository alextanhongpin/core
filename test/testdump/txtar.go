package testdump

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/alextanhongpin/core/internal"
	"golang.org/x/tools/txtar"
)

var update = flag.Bool("update", false, "update the dump file")

type TxTar struct {
	Name string
	File string
}

func NewTxTar(name, file string) *TxTar {
	if filepath.Ext(name) != ".txtar" {
		name += ".txtar"
	}

	return &TxTar{
		Name: name,
		File: file,
	}
}

func (rw *TxTar) Read() ([]byte, error) {
	arc, err := txtar.ParseFile(rw.Name)
	if err != nil {
		return nil, err
	}

	if len(arc.Files) != 1 {
		return nil, fmt.Errorf("testdump: file not found: %s", rw.Name)
	}

	file := arc.Files[0]
	if want, got := rw.File, file.Name; want != got {
		return nil, fmt.Errorf("testdump: file does not match: want %s, got %s", want, got)
	}

	return file.Data, nil
}

func (rw *TxTar) Write(data []byte) error {
	arc, err := txtar.ParseFile(rw.Name)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}

	if len(arc.Files) == 1 {
		if *update {
			arc.Files[0] = txtar.File{
				Name: rw.File,
				Data: data,
			}

			return internal.WriteFile(rw.Name, txtar.Format(arc), *update)
		}

		if arc.Files[0].Name != rw.File {
			return fmt.Errorf("testdump: file content has changed, run with -update flag to update: %s", rw.Name)
		}

		// Don't overwrite.
		return nil
	}

	// File does not exist yet, create new.
	arc.Files = append(arc.Files, txtar.File{
		Name: rw.File,
		Data: data,
	})

	return internal.WriteFile(rw.Name, txtar.Format(arc), false)
}
