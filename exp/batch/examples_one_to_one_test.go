package batch_test

import (
	"fmt"
	"sort"

	"github.com/alextanhongpin/core/exp/batch"
)

type Order struct {
	ID       int
	Shipment Shipment
}

type Shipment struct {
	ID      int
	OrderID int
}

func ExampleOneToOne() {
	l := newShipmentLoader()

	// We have a bunch of orders, and we want to load the author.
	orders := []Order{
		{ID: 1},
		{ID: 2}, // Same author as Book ID 1.
		{ID: 3},
	}

	for i := 0; i < len(orders); i++ {
		if err := l.Load(&orders[i].Shipment, orders[i].ID); err != nil {
			panic(err)
		}
	}

	if err := l.Wait(); err != nil {
		panic(err)
	}

	for _, o := range orders {
		fmt.Printf("%#v\n", o)
	}

	// Output:
	// batch_test.Order{ID:1, Shipment:batch_test.Shipment{ID:0, OrderID:1}}
	// batch_test.Order{ID:2, Shipment:batch_test.Shipment{ID:1, OrderID:2}}
	// batch_test.Order{ID:3, Shipment:batch_test.Shipment{ID:2, OrderID:3}}
}

func newShipmentLoader() *batch.Loader[int, Shipment] {
	batchFn := func(orderIds ...int) ([]Shipment, error) {
		// Sort for repeatability.
		sort.Ints(orderIds)

		shipments := make([]Shipment, len(orderIds))
		for i, id := range orderIds {
			shipments[i] = Shipment{
				ID:      i,
				OrderID: id,
			}
		}
		return shipments, nil
	}

	keyFn := func(a Shipment) (orderID int, err error) {
		orderID = a.OrderID
		return
	}

	return batch.New(batch.Option[int, Shipment]{
		BatchFn: batchFn,
		KeyFn:   keyFn,
	})
}
