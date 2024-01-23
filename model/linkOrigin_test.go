package model

import (
	"testing"

	"github.com/benpate/rosetta/schema"
)

func TestOriginLink(t *testing.T) {

	origin := NewOriginLink()

	s := schema.New(OriginLinkSchema())

	table := []tableTestItem{
		{"type", "PRIMARY", nil},
		{"followingId", "123412341234123412341234", nil},
		{"label", "TEST-LABEL", nil},
		{"url", "https://test.url", nil},
		{"imageUrl", "https://test.image.url", nil},
	}

	tableTest_Schema(t, &s, &origin, table)
}
