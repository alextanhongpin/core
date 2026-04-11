package stringcase

import (
	"iter"
	"slices"
	"strings"
	"unicode"

	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

var (
	DefaultInitialisms = strings.Fields("API APIs ASCII ASCIIs CPU CPUs CSS DNS EOF EOFs GUID GUIDs HTML HTMLs HTTP HTTPS HTTPs ID IDs IP IPs JSON JSONs LHS QPS RAM RAMs RHS RPC RPCs SLA SLAs SMTP SMTPs SQL SQLs SSH SSHs TCP TCPs TLS TTL TTLs UDP UDPs UI UID UIDs UIs URI URIs URL URLs UTF8 UTF8s UUID UUIDs VM VMs XML XMLs XSRF XSRFs XSS")
	tokenizer          = NewTokenizer(DefaultInitialisms...)
)

// ToKebab converts a string to kebab-case.
func ToKebab(s string) string {
	return tokenizer.Kebab(s)
}

// ToSnake converts a string to snake_case.
func ToSnake(s string) string {
	return tokenizer.Snake(s)
}

// ToCamel converts a string to camelCase.
func ToCamel(s string) string {
	return tokenizer.Camel(s)
}

// ToPascal converts a string to PascalCase.
func ToPascal(s string) string {
	return tokenizer.Pascal(s)
}

// ToTitle converts a string to Title Case.
func ToTitle(s string) string {
	return tokenizer.Title(s)
}

// FromKebab converts kebab-case to space-separated words.
func FromKebab(s string) string {
	return strings.ReplaceAll(s, "-", " ")
}

// FromSnake converts snake_case to space-separated words.
func FromSnake(s string) string {
	return strings.ReplaceAll(s, "_", " ")
}

type Tokenizer struct {
	initialisms  []string
	lowerToUpper map[string]string
}

func NewTokenizer(initialisms ...string) *Tokenizer {
	m := make(map[string]string)
	for _, i := range initialisms {
		m[strings.ToLower(i)] = i
	}

	return &Tokenizer{
		initialisms:  initialisms,
		lowerToUpper: m,
	}
}

func (t *Tokenizer) Title(text string) string {
	caser := cases.Title(language.English)
	var result []string
	for token := range t.Tokenize(text) {
		upper, ok := t.lowerToUpper[token]
		if ok {
			result = append(result, upper)
		} else {
			result = append(result, caser.String(token))
		}
	}

	return strings.Join(result, " ")
}

func (t *Tokenizer) Pascal(text string) string {
	var result []string
	for token := range t.Tokenize(text) {
		upper, ok := t.lowerToUpper[token]
		if ok {
			result = append(result, upper)
		} else {
			result = append(result, uppercaseFirst(token))
		}
	}

	return strings.Join(result, "")
}

func (t *Tokenizer) Camel(text string) string {
	var result []string
	for token := range t.Tokenize(text) {
		if len(result) == 0 {
			result = append(result, token)
			continue
		}

		upper, ok := t.lowerToUpper[token]
		if ok {
			result = append(result, upper)
		} else {
			result = append(result, uppercaseFirst(token))
		}
	}

	return strings.Join(result, "")
}

func (t *Tokenizer) Snake(text string) string {
	return strings.Join(slices.Collect(t.Tokenize(text)), "_")
}

func (t *Tokenizer) Kebab(text string) string {
	return strings.Join(slices.Collect(t.Tokenize(text)), "-")
}

func (t *Tokenizer) Tokenize(text string) iter.Seq[string] {
	return func(yield func(string) bool) {
		text = strings.Map(func(r rune) rune {
			if unicode.IsDigit(r) || unicode.IsLetter(r) {
				return r
			}
			return ' '
		}, text)

		for word := range strings.FieldsSeq(text) {
			for token := range t.tokenize(word) {
				if !yield(strings.ToLower(token)) {
					break
				}
			}
		}
	}
}

func (t *Tokenizer) tokenize(text string) iter.Seq[string] {
	return func(yield func(string) bool) {
		next, stop := iter.Pull(segment(text))
		defer stop()
		for {
			a, ok := next()
			if !ok {
				break
			}
			if strings.ToLower(a) == a {
				if !yield(a) {
					break
				}
				continue
			}
			b, ok := next()
			if !ok {
				yield(a)
				break
			}
			if len(a) == 1 {
				if !yield(a + b) {
					return
				}
				continue
			}

			if slices.Contains(t.initialisms, a+b) {
				if !yield(a + b) {
					return
				}
				continue
			}

			upper, last := a[:len(a)-1], a[len(a)-1:]
			if !yield(upper) {
				return
			}
			if !yield(last + b) {
				return
			}
		}
	}
}

func segment(text string) iter.Seq[string] {
	return func(yield func(string) bool) {
		runes := []rune(text)
		isUpper := unicode.IsUpper(runes[0])
		var start int
		for i, r := range runes {
			toggle := (isUpper && unicode.IsLower(r)) || (!isUpper && unicode.IsUpper(r))
			if !toggle {
				continue
			}
			isUpper = unicode.IsUpper(r)
			if !yield(string(runes[start:i])) {
				return
			}
			start = i
		}
		if s := string(runes[start:]); s != "" {
			yield(s)
		}
	}
}

func uppercaseFirst(s string) string {
	if len(s) == 0 {
		return s
	}
	r := []rune(s)
	r[0] = unicode.ToUpper(r[0])
	return string(r)
}
