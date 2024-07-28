package validator_test

import (
	"fmt"

	"github.com/alextanhongpin/core/validator"
)

var (
	urlField   = validator.StringExpr("required,url,min=3")
	linksField = validator.SliceExpr[Link]("required,min=2").EachFunc((Link).Valid)
)

type Link struct {
	URL string
}

func (l Link) Valid() error {
	return validator.NewErrors(map[string]error{
		"url": urlField.Validate(l.URL),
	})
}

type Page struct {
	Links []Link
}

func (p *Page) Valid() error {
	return validator.NewErrors(map[string]error{
		"links": linksField.Validate(p.Links),
	})
}

func ExampleSliceExpr() {

	invalid := &Page{Links: []Link{
		{"http://localhost 8080"},
		{"456"},
	}}
	less := &Page{Links: []Link{
		{"http://localhost"},
	}}
	valid := &Page{Links: []Link{
		{"http://localhost"},
		{"http://localhost:8080"},
	}}

	fmt.Printf("%v => %v\n", invalid.Links, invalid.Valid())
	fmt.Printf("%v => %v\n", less.Links, less.Valid())
	fmt.Printf("%v => %v\n", valid.Links, valid.Valid())
	// Output:
	// [{http://localhost 8080} {456}] => links: url: invalid url
	// [{http://localhost}] => links: min 2 items
	// [{http://localhost} {http://localhost:8080}] => <nil>
}
