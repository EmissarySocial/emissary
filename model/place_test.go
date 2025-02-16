package model

import (
	"testing"

	"github.com/benpate/rosetta/schema"
)

func TestPlace(t *testing.T) {

	s := schema.New(PlaceSchema())
	response := NewPlace()

	tests := []tableTestItem{
		{"name", "NAME", nil},
		{"fullAddress", "ADDRESS", nil},
		// Removing these tests because they are read-only
		// {"street1", "STREET1", nil},
		// {"street2", "STREET2", nil},
		// {"locality", "LOCALITY", nil},
		// {"region", "REGION", nil},
		// {"postalCode", "POSTAL CODE", nil},
		// {"country", "COUNTRY", nil},
		// {"latitude", 1234.56, nil},
		// {"longitude", 7890.12, nil},
		// {"radius", 13.0, nil},
		// {"units", "mm", nil},
	}

	tableTest_Schema(t, &s, &response, tests)
}
