package model

import (
	"testing"

	"github.com/benpate/rosetta/schema"
)

// TestSearchTag verifies that every SearchTag property round-trips through the schema
func TestSearchTag(t *testing.T) {

	s := schema.New(SearchTagSchema())
	searchTag := NewSearchTag()

	tests := []tableTestItem{
		{"searchTagId", "000000000000000000000001", nil},
		{"group", "GENRE", nil},
		{"name", "MYTAG", nil},
		{"colors.01", "#663399", nil},
		{"colors.02", "#AABBCC", nil},
		{"stateId", SearchTagStateAllowed, nil},
		{"related", "YOURTAG", nil},
		{"notes", "NOTES", nil},
		{"rank", 1234, nil},
		{"imageId", "086753090867530908675309", nil},
	}

	tableTest_Schema(t, &s, &searchTag, tests)
}
