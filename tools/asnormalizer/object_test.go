package asnormalizer

import (
	"testing"

	"github.com/benpate/hannibal/streams"
	"github.com/stretchr/testify/require"
)

func hashtagNote() map[string]any {
	return map[string]any{
		"type":    "Note",
		"id":      "https://example.com/notes/1",
		"content": "hello #cats",
		"tag": []any{
			map[string]any{"type": "Hashtag", "href": "https://example.com/tags/cats", "name": "#cats"},
		},
	}
}

// A post's tags belong to the post itself, so they must survive when the post
// arrives wrapped in a Create.
func TestObject_KeepsTagsOfWrappedPost(t *testing.T) {

	wrapper := streams.NewDocument(map[string]any{
		"type":   "Create",
		"actor":  "https://example.com/users/alice",
		"object": hashtagNote(),
	})

	tags, ok := Object(nil, wrapper)["tag"].([]map[string]any)

	require.True(t, ok)
	require.Len(t, tags, 1)
	require.Equal(t, "Hashtag", tags[0]["type"])
	require.Equal(t, "#cats", tags[0]["name"])
	require.Equal(t, "https://example.com/tags/cats", tags[0]["href"])
}

func TestObject_KeepsTagsOfUnwrappedPost(t *testing.T) {

	tags, ok := Object(nil, streams.NewDocument(hashtagNote()))["tag"].([]map[string]any)

	require.True(t, ok)
	require.Len(t, tags, 1)
}
