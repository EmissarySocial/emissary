package model

import (
	"testing"

	"github.com/benpate/rosetta/schema"
)

func TestRuleSchema(t *testing.T) {

	block := NewRule()
	s := schema.New(RuleSchema())

	table := []tableTestItem{
		{"ruleId", "123456781234567812345678", nil},
		{"userId", "876543218765432187654321", nil},
		{"followingId", "876543218765432187654321", nil},
		{"followingLabel", "Hoo boy", nil},
		{"type", "ACTOR", nil},
		{"action", "LABEL", nil},
		{"label", "LABEL", nil},
		{"trigger", "TRIGGER", nil},
		{"summary", "COMMENT", nil},
		{"isPublic", "true", true},
		{"publishDate", int64(1234567890), nil},
		{"reasonCode", "SPAM", nil},
	}

	tableTest_Schema(t, &s, &block, table)
}
