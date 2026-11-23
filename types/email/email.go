// Package email validates email address syntax and splits addresses into
// local part and domain. It does not check that a mailbox or domain exists.
package email

import (
	"regexp"
	"strings"
)

// Comprehensive email validation pattern following RFC 5322 specification
var comprehensivePattern = regexp.MustCompile("^(?:(?:(?:(?:[a-zA-Z]|\\d|[!#\\$%&'\\*\\+\\-\\/=\\?\\^_`{\\|}~]|[\\x{00A0}-\\x{D7FF}\\x{F900}-\\x{FDCF}\\x{FDF0}-\\x{FFEF}])+(?:\\.([a-zA-Z]|\\d|[!#\\$%&'\\*\\+\\-\\/=\\?\\^_`{\\|}~]|[\\x{00A0}-\\x{D7FF}\\x{F900}-\\x{FDCF}\\x{FDF0}-\\x{FFEF}])+)*)|(?:(?:\\x22)(?:(?:(?:(?:\\x20|\\x09)*(?:\\x0d\\x0a))?(?:\\x20|\\x09)+)?(?:(?:[\\x01-\\x08\\x0b\\x0c\\x0e-\\x1f\\x7f]|\\x21|[\\x23-\\x5b]|[\\x5d-\\x7e]|[\\x{00A0}-\\x{D7FF}\\x{F900}-\\x{FDCF}\\x{FDF0}-\\x{FFEF}])|(?:(?:[\\x01-\\x09\\x0b\\x0c\\x0d-\\x7f]|[\\x{00A0}-\\x{D7FF}\\x{F900}-\\x{FDCF}\\x{FDF0}-\\x{FFEF}]))))*(?:(?:(?:\\x20|\\x09)*(?:\\x0d\\x0a))?(\\x20|\\x09)+)?(?:\\x22))))@(?:(?:(?:[a-zA-Z]|\\d|[\\x{00A0}-\\x{D7FF}\\x{F900}-\\x{FDCF}\\x{FDF0}-\\x{FFEF}])|(?:(?:[a-zA-Z]|\\d|[\\x{00A0}-\\x{D7FF}\\x{F900}-\\x{FDCF}\\x{FDF0}-\\x{FFEF}])(?:[a-zA-Z]|\\d|-|\\.|~|[\\x{00A0}-\\x{D7FF}\\x{F900}-\\x{FDCF}\\x{FDF0}-\\x{FFEF}])*(?:[a-zA-Z]|\\d|[\\x{00A0}-\\x{D7FF}\\x{F900}-\\x{FDCF}\\x{FDF0}-\\x{FFEF}])))\\.)+(?:(?:[a-zA-Z]|[\\x{00A0}-\\x{D7FF}\\x{F900}-\\x{FDCF}\\x{FDF0}-\\x{FFEF}])|(?:(?:[a-zA-Z]|[\\x{00A0}-\\x{D7FF}\\x{F900}-\\x{FDCF}\\x{FDF0}-\\x{FFEF}])(?:[a-zA-Z]|\\d|-|\\.|~|[\\x{00A0}-\\x{D7FF}\\x{F900}-\\x{FDCF}\\x{FDF0}-\\x{FFEF}])*(?:[a-zA-Z]|[\\x{00A0}-\\x{D7FF}\\x{F900}-\\x{FDCF}\\x{FDF0}-\\x{FFEF}])))\\.?$")

// Basic email validation pattern for most common use cases
var basicPattern = regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)

// IsValid validates an email address using the comprehensive RFC 5322 pattern.
// This is the default validation method and handles most edge cases.
func IsValid(s string) bool {
	if len(s) > 254 { // RFC 5321 limit
		return false
	}
	return comprehensivePattern.MatchString(s)
}

// IsValidBasic validates an email address using a simpler pattern.
// This is faster but less comprehensive than IsValid.
// Use this for performance-critical applications where basic validation is sufficient.
func IsValidBasic(s string) bool {
	if len(s) > 254 { // RFC 5321 limit
		return false
	}
	return basicPattern.MatchString(s)
}

// Normalize normalizes an email address by converting to lowercase
// and trimming whitespace. This is useful for storing emails consistently.
func Normalize(s string) string {
	return strings.ToLower(strings.TrimSpace(s))
}

// Domain returns the part after the last "@", or "" if s has no "@" or
// either side is empty. It does not validate s.
func Domain(s string) string {
	_, domain, ok := split(s)
	if !ok {
		return ""
	}
	return domain
}

// LocalPart returns the part before the last "@", or "" if s has no "@" or
// either side is empty. It does not validate s.
func LocalPart(s string) string {
	local, _, ok := split(s)
	if !ok {
		return ""
	}
	return local
}

// split splits at the last "@" because a quoted local part may contain "@".
func split(s string) (local, domain string, ok bool) {
	i := strings.LastIndexByte(s, '@')
	if i <= 0 || i == len(s)-1 {
		return "", "", false
	}
	return s[:i], s[i+1:], true
}

// consumerDomains lists common free mailbox providers. It is deliberately
// short; it is not a disposable-address or free-provider database.
var consumerDomains = map[string]bool{
	"gmail.com":      true,
	"yahoo.com":      true,
	"hotmail.com":    true,
	"outlook.com":    true,
	"aol.com":        true,
	"icloud.com":     true,
	"mail.com":       true,
	"protonmail.com": true,
	"yandex.com":     true,
}

// IsBusinessEmail reports whether s has a domain that is not one of a short,
// fixed list of consumer providers (gmail.com, yahoo.com, outlook.com, ...).
// It returns false when s has no domain. It does not validate s.
func IsBusinessEmail(s string) bool {
	domain := strings.ToLower(Domain(s))
	if domain == "" {
		return false
	}
	return !consumerDomains[domain]
}
