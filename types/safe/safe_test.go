package safe_test

import (
	"bytes"
	"testing"

	"github.com/alextanhongpin/core/types/safe"
	"github.com/stretchr/testify/assert"
)

func TestSignature(t *testing.T) {
	var (
		secret = []byte("supersecret")
		data   = []byte("Hello, World!")
	)

	signature := safe.Signature(secret, data)
	is := assert.New(t)
	is.NotEmpty(signature)

	signature2 := safe.Signature(secret, data)
	is.True(bytes.Equal(signature, signature2))
}
