package email_test

import (
	"fmt"
	"log"
	"strings"

	"github.com/alextanhongpin/core/types/email"
)

// Example: User registration with email validation
func ExampleIsValid() {
	emails := []string{
		"user@example.com",
		"test.email+tag@domain.co.uk",
		"invalid.email",
		"user@",
		"@domain.com",
		"user..double.dot@example.com",
	}

	fmt.Println("Email validation results:")
	for _, e := range emails {
		valid := email.IsValid(e)
		fmt.Printf("%-30s: %t\n", e, valid)
	}
	// Output:
	// Email validation results:
	// user@example.com              : true
	// test.email+tag@domain.co.uk   : true
	// invalid.email                 : false
	// user@                         : false
	// @domain.com                   : false
	// user..double.dot@example.com  : false
}

// Example: Performance-focused validation for high-volume applications
func ExampleIsValidBasic() {
	// In high-performance scenarios, use basic validation
	emails := []string{
		"simple@example.com",
		"user.name@domain.org",
		"invalid@",
		"test@domain",
	}

	fmt.Println("Basic email validation:")
	for _, e := range emails {
		valid := email.IsValidBasic(e)
		fmt.Printf("%-20s: %t\n", e, valid)
	}
	// Output:
	// Basic email validation:
	// simple@example.com  : true
	// user.name@domain.org: true
	// invalid@            : false
	// test@domain         : false
}

// Example: Email normalization for consistent storage
func ExampleNormalize() {
	rawEmails := []string{
		"  User@Example.COM  ",
		"Test.Email@DOMAIN.org",
		"\tuser@example.com\n",
	}

	fmt.Println("Email normalization:")
	for _, e := range rawEmails {
		normalized := email.Normalize(e)
		fmt.Printf("'%s' -> '%s'\n", e, normalized)
	}
	// Output:
	// Email normalization:
	// '  User@Example.COM  ' -> 'user@example.com'
	// 'Test.Email@DOMAIN.org' -> 'test.email@domain.org'
	// '	user@example.com
	// ' -> 'user@example.com'
}

// Example: Extracting email components
func ExampleDomain() {
	emails := []string{
		"user@company.com",
		"admin@subdomain.example.org",
		"invalid-email",
	}

	fmt.Println("Domain extraction:")
	for _, e := range emails {
		domain := email.Domain(e)
		localPart := email.LocalPart(e)
		fmt.Printf("%-25s -> domain: '%s', local: '%s'\n", e, domain, localPart)
	}
	// Output:
	// Domain extraction:
	// user@company.com      -> domain: 'company.com', local: 'user'
	// admin@subdomain.example.org -> domain: 'subdomain.example.org', local: 'admin'
	// invalid-email         -> domain: '', local: ''
}

// Example: Distinguishing business vs consumer emails
func ExampleIsBusinessEmail() {
	emails := []string{
		"john@company.com",
		"user@gmail.com",
		"admin@startup.io",
		"personal@yahoo.com",
		"team@business.org",
		"me@hotmail.com",
	}

	fmt.Println("Business email detection:")
	for _, e := range emails {
		isBusiness := email.IsBusinessEmail(e)
		emailType := "consumer"
		if isBusiness {
			emailType = "business"
		}
		fmt.Printf("%-20s: %s\n", e, emailType)
	}
	// Output:
	// Business email detection:
	// john@company.com    : business
	// user@gmail.com      : consumer
	// admin@startup.io    : business
	// personal@yahoo.com  : consumer
	// team@business.org   : business
	// me@hotmail.com      : consumer
}

// Real-world example: Email subscription system
type NewsletterSubscription struct {
	Email     string `json:"email"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
	IsActive  bool   `json:"is_active"`
}

type SubscriptionService struct {
	subscriptions map[string]*NewsletterSubscription
}

func NewSubscriptionService() *SubscriptionService {
	return &SubscriptionService{
		subscriptions: make(map[string]*NewsletterSubscription),
	}
}

func (s *SubscriptionService) Subscribe(rawEmail, firstName, lastName string) error {
	// Normalize email for consistent storage
	normalizedEmail := email.Normalize(rawEmail)

	// Validate email
	if !email.IsValid(normalizedEmail) {
		return fmt.Errorf("invalid email address: %s", rawEmail)
	}

	// Check if already subscribed
	if _, exists := s.subscriptions[normalizedEmail]; exists {
		return fmt.Errorf("email already subscribed: %s", normalizedEmail)
	}

	// Create subscription
	subscription := &NewsletterSubscription{
		Email:     normalizedEmail,
		FirstName: firstName,
		LastName:  lastName,
		IsActive:  true,
	}

	s.subscriptions[normalizedEmail] = subscription
	return nil
}

func (s *SubscriptionService) GetBusinessSubscribers() []*NewsletterSubscription {
	var businessSubs []*NewsletterSubscription
	for _, sub := range s.subscriptions {
		if sub.IsActive && email.IsBusinessEmail(sub.Email) {
			businessSubs = append(businessSubs, sub)
		}
	}
	return businessSubs
}

func ExampleSubscriptionService() {
	service := NewSubscriptionService()

	// Subscribe various users
	users := []struct {
		email, firstName, lastName string
	}{
		{"  John.Doe@COMPANY.COM  ", "John", "Doe"},
		{"jane@startup.io", "Jane", "Smith"},
		{"personal@gmail.com", "Bob", "Wilson"},
		{"invalid-email", "Invalid", "User"},
		{"admin@business.org", "Admin", "User"},
	}

	fmt.Println("Newsletter subscription results:")
	for _, user := range users {
		err := service.Subscribe(user.email, user.firstName, user.lastName)
		if err != nil {
			fmt.Printf("❌ %s: %v\n", user.email, err)
		} else {
			fmt.Printf("✅ %s: successfully subscribed\n", email.Normalize(user.email))
		}
	}

	// Get business subscribers
	businessSubs := service.GetBusinessSubscribers()
	fmt.Printf("\nBusiness subscribers (%d):\n", len(businessSubs))
	for _, sub := range businessSubs {
		fmt.Printf("- %s %s <%s>\n", sub.FirstName, sub.LastName, sub.Email)
	}

	// Output:
	// Newsletter subscription results:
	// ✅   John.Doe@COMPANY.COM  : successfully subscribed
	// ✅ jane@startup.io: successfully subscribed
	// ✅ personal@gmail.com: successfully subscribed
	// ❌ invalid-email: invalid email address: invalid-email
	// ✅ admin@business.org: successfully subscribed
	//
	// Business subscribers (3):
	// - John Doe <john.doe@company.com>
	// - Jane Smith <jane@startup.io>
	// - Admin User <admin@business.org>
}

// Real-world example: Email domain analytics
type EmailAnalytics struct {
	domains map[string]int
	total   int
}

func NewEmailAnalytics() *EmailAnalytics {
	return &EmailAnalytics{
		domains: make(map[string]int),
	}
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

func (ea *EmailAnalytics) TopDomains(limit int) []struct {
	Domain  string
	Count   int
	Percent float64
} {
	var results []struct {
		Domain  string
		Count   int
		Percent float64
	}

	for domain, count := range ea.domains {
		percent := float64(count) / float64(ea.total) * 100
		results = append(results, struct {
			Domain  string
			Count   int
			Percent float64
		}{
			Domain:  domain,
			Count:   count,
			Percent: percent,
		})
	}

	// Simple sort by count (descending)
	for i := 0; i < len(results)-1; i++ {
		for j := i + 1; j < len(results); j++ {
			if results[j].Count > results[i].Count {
				results[i], results[j] = results[j], results[i]
			}
		}
	}

	if len(results) > limit {
		results = results[:limit]
	}

	return results
}

func ExampleEmailAnalytics() {
	analytics := NewEmailAnalytics()

	// Simulate email addresses from user registrations
	emails := []string{
		"user1@gmail.com", "user2@gmail.com", "user3@gmail.com",
		"admin@company.com", "support@company.com",
		"john@startup.io", "jane@startup.io",
		"customer@business.org",
		"personal@yahoo.com",
		"invalid-email", // This will be ignored
	}

	for _, email := range emails {
		analytics.AddEmail(email)
	}

	fmt.Printf("Email Domain Analytics (Total: %d valid emails)\n", analytics.total)
	fmt.Println(strings.Repeat("-", 40))

	topDomains := analytics.TopDomains(5)
	for i, domain := range topDomains {
		fmt.Printf("%d. %-20s %2d emails (%.1f%%)\n",
			i+1, domain.Domain, domain.Count, domain.Percent)
	}

	// Output:
	// Email Domain Analytics (Total: 9 valid emails)
	// ----------------------------------------
	// 1. gmail.com             3 emails (33.3%)
	// 2. company.com           2 emails (22.2%)
	// 3. startup.io            2 emails (22.2%)
	// 4. business.org          1 emails (11.1%)
	// 5. yahoo.com             1 emails (11.1%)
}

// Real-world example: Email verification system
type EmailVerificationService struct {
	pendingVerifications map[string]string // email -> verification token
	verifiedEmails       map[string]bool
}

func NewEmailVerificationService() *EmailVerificationService {
	return &EmailVerificationService{
		pendingVerifications: make(map[string]string),
		verifiedEmails:       make(map[string]bool),
	}
}

func (evs *EmailVerificationService) SendVerification(rawEmail string) error {
	normalizedEmail := email.Normalize(rawEmail)

	if !email.IsValid(normalizedEmail) {
		return fmt.Errorf("invalid email address")
	}

	if evs.verifiedEmails[normalizedEmail] {
		return fmt.Errorf("email already verified")
	}

	// Generate verification token (simplified)
	token := fmt.Sprintf("token_%s", strings.ReplaceAll(normalizedEmail, "@", "_"))
	evs.pendingVerifications[normalizedEmail] = token

	log.Printf("Verification email sent to %s with token: %s", normalizedEmail, token)
	return nil
}

func (evs *EmailVerificationService) VerifyEmail(rawEmail, token string) error {
	normalizedEmail := email.Normalize(rawEmail)

	expectedToken, exists := evs.pendingVerifications[normalizedEmail]
	if !exists {
		return fmt.Errorf("no pending verification for this email")
	}

	if token != expectedToken {
		return fmt.Errorf("invalid verification token")
	}

	// Mark as verified
	evs.verifiedEmails[normalizedEmail] = true
	delete(evs.pendingVerifications, normalizedEmail)

	log.Printf("Email verified successfully: %s", normalizedEmail)
	return nil
}

func ExampleEmailVerificationService() {
	service := NewEmailVerificationService()

	testEmail := "  Test.User@Example.COM  "

	// Send verification
	if err := service.SendVerification(testEmail); err != nil {
		log.Printf("Failed to send verification: %v", err)
		return
	}

	// Simulate user clicking verification link
	normalizedEmail := email.Normalize(testEmail)
	token := fmt.Sprintf("token_%s", strings.ReplaceAll(normalizedEmail, "@", "_"))

	if err := service.VerifyEmail(testEmail, token); err != nil {
		log.Printf("Failed to verify email: %v", err)
		return
	}

	fmt.Println("Email verification flow completed successfully")
	// Output:
	// Email verification flow completed successfully
}
