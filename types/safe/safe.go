// Package safe wraps crypto/rand, AES-GCM, and HMAC-SHA256 for common tasks:
// generating secrets and keys, encrypting with a random nonce prepended to
// the ciphertext, and signing and verifying data in constant time.
//
// String helpers use URL-safe base64 with padding (base64.URLEncoding).
package safe

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
)

// Secret generates a cryptographically secure random byte slice of the specified length.
// This is suitable for generating API keys, session tokens, and other secrets.
//
// Example:
//
//	secret, err := safe.Secret(32) // 256-bit secret
func Secret(n int) ([]byte, error) {
	if n <= 0 {
		return nil, fmt.Errorf("safe: invalid secret length %d", n)
	}

	secret := make([]byte, n)
	_, err := rand.Read(secret)
	return secret, err
}

// SecretString generates a cryptographically secure random string encoded as base64.
// This is convenient for generating human-readable secrets.
//
// Example:
//
//	token, err := safe.SecretString(32) // Base64-encoded 32-byte secret
func SecretString(n int) (string, error) {
	secret, err := Secret(n)
	if err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(secret), nil
}

// SecretHex generates a cryptographically secure random string encoded as hexadecimal.
// This is useful for generating hex-encoded keys and tokens.
//
// Example:
//
//	key, err := safe.SecretHex(16) // 32-character hex string
func SecretHex(n int) (string, error) {
	secret, err := Secret(n)
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(secret), nil
}

// Signature creates a HMAC-SHA256 signature of the given data using the provided secret.
// This can be used to ensure data integrity and authenticity.
//
// Example:
//
//	signature := safe.Signature(secret, []byte("important data"))
func Signature(secret, data []byte) []byte {
	h := hmac.New(sha256.New, secret)
	h.Write(data)
	return h.Sum(nil)
}

// SignatureString creates a HMAC-SHA256 signature and returns it as a base64-encoded string.
// This is convenient for use in HTTP headers, URLs, and other text-based protocols.
//
// Example:
//
//	sig := safe.SignatureString(secret, []byte("message"))
func SignatureString(secret, data []byte) string {
	signature := Signature(secret, data)
	return base64.URLEncoding.EncodeToString(signature)
}

// SignatureHex creates a HMAC-SHA256 signature and returns it as a hexadecimal string.
// This is useful when you need hex-encoded signatures.
//
// Example:
//
//	sig := safe.SignatureHex(secret, []byte("message"))
func SignatureHex(secret, data []byte) string {
	signature := Signature(secret, data)
	return hex.EncodeToString(signature)
}

// VerifySignature verifies a HMAC-SHA256 signature against the given data and secret.
// This function uses constant-time comparison to prevent timing attacks.
//
// Example:
//
//	valid := safe.VerifySignature(secret, data, signature)
func VerifySignature(secret, data, signature []byte) bool {
	return hmac.Equal(signature, Signature(secret, data))
}

// VerifySignatureString verifies a base64-encoded HMAC-SHA256 signature.
//
// Example:
//
//	valid := safe.VerifySignatureString(secret, data, "base64signature")
func VerifySignatureString(secret, data []byte, signature string) bool {
	sig, err := base64.URLEncoding.DecodeString(signature)
	if err != nil {
		return false
	}
	return VerifySignature(secret, data, sig)
}

// VerifySignatureHex verifies a hex-encoded HMAC-SHA256 signature.
//
// Example:
//
//	valid := safe.VerifySignatureHex(secret, data, "hexsignature")
func VerifySignatureHex(secret, data []byte, signature string) bool {
	sig, err := hex.DecodeString(signature)
	if err != nil {
		return false
	}
	return VerifySignature(secret, data, sig)
}
