package httputil

import (
	"bytes"
	"io"
)

type ReusableReader struct {
	io.Reader
	b  *bytes.Buffer
	bk *bytes.Buffer
}

func NewReusableReader(r io.Reader) *ReusableReader {
	b := new(bytes.Buffer)
	bk := new(bytes.Buffer)
	_, err := bk.ReadFrom(r)
	if err != nil {
		panic(err)
	}
	return &ReusableReader{
		Reader: io.TeeReader(bk, b),
		b:      b,
		bk:     bk,
	}
}

func (r *ReusableReader) Read(b []byte) (int, error) {
	n, err := r.Reader.Read(b)
	if err == io.EOF {
		if err := r.reset(); err != nil {
			return 0, err
		}
	}

	return n, err
}

func (r *ReusableReader) Close() error {
	return nil
}

func (r *ReusableReader) reset() error {
	_, err := io.Copy(r.bk, r.b)
	return err
}
