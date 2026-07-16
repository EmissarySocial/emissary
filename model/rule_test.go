package model

import (
	"testing"

	"github.com/benpate/rosetta/schema"
	"github.com/stretchr/testify/require"
)

func TestRuleSchema(t *testing.T) {

	block := NewRule()
	s := schema.New(RuleSchema())

	table := []tableTestItem{
		{"ruleId", "123456781234567812345678", nil},
		{"userId", "876543218765432187654321", nil},
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

// TestRuleSchema_NoForgeableFields confirms the two machinery fields cannot be written through the
// schema: `followingId` (which would forge a Rule's origin -- D9) and `matchKey` (which would let a
// form desync the match key from the Trigger). Both are set only by trusted server code.
func TestRuleSchema_NoForgeableFields(t *testing.T) {

	rule := NewRule()
	s := schema.New(RuleSchema())

	require.Error(t, s.Set(&rule, "followingId", "876543218765432187654321"))
	require.True(t, rule.FollowingID.IsZero(), "followingId must not be settable via the schema")

	require.Error(t, s.Set(&rule, "matchKey", "ACTOR:https://evil.example/@forged"))
	require.Equal(t, "", rule.MatchKey, "matchKey must not be settable via the schema")
}
