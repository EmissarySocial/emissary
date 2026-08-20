package model

import (
	"testing"

	"github.com/benpate/rosetta/schema"
)

// TestImportItemSchema returns the rosetta schema that describes a TestImportItem
func TestImportItemSchema(t *testing.T) {

	importItem := NewImportItem()
	s := schema.New(ImportItemSchema())

	table := []tableTestItem{
		{"importItemId", "5e5e5e5e5e5e5e5e5e5e5e5e", nil},
		{"importId", "5e5e5e5e5e5e5e5e5e5e5e5e", nil},
		{"userId", "5e5e5e5e5e5e5e5e5e5e5e5b", nil},
		{"localId", "5e5e5e5e5e5e5e5e5e5e5e5e", nil},
		{"type", "Stream", nil},
		{"importUrl", "http://test.com/", nil},
		{"remoteUrl", "http://test.com/", nil},
		{"localUrl", "http://test.com/", nil},
		{"stateId", ImportItemStateDone, nil},
		{"message", "does eat oats", nil},
	}

	tableTest_Schema(t, &s, &importItem, table)
}
