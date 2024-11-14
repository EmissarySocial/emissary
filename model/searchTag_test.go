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
		{"parentId", "000000000000000000000002", nil},
		{"tag", "TAG", nil},
		{"stateId", SearchTagStateAllowed, nil},
		{"notes", "NOTES", nil},
		{"rank", 1234, nil},
	}

	tableTest_Schema(t, &s, &searchTag, tests)
}
