package model

import (
	"testing"

	"github.com/benpate/rosetta/schema"
)

func TestOutboxMessageSchema(t *testing.T) {

	activity := NewOutboxMessage()
	s := schema.New(OutboxMessageSchema())

	table := []tableTestItem{
		{"outboxMessageId", "123456781234567812345678", nil},
		{"objectType", "test", nil},
		{"objectId", "123456781234567812345678", nil},
		{"userId", "876543218765432187654321", nil},
		{"parentId", "123456781234567812345678", nil},
		{"rank", "123", int64(123)},
	}

	tableTest_Schema(t, &s, &activity, table)
}
