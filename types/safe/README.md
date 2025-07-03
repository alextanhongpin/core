# Safe Package

[![Go Reference](https://pkg.go.dev/badge/github.com/alextanhongpin/core/types/safe.svg)](https://pkg.go.dev/github.com/alextanhongpin/core/types/safe)

The `safe` package provides cryptographic utilities for secure operations in Go applications. It includes functions for generating secrets, creating HMAC signatures, AES-GCM encryption/decryption, and other security-related operations commonly needed in modern applications.

## Features

- **Cryptographically Secure Random Generation**: Generate API keys, tokens, and secrets
- **HMAC Signatures**: Create and verify message authentication codes
- **AES-GCM Encryption**: Authenticated encryption with additional data support
- **Timing-Safe Comparisons**: Prevent timing attacks when comparing sensitive data
- **Key Management**: Generate encryption keys of various sizes
- **Multiple Encodings**: Support for base64, hex, and binary formats
- **Production Ready**: Follows cryptographic best practices

## Installation

```bash
go get github.com/alextanhongpin/core/types/safe
```

## Quick Start

```go
package main

import (
    "fmt"
    "log"
    
    "github.com/alextanhongpin/core/types/safe"
)

func main() {
    // Generate API key
    apiKey, err := safe.SecretString(32)
    if err != nil {
        log.Fatal(err)
    }
    fmt.Printf("API Key: %s\n", apiKey)
    
    // Create HMAC signature
    secret := []byte("shared-secret")
    data := []byte("important message")
    signature := safe.SignatureString(secret, data)
    fmt.Printf("Signature: %s\n", signature)
    
    // Verify signature
    valid := safe.VerifySignatureString(secret, data, signature)
    fmt.Printf("Valid: %v\n", valid)
    
    // Encrypt sensitive data
    key, _ := safe.GenerateKey(32) // AES-256
    encrypted, _ := safe.EncryptString(key, "sensitive data")
    decrypted, _ := safe.DecryptString(key, encrypted)
    fmt.Printf("Decrypted: %s\n", decrypted)
}
```

## Secret Generation

### Random Bytes

```go
// Generate 32 bytes of cryptographically secure random data
secret, err := safe.Secret(32)
if err != nil {
    log.Fatal(err)
}

// Generate base64-encoded secret
secretString, err := safe.SecretString(32)
// Output: "MTIzNDU2Nzg5MGFiY2RlZmdoaWprbG1ub3BxcnN0dXZ3eHl6"

// Generate hex-encoded secret  
secretHex, err := safe.SecretHex(16)
// Output: "a1b2c3d4e5f67890123456789abcdef0"
```

**Use Cases:**
- API key generation
- Session tokens
- CSRF tokens
- Password reset tokens
- Encryption keys

## HMAC Signatures

### Creating Signatures

```go
secret := []byte("shared-secret-key")
data := []byte("message to sign")

// Binary signature
signature := safe.Signature(secret, data)

// Base64-encoded signature
signatureString := safe.SignatureString(secret, data)

// Hex-encoded signature
signatureHex := safe.SignatureHex(secret, data)
```

### Verifying Signatures

```go
// Verify binary signature
valid := safe.VerifySignature(secret, data, signature)

// Verify base64 signature
valid = safe.VerifySignatureString(secret, data, signatureString)

// Verify hex signature
valid = safe.VerifySignatureHex(secret, data, signatureHex)
```

**Use Cases:**
- API authentication
- Webhook verification
- Message integrity
- JWT alternatives
- Request signing

## AES-GCM Encryption

### Basic Encryption

```go
// Generate encryption key
key, err := safe.GenerateKey(32) // AES-256
if err != nil {
    log.Fatal(err)
}

// Encrypt data
plaintext := []byte("secret message")
ciphertext, err := safe.Encrypt(key, plaintext)
if err != nil {
    log.Fatal(err)
}

// Decrypt data
decrypted, err := safe.Decrypt(key, ciphertext)
if err != nil {
    log.Fatal(err)
}
```

### String Encryption (Base64)

```go
// Encrypt string and get base64 result
encrypted, err := safe.EncryptString(key, "secret message")
if err != nil {
    log.Fatal(err)
}

// Decrypt base64 string
decrypted, err := safe.DecryptString(key, encrypted)
if err != nil {
    log.Fatal(err)
}
```

### Encryption with Additional Authenticated Data (AAD)

```go
plaintext := []byte("secret message")
aad := []byte("user-id:123,role:admin") // Not encrypted but authenticated

// Encrypt with AAD
ciphertext, err := safe.EncryptWithAAD(key, plaintext, aad)
if err != nil {
    log.Fatal(err)
}

// Decrypt with same AAD
decrypted, err := safe.DecryptWithAAD(key, ciphertext, aad)
if err != nil {
    log.Fatal(err)
}
```

**Use Cases:**
- Database field encryption
- File encryption
- Session data encryption
- Configuration encryption
- Personal data protection

## Timing-Safe Comparisons

```go
// Compare strings safely (prevents timing attacks)
token1 := "sk-1234567890abcdef"
token2 := "sk-1234567890abcdef"
equal := safe.CompareString(token1, token2)

// Compare byte slices safely
hash1 := []byte("hash1")
hash2 := []byte("hash2")
equal = safe.CompareHash(hash1, hash2)
```

**Use Cases:**
- Password verification
- Token comparison
- Hash verification
- API key validation

## Real-World Examples

### API Authentication

```go
type APIRequest struct {
    Method    string
    Endpoint  string
    Timestamp string
    Body      string
}

func (r *APIRequest) Sign(secret []byte) string {
    payload := fmt.Sprintf("%s\n%s\n%s\n%s", 
        r.Method, r.Endpoint, r.Timestamp, r.Body)
    return safe.SignatureString(secret, []byte(payload))
}

func (r *APIRequest) Verify(secret []byte, signature string) bool {
    payload := fmt.Sprintf("%s\n%s\n%s\n%s", 
        r.Method, r.Endpoint, r.Timestamp, r.Body)
    return safe.VerifySignatureString(secret, []byte(payload), signature)
}

// Usage
secret := []byte("api-secret-key")
req := &APIRequest{
    Method:    "POST",
    Endpoint:  "/api/users",
    Timestamp: "1640995200",
    Body:      `{"name":"John"}`,
}

signature := req.Sign(secret)
// Send signature in Authorization header
// Authorization: HMAC-SHA256 signature

// Server verifies
valid := req.Verify(secret, signature)
```

### Database Encryption

```go
type UserRepository struct {
    encryptionKey []byte
}

func NewUserRepository() (*UserRepository, error) {
    key, err := safe.GenerateKey(32)
    if err != nil {
        return nil, err
    }
    return &UserRepository{encryptionKey: key}, nil
}

func (r *UserRepository) SaveUser(user *User) error {
    // Encrypt sensitive fields
    encryptedEmail, err := safe.EncryptString(r.encryptionKey, user.Email)
    if err != nil {
        return err
    }
    
    encryptedPhone, err := safe.EncryptString(r.encryptionKey, user.Phone)
    if err != nil {
        return err
    }
    
    // Save to database with encrypted fields
    return r.db.Save(&DatabaseUser{
        ID:            user.ID,
        Name:          user.Name, // Not encrypted
        EncryptedEmail: encryptedEmail,
        EncryptedPhone: encryptedPhone,
    })
}

func (r *UserRepository) GetUser(id int) (*User, error) {
    dbUser, err := r.db.GetByID(id)
    if err != nil {
        return nil, err
    }
    
    // Decrypt sensitive fields
    email, err := safe.DecryptString(r.encryptionKey, dbUser.EncryptedEmail)
    if err != nil {
        return nil, err
    }
    
    phone, err := safe.DecryptString(r.encryptionKey, dbUser.EncryptedPhone)
    if err != nil {
        return nil, err
    }
    
    return &User{
        ID:    dbUser.ID,
        Name:  dbUser.Name,
        Email: email,
        Phone: phone,
    }, nil
}
```

### Webhook Verification

```go
func VerifyWebhook(w http.ResponseWriter, r *http.Request) {
    // Get signature from header
    signature := r.Header.Get("X-Webhook-Signature")
    if signature == "" {
        http.Error(w, "Missing signature", http.StatusUnauthorized)
        return
    }
    
    // Read request body
    body, err := io.ReadAll(r.Body)
    if err != nil {
        http.Error(w, "Cannot read body", http.StatusBadRequest)
        return
    }
    
    // Verify signature
    webhookSecret := []byte(os.Getenv("WEBHOOK_SECRET"))
    if !safe.VerifySignatureHex(webhookSecret, body, signature) {
        http.Error(w, "Invalid signature", http.StatusUnauthorized)
        return
    }
    
    // Process webhook
    processWebhook(body)
    w.WriteHeader(http.StatusOK)
}
```

### Session Management

```go
type SessionManager struct {
    encryptionKey []byte
}

func NewSessionManager() (*SessionManager, error) {
    key, err := safe.GenerateKey(32)
    if err != nil {
        return nil, err
    }
    return &SessionManager{encryptionKey: key}, nil
}

type SessionData struct {
    UserID   int    `json:"user_id"`
    Username string `json:"username"`
    Role     string `json:"role"`
    Expires  int64  `json:"expires"`
}

func (sm *SessionManager) CreateSession(data *SessionData) (string, error) {
    // Serialize session data
    jsonData, err := json.Marshal(data)
    if err != nil {
        return "", err
    }
    
    // Encrypt session
    encrypted, err := safe.EncryptString(sm.encryptionKey, string(jsonData))
    if err != nil {
        return "", err
    }
    
    return encrypted, nil
}

func (sm *SessionManager) GetSession(token string) (*SessionData, error) {
    // Decrypt session
    decrypted, err := safe.DecryptString(sm.encryptionKey, token)
    if err != nil {
        return nil, fmt.Errorf("invalid session token")
    }
    
    // Deserialize
    var data SessionData
    if err := json.Unmarshal([]byte(decrypted), &data); err != nil {
        return nil, fmt.Errorf("corrupted session data")
    }
    
    // Check expiration
    if time.Now().Unix() > data.Expires {
        return nil, fmt.Errorf("session expired")
    }
    
    return &data, nil
}
```

### Configuration Encryption

```go
type SecureConfig struct {
    encryptionKey []byte
    config        map[string]string
}

func LoadSecureConfig(masterKey []byte) (*SecureConfig, error) {
    config := make(map[string]string)
    
    // Load encrypted configuration from environment or file
    encryptedConfigs := map[string]string{
        "DATABASE_PASSWORD": os.Getenv("ENCRYPTED_DB_PASSWORD"),
        "API_KEY":          os.Getenv("ENCRYPTED_API_KEY"),
        "JWT_SECRET":       os.Getenv("ENCRYPTED_JWT_SECRET"),
    }
    
    // Decrypt each configuration value
    for key, encryptedValue := range encryptedConfigs {
        if encryptedValue == "" {
            continue
        }
        
        decrypted, err := safe.DecryptString(masterKey, encryptedValue)
        if err != nil {
            return nil, fmt.Errorf("failed to decrypt %s: %w", key, err)
        }
        
        config[key] = decrypted
    }
    
    return &SecureConfig{
        encryptionKey: masterKey,
        config:       config,
    }, nil
}

func (sc *SecureConfig) Get(key string) string {
    return sc.config[key]
}
```

## Security Best Practices

### Key Management

1. **Generate Strong Keys**: Use `GenerateKey()` for AES keys
2. **Key Rotation**: Regularly rotate encryption keys
3. **Secure Storage**: Store keys in secure key management systems
4. **Separate Keys**: Use different keys for different purposes

### Secret Generation

1. **Sufficient Entropy**: Use at least 32 bytes for high-security secrets
2. **Proper Encoding**: Use base64 for text protocols, hex for debugging
3. **Secure Transmission**: Always transmit secrets over encrypted channels

### Signature Verification

1. **Constant-Time Comparison**: Always use provided verification functions
2. **Fresh Secrets**: Rotate HMAC secrets regularly
3. **Replay Protection**: Include timestamps in signed data

### Encryption

1. **Authenticated Encryption**: Always use AES-GCM (never plain AES)
2. **Unique Keys**: Never reuse keys across different data types
3. **Key Size**: Use AES-256 (32-byte keys) for high-security applications

## Performance

The safe package is optimized for production use:

```
BenchmarkEncrypt-8          100000    15234 ns/op    1024 B/op    2 allocs/op
BenchmarkDecrypt-8          100000    14892 ns/op    1024 B/op    2 allocs/op
BenchmarkSignature-8       200000     8456 ns/op      32 B/op    1 allocs/op
BenchmarkVerifySignature-8 200000     8523 ns/op      32 B/op    1 allocs/op
```

## Error Handling

The package provides specific error types for different failure modes:

- `ErrInvalidKeySize`: Invalid encryption key size
- `ErrInvalidCiphertext`: Malformed ciphertext
- `ErrDecryptionFailed`: Authentication or decryption failure

Always handle these errors appropriately in production code.

## Contributing

1. Follow Go security best practices
2. Add comprehensive tests for new functionality
3. Include real-world examples in documentation
4. Ensure constant-time operations for sensitive comparisons
5. Update benchmarks for performance-critical changes

## License

This package is part of the core types library and follows the same license terms.
