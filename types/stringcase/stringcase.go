package stringcase

import (
	"regexp"
	"strings"
	"unicode"
)

var (
	matchFirstCap = regexp.MustCompile("(.)([A-Z][a-z]+)")
	matchAllCap   = regexp.MustCompile("([a-z0-9])([A-Z])")
	initialisms   = map[string]struct{}{
		"API": {}, "ASCII": {}, "CPU": {}, "CSS": {}, "DNS": {}, "EOF": {}, "GUID": {}, "HTML": {}, "HTTP": {}, "HTTPS": {}, "ID": {}, "IP": {}, "JSON": {}, "LHS": {}, "QPS": {}, "RAM": {}, "RHS": {}, "RPC": {}, "SLA": {}, "SMTP": {}, "SQL": {}, "SSH": {}, "TCP": {}, "TLS": {}, "TTL": {}, "UDP": {}, "UI": {}, "UID": {}, "UUID": {}, "URI": {}, "URL": {}, "UTF8": {}, "VM": {}, "XML": {}, "XSRF": {}, "XSS": {},
	}
)

// ToKebab converts a string to kebab-case.
func ToKebab(s string) string {
	s = toWords(s)
	return strings.ToLower(strings.ReplaceAll(s, " ", "-"))
}

// ToSnake converts a string to snake_case.
func ToSnake(s string) string {
	s = toWords(s)
	return strings.ToLower(strings.ReplaceAll(s, " ", "_"))
}

// ToCamel converts a string to camelCase.
func ToCamel(s string) string {
	words := splitWordsWithInitialism(toWords(s))
	if len(words) == 0 {
		return ""
	}
	for i := range words {
		words[i] = strings.ToLower(words[i])
	}
	words[0] = strings.ToLower(words[0])
	for i := 1; i < len(words); i++ {
		if _, ok := initialisms[strings.ToUpper(words[i])]; ok {
			words[i] = strings.ToUpper(words[i])
		} else {
			words[i] = upperFirst(words[i])
		}
	}
	return strings.Join(words, "")
}

// ToPascal converts a string to PascalCase.
func ToPascal(s string) string {
	words := splitWordsWithInitialism(toWords(s))
	for i := range words {
		words[i] = upperFirstInitialism(strings.ToLower(words[i]))
	}
	return strings.Join(words, "")
}

// ToTitle converts a string to Title Case.
func ToTitle(s string) string {
	words := splitWordsWithInitialism(toWords(s))
	for i := range words {
		words[i] = upperFirstInitialism(strings.ToLower(words[i]))
	}
	return strings.Join(words, " ")
}

// ToWords splits a string into space-separated words, collapsing multiple spaces.
func toWords(s string) string {
	s = matchFirstCap.ReplaceAllString(s, "$1 $2")
	s = matchAllCap.ReplaceAllString(s, "$1 $2")
	s = strings.ReplaceAll(s, "_", " ")
	s = strings.ReplaceAll(s, "-", " ")
	s = strings.Join(strings.Fields(s), " ") // collapse multiple spaces
	return s
}

// FromKebab converts kebab-case to space-separated words.
func FromKebab(s string) string {
	return strings.ReplaceAll(s, "-", " ")
}

// FromSnake converts snake_case to space-separated words.
func FromSnake(s string) string {
	return strings.ReplaceAll(s, "_", " ")
}

// splitWordsWithInitialism splits words and preserves initialisms as uppercase.
func splitWordsWithInitialism(s string) []string {
	words := strings.Fields(s)
	for i, w := range words {
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

// upperFirstInitialism uppercases the first rune, but preserves initialisms.
func upperFirstInitialism(s string) string {
	upper := strings.ToUpper(s)
	if _, ok := initialisms[upper]; ok {
		return upper
	}
	return upperFirst(s)
}
