package model

import (
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/benpate/rosetta/schema"
	"github.com/stretchr/testify/require"
)

// TestContentSchema returns the rosetta schema that describes a TestContent
func TestContentSchema(t *testing.T) {

	content := NewHTMLContent("TEST")
	s := schema.New(ContentSchema())

	table := []tableTestItem{
		{"format", ContentFormatHTML, nil},
		{"html", "TEST-HTML", nil},
		{"raw", "TEST-RAW", nil},
	}

	tableTest_Schema(t, &s, &content, table)
}

// TestContentSchema_MaxLength_WithinLimit confirms that content at the ceiling validates cleanly.
func TestContentSchema_MaxLength_WithinLimit(t *testing.T) {

	s := schema.New(ContentSchema())

	body := strings.Repeat("a", ContentMaxLength)
	content := Content{Format: ContentFormatHTML, Raw: body, HTML: body}

	_, err := s.Validate(&content)
	require.Nil(t, err)
}

// TestContentSchema_MaxLength_RejectsOversize confirms that Validate rejects an oversized body.
func TestContentSchema_MaxLength_RejectsOversize(t *testing.T) {

	s := schema.New(ContentSchema())

	body := strings.Repeat("a", ContentMaxLength+1)
	content := Content{Format: ContentFormatHTML, Raw: body, HTML: body}

	_, err := s.Validate(&content)
	require.NotNil(t, err)
}

// TestContentSchema_MaxLength_NormalizeTruncates confirms that the save path (Normalize)
// clamps an oversized body to the ceiling instead of storing it verbatim.  This is the
// last-resort backstop for write paths that bypass the edit-content step (federation
// ingest, imports).
func TestContentSchema_MaxLength_NormalizeTruncates(t *testing.T) {

	s := schema.New(ContentSchema())

	body := strings.Repeat("a", ContentMaxLength+100)
	content := Content{Format: ContentFormatHTML, Raw: body, HTML: body}

	_, err := s.Normalize(&content)
	require.Nil(t, err)
	require.Equal(t, ContentMaxLength, utf8.RuneCountInString(content.Raw))
	require.Equal(t, ContentMaxLength, utf8.RuneCountInString(content.HTML))
}
