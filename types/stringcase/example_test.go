package stringcase_test

import (
	"fmt"

	"github.com/alextanhongpin/core/types/stringcase"
)

func Example() {
	fmt.Println(stringcase.ToSnake("userIDsAPIURL"))
	fmt.Println(stringcase.ToCamel("user_id"))
	fmt.Println(stringcase.ToPascal("api-response"))
	fmt.Println(stringcase.ToTitle("httpServer"))

	t := stringcase.NewTokenizer("SKU")
	fmt.Println(t.Pascal("product_sku"))
	// Output:
	// user_ids_api_url
	// userID
	// APIResponse
	// HTTP Server
	// ProductSKU
}
