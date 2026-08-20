package model

import (
	"testing"

	"github.com/benpate/rosetta/schema"
)

// TestProductSchema returns the rosetta schema that describes a TestProduct
func TestProductSchema(t *testing.T) {

	product := NewProduct()
	s := schema.New(ProductSchema())

	table := []tableTestItem{
		{"productId", "123456781234567812345678", nil},
		{"userId", "876543218765432187654321", nil},
		{"merchantAccountId", "111111111111111111111111", nil},
		{"remoteId", "REMOTE-ID", nil},
		{"name", "Premium Membership", nil},
		{"price", "$9.99", nil},
		{"icon", "star", nil},
		{"adminHref", "https://example.com/admin/product/1", nil},
	}

	tableTest_Schema(t, &s, &product, table)
}
