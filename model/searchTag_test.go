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
		{"group", "GENRE", nil},
		{"name", "MYTAG", nil},
		{"colors.01", "#663399", nil},
		{"colors.02", "#AABBCC", nil},
		{"stateId", SearchTagStateAllowed, nil},
		{"related", "YOURTAG", nil},
		{"notes", "NOTES", nil},
		{"rank", 1234, nil},
	}

	tableTest_Schema(t, &s, &searchTag, tests)
}
