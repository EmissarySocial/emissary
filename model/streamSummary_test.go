package model

import (
	"reflect"
	"strings"
	"testing"

	"github.com/benpate/hannibal/vocab"
	"github.com/stretchr/testify/require"
)

// streamSummaryBSONNames returns the bson name of every field on StreamSummary.
func streamSummaryBSONNames(t *testing.T) []string {

	t.Helper()

	structType := reflect.TypeOf(StreamSummary{})
	result := make([]string, 0, structType.NumField())

	for index := range structType.NumField() {

		tag := structType.Field(index).Tag.Get("bson")

		if name, _, _ := strings.Cut(tag, ","); name != "" {
			result = append(result, name)
		}
	}

	return result
}

// TestStreamSummaryFields guards the Mongo projection list. StreamSummary has no schema and no
// GetPointer -- it is loaded straight from the database using Fields() -- so a name that is missing,
// or that no longer matches a struct field, produces no compile or runtime error anywhere. The
// field just silently arrives empty.
func TestStreamSummaryFields(t *testing.T) {

	fields := StreamSummaryFields()

	require.Contains(t, fields, "tags", "Tags would silently load as empty without this")
	require.Contains(t, fields, "hashtags", "still dual-written; see projects/TAGS-UNIFICATION.md")

	// EVERY projected name must exist on the struct. A name that does not asks Mongo for a field
	// that can never be unmarshalled, and the field it was meant to name silently loads as its
	// zero value -- which is exactly how "places" outlived the rename to "location".
	require.Subset(t, streamSummaryBSONNames(t), fields)

	// Both halves of that rename, pinned.
	require.Contains(t, fields, "location")
	require.NotContains(t, fields, "places", "renamed to location; see queries/upgrades/v020.go")
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
