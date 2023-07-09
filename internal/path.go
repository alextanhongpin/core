package internal

import "path/filepath"

type Path struct {
	TestDir  string
	FilePath string
	FileName string
	FileExt  string
}

func (o *Path) String() string {
	if len(o.FileName) == 0 {
		return filepath.Join(
			o.TestDir,
			o.FilePath+o.FileExt,
		)
	}

	// Get the file extension.
	fileName := string(o.FileName)
	fileExt := filepath.Ext(fileName)
	if fileExt != o.FileExt {
		fileName = fileName + o.FileExt
	}

	return filepath.Join(
		o.TestDir,
		o.FilePath,
		fileName,
	)
}
