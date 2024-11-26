package safe_test

import (
	"bytes"
	"testing"

	"github.com/alextanhongpin/core/types/safe"
	"github.com/stretchr/testify/assert"
)

func TestEncryptDecrypt(t *testing.T) {
	var (
		secret = []byte("supersecret12345")
		data   = []byte("Hello, World!")
	)

	is := assert.New(t)
	ciphertext, err := safe.Encrypt(secret, data)
	is.Nil(err)

	plaintext, err := safe.Decrypt(secret, ciphertext)
	is.Nil(err)
	is.True(bytes.Equal(data, plaintext))
}
