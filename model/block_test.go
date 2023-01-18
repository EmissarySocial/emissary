package model

import (
	"testing"

	"github.com/benpate/rosetta/schema"
)

func TestBlockSchema(t *testing.T) {

	block := NewBlock()
	s := schema.New(BlockSchema())

	table := []tableTestItem{
		{"blockId", "123456781234567812345678", nil},
		{"userId", "876543218765432187654321", nil},
		{"source", "INTERNAL", nil},
		{"type", "URL", nil},
		{"trigger", "TRIGGER", nil},
		{"comment", "COMMENT", nil},
		{"isPublic", "true", true},
		{"isActive", "true", true},
	}

	tableTest_Schema(t, &s, &block, table)
}
