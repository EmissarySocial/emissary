package mastodon

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// newActorClient returns a client serving the given documents and an actor whose featured address is set.
func newActorClient(featured any, documents map[string]map[string]any) *fakeClient {

	actor := map[string]any{"id": "https://a.example.com/users/ben", "type": "Person"}

	if featured != nil {
		actor["featured"] = featured
	}

	documents["https://a.example.com/users/ben"] = actor
	return &fakeClient{documents: documents}
}

// TestFeaturedPostURLs_ReadsEmbeddedPosts covers Mastodon's shape: a featured address whose collection holds the full posts.
func TestFeaturedPostURLs_ReadsEmbeddedPosts(t *testing.T) {

	client := newActorClient("https://a.example.com/users/ben/collections/featured", map[string]map[string]any{
		"https://a.example.com/users/ben/collections/featured": {
			"id": "https://a.example.com/users/ben/collections/featured", "type": "OrderedCollection", "totalItems": 2,
			"orderedItems": []any{
				map[string]any{"id": "https://a.example.com/users/ben/statuses/1", "type": "Note", "content": "pinned one"},
				map[string]any{"id": "https://a.example.com/users/ben/statuses/2", "type": "Note", "content": "pinned two"},
			},
		},
	})

	actor, err := client.Load("https://a.example.com/users/ben")
	require.NoError(t, err)

	require.Equal(t, []string{"https://a.example.com/users/ben/statuses/1", "https://a.example.com/users/ben/statuses/2"}, featuredPostURLs(client, actor, featuredMaxPosts))
}

// TestFeaturedPostURLs_ReadsAddressesAndPages covers a collection that lists plain addresses across two pages.
func TestFeaturedPostURLs_ReadsAddressesAndPages(t *testing.T) {

	client := newActorClient("https://a.example.com/featured", map[string]map[string]any{
		"https://a.example.com/featured": {
			"id": "https://a.example.com/featured", "type": "OrderedCollection",
			"first": "https://a.example.com/featured?page=1",
		},
		"https://a.example.com/featured?page=1": {
			"id": "https://a.example.com/featured?page=1", "type": "OrderedCollectionPage",
			"orderedItems": []any{"https://a.example.com/p/1", "https://a.example.com/p/2"}, "next": "https://a.example.com/featured?page=2",
		},
		"https://a.example.com/featured?page=2": {
			"id": "https://a.example.com/featured?page=2", "type": "OrderedCollectionPage",
			"orderedItems": []any{"https://a.example.com/p/3"},
		},
	})

	actor, err := client.Load("https://a.example.com/users/ben")
	require.NoError(t, err)

	require.Equal(t, []string{"https://a.example.com/p/1", "https://a.example.com/p/2", "https://a.example.com/p/3"}, featuredPostURLs(client, actor, featuredMaxPosts))
	require.Equal(t, []string{"https://a.example.com/p/1", "https://a.example.com/p/2"}, featuredPostURLs(client, actor, 2), "the limit is honoured")
}

// TestFeaturedPostURLs_YieldsNothingWhenThereIsNothingToRead covers no featured collection, an empty one, and one that cannot be loaded.
func TestFeaturedPostURLs_YieldsNothingWhenThereIsNothingToRead(t *testing.T) {

	none := newActorClient(nil, map[string]map[string]any{})
	actor, err := none.Load("https://a.example.com/users/ben")
	require.NoError(t, err)
	require.Empty(t, featuredPostURLs(none, actor, featuredMaxPosts))

	empty := newActorClient("https://a.example.com/featured", map[string]map[string]any{
		"https://a.example.com/featured": {"id": "https://a.example.com/featured", "type": "OrderedCollection", "totalItems": 0, "orderedItems": []any{}},
	})
	actor, err = empty.Load("https://a.example.com/users/ben")
	require.NoError(t, err)
	require.Empty(t, featuredPostURLs(empty, actor, featuredMaxPosts))

	broken := newActorClient("https://a.example.com/gone", map[string]map[string]any{})
	actor, err = broken.Load("https://a.example.com/users/ben")
	require.NoError(t, err)
	require.Empty(t, featuredPostURLs(broken, actor, featuredMaxPosts), "an unreadable collection is not an error")
}
