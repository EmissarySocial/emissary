package model

import (
	"testing"

	"github.com/benpate/rosetta/schema"
)

func TestCircleSchema(t *testing.T) {

	group := NewCircle()
	s := schema.New(CircleSchema())

	table := []tableTestItem{
		{"circleId", "5e5e5e5e5e5e5e5e5e5e5e5e", nil},
		{"userId", "123456781234567812345678", nil},
		{"name", "Name", nil},
		{"color", "#000000", nil},
		{"icon", "circle", nil},
		{"productIds.0", "123456789", nil},
		{"productIds.1", "987654321", nil},
		{"description", "Description", nil},
		{"isFeatured", true, nil},
	}

	tableTest_Schema(t, &s, &group, table)
}
