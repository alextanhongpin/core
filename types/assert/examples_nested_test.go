package assert_test

import (
	"encoding/json"
	"fmt"

	"github.com/alextanhongpin/core/types/assert"
)

var (
	optional = assert.Optional
	required = assert.Required
	is       = assert.Is
)

func ExampleIs() {
	var req CreateOrderRequest
	req.Discount = -1
	req.LineItems = append(req.LineItems, LineItemRequest{Quantity: -1}, LineItemRequest{})
	b, err := json.MarshalIndent(req.Valid(), "", " ")
	if err != nil {
		panic(err)
	}
	fmt.Println(string(b))

	// Output:
	// {
	//  "discount": "must be greater than 0",
	//  "lineItems[0].price": "required, must be greater than 0",
	//  "lineItems[0].productId": "required",
	//  "lineItems[0].quantity": "must be greater than 0",
	//  "lineItems[1].price": "required, must be greater than 0",
	//  "lineItems[1].productId": "required",
	//  "totalPrice": "required, must be greater than 0"
	// }
}

type CreateOrderRequest struct {
	Discount   int64
	LineItems  []LineItemRequest
	TotalPrice int64
}

func (req *CreateOrderRequest) Valid() map[string]string {
	res := map[string]string{
		"discount":   validateDiscount(req.Discount),
		"totalPrice": validatePrice(req.TotalPrice),
		"lineItems":  required(len(req.LineItems)),
	}

	for i, item := range req.LineItems {
		for k, v := range item.Valid() {
			res[fmt.Sprintf("lineItems[%d].%s", i, k)] = v
		}
	}

	return assert.Map(res)
}

type LineItemRequest struct {
	Price     int64
	ProductID string
	Quantity  int64
}

func (req *LineItemRequest) Valid() map[string]string {
	return assert.Map(map[string]string{
		"price":     validatePrice(req.Price),
		"productId": required(req.ProductID),
		"quantity":  validateQuantity(req.Quantity),
	})
}

func validatePrice(n int64) string {
	return required(n,
		is(n > 0, "must be greater than 0"))
}

func validateDiscount(n int64) string {
	return optional(n,
		is(n > 0, "must be greater than 0"),
		is(n <= 100, "maximum discount is 100%"),
	)
}

func validateQuantity(n int64) string {
	return optional(n,
		is(n > 0, "must be greater than 0"))
}
