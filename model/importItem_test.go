package model

import (
	"testing"

	"github.com/benpate/rosetta/schema"
)

func TestImportItemSchema(t *testing.T) {

	group := NewImportItem()
	s := schema.New(ImportItemSchema())

	table := []tableTestItem{
		{"importItemId", "5e5e5e5e5e5e5e5e5e5e5e5e", nil},
		{"importId", "5e5e5e5e5e5e5e5e5e5e5e5e", nil},
		{"userId", "5e5e5e5e5e5e5e5e5e5e5e5b", nil},
		{"localId", "5e5e5e5e5e5e5e5e5e5e5e5e", nil},
		{"type", "Stream", nil},
		{"url", "http://test.com/", nil},
		{"stateId", "AUTHORIZING", nil},
		{"message", "does eat oats", nil},
	}

	tableTest_Schema(t, &s, &group, table)
}
