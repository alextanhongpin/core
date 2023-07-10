package sqldump

import (
	"regexp"
	"strings"
)

var placePat = regexp.MustCompile(`\$\d+`)

type Var struct {
	Name  string
	Value string
}

func PostgresVars(normalized, original string) []Var {
	a := normalized
	b := original

	var res []Var

	placeholders := placePat.FindAllString(a, -1)

	for i, p := range placeholders {
		in := strings.Index(a, p)
		var next int
		if i == len(placeholders)-1 {
			next = len(a)
		} else {
			next = strings.Index(a, placeholders[i+1])
		}

		j := strings.Index(b[in:], a[in+len(p):next])
		if j == -1 {
			break
		}

		res = append(res, Var{
			Name:  p,
			Value: b[in : in+j],
		})
		a = a[in+len(p):]
		b = b[in+j:]
	}

	return res
}
