package model

import (
	"testing"

	"github.com/benpate/rosetta/schema"
)

// TestInboxActivitySchema returns the rosetta schema that describes a TestInboxActivity
func TestInboxActivitySchema(t *testing.T) {

	activity := NewInboxActivity()
	s := schema.New(InboxActivitySchema())

	table := []tableTestItem{
		{"inboxActivityId", "123456781234567812345678", nil},
		{"userId", "876543218765432187654321", nil},
		{"activityId", "https://example.com/activities/12345", nil},
		{"actorId", "https://example.com/users/alice", nil},
		{"objectId", "https://example.com/posts/12345", nil},
		{"activityType", "Create", nil},
		{"objectType", "Note", nil},
		{"mediaType", "message/mls", nil},
		{"publishedDate", int64(1625097600000), nil},
		{"receivedDate", int64(1625097600000), nil},
		{"isPublic", true, nil},
		{"rawActivity.type", "Create", nil},
		{"rawActivity.id", "https://example.com/activities/12345", nil},
	}

	tableTest_Schema(t, &s, &activity, table)
}
