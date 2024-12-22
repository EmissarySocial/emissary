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
		{"color", "#663399", nil},
		{"stateId", SearchTagStateAllowed, nil},
		{"notes", "NOTES", nil},
		{"rank", 1234, nil},
	}

	tableTest_Schema(t, &s, &searchTag, tests)
}
