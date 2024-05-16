package model

import (
	"testing"

	"github.com/benpate/rosetta/schema"
)

func TestGroupSchema(t *testing.T) {

	group := NewGroup()
	s := schema.New(GroupSchema())

	table := []tableTestItem{
		{"groupId", "5e5e5e5e5e5e5e5e5e5e5e5e", nil},
		{"token", "professional", nil},
		{"label", "LABEL", nil},
	}

	tableTest_Schema(t, &s, &group, table)
}
