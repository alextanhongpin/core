package testutil

import (
	"fmt"
	"go/format"
	"os"
	"strings"
	"testing"
	"unicode"

	"github.com/google/go-cmp/cmp"
)

func countLeadingSpace(s string) int {
	return len(s) - len(strings.TrimSpace(s))
}

func DumpStruct(t *testing.T, v any, opts ...cmp.Option) {
	t.Helper()

	got := []byte(prettyStruct(v))

	fileName := fmt.Sprintf("./testdata/%s.struct", t.Name())
	if err := writeToNewFile(fileName, got); err != nil {
		t.Fatal(err)
	}
	want, err := os.ReadFile(fileName)
	if err != nil {
		t.Fatal(err)
	}

	// NOTE: The want and got is reversed here.
	if diff := cmp.Diff(got, want, opts...); diff != "" {
		t.Fatal(diffError(diff))
	}

	return
}

func prettyStruct(v any) string {
	s := fmt.Sprintf("%#v", v)
	b, err := format.Source([]byte(s))
	if err != nil {
		panic(err)
	}

	s = string(b)
	var res []string
	push := func(s string) {
		res = append(res, strings.TrimSuffix(s, " "))
	}

	sb := new(strings.Builder)

	for i, r := range s {
		h := i - 1
		j := i + 1
		k := i + 2
		if i == len(s)-1 {
			j = i
			k = i
		} else if i == 0 {
			h = i
		}
		nextIsDigit := unicode.IsDigit(rune(s[j]))
		nextIsLetter := unicode.IsLetter(rune(s[j]))
		nextIsQuote := s[j] == '"'
		nextIsArray := s[j] == '[' && s[k] == ']'

		if r == '{' && s[j] != '}' { // Skip if map[string]interface{}
			sb.WriteRune(r)
			push(sb.String())
			sb.Reset()

			var n int
			if len(res) > 0 {
				last := res[len(res)-1]
				n = countLeadingSpace(last)
			}
			n += 2
			sb.WriteString(strings.Repeat(" ", n))
		} else if r == '}' && s[h] != '{' {
			push(sb.String())
			sb.Reset()

			var n int
			if len(res) > 0 {
				last := res[len(res)-1]
				n = countLeadingSpace(last)
			}
			n -= 2
			sb.WriteString(strings.Repeat(" ", n))
			sb.WriteRune(r)
		} else if r == ',' && s[j] == ' ' {
			// New line.
			sb.WriteRune(r)
			push(sb.String())
			sb.Reset()

			var n int
			if len(res) > 0 {
				last := res[len(res)-1]
				n = countLeadingSpace(last)
			}
			n--
			sb.WriteString(strings.Repeat(" ", n))
		} else if r == ':' && (nextIsDigit || nextIsLetter || nextIsArray || nextIsQuote) { // To avoid matching http://
			sb.WriteRune(r)
			sb.WriteRune(' ') // Add a space after colon.
		} else {
			sb.WriteRune(r)
		}
	}
	push(sb.String())
	return strings.Join(res, "\n")
}
