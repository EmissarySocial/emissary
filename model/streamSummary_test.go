package model

import (
	"testing"

	"github.com/benpate/hannibal/vocab"
	"github.com/stretchr/testify/require"
)

// TestStreamSummaryFields guards the two tag projections. The dangling-name check that covers
// every model in this package lives in TestFieldProjections.
func TestStreamSummaryFields(t *testing.T) {

	fields := StreamSummaryFields()

	require.Contains(t, fields, "tags", "Tags would silently load as empty without this")
	require.Contains(t, fields, "hashtags", "still dual-written; see projects/TAGS-UNIFICATION.md")
}

// TestStreamSummary_TagsRoundTrip confirms the Tags field carries mixed tag types intact, and that
// the model helpers work against a summary exactly as they do against a full Stream.
func TestStreamSummary_TagsRoundTrip(t *testing.T) {

	summary := NewStreamSummary()
	summary.Tags = TagList{
		NewTag(vocab.LinkTypeHashtag, "travel"),
		{Type: vocab.LinkTypeMention, Name: "bob@server.social", Href: "https://server.social/@bob"},
	}

	require.Equal(t, []string{"travel"}, []string(TagNames(summary.Tags, vocab.LinkTypeHashtag)))
	require.Equal(t, []string{"bob@server.social"}, []string(TagNames(summary.Tags, vocab.LinkTypeMention)))
	require.Equal(t, "https://server.social/@bob", TagsOfType(summary.Tags, vocab.LinkTypeMention)[0].Href)
}
