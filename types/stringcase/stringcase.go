package stringcase

import (
	"fmt"
	"math"
	"regexp"
	"slices"
	"strings"
	"unicode"

	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

var (
	// In Go, the "s" in a plural initialism or acronym should be lowercase
	// (e.g., IDs, URLs, JSONs), keeping the base initialism fully uppercase
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
	for _, token := range t.Tokenize(text) {
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
	for _, token := range t.Tokenize(text) {
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
	for _, token := range t.Tokenize(text) {
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
	return strings.Join(t.Tokenize(text), "_")
}

func (t *Tokenizer) Kebab(text string) string {
	return strings.Join(t.Tokenize(text), "-")
}

func (t *Tokenizer) Tokenize(text string) []string {
	return Tokenize(text, t.initialisms)
}

func uppercaseFirst(s string) string {
	if len(s) == 0 {
		return s
	}
	r := []rune(s)
	r[0] = unicode.ToUpper(r[0])
	return string(r)
}

func Tokenize(s string, initialisms []string) []string {
	// Replace all special characters with space.
	s = strings.Map(func(r rune) rune {
		if unicode.IsDigit(r) || unicode.IsLetter(r) {
			return r
		}
		return ' '
	}, s)
	var words []string
	for c := range strings.FieldsSeq(s) {
		for _, word := range segment(c, initialisms) {
			if slices.Contains(initialisms, word) {
				words = append(words, strings.ToLower(word))
				continue
			}
			bounds := splitBoundary(word)
			for _, b := range bounds {
				words = append(words, strings.ToLower(b))
			}
		}
	}
	return words
}

type item struct {
	score  int
	text   string
	before []string
	after  []string
}

func (i item) String() string {
	return fmt.Sprintf("%d:%s", i.score, strings.Join(append(append(i.before, i.text), i.after...), " "))
}

func (i item) Tokens() []string {
	result := append(i.before, i.text)
	result = append(result, i.after...)
	return strings.Fields(strings.Join(result, " "))
}

// segment scores the fragments, taking only the ones with the highest score.
func segment(s string, initialisms []string) []string {
	q := []item{{text: s}}

	var bestScore int = -math.MaxInt
	var bestResult []string
	seen := make(map[string]bool)
	for len(q) > 0 {
		var h item
		h, q = q[0], q[1:]
		if h.text == "" {
			result := h.Tokens()
			if seen[fmt.Sprint(result)] {
				continue
			}
			seen[fmt.Sprint(result)] = true
			for _, res := range result {
				if slices.Contains(initialisms, res) {
					h.score += 1
				}
			}
			if h.score > bestScore {
				bestScore = h.score
				bestResult = result
				continue
			}

			continue
		}
		for _, d := range initialisms {
			if after, ok := strings.CutPrefix(h.text, d); ok && isValidAfter(after) {
				q = append(q, item{
					score:  h.score + 1,
					text:   after,
					before: append(h.before, d),
					after:  h.after,
				})
			}
			if before, ok := strings.CutSuffix(h.text, d); ok {
				q = append(q, item{
					score:  h.score + 1,
					text:   before,
					before: h.before,
					after:  append([]string{d}, h.after...),
				})
			}
		}
		q = append(q, item{
			score:  h.score,
			before: append(h.before, h.text),
			after:  h.after,
		})
	}
	return bestResult
}

// isValid checks if the segment that follows the initialism is valid.
// it must start with uppercase or a digit.
func isValidAfter(s string) bool {
	r := []rune(s)
	if len(r) == 0 {
		return true
	}
	valid := unicode.IsUpper(r[0]) || unicode.IsDigit(r[0])
	if !valid {
		return false
	}
	if len(r) == 1 {
		return true
	}

	return unicode.IsLower(r[1]) || unicode.IsDigit(r[1])
}

// 1. digits followed by uppercase, e.g. 1990Year, spilt at 1990<split>Year
// 2. lowercase followed by uppercase, e.g. apiServer, split at api<split>Server
// 3. two uppercase followed by lowercase, e.g. HTTPSServer, split at HTTPS<split>Server.
// 4. an uppercase followed by digit, initialism doesn't mix with digits, e.g. UserID1, split at UserID<split>1
var re = regexp.MustCompile(`([0-9][A-Z]|[a-z][A-Z]|[A-Z][A-Z][a-z]|[A-Z][0-9])`)

func splitBoundary(s string) []string {
	s = re.ReplaceAllStringFunc(s, func(s string) string {
		return s[:1] + " " + s[1:]
	})
	return strings.Fields(s)
}
