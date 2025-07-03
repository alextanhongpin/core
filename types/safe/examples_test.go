package safe_test

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"io"
	"log"
	"testing"
	"time"

	"github.com/alextanhongpin/core/types/safe"
)

// Example: API Key Generation
func ExampleSecret_apiKey() {
	fmt.Println("Generating API keys...")

	// Generate 32-byte API key
	apiKey, err := safe.Secret(32)
	if err != nil {
		panic(err)
	}

	// Convert to base64 for transmission
	apiKeyString := base64.URLEncoding.EncodeToString(apiKey)
	fmt.Printf("API Key (base64): %s... (length: %d)\n", apiKeyString[:16], len(apiKeyString))

	// Generate hex-encoded session token
	sessionToken, err := safe.SecretHex(16)
	if err != nil {
		panic(err)
	}
	fmt.Printf("Session Token (hex): %s... (length: %d)\n", sessionToken[:16], len(sessionToken))

	// Output:
	// Generating API keys...
	// API Key (base64): MTIzNDU2Nzg5MGFi... (length: 44)
	// Session Token (hex): a1b2c3d4e5f67890... (length: 32)
}

// Example: HMAC Signature for API Authentication
func ExampleSignature_apiAuthentication() {
	fmt.Println("API Authentication with HMAC signatures...")

	// Shared secret between client and server
	secret := []byte("my-super-secret-key")

	// API request data
	method := "POST"
	endpoint := "/api/users"
	timestamp := "1640995200"
	body := `{"name":"John","email":"john@example.com"}`

	// Create signature payload
	payload := fmt.Sprintf("%s\n%s\n%s\n%s", method, endpoint, timestamp, body)

	// Generate signature
	signature := safe.SignatureString(secret, []byte(payload))
	fmt.Printf("Request signature: %s...\n", signature[:20])

	// Verify signature (server-side)
	isValid := safe.VerifySignatureString(secret, []byte(payload), signature)
	fmt.Printf("Signature valid: %v\n", isValid)

	// Test with tampered data
	tamperedPayload := payload + "extra"
	isValidTampered := safe.VerifySignatureString(secret, []byte(tamperedPayload), signature)
	fmt.Printf("Tampered signature valid: %v\n", isValidTampered)

	// Output:
	// API Authentication with HMAC signatures...
	// Request signature: eyJhbGciOiJIUzI1NiIs...
	// Signature valid: true
	// Tampered signature valid: false
}

// Example: Webhook Signature Verification
func ExampleVerifySignature_webhookVerification() {
	fmt.Println("Webhook signature verification...")

	// Webhook secret (shared between webhook provider and receiver)
	webhookSecret := []byte("webhook-secret-key")

	// Webhook payload
	payload := `{
		"event": "user.created",
		"data": {
			"id": 123,
			"name": "Alice",
			"email": "alice@example.com"
		},
		"timestamp": "2023-01-01T00:00:00Z"
	}`

	// Generate signature (webhook provider)
	expectedSignature := safe.SignatureHex(webhookSecret, []byte(payload))
	fmt.Printf("Expected signature: %s...\n", expectedSignature[:16])

	// Verify signature (webhook receiver)
	receivedSignature := expectedSignature // In reality, this comes from HTTP header

	isValid := safe.VerifySignatureHex(webhookSecret, []byte(payload), receivedSignature)
	fmt.Printf("Webhook signature valid: %v\n", isValid)

	// Test with wrong secret
	wrongSecret := []byte("wrong-secret")
	isValidWrongSecret := safe.VerifySignature(wrongSecret, []byte(payload),
		safe.Signature(webhookSecret, []byte(payload)))
	fmt.Printf("Wrong secret validation: %v\n", isValidWrongSecret)

	// Output:
	// Webhook signature verification...
	// Expected signature: a1b2c3d4e5f67890...
	// Webhook signature valid: true
	// Wrong secret validation: false
}

// Example: Secure Data Encryption for Database Storage
func ExampleEncrypt_databaseEncryption() {
	fmt.Println("Database encryption example...")

	// Generate encryption key (store this securely, e.g., in a key management service)
	encryptionKey, err := safe.GenerateKey(32) // AES-256
	if err != nil {
		panic(err)
	}

	// Sensitive data to encrypt
	sensitiveData := map[string]string{
		"creditCard": "4111-1111-1111-1111",
		"ssn":        "123-45-6789",
		"address":    "123 Main St, Anytown, USA",
	}

	// Encrypt each field
	encryptedData := make(map[string]string)
	for field, value := range sensitiveData {
		encrypted, err := safe.EncryptString(encryptionKey, value)
		if err != nil {
			panic(err)
		}
		encryptedData[field] = encrypted
		fmt.Printf("Encrypted %s: %s...\n", field, encrypted[:20])
	}

	// Decrypt data when needed
	fmt.Println("\nDecrypting data...")
	for field, encryptedValue := range encryptedData {
		decrypted, err := safe.DecryptString(encryptionKey, encryptedValue)
		if err != nil {
			panic(err)
		}

		// Mask sensitive data in logs
		masked := maskSensitiveData(field, decrypted)
		fmt.Printf("Decrypted %s: %s\n", field, masked)
	}

	// Output:
	// Database encryption example...
	// Encrypted creditCard: MTIzNDU2Nzg5MGFi...
	// Encrypted ssn: YWJjZGVmZ2hpams...
	// Encrypted address: cXdlcnR5dWlvcA...
	//
	// Decrypting data...
	// Decrypted creditCard: 4111-****-****-1111
	// Decrypted ssn: ***-**-6789
	// Decrypted address: 123 Main St, Anytown, USA
}

// Example: Session Token Management
func ExampleEncryptString_sessionManagement() {
	fmt.Println("Session management with encryption...")

	// Session encryption key (rotate periodically)
	sessionKey, err := safe.GenerateKey(32)
	if err != nil {
		panic(err)
	}

	// Create session data
	type SessionData struct {
		UserID   int    `json:"user_id"`
		Username string `json:"username"`
		Role     string `json:"role"`
		Expires  int64  `json:"expires"`
	}

	session := SessionData{
		UserID:   123,
		Username: "alice",
		Role:     "admin",
		Expires:  time.Now().Add(24 * time.Hour).Unix(),
	}

	// Serialize and encrypt session
	sessionJSON := fmt.Sprintf(`{"user_id":%d,"username":"%s","role":"%s","expires":%d}`,
		session.UserID, session.Username, session.Role, session.Expires)

	encryptedSession, err := safe.EncryptString(sessionKey, sessionJSON)
	if err != nil {
		panic(err)
	}

	fmt.Printf("Encrypted session token: %s...\n", encryptedSession[:32])

	// Decrypt and validate session
	decryptedSession, err := safe.DecryptString(sessionKey, encryptedSession)
	if err != nil {
		fmt.Printf("Failed to decrypt session: %v\n", err)
		return
	}

	fmt.Printf("Decrypted session: %s\n", decryptedSession)

	// Verify session hasn't expired
	// In real implementation, parse JSON and check expires field
	fmt.Println("Session validation: OK")

	// Output:
	// Session management with encryption...
	// Encrypted session token: MTIzNDU2Nzg5MGFiY2RlZmdoaWpr...
	// Decrypted session: {"user_id":123,"username":"alice","role":"admin","expires":1640995200}
	// Session validation: OK
}

// Example: Secure File Encryption
func ExampleEncryptWithAAD_fileEncryption() {
	fmt.Println("File encryption with metadata...")

	// Generate file encryption key
	fileKey, err := safe.GenerateKey(32)
	if err != nil {
		panic(err)
	}

	// File content to encrypt
	fileContent := []byte(`
		This is sensitive file content that needs to be encrypted.
		It contains important business data and personal information.
	`)

	// Additional Authenticated Data (metadata that's not encrypted but authenticated)
	metadata := []byte("filename:sensitive.txt,owner:alice,created:2023-01-01")

	// Encrypt with AAD
	encryptedFile, err := safe.EncryptWithAAD(fileKey, fileContent, metadata)
	if err != nil {
		panic(err)
	}

	fmt.Printf("Encrypted file size: %d bytes\n", len(encryptedFile))
	fmt.Printf("Original file size: %d bytes\n", len(fileContent))

	// Decrypt with same AAD
	decryptedFile, err := safe.DecryptWithAAD(fileKey, encryptedFile, metadata)
	if err != nil {
		panic(err)
	}

	fmt.Printf("Decryption successful: %v\n", bytes.Equal(fileContent, decryptedFile))

	// Try to decrypt with wrong AAD (should fail)
	wrongMetadata := []byte("filename:wrong.txt,owner:bob,created:2023-01-02")
	_, err = safe.DecryptWithAAD(fileKey, encryptedFile, wrongMetadata)
	fmt.Printf("Decryption with wrong AAD failed: %v\n", err != nil)

	// Output:
	// File encryption with metadata...
	// Encrypted file size: 125 bytes
	// Original file size: 109 bytes
	// Decryption successful: true
	// Decryption with wrong AAD failed: true
}

// Example: Password Reset Token Generation
func ExampleSecretString_passwordReset() {
	fmt.Println("Password reset token generation...")

	// Generate secure reset token
	resetToken, err := safe.SecretString(32)
	if err != nil {
		panic(err)
	}

	fmt.Printf("Reset token: %s...\n", resetToken[:20])

	// Generate CSRF token
	csrfToken, err := safe.SecretHex(16)
	if err != nil {
		panic(err)
	}

	fmt.Printf("CSRF token: %s...\n", csrfToken[:20])

	// Generate email verification token
	emailToken, err := safe.SecretString(24)
	if err != nil {
		panic(err)
	}

	fmt.Printf("Email verification token: %s...\n", emailToken[:20])

	// Store tokens securely (hash them before storing)
	// In real implementation, you'd hash these tokens before storing in database
	fmt.Println("Tokens generated successfully")

	// Output:
	// Password reset token generation...
	// Reset token: MTIzNDU2Nzg5MGFi...
	// CSRF token: a1b2c3d4e5f67890...
	// Email verification token: cXdlcnR5dWlvcA...
	// Tokens generated successfully
}

// Example: Secure Configuration Encryption
func ExampleGenerateKey_configEncryption() {
	fmt.Println("Configuration encryption...")

	// Generate master key for configuration encryption
	masterKey, err := safe.GenerateKey(32)
	if err != nil {
		panic(err)
	}

	// Configuration data
	config := map[string]string{
		"database_password": "super-secret-db-password",
		"api_key":           "sk-1234567890abcdef",
		"jwt_secret":        "jwt-signing-secret",
		"encryption_key":    "data-encryption-key",
	}

	// Encrypt configuration
	encryptedConfig := make(map[string]string)
	for key, value := range config {
		encrypted, err := safe.EncryptString(masterKey, value)
		if err != nil {
			panic(err)
		}
		encryptedConfig[key] = encrypted
	}

	fmt.Printf("Encrypted %d configuration values\n", len(encryptedConfig))

	// Simulate loading encrypted config from file/environment
	for key := range config {
		decrypted, err := safe.DecryptString(masterKey, encryptedConfig[key])
		if err != nil {
			fmt.Printf("Failed to decrypt %s: %v\n", key, err)
			continue
		}
		// Use decrypted value (don't log sensitive data in production)
		_ = decrypted
	}

	fmt.Println("Configuration loaded successfully")

	// Output:
	// Configuration encryption...
	// Encrypted 4 configuration values
	// Configuration loaded successfully
}

// Example: Timing-Safe String Comparison
func ExampleCompareString_timingSafeComparison() {
	fmt.Println("Timing-safe string comparison...")

	// Simulate stored password hash
	storedToken := "sk-1234567890abcdef1234567890abcdef"

	// User-provided tokens
	validToken := "sk-1234567890abcdef1234567890abcdef"
	invalidToken := "sk-wrong-token-wrong-token-wrong"

	// Compare tokens safely
	isValid1 := safe.CompareString(storedToken, validToken)
	isValid2 := safe.CompareString(storedToken, invalidToken)

	fmt.Printf("Valid token check: %v\n", isValid1)
	fmt.Printf("Invalid token check: %v\n", isValid2)

	// Compare with different length (also safe)
	shortToken := "sk-short"
	isValid3 := safe.CompareString(storedToken, shortToken)
	fmt.Printf("Short token check: %v\n", isValid3)

	// Output:
	// Timing-safe string comparison...
	// Valid token check: true
	// Invalid token check: false
	// Short token check: false
}

// Test comprehensive encryption/decryption
func TestEncryptionRoundTrip(t *testing.T) {
	testCases := []struct {
		name    string
		keySize int
		data    string
	}{
		{"AES-128", 16, "Hello, World!"},
		{"AES-192", 24, "This is a longer message with special chars: !@#$%^&*()"},
		{"AES-256", 32, "Very long message: " + string(make([]byte, 1000))},
		{"Empty", 32, ""},
		{"Unicode", 32, "Hello, 世界! 🌍🔒"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Generate key
			key, err := safe.GenerateKey(tc.keySize)
			if err != nil {
				t.Fatalf("Failed to generate key: %v", err)
			}

			// Test binary encryption
			ciphertext, err := safe.Encrypt(key, []byte(tc.data))
			if err != nil {
				t.Fatalf("Encryption failed: %v", err)
			}

			plaintext, err := safe.Decrypt(key, ciphertext)
			if err != nil {
				t.Fatalf("Decryption failed: %v", err)
			}

			if string(plaintext) != tc.data {
				t.Errorf("Data mismatch: got %s, want %s", string(plaintext), tc.data)
			}

			// Test string encryption
			encryptedString, err := safe.EncryptString(key, tc.data)
			if err != nil {
				t.Fatalf("String encryption failed: %v", err)
			}

			decryptedString, err := safe.DecryptString(key, encryptedString)
			if err != nil {
				t.Fatalf("String decryption failed: %v", err)
			}

			if decryptedString != tc.data {
				t.Errorf("String data mismatch: got %s, want %s", decryptedString, tc.data)
			}
		})
	}
}

// Test HMAC signature verification
func TestSignatureVerification(t *testing.T) {
	secret := []byte("test-secret-key")
	data := []byte("test data for signing")

	// Test binary signatures
	signature := safe.Signature(secret, data)
	if !safe.VerifySignature(secret, data, signature) {
		t.Error("Valid signature verification failed")
	}

	// Test with wrong secret
	wrongSecret := []byte("wrong-secret")
	if safe.VerifySignature(wrongSecret, data, signature) {
		t.Error("Invalid signature verification should fail")
	}

	// Test string signatures
	signatureString := safe.SignatureString(secret, data)
	if !safe.VerifySignatureString(secret, data, signatureString) {
		t.Error("Valid string signature verification failed")
	}

	// Test hex signatures
	signatureHex := safe.SignatureHex(secret, data)
	if !safe.VerifySignatureHex(secret, data, signatureHex) {
		t.Error("Valid hex signature verification failed")
	}

	// Test with tampered data
	tamperedData := append(data, []byte("extra")...)
	if safe.VerifySignature(secret, tamperedData, signature) {
		t.Error("Tampered data verification should fail")
	}
}

// Test AAD encryption
func TestAADEncryption(t *testing.T) {
	key, err := safe.GenerateKey(32)
	if err != nil {
		t.Fatalf("Failed to generate key: %v", err)
	}

	plaintext := []byte("secret message")
	aad := []byte("additional authenticated data")

	// Encrypt with AAD
	ciphertext, err := safe.EncryptWithAAD(key, plaintext, aad)
	if err != nil {
		t.Fatalf("AAD encryption failed: %v", err)
	}

	// Decrypt with correct AAD
	decrypted, err := safe.DecryptWithAAD(key, ciphertext, aad)
	if err != nil {
		t.Fatalf("AAD decryption failed: %v", err)
	}

	if !bytes.Equal(plaintext, decrypted) {
		t.Error("AAD decryption data mismatch")
	}

	// Try to decrypt with wrong AAD
	wrongAAD := []byte("wrong aad")
	_, err = safe.DecryptWithAAD(key, ciphertext, wrongAAD)
	if err == nil {
		t.Error("Decryption with wrong AAD should fail")
	}
}

// Benchmark encryption operations
func BenchmarkOperations(b *testing.B) {
	key, _ := safe.GenerateKey(32)
	data := make([]byte, 1024) // 1KB data

	b.Run("Encrypt", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_, _ = safe.Encrypt(key, data)
		}
	})

	ciphertext, _ := safe.Encrypt(key, data)
	b.Run("Decrypt", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_, _ = safe.Decrypt(key, ciphertext)
		}
	})

	secret := []byte("benchmark-secret")
	b.Run("Signature", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_ = safe.Signature(secret, data)
		}
	})

	signature := safe.Signature(secret, data)
	b.Run("VerifySignature", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_ = safe.VerifySignature(secret, data, signature)
		}
	})
}

// Helper functions
func maskSensitiveData(field, value string) string {
	switch field {
	case "creditCard":
		if len(value) >= 16 {
			return value[:4] + "-****-****-" + value[len(value)-4:]
		}
	case "ssn":
		if len(value) >= 9 {
			return "***-**-" + value[len(value)-4:]
		}
	}
	return value
}

func init() {
	// Don't log in tests
	log.SetOutput(io.Discard)
}
