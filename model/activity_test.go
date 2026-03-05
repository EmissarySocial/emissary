package model

import (
	"testing"

	"github.com/benpate/rosetta/schema"
)

func TestActivitySchema(t *testing.T) {

	annotation := NewActivity()
	s := schema.New(ActivitySchema())

	table := []tableTestItem{
		{"activityId", "123456781234567812345678", nil},
		{"actorType", "User", nil},
		{"actorId", "876543218765432187654321", nil},
		{"recipients.0", "as:Public", nil},
		{"object.to", "as:Public", nil},
		{"object.id", "http://example.com/activities/1", nil},
		{"object.type", "Note", nil},
		{"object.published", "2024-01-01T12:00:00Z", nil},
		{"object.attributedTo", "http://example.com/users/alice", nil},
		{"object.content", "<p>Hello, world!</p>", nil},
	}

	tableTest_Schema(t, &s, &annotation, table)
}
