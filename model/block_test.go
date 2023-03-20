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
		{"type", "ACTOR", nil},
		{"trigger", "TRIGGER", nil},
		{"behavior", "BLOCK", nil},
		{"comment", "COMMENT", nil},
		{"isPublic", "true", true},
		{"origin.internalId", "123456781234567812345678", nil},
		{"origin.type", "INTERNAL", nil},
		{"origin.url", "https://example.com", nil},
		{"origin.label", "LABEL", nil},
		{"origin.summary", "SUMMARY", nil},
		{"origin.imageUrl", "ICON.URL.HERE", nil},
	}

	tableTest_Schema(t, &s, &block, table)
}
