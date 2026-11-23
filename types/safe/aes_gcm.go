package safe

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
)

var (
	// ErrInvalidKeySize is returned when the encryption key has an invalid size.
	ErrInvalidKeySize = errors.New("safe: invalid key size, must be 16, 24, or 32 bytes")

	// ErrInvalidCiphertext is returned when the ciphertext is too short or malformed.
	ErrInvalidCiphertext = errors.New("safe: invalid ciphertext")

	// ErrDecryptionFailed is returned when decryption fails due to authentication failure.
	ErrDecryptionFailed = errors.New("safe: decryption failed")
)

// Encrypt encrypts plaintext using AES-GCM with the provided secret key.
// The function automatically generates a random nonce and prepends it to the ciphertext.
// The secret key must be 16, 24, or 32 bytes (AES-128, AES-192, or AES-256).
//
// Example:
//
//	key := make([]byte, 32) // AES-256
//	rand.Read(key)
//	ciphertext, err := safe.Encrypt(key, []byte("secret message"))
func Encrypt(secret, plaintext []byte) ([]byte, error) {
	return EncryptWithAAD(secret, plaintext, nil)
}

// EncryptString encrypts a string and returns the result as a base64-encoded string.
// This is convenient for storing encrypted data in databases or transmitting over text protocols.
//
// Example:
//
//	encrypted, err := safe.EncryptString(key, "secret message")
func EncryptString(secret []byte, plaintext string) (string, error) {
	ciphertext, err := Encrypt(secret, []byte(plaintext))
	if err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(ciphertext), nil
}

// Decrypt decrypts ciphertext that was encrypted with Encrypt.
// The function extracts the nonce from the beginning of the ciphertext.
//
// Example:
//
//	plaintext, err := safe.Decrypt(key, ciphertext)
func Decrypt(secret, ciphertext []byte) ([]byte, error) {
	return DecryptWithAAD(secret, ciphertext, nil)
}

// DecryptString decrypts a base64-encoded ciphertext and returns the result as a string.
//
// Example:
//
//	plaintext, err := safe.DecryptString(key, "base64ciphertext")
func DecryptString(secret []byte, ciphertext string) (string, error) {
	data, err := base64.URLEncoding.DecodeString(ciphertext)
	if err != nil {
		return "", fmt.Errorf("safe: failed to decode base64: %w", err)
	}

	plaintext, err := Decrypt(secret, data)
	if err != nil {
		return "", err
	}

	return string(plaintext), nil
}

// GenerateKey generates a random AES key of the specified size.
// Valid sizes are 16 (AES-128), 24 (AES-192), or 32 (AES-256) bytes.
//
// Example:
//
//	key, err := safe.GenerateKey(32) // AES-256 key
func GenerateKey(size int) ([]byte, error) {
	if size != 16 && size != 24 && size != 32 {
		return nil, ErrInvalidKeySize
	}

	key := make([]byte, size)
	if _, err := rand.Read(key); err != nil {
		return nil, fmt.Errorf("safe: failed to generate key: %w", err)
	}

	return key, nil
}

// EncryptWithAAD encrypts plaintext with Additional Authenticated Data (AAD).
// The AAD is not encrypted but is authenticated along with the plaintext.
//
// Example:
//
//	aad := []byte("user-id:123")
//	ciphertext, err := safe.EncryptWithAAD(key, plaintext, aad)
func EncryptWithAAD(secret, plaintext, aad []byte) ([]byte, error) {
	gcm, err := newGCM(secret)
	if err != nil {
		return nil, err
	}
	nonce := make([]byte, gcm.NonceSize(), gcm.NonceSize()+len(plaintext)+gcm.Overhead())
	if _, err := rand.Read(nonce); err != nil {
		return nil, fmt.Errorf("safe: failed to generate nonce: %w", err)
	}
	return gcm.Seal(nonce, nonce, plaintext, aad), nil
}

// DecryptWithAAD decrypts ciphertext that was encrypted with EncryptWithAAD.
// The same AAD must be provided for successful decryption.
//
// Example:
//
//	aad := []byte("user-id:123")
//	plaintext, err := safe.DecryptWithAAD(key, ciphertext, aad)
func DecryptWithAAD(secret, ciphertext, aad []byte) ([]byte, error) {
	gcm, err := newGCM(secret)
	if err != nil {
		return nil, err
	}
	if len(ciphertext) < gcm.NonceSize()+gcm.Overhead() {
		return nil, ErrInvalidCiphertext
	}
	nonce, sealed := ciphertext[:gcm.NonceSize()], ciphertext[gcm.NonceSize():]
	plaintext, err := gcm.Open(nil, nonce, sealed, aad)
	if err != nil {
		return nil, ErrDecryptionFailed
	}
	return plaintext, nil
}

func newGCM(secret []byte) (cipher.AEAD, error) {
	switch len(secret) {
	case 16, 24, 32:
	default:
		return nil, ErrInvalidKeySize
	}
	block, err := aes.NewCipher(secret)
	if err != nil {
		return nil, fmt.Errorf("safe: failed to create cipher: %w", err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("safe: failed to create GCM: %w", err)
	}
	return gcm, nil
}
