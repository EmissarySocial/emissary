package model

import (
	"testing"

	"github.com/benpate/rosetta/schema"
	"github.com/stretchr/testify/require"
)

// TestOriginLink verifies that every OriginLink property round-trips through the schema
func TestOriginLink(t *testing.T) {

	origin := NewOriginLink()

	s := schema.New(OriginLinkSchema())

	table := []tableTestItem{
		{"type", "PRIMARY", nil},
		{"followingId", "123412341234123412341234", nil},
		{"label", "TEST-LABEL", nil},
		{"url", "https://test.url", nil},
		{"iconUrl", "https://test.image.url", nil},
	}

	tableTest_Schema(t, &s, &origin, table)
}

// TestOriginLink_Hashtag verifies that a tag-follow origin passes the schema
func TestOriginLink_Hashtag(t *testing.T) {

	origin := NewOriginLink()
	origin.Type = OriginTypeHashtag
	origin.URL = "https://local.example/tags/Cats"
	origin.Label = "#Cats"

	s := schema.New(OriginLinkSchema())
	_, err := s.Validate(&origin)
	require.NoError(t, err)
	require.Equal(t, "hash", origin.Icon())
}
