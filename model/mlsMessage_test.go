package model

import (
	"testing"

	"github.com/benpate/rosetta/schema"
)

func TestMLSMessageSchema(t *testing.T) {

	activity := NewMLSMessage()
	s := schema.New(MLSMessageSchema())

	table := []tableTestItem{
		{"mlsMessageId", "123456781234567812345678", nil},
		{"userId", "876543218765432187654321", nil},
		{"type", "mls:Welcome", nil},
		{"content", "base-64-encrypted", nil},
	}

	tableTest_Schema(t, &s, &activity, table)
}
