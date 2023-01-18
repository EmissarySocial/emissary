package model

import (
	"testing"

	"github.com/benpate/rosetta/schema"
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
}
