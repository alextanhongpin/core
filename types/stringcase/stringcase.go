package stringcase

import (
	"regexp"
	"slices"
	"strings"
	"unicode"
)

var (
	// The order matters as regexp will match the longest first.
	// We want to avoid false match with UUID/UID/UI.
	split       = regexp.MustCompile(`([A-Z][a-z0-9]+|ASCII|HTTPS|GUID|HTML|HTTP|JSON|SMTP|UTF8|UUID|XSRF|API|CPU|CSS|DNS|EOF|LHS|QPS|RAM|RHS|RPC|SLA|SQL|SSH|TCP|TLS|TTL|UDP|UID|URI|URL|XML|XSS|ID|IP|UI|VM)|(ASCII|HTTPS|GUID|HTML|HTTP|JSON|SMTP|UTF8|UUID|XSRF|API|CPU|CSS|DNS|EOF|LHS|QPS|RAM|RHS|RPC|SLA|SQL|SSH|TCP|TLS|TTL|UDP|UID|URI|URL|XML|XSS|ID|IP|UI|VM|-|_)`)
	initialisms = map[string]struct{}{
		"API": {}, "ASCII": {}, "CPU": {}, "CSS": {}, "DNS": {}, "EOF": {}, "GUID": {}, "HTML": {}, "HTTP": {}, "HTTPS": {}, "ID": {}, "IP": {}, "JSON": {}, "LHS": {}, "QPS": {}, "RAM": {}, "RHS": {}, "RPC": {}, "SLA": {}, "SMTP": {}, "SQL": {}, "SSH": {}, "TCP": {}, "TLS": {}, "TTL": {}, "UDP": {}, "UI": {}, "UID": {}, "UUID": {}, "URI": {}, "URL": {}, "UTF8": {}, "VM": {}, "XML": {}, "XSRF": {}, "XSS": {}}
)

// ToKebab converts a string to kebab-case.
func ToKebab(s string) string {
	return strings.Join(tokenize(s), "-")
}

// ToSnake converts a string to snake_case.
func ToSnake(s string) string {
	return strings.Join(tokenize(s), "_")
}

// ToCamel converts a string to camelCase.
func ToCamel(s string) string {
	tokens := normalize(s)
	if len(tokens) == 0 {
		return ""
	}

	tokens[0] = strings.ToLower(tokens[0])
	return strings.Join(tokens, "")
}

// ToPascal converts a string to PascalCase.
func ToPascal(s string) string {
	return strings.Join(normalize(s), "")
}

// ToTitle converts a string to Title Case.
func ToTitle(s string) string {
	return strings.Join(normalize(s), " ")
}

func normalize(s string) []string {
	words := preserveInitialism(tokenize(s))
	for i := range words {
		words[i] = upperFirst(words[i])
	}
	return words
}

// ToWords splits a string into space-separated words, collapsing multiple spaces.
func tokenize(s string) []string {
	var result []string
	s = split.ReplaceAllString(s, " $1 ")
	for w := range strings.FieldsSeq(s) {
		result = append(result, strings.ToLower(w))
	}

	return result
}

// FromKebab converts kebab-case to space-separated words.
func FromKebab(s string) string {
	return strings.ReplaceAll(s, "-", " ")
}

// FromSnake converts snake_case to space-separated words.
func FromSnake(s string) string {
	return strings.ReplaceAll(s, "_", " ")
}

// preserveInitialism preserves initialisms as uppercase.
func preserveInitialism(s []string) []string {
	words := slices.Clone(s)
	for i, w := range s {
		upper := strings.ToUpper(w)
		if _, ok := initialisms[upper]; ok {
			words[i] = upper
		}
	}
	return words
}

// upperFirst uppercases the first rune in a string.
func upperFirst(s string) string {
	if s == "" {
		return ""
	}
	runes := []rune(s)
	runes[0] = unicode.ToUpper(runes[0])
	return string(runes)
}
