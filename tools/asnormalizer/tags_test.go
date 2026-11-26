package asnormalizer

import (
	"testing"

	"github.com/benpate/hannibal/streams"
	"github.com/benpate/hannibal/vocab"
	"github.com/stretchr/testify/require"
)

// TestTags_InternalRelAnywhere drops a tag whose internal "--emissary-" rel is not its first rel
func TestTags_InternalRelAnywhere(t *testing.T) {

	document := streams.NewDocument([]any{
		map[string]any{vocab.PropertyType: vocab.LinkTypeHashtag, vocab.PropertyName: "#cats"},
		map[string]any{vocab.PropertyType: vocab.CoreTypeLink, vocab.PropertyHref: "https://remote.example/x", vocab.PropertyRel: []any{"alternate", "--emissary-internal"}},
	})

	result := Tags(document)

	require.Len(t, result, 1)
	require.Equal(t, "#cats", result[0][vocab.PropertyName])
}
