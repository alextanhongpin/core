# Email - Email Validation and Utilities

[![Go Reference](https://pkg.go.dev/badge/github.com/alextanhongpin/core/types/email.svg)](https://pkg.go.dev/github.com/alextanhongpin/core/types/email)

Package `email` provides comprehensive email address validation and utilities for Go applications. It offers both RFC 5322 compliant validation and simpler patterns for different use cases, along with utilities for email normalization and analysis.

## Features

- **RFC 5322 Compliant Validation**: Comprehensive email validation following official standards
- **Basic Validation**: Faster, simpler validation for performance-critical applications  
- **Email Normalization**: Consistent email formatting for storage and comparison
- **Domain Extraction**: Extract domain and local parts from email addresses
- **Business Email Detection**: Distinguish between business and consumer email providers
- **Zero Dependencies**: Pure Go implementation with no external dependencies

## Installation

```bash
go get github.com/alextanhongpin/core/types/email
```

## Quick Start

```go
package main

import (
    "fmt"
    "github.com/alextanhongpin/core/types/email"
)

func main() {
    // Basic validation
    if email.IsValid("user@example.com") {
        fmt.Println("Valid email!")
    }

    // Normalize for consistent storage
    normalized := email.Normalize("  User@Example.COM  ")
    fmt.Println(normalized) // Output: user@example.com

    // Extract components
    domain := email.Domain("user@company.com")      // company.com
    local := email.LocalPart("user@company.com")    // user

    // Check if business email
    isBusiness := email.IsBusinessEmail("admin@company.com") // true
    isConsumer := email.IsBusinessEmail("user@gmail.com")    // false
}
```

## API Reference

### Validation Functions

#### `IsValid(s string) bool`
Validates an email address using the comprehensive RFC 5322 pattern. This is the recommended validation method for most applications.

```go
valid := email.IsValid("test.email+tag@domain.co.uk") // true
valid = email.IsValid("invalid.email")                // false
```

#### `IsValidBasic(s string) bool`
Validates an email address using a simpler, faster pattern. Use this for high-performance scenarios where basic validation is sufficient.

```go
valid := email.IsValidBasic("simple@example.com") // true
valid = email.IsValidBasic("user@")               // false
```

### Utility Functions

#### `Normalize(s string) string`
Normalizes an email address by converting to lowercase and trimming whitespace.

```go
normalized := email.Normalize("  User@Example.COM  ") // "user@example.com"
```

#### `Domain(s string) string`
Extracts the domain part from an email address.

```go
domain := email.Domain("user@company.com") // "company.com"
```

#### `LocalPart(s string) string`
Extracts the local part (before @) from an email address.

```go
local := email.LocalPart("user@company.com") // "user"
```

#### `IsBusinessEmail(s string) bool`
Checks if an email uses a business domain (not a common consumer email provider).

```go
isBusiness := email.IsBusinessEmail("admin@company.com") // true
isConsumer := email.IsBusinessEmail("user@gmail.com")    // false
```

## Real-World Examples

### User Registration System

```go
type UserRegistration struct {
    Email     string
    FirstName string
    LastName  string
}

func (ur *UserRegistration) Validate() error {
    // Normalize email for consistent storage
    ur.Email = email.Normalize(ur.Email)
    
    // Validate email format
    if !email.IsValid(ur.Email) {
        return fmt.Errorf("invalid email address: %s", ur.Email)
    }
    
    return nil
}

func registerUser(rawEmail, firstName, lastName string) error {
    user := &UserRegistration{
        Email:     rawEmail,
        FirstName: firstName,
        LastName:  lastName,
    }
    
    if err := user.Validate(); err != nil {
        return err
    }
    
    // Save to database...
    return nil
}
```

### Newsletter Subscription Service

```go
type SubscriptionService struct {
    subscriptions map[string]*Subscription
}

func (s *SubscriptionService) Subscribe(rawEmail string) error {
    normalizedEmail := email.Normalize(rawEmail)
    
    if !email.IsValid(normalizedEmail) {
        return fmt.Errorf("invalid email address")
    }
    
    if _, exists := s.subscriptions[normalizedEmail]; exists {
        return fmt.Errorf("email already subscribed")
    }
    
    s.subscriptions[normalizedEmail] = &Subscription{
        Email:    normalizedEmail,
        IsActive: true,
    }
    
    return nil
}

func (s *SubscriptionService) GetBusinessSubscribers() []*Subscription {
    var businessSubs []*Subscription
    for _, sub := range s.subscriptions {
        if sub.IsActive && email.IsBusinessEmail(sub.Email) {
            businessSubs = append(businessSubs, sub)
        }
    }
    return businessSubs
}
```

### Email Domain Analytics

```go
type EmailAnalytics struct {
    domains map[string]int
    total   int
}

func (ea *EmailAnalytics) AddEmail(emailAddr string) {
    if !email.IsValid(emailAddr) {
        return
    }
    
    domain := email.Domain(email.Normalize(emailAddr))
    if domain != "" {
        ea.domains[domain]++
        ea.total++
    }
}

func (ea *EmailAnalytics) GetTopDomains(limit int) []DomainStats {
    // Implementation for getting most common domains
    // Useful for understanding your user base
}
```

### Email Verification System

```go
type EmailVerificationService struct {
    pendingVerifications map[string]string
    verifiedEmails      map[string]bool
}

func (evs *EmailVerificationService) SendVerification(rawEmail string) error {
    normalizedEmail := email.Normalize(rawEmail)
    
    if !email.IsValid(normalizedEmail) {
        return fmt.Errorf("invalid email address")
    }
    
    if evs.verifiedEmails[normalizedEmail] {
        return fmt.Errorf("email already verified")
    }
    
    token := generateVerificationToken()
    evs.pendingVerifications[normalizedEmail] = token
    
    // Send verification email...
    return nil
}

func (evs *EmailVerificationService) VerifyEmail(rawEmail, token string) error {
    normalizedEmail := email.Normalize(rawEmail)
    
    expectedToken, exists := evs.pendingVerifications[normalizedEmail]
    if !exists || token != expectedToken {
        return fmt.Errorf("invalid verification")
    }
    
    evs.verifiedEmails[normalizedEmail] = true
    delete(evs.pendingVerifications, normalizedEmail)
    
    return nil
}
```

## Performance Considerations

- **Use `IsValidBasic` for high-volume scenarios**: If you're processing thousands of emails per second, the basic validation is significantly faster
- **Normalize emails early**: Always normalize emails when storing them to ensure consistency
- **Cache validation results**: For repeated validation of the same emails, consider caching results

```go
// For high-performance scenarios
func validateEmailsFast(emails []string) []string {
    var valid []string
    for _, e := range emails {
        if email.IsValidBasic(e) {
            valid = append(valid, email.Normalize(e))
        }
    }
    return valid
}
```

## Integration with Validation Libraries

Use with the `assert` package for structured validation:

```go
import "github.com/alextanhongpin/core/types/assert"

type User struct {
    Email string `json:"email"`
}

func (u *User) Validate() map[string]string {
    return assert.Map(map[string]string{
        "email": assert.Required(u.Email,
            assert.Is(email.IsValid(u.Email), "must be a valid email address"),
        ),
    })
}
```

## Best Practices

1. **Always normalize emails before storage** to ensure consistent lookups
2. **Use comprehensive validation by default** unless performance is critical
3. **Consider business vs consumer emails** for B2B applications
4. **Validate early** at API boundaries and form inputs
5. **Handle edge cases** like very long emails (>254 characters per RFC 5321)

## Consumer Email Providers

The package recognizes these common consumer email providers:
- gmail.com
- yahoo.com  
- hotmail.com
- outlook.com
- aol.com
- icloud.com
- mail.com
- protonmail.com
- yandex.com

All other domains are considered "business" domains.

## License

MIT License - see LICENSE file for details.
