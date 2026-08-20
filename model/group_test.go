package model

import (
	"testing"

	"github.com/benpate/rosetta/schema"
)

// TestGroupSchema returns the rosetta schema that describes a TestGroup
func TestGroupSchema(t *testing.T) {

	group := NewGroup()
	s := schema.New(GroupSchema())

	table := []tableTestItem{
		{"groupId", "5e5e5e5e5e5e5e5e5e5e5e5e", nil},
		{"token", "professional", nil},
		{"label", "LABEL", nil},
		{"description", "This is a description of the group.", nil},
		{"icon", "people", nil},
	}

	tableTest_Schema(t, &s, &group, table)
}
