package batch_test

import (
	"fmt"

	"github.com/alextanhongpin/core/exp/batch"
)

type Product struct {
	ID            int
	CategoryIDs   []int
	SubCategories []SubCategory
}

type SubCategory struct {
	ID         int
	CategoryID int
	Meta       map[string]string
}

func ExampleManyToMany() {
	l := newSubCategoryLoader()

	// We have a bunch of subCategories, and we want to load the product.

	pdts := make([]Product, 5)
	for i := 0; i < len(pdts); i++ {
		pdts[i].ID = i

		for j := 0; j < i; j++ {
			pdts[i].CategoryIDs = append(pdts[i].CategoryIDs, j+1)
		}

		// Load subcategories by category id.
		if err := l.LoadMany(&pdts[i].SubCategories, pdts[i].CategoryIDs...); err != nil {
			panic(err)
		}
	}

	// Initiate the fetch.
	if err := l.Wait(); err != nil {
		panic(err)
	}

	for i := range pdts {
		fmt.Printf("product %d has %d subCategories\n", pdts[i].ID, len(pdts[i].SubCategories))
	}
	// Output:
	// product 0 has 0 subCategories
	// product 1 has 1 subCategories
	// product 2 has 3 subCategories
	// product 3 has 6 subCategories
	// product 4 has 10 subCategories
}

func newSubCategoryLoader() *batch.Loader[int, SubCategory] {
	batchFn := func(categoryIds ...int) ([]SubCategory, error) {
		var subCategories []SubCategory
		for _, id := range categoryIds {
			// The number of subCategories is proportional to the CategoryID.
			for i := 0; i < id; i++ {
				subCategories = append(subCategories, SubCategory{
					ID:         id*100 + i,
					CategoryID: id,
					Meta:       make(map[string]string),
				})
			}
		}

		return subCategories, nil
	}

	keyFn := func(s SubCategory) (categoryID int, err error) {
		categoryID = s.CategoryID
		return
	}

	return batch.New(batch.Option[int, SubCategory]{
		BatchFn: batchFn,
		KeyFn:   keyFn,
	})
}
