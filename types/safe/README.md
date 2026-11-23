# safe

Small wrappers over `crypto/rand`, AES-GCM, and HMAC-SHA256: generate secrets,
encrypt with a fresh random nonce, and sign and verify data in constant time.

**Use it for:** encrypting values at rest with a key you manage, signing
webhook payloads or URLs, and generating API keys and session tokens.

**Don't use it for:** password hashing (use bcrypt, scrypt, or argon2), key
derivation from passwords, key rotation or storage, or public-key signatures.

## Installation

```sh
go get github.com/alextanhongpin/core/types
```

`safe` is a package in the `types` module; import it as
`github.com/alextanhongpin/core/types/safe`. Check the module's `go.mod` for the
required Go version.

## Quick start

```go
package main

import (
	"fmt"

	"github.com/alextanhongpin/core/types/safe"
)

func main() {
	key, err := safe.GenerateKey(32) // AES-256
	if err != nil {
		panic(err)
	}

	ct, err := safe.EncryptString(key, "card=4242")
	if err != nil {
		panic(err)
	}
	pt, err := safe.DecryptString(key, ct)
	fmt.Println(pt, err)

	sig := safe.SignatureHex(key, []byte("order:42"))
	fmt.Println(safe.VerifySignatureHex(key, []byte("order:42"), sig))
	fmt.Println(safe.VerifySignatureHex(key, []byte("order:43"), sig))

	token, _ := safe.SecretString(32)
	fmt.Println(len(token))
}
```

It prints:

```
card=4242 <nil>
true
false
44
```

## API

```go
var ErrInvalidKeySize, ErrInvalidCiphertext, ErrDecryptionFailed error

func GenerateKey(size int) ([]byte, error) // 16, 24, or 32 bytes

func Encrypt(key, plaintext []byte) ([]byte, error)
func Decrypt(key, ciphertext []byte) ([]byte, error)
func EncryptWithAAD(key, plaintext, aad []byte) ([]byte, error)
func DecryptWithAAD(key, ciphertext, aad []byte) ([]byte, error)
func EncryptString(key []byte, plaintext string) (string, error)  // base64
func DecryptString(key []byte, ciphertext string) (string, error)

func Secret(n int) ([]byte, error)
func SecretString(n int) (string, error) // base64 of n random bytes
func SecretHex(n int) (string, error)    // hex of n random bytes

func Signature(key, data []byte) []byte
func SignatureString(key, data []byte) string // base64
func SignatureHex(key, data []byte) string
func VerifySignature(key, data, sig []byte) bool
func VerifySignatureString(key, data []byte, sig string) bool
func VerifySignatureHex(key, data []byte, sig string) bool
```

For constant-time comparison of other values, use `crypto/subtle` or
`hmac.Equal` directly.

## Behavior and guarantees

- Encryption is AES-GCM. The key length selects AES-128, -192, or -256; any
  other length returns `ErrInvalidKeySize`.
- Each call draws a new 12-byte random nonce and returns
  `nonce || ciphertext || 16-byte tag`, so the output is 28 bytes longer than the
  plaintext and encrypting the same plaintext twice gives different results.
  Random nonces are safe for up to about 2^32 messages per key; rotate keys
  before that.
- Input shorter than nonce plus tag returns `ErrInvalidCiphertext`. A wrong key,
  modified ciphertext, or mismatched AAD returns `ErrDecryptionFailed`, without
  saying which.
- AAD is authenticated but not encrypted, and must be passed again to decrypt.
  Data encrypted with AAD cannot be decrypted by `Decrypt`, and vice versa.
- String helpers use URL-safe base64 with padding (`base64.URLEncoding`).
- Signatures are HMAC-SHA256 (32 bytes). Verification uses `hmac.Equal` and
  returns false for malformed encodings.
- `Secret` returns an error for `n <= 0`. Note that `SecretString(n)` and
  `SecretHex(n)` take a byte count, so the string is longer than `n`.
- All functions are safe for concurrent use.

## Testing

```sh
cd types
go vet ./safe
go test -race -count=1 ./safe
```
