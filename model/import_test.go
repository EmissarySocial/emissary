package model

import (
	"testing"

	"github.com/benpate/rosetta/schema"
)

func TestImportSchema(t *testing.T) {

	group := NewImport()
	s := schema.New(ImportSchema())

	table := []tableTestItem{
		{"importId", "5e5e5e5e5e5e5e5e5e5e5e5e", nil},
		{"userId", "5e5e5e5e5e5e5e5e5e5e5e5b", nil},
		{"stateId", "AUTHORIZING", nil},
		{"sourceId", "@source@source.social", nil},
		{"sourceUrl", "http://source.com/sourceurl", nil},
		{"message", "does eat oats", nil},
		{"totalItems", 7, nil},
		{"completeItems", 5, nil},
	}

	tableTest_Schema(t, &s, &group, table)
}
