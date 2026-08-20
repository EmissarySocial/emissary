package model

import (
	"testing"

	"github.com/benpate/rosetta/schema"
)

// TestObjectLink verifies that every ObjectLink property round-trips through the schema
func TestObjectLink(t *testing.T) {

	s := schema.New(ObjectLinkSchema())
	objectLink := NewObjectLink()

	tests := []tableTestItem{
		{"objectLinkId", "000000000000000000000001", nil},
		{"context", "http://example.com/context", nil},
		{"inReplyTo", "http://example.com/inReplyTo", nil},
		{"recipients.0", "http://example.com/recipient", nil},
		{"recipients.1", "http://example.com/other-recipient", nil},
		{"object", "https://example/object", nil},
	}

	tableTest_Schema(t, &s, &objectLink, tests)
}
