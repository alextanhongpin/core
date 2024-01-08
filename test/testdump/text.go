package testdump

import "github.com/alextanhongpin/core/internal"

func Text(rw readerWriter, received string) error {
	if err := rw.Write([]byte(received)); err != nil {
		return err
	}

	b, err := rw.Read()
	if err != nil {
		return err
	}

	snapshot := string(b)
	return internal.ANSIDiff(snapshot, received)
}
