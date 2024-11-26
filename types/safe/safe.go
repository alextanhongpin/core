package safe

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
)

func Secret(n int) ([]byte, error) {
	secret := make([]byte, n)
	_, err := rand.Read(secret)
	return secret, err
}

// Signature can be used to generate a HMAC-SHA256 hash of a given data to
// ensure its integrity.
func Signature(secret, data []byte) []byte {
	hmac := hmac.New(sha256.New, secret)
	hmac.Write([]byte(data))
	return hmac.Sum(nil)
}
