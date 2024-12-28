package model

import (
	"testing"

	"github.com/benpate/rosetta/schema"
)

func TestSearchTag(t *testing.T) {

	s := schema.New(SearchTagSchema())
	searchTag := NewSearchTag()

	tests := []tableTestItem{
		{"searchTagId", "000000000000000000000001", nil},
		{"parent", "YOURTAG", nil},
		{"name", "MYTAG", nil},
		{"description", "DESCRIPTION", nil},
		{"colors.01", "#663399", nil},
		{"colors.02", "#AABBCC", nil},
		{"stateId", SearchTagStateAllowed, nil},
		{"notes", "NOTES", nil},
		{"rank", 1234, nil},
	}

	tableTest_Schema(t, &s, &searchTag, tests)
}
