package model

import (
	"testing"

	"github.com/benpate/rosetta/schema"
)

func TestOutboxMessage(t *testing.T) {

	s := schema.New(OutboxMessageSchema())
	response := NewOutboxMessage()

	tests := []tableTestItem{
		{"outboxMessageId", "000000000000000000000001", nil},
		{"parentType", "User", nil},
		{"parentId", "000000000000000000000001", nil},
		{"activityType", "Create", nil},
		{"url", "https://john.connor.mil", nil},
	}

	tableTest_Schema(t, &s, &response, tests)
}
