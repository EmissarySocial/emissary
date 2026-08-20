package model

import (
	"testing"

	"github.com/benpate/rosetta/schema"
)

// TestOutboxItemSchema returns the rosetta schema that describes a TestOutboxItem
func TestOutboxItemSchema(t *testing.T) {

	annotation := NewOutboxItem()
	s := schema.New(OutboxItemSchema())

	table := []tableTestItem{
		{"activityId", "123456781234567812345678", nil},
		{"actorType", "User", nil},
		{"actorId", "876543218765432187654321", nil},
		{"recipients.0", "Public", nil},
		{"activity.to", "Public", nil},
		{"activity.id", "http://example.com/activities/1", nil},
		{"activity.type", "Note", nil},
		{"activity.published", "2024-01-01T12:00:00Z", nil},
		{"activity.attributedTo", "http://example.com/users/alice", nil},
		{"activity.content", "<p>Hello, world!</p>", nil},
		{"url", "https://example.com/outbox/1", nil},
	}

	tableTest_Schema(t, &s, &annotation, table)
}
