package model

import (
	"testing"

	"github.com/benpate/rosetta/schema"
	"github.com/stretchr/testify/require"
)

func TestContentSchema(t *testing.T) {

	content := NewHTMLContent("TEST")
	s := schema.New(ContentSchema())

	table := []tableTestItem{
		{"format", "TEST-FORMAT", nil},
		{"html", "TEST-HTML", nil},
		{"raw", "TEST-RAW", nil},
	}

	tableTest_Schema(t, &s, &content, table)

	require.Error(t, s.Validate(&content))

	s.Set(&content, "format", ContentFormatHTML)
	require.Nil(t, s.Validate(&content))
}
