package internal

import (
	"path/filepath"
	"strings"
)

type Path struct {
	Dir      string
	FilePath string
	FileName string
	FileExt  string
}

func (o *Path) String() string {
	if len(o.FileName) == 0 {
		filePath := strings.TrimSuffix(o.FilePath, "/")
		return filepath.Join(
			o.Dir,
			filePath+o.FileExt,
		)
	}

	// Get the file extension.
	fileName := string(o.FileName)
	fileExt := filepath.Ext(fileName)
	if fileExt != o.FileExt {
		fileName = fileName + o.FileExt
	}

	return filepath.Join(
		o.Dir,
		o.FilePath,
		fileName,
	)
}
