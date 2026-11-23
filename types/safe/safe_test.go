package safe_test

import (
	"bytes"
	"errors"
	"testing"

	"github.com/alextanhongpin/core/types/safe"
	"github.com/go-openapi/testify/assert"
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

func TestDecryptRejectsTruncatedCiphertext(t *testing.T) {
	key, err := safe.GenerateKey(32)
	if err != nil {
		t.Fatal(err)
	}
	ct, err := safe.Encrypt(key, []byte("hello"))
	if err != nil {
		t.Fatal(err)
	}
	// Nonce present but tag cut short.
	if _, err := safe.Decrypt(key, ct[:12+4]); !errors.Is(err, safe.ErrInvalidCiphertext) {
		t.Fatalf("want ErrInvalidCiphertext, got %v", err)
	}
	ct[len(ct)-1] ^= 1
	if _, err := safe.Decrypt(key, ct); !errors.Is(err, safe.ErrDecryptionFailed) {
		t.Fatalf("want ErrDecryptionFailed, got %v", err)
	}
}

func TestAADMustMatch(t *testing.T) {
	key, _ := safe.GenerateKey(16)
	ct, err := safe.EncryptWithAAD(key, []byte("hello"), []byte("user:1"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := safe.DecryptWithAAD(key, ct, []byte("user:2")); !errors.Is(err, safe.ErrDecryptionFailed) {
		t.Fatalf("want ErrDecryptionFailed, got %v", err)
	}
	if _, err := safe.Decrypt(key, ct); err == nil {
		t.Fatal("Decrypt without AAD must fail")
	}
}

func TestInvalidKeySize(t *testing.T) {
	if _, err := safe.Encrypt(make([]byte, 10), nil); !errors.Is(err, safe.ErrInvalidKeySize) {
		t.Fatalf("want ErrInvalidKeySize, got %v", err)
	}
}
