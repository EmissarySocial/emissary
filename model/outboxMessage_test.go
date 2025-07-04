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
		{"actorType", "User", nil},
		{"actorId", "000000000000000000000001", nil},
		{"activityType", "Create", nil},
		{"objectId", "https://john.connor.mil", nil},
		{"permissions.0", "086753090867530908675309", nil},
		{"permissions.1", "086753090867530908675309", nil},
	}

	tableTest_Schema(t, &s, &response, tests)
}
