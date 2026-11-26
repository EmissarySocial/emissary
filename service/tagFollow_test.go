package service

import (
	"testing"

	"github.com/benpate/hannibal/streams"
	"github.com/benpate/hannibal/vocab"
	"github.com/stretchr/testify/require"
)

// TestTagFollowName strips whitespace and the leading # from a tag
func TestTagFollowName(t *testing.T) {
	require.Equal(t, "Cats", TagFollowName("  #Cats "))
	require.Equal(t, "cats", TagFollowName("cats"))
	require.Equal(t, "", TagFollowName("#"))
}

// TestDocumentTagValues reads only Hashtag tags, normalized, skipping mentions and bare strings
func TestDocumentTagValues(t *testing.T) {

	document := streams.NewDocument(map[string]any{
		vocab.PropertyID: "https://remote.example/notes/1",
		vocab.PropertyTag: []any{
			map[string]any{vocab.PropertyType: vocab.LinkTypeHashtag, vocab.PropertyName: "#Cats"},
			map[string]any{vocab.PropertyType: vocab.LinkTypeMention, vocab.PropertyName: "@alice@remote.example"},
			map[string]any{vocab.PropertyType: vocab.LinkTypeHashtag, vocab.PropertyName: "#"},
			"https://remote.example/tags/dogs",
			map[string]any{vocab.PropertyType: vocab.LinkTypeHashtag, vocab.PropertyName: "#Playstation"},
		},
	})

	require.Equal(t, []string{"cats", "playstation"}, DocumentTagValues(document))
}

// TestIsPublicPost accepts only public posts, whether cached (a "visibility" word) or raw (to/cc)
func TestIsPublicPost(t *testing.T) {

	post := func(values map[string]any) streams.Document {
		return streams.NewDocument(values)
	}

	require.True(t, IsPublicPost(post(map[string]any{"visibility": "public"})), "cached public")
	require.False(t, IsPublicPost(post(map[string]any{"visibility": "unlisted"})), "cached unlisted")
	require.True(t, IsPublicPost(post(map[string]any{vocab.PropertyTo: vocab.NamespaceActivityStreamsPublic})), "raw public")
	require.False(t, IsPublicPost(post(map[string]any{vocab.PropertyCC: vocab.NamespaceActivityStreamsPublic})), "raw unlisted")
	require.False(t, IsPublicPost(post(map[string]any{vocab.PropertyTo: "https://remote.example/users/alice/followers"})), "raw followers-only")
	require.False(t, IsPublicPost(post(map[string]any{})), "no audience")
}
