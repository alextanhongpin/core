package testutil

import (
	"io"
	"net/http"
	"os"
)

func LoadURLAndCompare(uri string, fileName string, cmp comparer) error {
	resp, err := http.Get(uri)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	received, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	err = writeToNewFile(fileName, received)
	if err != nil {
		return err
	}

	snapshot, err := os.ReadFile(fileName)
	if err != nil {
		return err
	}

	return cmp.Compare(snapshot, received)
}
