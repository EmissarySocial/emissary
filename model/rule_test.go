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
		{"followingId", "876543218765432187654321", nil},
		{"followingLabel", "Hoo boy", nil},
		{"type", "ACTOR", nil},
		{"action", "LABEL", nil},
		{"label", "LABEL", nil},
		{"trigger", "TRIGGER", nil},
		{"summary", "COMMENT", nil},
		{"isPublic", "true", true},
		{"publishDate", int64(1234567890), nil},
	}

	tableTest_Schema(t, &s, &block, table)
}

func TestRule_FilterByActorEmail(t *testing.T) {

	block := Rule{
		Type:    RuleTypeActor,
		Trigger: "john@connor.com",
	}

	require.True(t, block.FilterByActor("john@connor.com"))
	require.True(t, block.FilterByActor("John Connor <john@connor.com>"))
	require.False(t, block.FilterByActor("sara@sky.net"))
}

func TestRule_FilterByActorURI(t *testing.T) {

	block := Rule{
		Type:    RuleTypeActor,
		Trigger: "https://connor.com/@john",
	}

	require.True(t, block.FilterByActor("https://connor.com/@john"))
	require.False(t, block.FilterByActor("https://sky.net/@sarah"))
}

func TestRule_FilterByDomain(t *testing.T) {

	block := Rule{
		Type:    RuleTypeDomain,
		Trigger: "connor.com",
	}

	require.True(t, block.FilterByActor("john@connor.com"))
	require.True(t, block.FilterByActor("John Connor <john@connor.com>"))
	require.True(t, block.FilterByActor("https://connor.com/@john"))
	require.True(t, block.FilterByActor("https://john.connor.com"))
	require.False(t, block.FilterByActor("sara@sky.net"))
	require.False(t, block.FilterByActor("https://sky.net/@sarah"))
}
